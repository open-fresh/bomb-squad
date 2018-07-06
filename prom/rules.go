package prom

import (
	"context"
	"fmt"
	"log"

	"github.com/Fresh-Tracks/bomb-squad/configmap"
	promcfg "github.com/Fresh-Tracks/bomb-squad/prom/config"
	yaml "gopkg.in/yaml.v2"
)

func GetPrometheusConfig(ctx context.Context, cm configmap.ConfigMap) {
	var cfg promcfg.Config
	raw := cm.ReadRawData(ctx, cm.Key)
	err := yaml.Unmarshal(raw, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("UNMARSHALED PROM CFG:\n%+v\n", cfg)
	fmt.Printf("\nSCRAPE CONFIGS:\n%s\n", cfg.ScrapeConfigs)

	// Re-marshal to see what we get
	b, err := yaml.Marshal(cfg)
	fmt.Printf("\nREMARSHALLED CONFIG:\n%s\n", b)
}

func InsertRuleFile(ctx context.Context, filename string, cm configmap.ConfigMap) {
	//ruleFiles := ConfigGetRuleFiles()
	//fmt.Printf("RULE FILES:\n%s\n", ruleFiles)
}
