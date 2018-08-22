package config

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/prometheus/common/model"
	promcfg "github.com/prometheus/prometheus/config"
	yaml "gopkg.in/yaml.v2"
)

func Encode(rc promcfg.RelabelConfig) string {
	b, err := yaml.Marshal(rc)
	if err != nil {
		log.Fatal(err)
	}

	s := fmt.Sprintf("%s", string(b))
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func ConfigGetRuleFiles() []string {
	return []string{"nope", "not yet"}
}

// HighCardSeries represents a Prometheus series that has been idenitified as
// high cardinality
type HighCardSeries struct {
	MetricName        string
	HighCardLabelName model.LabelName
}

// TODO: Only generate the relabel config for the appropriate job that is spitting out
// the high-cardinality metric
// TODO: Within a job, some series may never be exploding on this label. Consider including
// all relevant labels in source_labels...?
func GenerateMetricRelabelConfig(s HighCardSeries) promcfg.RelabelConfig {
	valueReplace := "bs_silence"
	regexpOriginal := fmt.Sprintf("^%s;.*$", s.MetricName)
	promRegex, err := promcfg.NewRegexp(regexpOriginal)
	if err != nil {
		log.Fatal(err)
	}

	newMetricRelabelConfig := promcfg.RelabelConfig{
		SourceLabels: model.LabelNames{"__name__", s.HighCardLabelName},
		Regex:        promRegex,
		TargetLabel:  string(s.HighCardLabelName),
		Replacement:  valueReplace,
		Action:       "replace",
	}
	return newMetricRelabelConfig
}
