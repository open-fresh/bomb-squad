package configmap

import (
	"context"
	"log"
	"time"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
)

// ConfigMap Struct to hold relevant details of a ConfigMap
type ConfigMap struct {
	Client      *k8s.Client
	Namespace   string
	Name        string
	Key         string
	LastUpdated time.Duration
	Ctx         context.Context
}

// Init just tries to get a K8s client created, and if it can't, bail
func (c *ConfigMap) Init() {
	client, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal(err)
	}
	c.Client = client
	_ = c.ReadRawData(c.Ctx, c.Key)
}

// ReadRawData pulls in value from the `data` key in the ConfigMap as-is
func (c *ConfigMap) ReadRawData(ctx context.Context, key string) []byte {
	var configMap corev1.ConfigMap
	err := c.Client.Get(ctx, c.Namespace, c.Name, &configMap)
	if err != nil {
		log.Fatal(err)
	}
	b := []byte(configMap.Data[key])
	return b
}
