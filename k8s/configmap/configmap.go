package configmap

import (
	"context"
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

// By most accounts, this type of error can be ignored. Let's hope that's true.
const optimisticMergeConflictError = "kubernetes api: Failure 409 Operation cannot be fulfilled on configmaps \"prometheus\": the object has been modified; please apply your changes to the latest version and try again"

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

	if res, ok := c.CM.Data[key]; !ok {
		return []byte{}
	} else {
		return []byte(res)
	}
}

func (c *ConfigMap) Update(ctx context.Context, cfg promcfg.Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	c.CM.Data[c.Key] = string(b)
	err = c.UpdateWithRetries(5)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (c *ConfigMap) UpdateWithRetries(retries int) error {
	var err error
	var newCM corev1.ConfigMap
	err = nil

	log.Println("Updating ConfigMap")
	for tries := 1; tries <= retries; tries++ {
		err = c.Client.Update(c.Ctx, c.CM)
		if err != nil && err.Error() == optimisticMergeConflictError {
			log.Println("Retrying ConfigMap Update")
			log.Println("Error type is optimisticMergeConflictError")
			err := c.Client.Get(c.Ctx, c.Client.Namespace, c.Name, &newCM)
			if err != nil {
				return err
			}

			newResourceVersion := newCM.Metadata.GetResourceVersion()
			log.Printf("newResourceVersion: %s\n", newResourceVersion)
			err = c.Client.Update(c.Ctx, c.CM, k8s.ResourceVersion(newResourceVersion))
			if err != nil {
				log.Fatal(err)
			}
		} else if err != nil {
			log.Println("Error type is NOT optimisticMergeConflictError")
			return err
		} else {
			return nil
		}
	}
	return err
}
