package configmap

import (
	"context"
	"fmt"
	"log"
	"time"

	promcfg "github.com/Fresh-Tracks/bomb-squad/prom/config"
	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	yaml "gopkg.in/yaml.v2"
)

// ConfigMap Struct to hold relevant details of a ConfigMap
type ConfigMap struct {
	Client      *k8s.Client
	Name        string
	CM          *corev1.ConfigMap
	Key         string
	LastUpdated time.Duration
	Ctx         context.Context
}

// Init just tries to get a K8s client created, and if it can't, bail
func (c *ConfigMap) Init(ctx context.Context) {
	cm := corev1.ConfigMap{}
	// NewInClusterClient() creates a client that is forced into the same namespace
	// as the entity in which the Client is created. So as long as bomb-squad runs
	// in a container that resides in the same namespace as the Prometheus ConfigMap,
	// we're good
	client, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal(err)
	}
	c.Client = client

	err = c.Client.Get(ctx, c.Client.Namespace, c.Name, &cm)
	if err != nil {
		log.Fatal(err)
	}
	c.CM = &cm
}

// ReadRawData pulls in value from the `data` key in the ConfigMap as-is
func (c *ConfigMap) ReadRawData(ctx context.Context, key string) []byte {
	cm := corev1.ConfigMap{}
	err := c.Client.Get(ctx, c.Client.Namespace, c.Name, &cm)
	if err != nil {
		log.Fatal(err)
	}
	c.CM = &cm

	return []byte(c.CM.Data[key])
}

func (c *ConfigMap) Update(ctx context.Context, cfg promcfg.Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	c.CM.Data[c.Key] = string(b)
	if err := c.Client.Update(ctx, c.CM); err != nil {
		return err
	}

	fmt.Println("Successfully updated ConfigMap")
	return nil
}
