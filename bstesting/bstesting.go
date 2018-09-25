package bstesting

import (
	"testing"

	"github.com/open-fresh/bomb-squad/config"
)

func helperGetConfigFileBytes(t *testing.T, filename string) []byte {
	if filename == "prometheus.yml" {
		return promConfigBytes
	}
	return []byte{}
}
func NewConfigurator(t *testing.T) config.Configurator {
	return &TestConfigurator{
		T: t,
	}
}

type TestConfigurator struct {
	T *testing.T
}

func (c *TestConfigurator) Read() ([]byte, error) {
	return helperGetConfigFileBytes(c.T, "prometheus.yml"), nil
}

func (c *TestConfigurator) Write([]byte) error {
	return nil
}

func (c *TestConfigurator) GetLocation() string {
	return "testLocal"
}
