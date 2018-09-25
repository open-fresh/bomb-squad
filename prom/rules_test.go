package prom_test

import (
	"testing"

	"github.com/open-fresh/bomb-squad/bstesting"
	"github.com/open-fresh/bomb-squad/config"
	"github.com/open-fresh/bomb-squad/prom"
	"github.com/stretchr/testify/require"
)

func TestCanAppendRulesFile(t *testing.T) {
	c := bstesting.NewConfigurator(t)
	promcfg, err := config.ReadPromConfig(c)
	require.NoError(t, err)
	promcfg, err = prom.AppendRuleFile("/test/rules/file.yaml", c)
	require.Equal(t, "/test/rules/file.yaml", promcfg.RuleFiles[1])
}
