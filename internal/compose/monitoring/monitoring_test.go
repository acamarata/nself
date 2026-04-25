package monitoring

import (
	"strings"
	"testing"
)

func TestRenderPrometheusYAMLIncludesBuiltins(t *testing.T) {
	cfg := Defaults()
	out, err := RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)

	wants := []string{
		"job_name: prometheus",
		"job_name: cadvisor",
		"job_name: node",
		"job_name: postgres",
		"job_name: nginx",
		"job_name: hasura",
		"job_name: minio",
		"job_name: auth",
		"scrape_interval: 15s",
		"alertmanager:9093",
		// Hasura metrics path and bearer token placeholder
		"/v1/metrics",
		"HASURA_GRAPHQL_METRICS_SECRET",
		// MinIO cluster metrics path
		"/minio/v2/metrics/cluster",
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Fatalf("expected rendered config to contain %q\n---\n%s", w, s)
		}
	}
}

func TestRenderPrometheusYAMLBearerToken(t *testing.T) {
	cfg := Defaults()
	out, err := RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "authorization:") {
		t.Fatalf("expected authorization block for hasura target\n---\n%s", s)
	}
	if !strings.Contains(s, "credentials: ${HASURA_GRAPHQL_METRICS_SECRET}") {
		t.Fatalf("expected bearer token placeholder in authorization block\n---\n%s", s)
	}
}

func TestRenderPrometheusYAMLStableOrder(t *testing.T) {
	cfg := Defaults()
	cfg.Targets = []ScrapeTarget{
		{JobName: "zeta", ServiceName: "z", Port: 1},
		{JobName: "alpha", ServiceName: "a", Port: 1},
		{JobName: "mu", ServiceName: "m", Port: 1},
	}
	out1, err := RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render 1: %v", err)
	}
	out2, err := RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render 2: %v", err)
	}
	if string(out1) != string(out2) {
		t.Fatal("render is not deterministic")
	}
	idxAlpha := strings.Index(string(out1), "job_name: alpha")
	idxMu := strings.Index(string(out1), "job_name: mu")
	idxZeta := strings.Index(string(out1), "job_name: zeta")
	if !(idxAlpha < idxMu && idxMu < idxZeta) {
		t.Fatalf("targets not in alpha order: alpha=%d mu=%d zeta=%d", idxAlpha, idxMu, idxZeta)
	}
}

func TestTargetFromPlugin(t *testing.T) {
	tg := TargetFromPlugin("ai", 8081, "pro")
	if tg.JobName != "ai" || tg.ServiceName != "ai" || tg.Port != 8081 {
		t.Fatalf("unexpected target: %+v", tg)
	}
	if tg.Labels["plugin"] != "ai" || tg.Labels["tier"] != "pro" {
		t.Fatalf("unexpected labels: %+v", tg.Labels)
	}
	if tg.Path != "/metrics" {
		t.Fatalf("unexpected path: %q", tg.Path)
	}
}

func TestRenderPrometheusYAMLNilConfig(t *testing.T) {
	if _, err := RenderPrometheusYAML(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestRenderLokiYAMLDefaults(t *testing.T) {
	out, err := RenderLokiYAML(DefaultLokiConfig())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)

	wants := []string{
		"retention_period: 720h",
		"auth_enabled: false",
		"ruler:",
		"enable_api: true",
		"reporting_enabled: false",
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Fatalf("expected loki yaml to contain %q\n---\n%s", w, s)
		}
	}
}

func TestRenderLokiYAMLMultiTenant(t *testing.T) {
	cfg := DefaultLokiConfig()
	cfg.MultiTenantEnabled = true
	out, err := RenderLokiYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "auth_enabled: true") {
		t.Fatal("expected multi-tenant to enable auth")
	}
	if !strings.Contains(s, "per_tenant_override_config") {
		t.Fatal("expected per-tenant override config in multi-tenant mode")
	}
}

func TestRenderLokiYAMLNilConfig(t *testing.T) {
	if _, err := RenderLokiYAML(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestRenderPromtailYAMLDefaults(t *testing.T) {
	out, err := RenderPromtailYAML(DefaultPromtailConfig("my-proj"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)

	wants := []string{
		"http://loki:3100/loki/api/v1/push",
		"nself_project: my-proj",
		"docker_sd_configs:",
		"container:",
		"plugin:",
		"tier:",
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Fatalf("expected promtail yaml to contain %q\n---\n%s", w, s)
		}
	}
}

func TestRenderPromtailYAMLWithTenant(t *testing.T) {
	cfg := DefaultPromtailConfig("my-proj")
	cfg.TenantID = "tenant-1"
	out, err := RenderPromtailYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(string(out), "tenant_id: tenant-1") {
		t.Fatalf("expected tenant_id in rendered yaml:\n%s", string(out))
	}
}

func TestRenderPromtailYAMLNilConfig(t *testing.T) {
	if _, err := RenderPromtailYAML(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
}
