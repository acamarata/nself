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

// --- G6-T04: OTEL alert rules + Alertmanager ---

func TestRenderOTELAlertsYAMLContainsFourRules(t *testing.T) {
	cfg := DefaultOTELAlertRulesConfig()
	out, err := RenderOTELAlertsYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)

	rules := []string{
		"alert: OtelCollectorDown",
		"alert: TempoDown",
		"alert: OtelSpanIngestDrop",
		"alert: TempoScrapeErrors",
	}
	for _, rule := range rules {
		if !strings.Contains(s, rule) {
			t.Fatalf("expected otel-alerts.yml to contain %q\n---\n%s", rule, s)
		}
	}
}

func TestRenderOTELAlertsYAMLSeverityLabels(t *testing.T) {
	cfg := DefaultOTELAlertRulesConfig()
	out, err := RenderOTELAlertsYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)

	if !strings.Contains(s, "severity: critical") {
		t.Fatal("expected at least one critical severity label")
	}
	if !strings.Contains(s, "severity: warning") {
		t.Fatal("expected at least one warning severity label")
	}
}

func TestRenderOTELAlertsYAMLDownDuration(t *testing.T) {
	cfg := DefaultOTELAlertRulesConfig()
	out, err := RenderOTELAlertsYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	// Both down alerts must fire after DownDuration (default 5m), not immediately.
	if count := strings.Count(s, "for: 5m"); count < 2 {
		t.Fatalf("expected at least 2 'for: 5m' duration guards, got %d\n---\n%s", count, s)
	}
}

func TestRenderOTELAlertsYAMLRunbookURLsPresent(t *testing.T) {
	cfg := DefaultOTELAlertRulesConfig()
	out, err := RenderOTELAlertsYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "runbook_url:") {
		t.Fatal("expected runbook_url annotations in otel-alerts.yml")
	}
}

func TestRenderOTELAlertsYAMLNilConfig(t *testing.T) {
	if _, err := RenderOTELAlertsYAML(nil); err == nil {
		t.Fatal("expected error for nil OTELAlertRulesConfig")
	}
}

func TestRenderOTELAlertsYAMLSpanDropThreshold(t *testing.T) {
	cfg := DefaultOTELAlertRulesConfig()
	cfg.SpanIngestDropRatio = 0.5
	out, err := RenderOTELAlertsYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "0.5") {
		t.Fatalf("expected span ingest ratio 0.5 in rendered config\n---\n%s", s)
	}
}

func TestRenderOTELAlertsYAMLOtelcolJobSelector(t *testing.T) {
	cfg := DefaultOTELAlertRulesConfig()
	out, err := RenderOTELAlertsYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `up{job="otelcol"}`) {
		t.Fatalf("expected otelcol job selector in OtelCollectorDown expr\n---\n%s", s)
	}
	if !strings.Contains(s, `up{job="tempo"}`) {
		t.Fatalf("expected tempo job selector in TempoDown expr\n---\n%s", s)
	}
}

func TestRenderAlertmanagerYAMLContainsOncallStub(t *testing.T) {
	cfg := DefaultAlertmanagerConfig()
	out, err := RenderAlertmanagerYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)

	wants := []string{
		"oncall-stub",
		"null-receiver",
		"severity = critical",
		"severity = warning",
		"P98",
		"email_configs:",
		"send_resolved: true",
		"group_wait: 30s",
		"group_interval: 5m",
		"repeat_interval: 4h",
		"inhibit_rules:",
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Fatalf("expected alertmanager.yml to contain %q\n---\n%s", w, s)
		}
	}
}

func TestRenderAlertmanagerYAMLEmailConfigurable(t *testing.T) {
	cfg := DefaultAlertmanagerConfig()
	out, err := RenderAlertmanagerYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	// Email address must use the env-var placeholder — never a hardcoded address.
	if !strings.Contains(s, "ALERTMANAGER_ONCALL_EMAIL") {
		t.Fatalf("expected ALERTMANAGER_ONCALL_EMAIL env-var placeholder in alertmanager.yml\n---\n%s", s)
	}
}

func TestRenderAlertmanagerYAMLNilConfig(t *testing.T) {
	if _, err := RenderAlertmanagerYAML(nil); err == nil {
		t.Fatal("expected error for nil AlertmanagerConfig")
	}
}

func TestPrometheusDefaultsIncludeOtelcolTarget(t *testing.T) {
	cfg := Defaults()
	out, err := RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "job_name: otelcol") {
		t.Fatalf("expected otelcol scrape target in default prometheus config\n---\n%s", s)
	}
}

func TestPrometheusDefaultsIncludeOtelAlertRulesFile(t *testing.T) {
	cfg := Defaults()
	out, err := RenderPrometheusYAML(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "otel-alerts.yml") {
		t.Fatalf("expected otel-alerts.yml in default prometheus rule_files\n---\n%s", s)
	}
}
