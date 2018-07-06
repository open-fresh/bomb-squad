package config

type ServiceDiscoveryConfig struct {
	// List of labeled target groups for this job.
	StaticConfigs []*TargetGroup `yaml:"static_configs,omitempty"`
	// List of Kubernetes service discovery configurations.
	KubernetesSDConfigs []*KubernetesSDConfig `yaml:"kubernetes_sd_configs,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *ServiceDiscoveryConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ServiceDiscoveryConfig
	return unmarshal((*plain)(c))
}
