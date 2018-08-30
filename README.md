# Bomb Squad: Suppressing Prometheus Cardinality Explosions
[![CircleCI](https://circleci.com/gh/Fresh-Tracks/bomb-squad.svg?style=svg)](https://circleci.com/gh/Fresh-Tracks/bomb-squad)
[![Docker Repository on Quay](https://quay.io/repository/freshtracks.io/bomb-squad/status "Docker Repository on Quay")](https://quay.io/repository/freshtracks.io/bomb-squad)

Bomb Squad is a sidecar to Kubernetes-deployed Prometheus instances that detects and suppresses cardinality explosions. It is a tool intended to bring operational stability and greater visibility in times of rapid cardinality inflation, keeping your Prometheus instances online and usable while providing clear indications that something is trying to blow up.

## Status
Bomb Squad is currently an **alpha** project, with a few caveats of which you should be aware:
* It is currently very Kubernetes-centric in implementation, though not conceptually
* ~~It is currently quite limited in how many Prometheus configurations it can support, as it's non-trivial to vendor Prometheus' `config` package (doing so naively will pull in _all_ of the service discovery vendor code, which hurts).~~
  ~~* For now, only static scrape jobs and Kubernetes service discovery configs are supported~~
  ~~* Any other service discovery configuration will be rendered incorrectly upon writing the configuration back to the ConfigMap~~
* There have been some assumptions made for the sake of solving specific problems, which we intend to refactor properly and make more broadly applicable
* For now, it can only one class of cardinality explosion (exploding label _values_), while there are at least two more classes that we'd love to support:
  * Exploding label _names_
  * Exploding _metric_ names
* PRs and issues are welcome!

## Suppressing the what now?
You might find now and again that one or more of your Prometheus scrape targets begins to expose some manner of super high-cardinality data as metric labels. Prometheus is awesome at handling "typical" high-cardinality behavior:
* Normal pod churn from Kubernetes
* Elastic workloads scaling up and down to handle additional work
* Infrequent and unsustained infrastructure changes (ex. bringing new test clusters online and/or tearing them down)

There are events, however, in which there is dramatic, sustained growth in the cardinality of one or more metrics. We call these events "cardinality explosions", and they can reduce your Prometheus instance(s) and any downstream receiving services to a smoldering heap in very short order.

Some examples of these events include:
* Most commonly (and usually of the greatest magnitude): bad code deploys that stuff high-cardinality data (request IDs, timestamps, user-provided values, etc.) into one or more labels of one or more metrics
* Runaway autoscaling of services that expose a large number of metrics (particularly if the series therein have a lot of variability across many labels)
* Rapid successive container restarts in one or more busy services

Bomb Squad is designed to detect these events and, by way of standard Prometheus capabilities, suppress the negative behavior so that Prometheus can stay online and downstream services can continue to function reliably.

## How does it work?
Bomb Squad is deployed as a sidecar within your Kubernetes Prometheus pods. One this is done, it does the following:
* Bootstraps necessary recording rules into the local Prometheus config
* Monitors the resulting metrics for evidence of cardinality explosions
* When an explosion is detected, inserts "silencing rules" (generated metric\_relabel\_configs) into ALL scrape configs
* Expose metrics related to the exploding metric and label name
* Store silenced `metric.labelName` in Bomb Squad ConfigMap entry
* (TODO) Hot-reloads the Prometheus config
* When the issue causing the explosion has been remediated and code redeployed, allow removal of silencing rules by way of command line tool

## Run Bomb Squad Locally
There is a handy script, `run-local/run-minikube.sh` that will spin up a minikube environment for you that will contain the necessary components to play with and try out Bomb Squad locally.
Steps:
```bash
make clean # just in case the image needs to be rebuilt by the minikube docker engine
cd run-local
./run-minikube.sh
minikube service prometheus
```

This will get things spun up and open a browser window with the stock Prometheus query UI. You can check out the test metric by querying for `statspitter_high_card_test_gauge_vec`. It's also worth taking a look at the bootstrapped recording rules, if you're curious, by visiting the Status -> Rules page.

Before triggering a cardinality explosion, it's recommended that you tail the bomb-squad container's logs. Our preferred method is with `stern`:
```bash
stern . -c bomb-squad
```

To trigger a cardinality explosion and consequently a suppression event by Bomb Squad, run:
```bash
# StatSpitter is a toy app that spits out ~100 new series per second on request
curl -i $(minikube service statspitter --url)/toggle
```

If you watch the bomb-squad container logs, you should see some detection and rule insertion messages go by after a few seconds. Bomb Squad automatically reloads the Prometheus config, so you won't need to take any further action to suppress the explosion!

You can view Bomb Squad's metrics in Prometheus by querying for `bomb_squad_exploding_label_distinct_values`.
You can also view what `metric.label` combinations Bomb Squad is currently silencing by using the CLI in the running container:
```bash
kubectl exec <prometheus_pod_name> -c bomb-squad -- bs list
```

To remediate our simulated "bad code deploy" that caused the explosion, delete the statspitter pod to stop the explosion and dump the old exploded series from its registry:
```bash
kubectl delete pod -l app=statspitter
```

Finally, to remove the silence on our test metric:
```bash
kubectl exec <prometheus_pod_name> -c bomb-squad -- bs unsilence <metric.label as shown by bs list above>
```

## Deploying Bomb Squad
Bomb Squad needs to be deployed as a sidecar container inside your Prometheus pod(s), and there are a couple of requirements to note:
* Bomb Squad should start up after Prometheus to avoid failed API calls while Prometheus initializes
* Bomb Squad needs to mount an `emptyDir` volume so that it has a place from which to bootstrap its rules

A container spec along the lines of the following, added to your Prometheus pod spec, should do the trick:
```bash
spec:
  ...
  template:
    ...
    spec:
    ...
      containers:
        ...
        <prometheus container spec>
        ...
        - name: bomb-squad
          image: gcr.io/freshtracks-io/bomb-squad:latest
          args:
          - -prom-url=localhost:9090 # In case you run Prometheus on a non-standard port
          ports:
          - containerPort: 8080
            protocol: TCP
          volumeMounts:
          - mountPath: /etc/config/bomb-squad
            name: bomb-squad-rules
      volumes:
        ...
        - emptyDir: {}
          name: bomb-squad-rules
```
