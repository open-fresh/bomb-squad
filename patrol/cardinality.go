package patrol

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/deckarep/golang-set"
	"github.com/open-fresh/bomb-squad/config"
	"github.com/open-fresh/bomb-squad/prom"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	yaml "gopkg.in/yaml.v2"
)

var (
	ExplodingLabelGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "bomb_squad",
			Name:      "exploding_label_distinct_values",
			Help:      "Track which series have been identified as having exploding cardinality",
		},
		[]string{"metric_name", "label_name"},
	)
)

// lValues is a simple map that holds all discrete label values for a given
// label within a single metric's collection of series
type labelTracker map[string]mapset.Set

func (p *Patrol) getTopCardinalities() error {
	var highCardSeries []config.HighCardSeries

	relativeURL, err := url.Parse("/api/v1/query")
	if err != nil {
		return fmt.Errorf("failed to parse relative api v1 query path: %s", err)
	}

	query := p.PromURL.Query()
	query.Set("query", fmt.Sprintf("topk(%d,delta(card_count[1m]))", p.HighCardN))
	relativeURL.RawQuery = query.Encode()

	queryURL := p.PromURL.ResolveReference(relativeURL)

	b, err := prom.Fetch(queryURL.String(), p.HTTPClient)
	if err != nil {
		return fmt.Errorf("failed to fetch query from prometheus: %s", err)
	}

	iq := &prom.InstantQuery{}
	err = json.Unmarshal(b, iq)
	if err != nil {
		// Bail here because we're cooked if we can't wrangle Prometheus output
		log.Fatal(err)
	}

	m := p.cardinalityTooHigh(iq)
	if len(m) > 0 {
		highCardSeries = p.findHighCardSeries(m)
	}

	for _, s := range highCardSeries {
		mrc, err := config.GenerateMetricRelabelConfig(s)
		if err != nil {
			log.Printf("Couldn't generate metric relabel config for metric %s: %s\n", s.MetricName, err)
			continue
		}

		err = prom.ReUnmarshal(&mrc)
		if err != nil {
			log.Println(err)
			continue
		}

		newPromConfig, err := config.InsertMetricRelabelConfigToPromConfig(mrc, p.PromConfigurator)
		if err != nil {
			log.Printf("Error inserting relabel config for metric %s: %s\n", s.MetricName, err)
			continue
		}

		newPromConfigBytes, err := yaml.Marshal(newPromConfig)
		if err != nil {
			log.Printf("Error marshalling Prometheus config: %s\n", err)
			continue
		}

		err = p.PromConfigurator.Write(newPromConfigBytes)
		if err != nil {
			log.Printf("Error writing Prometheus config: %s\n", err)
			continue
		}

		err = config.StoreMetricRelabelConfigBombSquad(s, mrc, p.BSConfigurator)
		if err != nil {
			log.Printf("Couldn't store metric relabel config for metric %s: %s\n", s.MetricName, err)
			continue
		}
	}

	return nil
}

func (p *Patrol) cardinalityTooHigh(iq *prom.InstantQuery) []string {
	out := []string{}
	for _, v := range iq.Data.Result {
		m := v.Metric["metric_name"]
		val := v.Value[1].(string)
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			log.Printf("Couldn't parse float64 from '%s': %s\n", val, err)
			continue
		}

		if f >= p.HighCardThreshold {
			out = append(out, m)
		}
	}
	return out
}

func (p *Patrol) getDistinctLabelValuesInSeries(s map[string]string, tracker labelTracker) {
	// Loop through the passed series and loop through the label:value pairs.
	// For each label, ensure we're ready to track discrete values.
	for label, value := range s {
		if _, ok := tracker[label]; !ok {
			tracker[label] = mapset.NewSet()
		}
		tracker[label].Add(value)
	}
}

func (p *Patrol) findHighCardSeries(metrics []string) []config.HighCardSeries {
	hwmLabel := ""
	var (
		s      prom.Series
		hwm, l int
	)
	res := []config.HighCardSeries{}

	for _, metricName := range metrics {

		relativeURL, err := url.Parse("/api/v1/series")
		query := p.PromURL.Query()
		query.Set("match[]", fmt.Sprint(metricName))
		relativeURL.RawQuery = query.Encode()

		queryURL := p.PromURL.ResolveReference(relativeURL)

		b, err := prom.Fetch(queryURL.String(), p.HTTPClient)
		if err != nil {
			// Bail here because we're cooked if we can't reach Prometheus
			log.Fatal(err)
		}

		err = json.Unmarshal(b, &s)
		if err != nil {
			// Bail here because we're cooked if we can't wrangle Prometheus results
			log.Fatal(err)
		}

		tracker := labelTracker{}
		for _, series := range s.Data {
			p.getDistinctLabelValuesInSeries(series, tracker)
		}

		// The label with the highest cardinality should be the exploding one,
		// so we track a high water mark and continue with the "winner"
		hwm = 0
		l = 0
		for label, values := range tracker {
			l = values.Cardinality()
			if l > hwm {
				hwm = l
				hwmLabel = label
			}
		}

		res = append(res,
			config.HighCardSeries{
				MetricName:        metricName,
				HighCardLabelName: model.LabelName(hwmLabel),
			},
		)
		fmt.Printf("Detected exploding label \"%s\" on metric \"%s\"\n", hwmLabel, metricName)
		ExplodingLabelGauge.WithLabelValues(metricName, hwmLabel).Set(float64(hwm))
	}

	return res
}
