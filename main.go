package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/open-fresh/bomb-squad/config"
	configmap "github.com/open-fresh/bomb-squad/k8s/configmap"
	"github.com/open-fresh/bomb-squad/patrol"
	"github.com/open-fresh/bomb-squad/prom"
	"github.com/open-fresh/bomb-squad/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	version            = "undefined"
	promVersion        = "undefined"
	promRulesVersion   = "undefined"
	inK8s              = flag.Bool("k8s", true, "Whether bomb-squad is being deployed in a Kubernetes cluster")
	k8sNamespace       = flag.String("k8s-namespace", "default", "Kubernetes namespace holding Prometheus ConfigMap")
	k8sConfigMapName   = flag.String("k8s-configmap", "prometheus", "Name of the Kubernetes ConfigMap holding Prometheus configuration")
	bsConfigLocation   = flag.String("bs-config-loc", "bomb-squad", "Where the Bomb Squad Config lives. For K8s deployments, this should be the ConfigMap.Data key. Otherwise, full path to file.")
	promConfigLocation = flag.String("prom-config-loc", "prometheus.yml", "Where the Prometheus lives. For K8s deployments, this should be the ConfigMap.Data key. Otherwise, full path to file.")
	metricsPort        = flag.Int("metrics-port", 8080, "Port on which to listen for metric scrapes")
	promURL            = flag.String("prom-url", "http://localhost:9090", "Prometheus URL to query")
	getVersion         = flag.Bool("version", false, "return version information and exit")
	versionGauge       = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "bomb_squad",
			Name:      "details",
			Help:      "Static series that tracks the current versions of all the things in Bomb Squad",
			ConstLabels: map[string]string{
				"version":                  version,
				"prometheus_version":       promVersion,
				"prometheus_rules_version": promRulesVersion,
			},
		},
	)
	k8sClientSet     kubernetes.Interface
	promConfigurator config.Configurator
	bsConfigurator   config.Configurator
)

func init() {
	prometheus.MustRegister(versionGauge)
	prometheus.MustRegister(patrol.ExplodingLabelGauge)
}

func bootstrap(c config.Configurator) {
	// TODO: Don't do this file write if the file already exists, but DO write the file
	// if it's not present on disk but still present in the ConfigMap
	b, err := ioutil.ReadFile("/etc/bomb-squad/rules.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("/etc/config/bomb-squad/rules.yaml", b, 0644)
	if err != nil {
		log.Fatalf("Error writing bootstrap recording rules: %s", err)
	}

	cfg, err := prom.AppendRuleFile("/etc/config/bomb-squad/rules.yaml", c)
	if err != nil {
		log.Fatalf("Error adding bootstrap recording rules to Prometheus config: %s", err)
	}

	err = config.WritePromConfig(cfg, c)
	if err != nil {
		log.Fatalf("Error adding bootstrap recording rules to Prometheus config: %s", err)
	}

}

func main() {
	flag.Parse()
	if *getVersion {
		out := fmt.Sprintf("version: %s\nprometheus: %s\nprometheus-rules: %s\n", version, promVersion, promRulesVersion)
		log.Fatal(out)
	}

	if *inK8s {
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatal(err)
		}

		k8sClientSet, err = kubernetes.NewForConfig(inClusterConfig)
		if err != nil {
			log.Fatal(err)
		}
		cmClient := k8sClientSet.CoreV1().ConfigMaps(*k8sNamespace)
		promConfigurator = configmap.NewConfigMapWrapper(cmClient, *k8sNamespace, *k8sConfigMapName, *promConfigLocation)
		bsConfigurator = configmap.NewConfigMapWrapper(cmClient, *k8sNamespace, *k8sConfigMapName, *bsConfigLocation)
	}

	promurl, err := url.Parse(*promURL)
	if err != nil {
		log.Fatalf("could not parse prometheus url: %s", err)
	}

	httpClient, err := util.HttpClient()
	if err != nil {
		log.Fatalf("could not create http client: %s", err)
	}

	p := patrol.Patrol{
		PromURL:           promurl,
		Interval:          5 * time.Second,
		HighCardN:         5,
		HighCardThreshold: 100,
		HTTPClient:        httpClient,
		PromConfigurator:  promConfigurator,
		BSConfigurator:    bsConfigurator,
	}

	if len(os.Args) > 1 {
		cmd := os.Args[1]
		if cmd == "list" {
			fmt.Println("Suppressed Labels (metricName.labelName):")
			config.ListSuppressedMetrics(p.BSConfigurator)
			os.Exit(0)
		}

		if cmd == "unsilence" {
			label := os.Args[2]
			fmt.Printf("Removing silence rule for suppressed label: %s\n", label)
			err := config.RemoveSilence(label, p.PromConfigurator, p.BSConfigurator)
			if err != nil {
				log.Fatalf("Could not remove silencing rule: %s\n", err)
			}

			os.Exit(0)
		}
	}

	if *inK8s {
		bootstrap(p.PromConfigurator)
	}
	go p.Run()

	mux := http.DefaultServeMux
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/metrics/reset", patrol.MetricResetHandler())
	versionGauge.Set(1.0)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *metricsPort),
		Handler: mux,
	}

	fmt.Println("Welcome to bomb-squad")
	log.Println("serving prometheus endpoints on port 8080")
	log.Fatal(server.ListenAndServe())
}
