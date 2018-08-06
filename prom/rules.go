package prom

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"

	configmap "github.com/Fresh-Tracks/bomb-squad/k8s/configmap"
	promcfg "github.com/Fresh-Tracks/bomb-squad/prom/config"
	yaml "gopkg.in/yaml.v2"
)

// GetPrometheusConfig pulls in the full base Prometheus config
// from the provided ConfigMap. Does not include rules nor AM configs.
func GetPrometheusConfig(ctx context.Context, c configmap.ConfigMap) promcfg.Config {
	var cfg promcfg.Config
	raw := c.ReadRawData(ctx, c.Key)

	err := yaml.Unmarshal(raw, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

// GetPrometheusConfigFromDisk pulls the actual currently-rendered config from disk
// so that we can check to see if we're ready for config reloads
func GetPrometheusConfigFromDisk(filename string) promcfg.Config {
	var cfg promcfg.Config
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(raw, &cfg)
	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

// AppendRuleFile Appends a static rule file that Bomb Squad needs into the
// array of rule files that may exist in the current Prometheus config
func AppendRuleFile(ctx context.Context, filename string, c configmap.ConfigMap) error {
	cfg := GetPrometheusConfig(ctx, c)
	configRuleFiles := cfg.RuleFiles
	ruleFileFound := promcfg.RecordingRuleInConfig(cfg, filename)

	if !ruleFileFound {
		newRuleFiles := append(configRuleFiles, filename)
		cfg.RuleFiles = newRuleFiles
		err := c.Update(ctx, cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReloadConfig(client http.Client) error {
	var (
		resp *http.Response
		err  error
	)
	endpt := "http://localhost:9090/-/reload"
	req, _ := http.NewRequest("POST", endpt, nil)

	resp, err = client.Do(req)
	if err != nil {
		log.Println("Error reloading Prometheus config", err)
		return err
	}

	log.Println("Successfully reloaded Prometheus config")
	// defer can't check error states, and GoMetaLinter complains
	_ = resp.Body.Close()

	return nil
}
