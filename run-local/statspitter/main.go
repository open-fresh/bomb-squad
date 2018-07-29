package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	testGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "statspitter",
			Subsystem: "high_card",
			Name:      "test_gauge_vec",
			Help:      "Do irresponsible things with a GuageVec.",
		},
		[]string{
			"seriesType",
			"highCard",
		},
	)

	explodeCardinalty = -1
)

func init() {
	prometheus.MustRegister(testGaugeVec)
}

func run(intervalMs int) {
	tck := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	for _ = range tck.C {
		for i := 1; i <= 10; i++ {
			testGaugeVec.WithLabelValues("stable", fmt.Sprintf("static_%s", strconv.Itoa(i))).Set(0.)
			if explodeCardinalty == 1 {
				for j := 1; j <= 10; j++ {
					testGaugeVec.WithLabelValues("exploding", fmt.Sprintf("boom_%s", time.Now().String())).Set(0.)
				}
			}
		}
		testGaugeVec.WithLabelValues("stable", "I'm_so_stable!").Set(0.)
	}
}

func toggleHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		explodeCardinalty *= -1
	})
}

func main() {
	fmt.Println("statspitter")
	mux := http.DefaultServeMux
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/toggle", toggleHandler())

	go run(1000)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", 8090),
		Handler: mux,
	}
	log.Fatal(server.ListenAndServe())
}
