package patrol

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	configmap "github.com/Fresh-Tracks/bomb-squad/k8s/configmap"
	"github.com/Fresh-Tracks/bomb-squad/prom"
	promcfg "github.com/Fresh-Tracks/bomb-squad/prom/config"
	"github.com/Fresh-Tracks/bomb-squad/util"
	yaml "gopkg.in/yaml.v2"
)

var (
	iq prom.InstantQuery
)

type Patrol struct {
	PromURL           string
	Interval          time.Duration
	HighCardN         int
	HighCardThreshold float64
	Client            *http.Client
	ConfigMap         *configmap.ConfigMap
	PromConfig        *promcfg.Config
	Ctx               context.Context
}

type BombSquadLabelConfig map[string]string

type BombSquadMetricConfig struct {
	SuppressedMetrics map[string]BombSquadLabelConfig
}

func (p *Patrol) Run() {
	ticker := time.NewTicker(time.Duration(p.Interval) * time.Second)
	for _ = range ticker.C {
		err := p.getTopCardinalities()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (p *Patrol) GetBombSquadConfig() BombSquadMetricConfig {
	b := BombSquadMetricConfig{}
	bsConfig := p.ConfigMap.ReadRawData(p.Ctx, "bomb-squad.yaml")
	if len(bsConfig) > 0 {
		err := yaml.Unmarshal(bsConfig, &b)
		if err != nil {
			log.Fatal(err)
		}
	}

	if b.SuppressedMetrics == nil {
		b.SuppressedMetrics = map[string]BombSquadLabelConfig{}
	}

	return b
}

func (p *Patrol) ListSuppressedMetrics() {
	b := p.GetBombSquadConfig()
	for metric, labels := range b.SuppressedMetrics {
		for label, _ := range labels {
			fmt.Printf("%s.%s\n", metric, label)
		}
	}
}

func (p *Patrol) RemoveSilence(label string) error {
	promConfig := promcfg.Config{}
	promBytes := p.ConfigMap.ReadRawData(p.Ctx, p.ConfigMap.Key)
	err := yaml.Unmarshal(promBytes, &promConfig)
	if err != nil {
		log.Fatal(err)
	}

	ml := strings.Split(label, ".")
	metricName, labelName := ml[0], ml[1]

	bsCfg := p.GetBombSquadConfig()
	bsRelabelConfigEncoded := bsCfg.SuppressedMetrics[metricName][labelName]

	for _, scrapeConfig := range promConfig.ScrapeConfigs {
		i := p.FindRelabelConfigInScrapeConfig(bsRelabelConfigEncoded, *scrapeConfig)
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

	bsCfgBytes, err := yaml.Marshal(bsCfg)
	if err != nil {
		log.Fatal(err)
	}

	promConfigBytes, err := yaml.Marshal(promConfig)
	if err != nil {
		log.Fatal(err)
	}

	p.ConfigMap.CM.Data["bomb-squad.yaml"] = string(bsCfgBytes)
	p.ConfigMap.CM.Data[p.ConfigMap.Key] = string(promConfigBytes)
	p.ConfigMap.UpdateWithRetries(5)

	resetMetric(metricName, labelName)

	return nil
}

func resetMetric(metricName, labelName string) {
	client, _ := util.HttpClient()
	endpt := fmt.Sprintf("http://localhost:8080/metrics/reset?metric=%s&label=%s", metricName, labelName)
	req, _ := http.NewRequest("GET", endpt, nil)

	_, err := client.Do(req)
	if err != nil {
		log.Println("Failed to reset metric for %s.%s. Not urgent - continuing.", err)
	}
}

func (p *Patrol) StoreMetricRelabelConfigBombSquad(s promcfg.HighCardSeries, mrc promcfg.RelabelConfig) {
	b := p.GetBombSquadConfig()
	if lc, ok := b.SuppressedMetrics[s.MetricName]; ok {
		lc[s.HighCardLabelName] = mrc.Encode()
	} else {
		b.SuppressedMetrics[s.MetricName] = lc
		lc = BombSquadLabelConfig{}
		lc[s.HighCardLabelName] = mrc.Encode()
	}

	res, err := yaml.Marshal(b)
	if err != nil {
		log.Fatal(err)
	}

	p.ConfigMap.CM.Data["bomb-squad.yaml"] = string(res)
	err = p.ConfigMap.UpdateWithRetries(5)
	if err != nil {
		log.Fatal(err)
	}
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

func (p *Patrol) FindRelabelConfigInScrapeConfig(encodedRule string, scrapeConfig promcfg.ScrapeConfig) int {
	for i, relabelConfig := range scrapeConfig.MetricRelabelConfigs {
		if relabelConfig.Encode() == encodedRule {
			return i
		}
	}

	return -1
}

func (p *Patrol) InsertMetricRelabelConfigToPromConfig(rc promcfg.RelabelConfig) promcfg.Config {
	promConfig := prom.GetPrometheusConfig(p.Ctx, *p.ConfigMap)
	rcEncoded := rc.Encode()
	for _, scrapeConfig := range promConfig.ScrapeConfigs {
		if p.FindRelabelConfigInScrapeConfig(rcEncoded, *scrapeConfig) == -1 {
			fmt.Printf("Did not find necessary silence rule in ScrapeConfig %s, adding now\n", scrapeConfig.JobName)
			scrapeConfig.MetricRelabelConfigs = append(scrapeConfig.MetricRelabelConfigs, &rc)
		}
	}
	return promConfig
}

func MetricResetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		metricName := req.URL.Query().Get("metric")
		labelName := req.URL.Query().Get("label")
		fmt.Printf("Resetting metrics for %s.%s\n", metricName, labelName)

		ExplodingLabelGauge.WithLabelValues(metricName, labelName).Set(float64(0.))
	})
}
