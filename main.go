package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/Fresh-Tracks/bomb-squad/configmap"
	"github.com/Fresh-Tracks/bomb-squad/patrol"
	"github.com/Fresh-Tracks/bomb-squad/prom"
	"github.com/Fresh-Tracks/bomb-squad/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version          = "undefined"
	promVersion      = "undefined"
	promRulesVersion = "undefined"
	metricsPort      = flag.Int("metrics-port", 8080, "Port on which to listen for metric scrapes")
	promURL          = flag.String("prom-url", "http://localhost:9090", "Prometheus URL to query")
	cmNamespace      = flag.String("configmap-namespace", "default", "Namespace that holds the Prometheus ConfigMap")
	cmName           = flag.String("configmap-name", "prometheus", "Name of the Prometheus ConfigMap")
	cmKey            = flag.String("configmap-prometheus-key", "prometheus.yml", "The key in the ConfigMap that holds the main Prometheus config")
	getVersion       = flag.Bool("version", false, "return version information and exit")
	versionGauge     = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "ft_agent",
			Subsystem: "version",
			Name:      "details",
			Help:      "Static series that tracks the current schema version in use by the FreshTracks Agent",
			ConstLabels: map[string]string{
				"bomb_squad_version":       version,
				"prometheus_version":       promVersion,
				"prometheus_rules_version": promRulesVersion,
			},
		},
	)
)

func init() {
	prometheus.MustRegister(versionGauge)
}

func bootstrap(ctx context.Context, cm configmap.ConfigMap) {
	//prom.InsertRuleFile("foo.yaml")
	prom.GetPrometheusConfig(ctx, cm)
}

func main() {
	fmt.Println("Welcome to bomb-squad")
	flag.Parse()
	if *getVersion {
		out := ""
		for k, v := range map[string]string{
			"version":          version,
			"prometheus":       promVersion,
			"prometheus-rules": promRulesVersion,
		} {
			out = out + fmt.Sprintf("%s: %s\n", k, v)
		}
		log.Fatal(out)
	}

	log.Println("serving prometheus endpoints on port 8080")

	client, err := util.HttpClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	cm := configmap.ConfigMap{
		Namespace:   *cmNamespace,
		Name:        *cmName,
		Key:         *cmKey,
		LastUpdated: 0,
		Ctx:         ctx,
	}
	cm.Init()

	p := patrol.Patrol{
		PromURL:           *promURL,
		Interval:          15,
		HighCardN:         5,
		HighCardThreshold: 100,
		Client:            client,
		ConfigMap:         &cm,
	}

	bootstrap(ctx, cm)
	go p.Run()

	mux := http.DefaultServeMux
	mux.Handle("/metrics", promhttp.Handler())
	versionGauge.Set(1.0)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *metricsPort),
		Handler: mux,
	}
	log.Fatal(server.ListenAndServe())
}
