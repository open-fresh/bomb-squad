package config_test

import (
	"testing"

	"github.com/Fresh-Tracks/bomb-squad/bstesting"
	"github.com/Fresh-Tracks/bomb-squad/config"
	"github.com/stretchr/testify/require"
)

func TestCanReadPromConfig(t *testing.T) {
	c := bstesting.NewConfigurator(t)
	promcfg, err := config.ReadPromConfig(c)
	require.NoError(t, err)
	require.NotEmpty(t, promcfg)
}

func TestCanWritePromConfig(t *testing.T) {
	c := bstesting.NewConfigurator(t)
	promcfg, _ := config.ReadPromConfig(c)
	err := config.WritePromConfig(promcfg, c)
	require.NoError(t, err)
}

func TestCanReadBombSquadConfig(t *testing.T) {
	c := bstesting.NewConfigurator(t)
	bscfg, err := config.ReadBombSquadConfig(c)
	require.NoError(t, err)
	require.NotEmpty(t, bscfg)
}

func TestCanWriteBombSquadConfig(t *testing.T) {
	c := bstesting.NewConfigurator(t)
	bscfg, _ := config.ReadBombSquadConfig(c)
	err := config.WriteBombSquadConfig(bscfg, c)
	require.NoError(t, err)
}

func TestCanInsertMetricRelabelConfigToPromConfig(t *testing.T) {
	c := bstesting.NewConfigurator(t)
	hcs := config.HighCardSeries{MetricName: "foo", HighCardLabelName: "bar"}
	mrc, err := config.GenerateMetricRelabelConfig(hcs)
	require.NoError(t, err)
	promcfg, err := config.InsertMetricRelabelConfigToPromConfig(mrc, c)
	require.NoError(t, err)
	insertedMRC := promcfg.ScrapeConfigs[0].MetricRelabelConfigs[0]
	require.Equal(t, "bar", insertedMRC.TargetLabel)
	require.Equal(t, "^(?:^foo;.*$)$", insertedMRC.Regex.String())
}
