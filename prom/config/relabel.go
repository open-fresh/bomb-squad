package config

import (
	"fmt"
	"regexp"
)

// HighCardSeries represents a Prometheus series that has been idenitified as
// high cardinality
type HighCardSeries struct {
	MetricName        string
	HighCardLabelName string
}

// TODO: Only generate the relabel config for the appropriate job that is spitting out
// the high-cardinality metric
// TODO: Within a job, some series may never be exploding on this label. Consider including
// all relevant labels in source_labels...?
func GenerateMetricRelabelConfig(s HighCardSeries) RelabelConfig {
	valueReplace := "bs_silence"
	regexpOriginal := fmt.Sprintf("^%s;.*$", s.MetricName)
	regex := regexp.MustCompile(regexpOriginal)
	promRegex := Regexp{regex, regexpOriginal}
	newMetricRelabelConfig := RelabelConfig{
		SourceLabels: []string{"__name__", s.HighCardLabelName},
		Regex:        promRegex,
		TargetLabel:  s.HighCardLabelName,
		Replacement:  valueReplace,
		Action:       "replace",
	}
	return newMetricRelabelConfig
}
