package bstesting

var (
	promConfigBytes = []byte(`
remote_write: []
rule_files:
- /etc/config/rules.yml
global:
  scrape_interval: 3s
  scrape_timeout: 3s
  evaluation_interval: 5s
scrape_configs:
- job_name: prometheus
  scrape_interval: 3s
  scrape_timeout: 3s
  metrics_path: /metrics
  scheme: http
  static_configs:
  - targets:
    - localhost:9090
- job_name: bomb-squad
  scrape_interval: 3s
  scrape_timeout: 3s
  metrics_path: /metrics
  scheme: http
  static_configs:
  - targets:
    - localhost:8080
- job_name: kubernetes-apiservers
  scrape_interval: 3s
  scrape_timeout: 3s
  metrics_path: /metrics
  scheme: https
  kubernetes_sd_configs:
  - api_server: null
    role: endpoints
    namespaces:
      names: []
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true
  relabel_configs:
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
    separator: ;
    regex: default;kubernetes;https
    replacement: $1
    action: keep
- job_name: 'ft-kubernetes-pods'
  kubernetes_sd_configs:
    - role: pod
  relabel_configs:
    - source_labels: [__meta_kubernetes_pod_annotation_freshtracks_io_scrape]
      action: keep
      regex: true
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
      action: replace
      target_label: __metrics_path__
      regex: (.+)
    - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
      action: replace
      regex: (.+):(?:\d+);(\d+)
      replacement: ${1}:${2}
      target_label: __address__
`)
)
