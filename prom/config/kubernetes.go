package config

// KubernetesSDConfig is the configuration for Kubernetes service discovery.
type KubernetesSDConfig struct {
	APIServer          URL                          `yaml:"api_server,omitempty"`
	Role               string                       `yaml:"role"`
	BasicAuth          *BasicAuth                   `yaml:"basic_auth,omitempty"`
	BearerToken        string                       `yaml:"bearer_token,omitempty"`
	BearerTokenFile    string                       `yaml:"bearer_token_file,omitempty"`
	TLSConfig          TLSConfig                    `yaml:"tls_config,omitempty"`
	NamespaceDiscovery KubernetesNamespaceDiscovery `yaml:"namespaces,omitempty"`
}

// KubernetesNamespaceDiscovery is the configuration for discovering
// Kubernetes namespaces.
type KubernetesNamespaceDiscovery struct {
	Names []string `yaml:"names"`
}
