// Package monitoring generates monitoring-stack configuration files for the
// nSelf CLI. Prometheus, Loki, Promtail, and Alertmanager configs are all
// built here and written into the target project's `monitoring/` directory by
// `nself build` when MONITORING_ENABLED=true.
//
// This package owns *config generation*, not the docker-compose service
// definitions (those live in docker-compose.monitoring.yml — or, soon, the
// nself-monitoring free plugin).
package monitoring

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"
)

// ScrapeTarget describes one plugin or service that Prometheus should scrape.
// Every nSelf plugin exposes /metrics on its HTTP port; admins may add custom
// services with the same contract via ScrapeTargets in the monitoring config.
type ScrapeTarget struct {
	// JobName uniquely identifies the target in Prometheus. Use the plugin
	// name (e.g. "ai", "mux") or the service name (e.g. "postgres", "nginx").
	JobName string

	// ServiceName is the docker-compose service name resolved via the Docker
	// network DNS. Prometheus scrapes <ServiceName>:<Port><Path>.
	ServiceName string

	// Port is the plugin's HTTP port as declared in plugin.json.
	Port int

	// Path is the metrics endpoint. Defaults to /metrics if empty.
	Path string

	// Interval overrides the global scrape_interval when set. Empty = global.
	Interval string

	// Labels are additional static labels attached to every scraped series.
	// Common use: {"plugin": "ai", "tier": "pro"}.
	Labels map[string]string

	// BearerToken, when set, is sent as an Authorization: Bearer header.
	// Used by Hasura /v1/metrics which requires HASURA_GRAPHQL_METRICS_SECRET.
	BearerToken string
}

// PrometheusConfig captures everything needed to render prometheus.yml.
type PrometheusConfig struct {
	// ScrapeInterval is the global default (e.g. "15s").
	ScrapeInterval string
	// EvaluationInterval is how often Prometheus evaluates alert rules.
	EvaluationInterval string
	// ExternalLabels apply to every time series (useful for multi-cluster).
	ExternalLabels map[string]string
	// AlertmanagerURL is the Alertmanager host:port; empty disables alerting.
	AlertmanagerURL string
	// RuleFiles are paths (inside the container) to load alert rules from.
	RuleFiles []string
	// Targets is the full list of scrape targets.
	Targets []ScrapeTarget
}

// Defaults returns a PrometheusConfig with nSelf's out-of-the-box settings.
// Callers append their own targets via `cfg.Targets = append(...)` before
// rendering.
func Defaults() *PrometheusConfig {
	return &PrometheusConfig{
		ScrapeInterval:     "15s",
		EvaluationInterval: "15s",
		ExternalLabels:     map[string]string{"cluster": "nself"},
		AlertmanagerURL:    "alertmanager:9093",
		RuleFiles: []string{
			"/etc/prometheus/alerts.yml",
			"/etc/prometheus/rules/otel-alerts.yml",
			"/etc/prometheus/rules/infra-alerts.yml",
		},
		Targets:            BuiltinTargets(),
	}
}

// BuiltinTargets returns the monitoring targets that ship with every nSelf
// stack: Prometheus itself, node-exporter, cadvisor, postgres-exporter, plus
// Hasura /v1/metrics, MinIO cluster metrics, and Auth /metrics.
// Plugin targets are appended by the caller.
func BuiltinTargets() []ScrapeTarget {
	return []ScrapeTarget{
		{JobName: "prometheus", ServiceName: "prometheus", Port: 9090, Path: "/metrics"},
		{JobName: "node", ServiceName: "node-exporter", Port: 9100, Path: "/metrics"},
		{JobName: "cadvisor", ServiceName: "cadvisor", Port: 8080, Path: "/metrics"},
		{JobName: "postgres", ServiceName: "postgres-exporter", Port: 9187, Path: "/metrics"},
		{JobName: "redis", ServiceName: "redis-exporter", Port: 9121, Path: "/metrics"},
		{JobName: "nginx", ServiceName: "nginx", Port: 9113, Path: "/metrics"},
		// Hasura GraphQL Engine Prometheus metrics endpoint.
		// Requires HASURA_GRAPHQL_METRICS_SECRET set in the hasura service env.
		// The bearer token is injected at build time from that env var.
		{
			JobName:     "hasura",
			ServiceName: "hasura",
			Port:        8080,
			Path:        "/v1/metrics",
			Interval:    "30s",
			Labels:      map[string]string{"service": "hasura"},
			BearerToken: "${HASURA_GRAPHQL_METRICS_SECRET}",
		},
		// MinIO S3-compatible storage cluster metrics.
		{
			JobName:     "minio",
			ServiceName: "minio",
			Port:        9000,
			Path:        "/minio/v2/metrics/cluster",
			Interval:    "30s",
			Labels:      map[string]string{"service": "minio"},
		},
		// Auth service Prometheus metrics.
		{
			JobName:     "auth",
			ServiceName: "auth",
			Port:        4000,
			Path:        "/metrics",
			Interval:    "30s",
			Labels:      map[string]string{"service": "auth"},
		},
		// Tempo distributed tracing backend metrics.
		// Requires MONITORING_TRACING_ENABLED=true (tempo + otel-collector in stack).
		{
			JobName:     "tempo",
			ServiceName: "tempo",
			Port:        3200,
			Path:        "/metrics",
			Interval:    "30s",
			Labels:      map[string]string{"service": "tempo"},
		},
		// OTEL Collector internal metrics.
		// The collector exposes its own health and pipeline metrics at /metrics.
		// Requires MONITORING_TRACING_ENABLED=true (same condition as tempo).
		{
			JobName:     "otelcol",
			ServiceName: "otel-collector",
			Port:        8888,
			Path:        "/metrics",
			Interval:    "30s",
			Labels:      map[string]string{"service": "otel-collector"},
		},
	}
}

// TargetFromPlugin builds a ScrapeTarget for a plugin declared with the given
// name, port, and tier ("free" or "pro"). It sets sensible labels so the
// Grafana dashboards can filter by plugin and tier.
func TargetFromPlugin(name string, port int, tier string) ScrapeTarget {
	return ScrapeTarget{
		JobName:     name,
		ServiceName: name,
		Port:        port,
		Path:        "/metrics",
		Labels: map[string]string{
			"plugin": name,
			"tier":   tier,
		},
	}
}

const prometheusTmpl = `# Generated by nself build — DO NOT HAND EDIT
global:
  scrape_interval: {{.ScrapeInterval}}
  evaluation_interval: {{.EvaluationInterval}}
{{- if .ExternalLabels}}
  external_labels:
{{- range $k, $v := .ExternalLabels}}
    {{$k}}: {{$v}}
{{- end}}
{{- end}}
{{- if .AlertmanagerURL}}

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - {{.AlertmanagerURL}}
{{- end}}
{{- if .RuleFiles}}

rule_files:
{{- range .RuleFiles}}
  - {{.}}
{{- end}}
{{- end}}

scrape_configs:
{{- range .Targets}}
  - job_name: {{.JobName}}
{{- if .Interval}}
    scrape_interval: {{.Interval}}
{{- end}}
    metrics_path: {{if .Path}}{{.Path}}{{else}}/metrics{{end}}
{{- if .BearerToken}}
    authorization:
      credentials: {{.BearerToken}}
{{- end}}
    static_configs:
      - targets:
          - {{.ServiceName}}:{{.Port}}
{{- if .Labels}}
        labels:
{{- range $k, $v := .Labels}}
          {{$k}}: {{$v}}
{{- end}}
{{- end}}
{{- end}}
`

// RenderPrometheusYAML returns the rendered prometheus.yml bytes for cfg.
// Targets are emitted in stable order (alphabetical by job name) so repeated
// builds produce byte-identical output — critical for snapshot tests and for
// avoiding noisy diffs in generated config.
func RenderPrometheusYAML(cfg *PrometheusConfig) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("monitoring: nil PrometheusConfig")
	}
	// Stable order.
	sorted := make([]ScrapeTarget, len(cfg.Targets))
	copy(sorted, cfg.Targets)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].JobName < sorted[j].JobName })
	shadow := *cfg
	shadow.Targets = sorted

	tmpl, err := template.New("prometheus").Parse(prometheusTmpl)
	if err != nil {
		return nil, fmt.Errorf("monitoring: parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, &shadow); err != nil {
		return nil, fmt.Errorf("monitoring: execute template: %w", err)
	}
	return buf.Bytes(), nil
}
