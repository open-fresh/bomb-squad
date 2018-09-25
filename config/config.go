package config

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/open-fresh/bomb-squad/util"
	"github.com/prometheus/common/model"
	promcfg "github.com/prometheus/prometheus/config"
	yaml "gopkg.in/yaml.v2"
)

type Configurator interface {
	Read() ([]byte, error)
	Write([]byte) error
	GetLocation() string
}

type BombSquadLabelConfig map[string]string

type BombSquadConfig struct {
	SuppressedMetrics map[string]BombSquadLabelConfig
}

func ReadBombSquadConfig(c Configurator) (BombSquadConfig, error) {
	b, err := c.Read()
	if err != nil {
		return BombSquadConfig{}, fmt.Errorf("Failed to read Bomb Squad config: %s", err)
	}

	bscfg := BombSquadConfig{}
	err = yaml.Unmarshal(b, &bscfg)
	if err != nil {
		return BombSquadConfig{}, fmt.Errorf("Couldn't unmarshal into config.BombSquadConfig: %s", err)
	}
	if bscfg.SuppressedMetrics == nil {
		bscfg.SuppressedMetrics = map[string]BombSquadLabelConfig{}
	}

	return bscfg, nil
}

func ReadPromConfig(c Configurator) (promcfg.Config, error) {
	b, err := c.Read()
	if err != nil {
		return promcfg.Config{}, fmt.Errorf("Failed to read Prometheus config: %s", err)
	}

	pcfg := promcfg.Config{}
	err = yaml.Unmarshal(b, &pcfg)
	if err != nil {
		return promcfg.Config{}, fmt.Errorf("Couldn't unmarshal into prometheus.Config: %s", err)
	}
	return pcfg, nil
}

func WriteBombSquadConfig(bscfg BombSquadConfig, c Configurator) error {
	b, err := yaml.Marshal(bscfg)
	if err != nil {
		log.Printf("Failed to write Bomb Squad config: %s\n", err)
		return err
	}
	return c.Write(b)
}

func WritePromConfig(pcfg promcfg.Config, c Configurator) error {
	b, err := yaml.Marshal(pcfg)
	if err != nil {
		log.Printf("Failed to write Prometheus config: %s\n", err)
		return err
	}
	return c.Write(b)
}

func ListSuppressedMetrics(c Configurator) {
	b, err := ReadBombSquadConfig(c)
	if err != nil {
		log.Fatalf("Couldn't list suppressed metrics: %s\n", err)
	}

	for metric, labels := range b.SuppressedMetrics {
		for label := range labels {
			fmt.Printf("%s.%s\n", metric, label)
		}
	}
}

func RemoveSilence(label string, pc, bc Configurator) error {
	promConfig, err := ReadPromConfig(pc)
	if err != nil {
		return err
	}

	ml := strings.Split(label, ".")
	metricName, labelName := ml[0], ml[1]

	bsCfg, err := ReadBombSquadConfig(bc)
	if err != nil {
		return err
	}

	bsRelabelConfigEncoded := bsCfg.SuppressedMetrics[metricName][labelName]

	for _, scrapeConfig := range promConfig.ScrapeConfigs {
		i := FindRelabelConfigInScrapeConfig(bsRelabelConfigEncoded, *scrapeConfig)
		if i >= 0 {
			scrapeConfig.MetricRelabelConfigs = DeleteRelabelConfigFromArray(scrapeConfig.MetricRelabelConfigs, i)
			fmt.Printf("Deleted silence rule from ScrapeConfig %s\n", scrapeConfig.JobName)
		}
	}

	if len(bsCfg.SuppressedMetrics[metricName]) == 1 {
		delete(bsCfg.SuppressedMetrics, metricName)
	} else {
		delete(bsCfg.SuppressedMetrics[metricName], labelName)
	}

	err = WriteBombSquadConfig(bsCfg, bc)
	if err != nil {
		return err
	}

	err = WritePromConfig(promConfig, pc)
	if err != nil {
		return err
	}

	resetMetric(metricName, labelName)

	return nil
}

func StoreMetricRelabelConfigBombSquad(s HighCardSeries, mrc promcfg.RelabelConfig, c Configurator) error {
	b, err := ReadBombSquadConfig(c)
	if err != nil {
		return err
	}

	if lc, ok := b.SuppressedMetrics[s.MetricName]; ok {
		lc[string(s.HighCardLabelName)] = encode(mrc)
	} else {
		b.SuppressedMetrics[s.MetricName] = lc
		lc = BombSquadLabelConfig{}
		lc[string(s.HighCardLabelName)] = encode(mrc)
	}

	err = WriteBombSquadConfig(b, c)
	if err != nil {
		log.Fatalf("Failed to write BombSquadConfig: %s\n", err)
	}

	return nil
}

func DeleteRelabelConfigFromArray(arr []*promcfg.RelabelConfig, index int) []*promcfg.RelabelConfig {
	res := []*promcfg.RelabelConfig{}
	if len(arr) > 1 {
		res = append(arr[:index], arr[index+1:]...)
	} else {
		res = []*promcfg.RelabelConfig{}
	}
	return res
}

func FindRelabelConfigInScrapeConfig(encodedRule string, scrapeConfig promcfg.ScrapeConfig) int {
	for i, relabelConfig := range scrapeConfig.MetricRelabelConfigs {
		if encode(*relabelConfig) == encodedRule {
			return i
		}
	}

	return -1
}

func InsertMetricRelabelConfigToPromConfig(rc promcfg.RelabelConfig, c Configurator) (promcfg.Config, error) {
	promConfig, err := ReadPromConfig(c)
	if err != nil {
		return promcfg.Config{}, err
	}

	rcEncoded := encode(rc)
	for _, scrapeConfig := range promConfig.ScrapeConfigs {
		if FindRelabelConfigInScrapeConfig(rcEncoded, *scrapeConfig) == -1 {
			fmt.Printf("Did not find necessary silence rule in ScrapeConfig %s, adding now\n", scrapeConfig.JobName)
			scrapeConfig.MetricRelabelConfigs = append(scrapeConfig.MetricRelabelConfigs, &rc)
		}
	}
	return promConfig, nil
}

func encode(rc promcfg.RelabelConfig) string {
	b, err := yaml.Marshal(rc)
	if err != nil {
		// Bail here, because there's no point continuing anything else if we can't encode to a string
		log.Fatalf("Failed to encode relabel config: %s\n", err)
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
func GenerateMetricRelabelConfig(s HighCardSeries) (promcfg.RelabelConfig, error) {
	valueReplace := "bs_silence"
	regexpOriginal := fmt.Sprintf("^%s;.*$", s.MetricName)
	promRegex, err := promcfg.NewRegexp(regexpOriginal)
	if err != nil {
		return promcfg.RelabelConfig{}, fmt.Errorf("Couldn't create promcfg.Regexp from '%s': %s", regexpOriginal, err)
	}

	newMetricRelabelConfig := promcfg.RelabelConfig{
		SourceLabels: model.LabelNames{"__name__", s.HighCardLabelName},
		Regex:        promRegex,
		TargetLabel:  string(s.HighCardLabelName),
		Replacement:  valueReplace,
		Action:       "replace",
	}
	return newMetricRelabelConfig, nil
}

func resetMetric(metricName, labelName string) {
	client, _ := util.HttpClient()
	// TODO This is a hack currently, to allow the CLI invocation of `unsilence` to actually get to the metric
	// that needs reset. Should not assume that CLI will be invoked from the running instance, and make this
	// configurable
	endpt := fmt.Sprintf("http://localhost:8080/metrics/reset?metric=%s&label=%s", metricName, labelName)
	req, _ := http.NewRequest("GET", endpt, nil)

	_, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to reset metric for %s.%s: %s. Not urgent - continuing.", metricName, labelName, err)
	}
}
