package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	configmap "github.com/Fresh-Tracks/bomb-squad/k8s/configmap"
	"github.com/Fresh-Tracks/bomb-squad/patrol"
	"github.com/Fresh-Tracks/bomb-squad/prom"
	promcfg "github.com/Fresh-Tracks/bomb-squad/prom/config"
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
	promConfig       = flag.String("prom-config", "/etc/config/prometheus.yml", "Full path to Prometheus config file")
	cmName           = flag.String("configmap-name", "prometheus", "Name of the Prometheus ConfigMap")
	cmKey            = flag.String("configmap-prometheus-key", "prometheus.yml", "The key in the ConfigMap that holds the main Prometheus config")
	getVersion       = flag.Bool("version", false, "return version information and exit")
	versionGauge     = prometheus.NewGauge(
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
)

func init() {
	prometheus.MustRegister(versionGauge)
	prometheus.MustRegister(patrol.ExplodingLabelGauge)
}

func bootstrap(ctx context.Context, c configmap.ConfigMap, p patrol.Patrol) {
	var (
		cfg promcfg.Config
	)
	log.Println("Bootstrapping...")
	// TODO: Don't do this file write if the file already exists, but DO write the file
	// if it's not present on disk but still present in the ConfigMap
	b, err := ioutil.ReadFile("/etc/bomb-squad/rules.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("/etc/config/bomb-squad/rules.yaml", b, 0644)

	prom.AppendRuleFile(ctx, "/etc/config/bomb-squad/rules.yaml", c)

	for i := 0; i < 12; i++ {
		cfg = prom.GetPrometheusConfigFromDisk(*promConfig)
		if promcfg.RecordingRuleInConfig(cfg, "/etc/config/bomb-squad/rules.yaml") {
			err := prom.ReloadConfig(*p.Client)
			if err != nil {
				log.Fatal(err)
			}
			break
		}
		time.Sleep(1 * time.Second)
	}
	log.Println("Successfully bootstrapped")

	// Sleep for a moment so that Prometheus' own metrics don't
	// get flagged as exploding
	log.Println("Global grace period start")
	//time.Sleep(30 * time.Second)
	log.Println("Global grace period end")
}

func main() {
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

	client, err := util.HttpClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	cm := configmap.ConfigMap{
		Name:        *cmName,
		Key:         *cmKey,
		LastUpdated: 0,
		Ctx:         ctx,
	}
	cm.Init(ctx)

	p := patrol.Patrol{
		PromURL:           *promURL,
		Interval:          5,
		HighCardN:         5,
		HighCardThreshold: 100,
		Client:            client,
		ConfigMap:         &cm,
		Ctx:               ctx,
	}

	cmd := os.Args[1]
	if cmd == "list" {
		fmt.Println("Suppressed Labels (metricName.labelName):")
		p.ListSuppressedMetrics()
		os.Exit(0)
	}

	if cmd == "unsilence" {
		label := os.Args[2]
		fmt.Printf("Removing silence rule for suppressed label: %s\n", label)
		p.RemoveSilence(label)
		os.Exit(0)
	}

	log.Println("Welcome to bomb-squad")
	bootstrap(ctx, cm, p)
	go p.Run(*promConfig)

	mux := http.DefaultServeMux
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/metrics/reset", patrol.MetricResetHandler())
	versionGauge.Set(1.0)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *metricsPort),
		Handler: mux,
	}

	log.Println("Serving metrics on port 8080")
	log.Fatal(server.ListenAndServe())
}
