package patrol

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/Fresh-Tracks/bomb-squad/prom"
	promcfg "github.com/Fresh-Tracks/bomb-squad/prom/config"
	"github.com/deckarep/golang-set"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
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
	var highCardSeries []promcfg.HighCardSeries

	relativeURL, err := url.Parse("/api/v1/query")
	if err != nil {
		return fmt.Errorf("failed to parse relative api v1 query path: %s", err)
	}

	query := p.PromURL.Query()
	query.Set("query", fmt.Sprintf("topk(%d,delta(card_count[1m]))", p.HighCardN))
	relativeURL.RawQuery = query.Encode()

	queryURL := p.PromURL.ResolveReference(relativeURL)

	b, err := prom.Fetch(queryURL.String(), p.Client)
	if err != nil {
		return fmt.Errorf("failed to fetch query from prometheus: %s", err)
	}

	iq := &prom.InstantQuery{}
	err = json.Unmarshal(b, iq)
	if err != nil {
		log.Fatal(err)
	}

	m := p.cardinalityTooHigh(iq)
	if len(m) > 0 {
		highCardSeries = p.findHighCardSeries(m)
	}

	for _, s := range highCardSeries {
		mrc := promcfg.GenerateMetricRelabelConfig(s)

		newPromConfig := p.InsertMetricRelabelConfigToPromConfig(mrc)

		err := p.ConfigMap.Update(p.Ctx, newPromConfig)
		if err != nil {
			log.Fatal(err)
		}

		p.StoreMetricRelabelConfigBombSquad(s, mrc)
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
			log.Fatal(err)
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

func (p *Patrol) tryToFindStableValues(metric, label string, currentSet mapset.Set) mapset.Set {
	var s prom.Series
	earlierSet := mapset.NewSet()
	end := time.Now().Unix() - 30
	start := end - 600
	attempts := 0
	maxAttempts := 100
	diff := currentSet.Difference(earlierSet).Cardinality()
	fmt.Println("Trying to find stable series...")

	for attempts < maxAttempts && diff > 0 {
		attempts++

		earlierSet = mapset.NewSet()

		end = start + 570
		start = end - 600

		urlString := fmt.Sprintf("http://%s/api/v1/series?match[]=%s&start=%d&end=%d", p.PromURL, metric, start, end)

		b, err := prom.Fetch(urlString, p.Client)
		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal(b, &s)
		if err != nil {
			log.Fatal(err)
		}

		for _, series := range s.Data {
			earlierSet.Add(series[label])
		}

		diff = currentSet.Difference(earlierSet).Cardinality()

		currentSet = earlierSet
	}

	if diff == 0 {
		fmt.Printf("All done! Found stable series:\n%s\nTook %d attempts\n", earlierSet.String(), attempts)
	} else {
		fmt.Printf("Didn't make it after %d attempts.\n", attempts)
	}
	return earlierSet
}

func (p *Patrol) findHighCardSeries(metrics []string) []promcfg.HighCardSeries {
	hwmLabel := ""
	var (
		s      prom.Series
		b      []byte
		hwm, l int
		err    error
	)
	res := []promcfg.HighCardSeries{}

	for _, metricName := range metrics {
		urlString := fmt.Sprintf("http://%s/api/v1/series?match[]=%s", p.PromURL, metricName)

		b, err = prom.Fetch(urlString, p.Client)
		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal(b, &s)
		if err != nil {
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
			promcfg.HighCardSeries{
				MetricName:        metricName,
				HighCardLabelName: model.LabelName(hwmLabel),
			},
		)
		fmt.Printf("Detected exploding label \"%s\" on metric \"%s\"\n", hwmLabel, metricName)
		ExplodingLabelGauge.WithLabelValues(metricName, hwmLabel).Set(float64(hwm))
	}

	return res
}
