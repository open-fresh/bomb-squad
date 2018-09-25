package patrol

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/open-fresh/bomb-squad/config"
	"github.com/open-fresh/bomb-squad/prom"
)

var (
	iq prom.InstantQuery
)

type Patrol struct {
	PromURL           *url.URL
	Interval          time.Duration
	HighCardN         int
	HighCardThreshold float64
	HTTPClient        *http.Client
	PromConfigurator  config.Configurator
	BSConfigurator    config.Configurator
}

func (p *Patrol) Run() {
	ticker := time.NewTicker(p.Interval)
	for range ticker.C {
		err := p.getTopCardinalities()
		if err != nil {
			log.Fatalf("Couldn't retrieve top cardinalities: %s\n", err)
		}
	}
}

func MetricResetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		metricName := req.URL.Query().Get("metric")
		labelName := req.URL.Query().Get("label")
		fmt.Printf("Resetting metrics for %s.%s\n", metricName, labelName)

		ExplodingLabelGauge.WithLabelValues(metricName, labelName).Set(float64(0.))
	})
}
