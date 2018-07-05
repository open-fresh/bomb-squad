# StatSpitter

## Wat?
This is a tiny binary Chowny threw together to simulate a high-cardinality event in Prometheus.
It's extremely simple, but could be expanded as you please.

## How to Run it
### Build
```
GOOS=linux GOARCH=amd64 go build
```

### Deploy to minikube
```
minikube start --cpus 4 --memory 8192
cd ksonnet
ks apply --insecure-skip-tls-verify local
```

If that doesn't work, it's most likely because your minikube context differs from mine.
Tweak ksonnet's `app.yaml` and it should work.

### Usage
Once it starts, `statspitter` will simply expose a single gauge metric named `statspitter_high_card_test_gauge_vec`.

There will be one "nice" series that kicks out a normal set of data points:
```
statspitter_high_card_test_gauge_vec{type="stable", value="foo"}
```
This acts as a control of sorts.

There will also be a set of series that are created at a rate of approximately 10/s:
```
statspitter_high_card_test_gauge_vec{type="volatile", value=<time.Now()>}
```

All series are perpetually set to 0.0, as values don't matter here.

If you want to turn off the explosion of series, simply run the following:
```
curl $(mk service ss --url)/toggle
```

As the endpoint implies, GET'ing this URL again will re-enabel series explosion.
