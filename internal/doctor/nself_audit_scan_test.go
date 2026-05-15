package doctor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckNSelfAuditScan_PluginUnreachable_ReturnsSkippedPass verifies the
// graceful-degradation contract: if the plugin is not running, doctor must
// emit a single "pass" check (skipped) rather than failing the run.
func TestCheckNSelfAuditScan_PluginUnreachable_ReturnsSkippedPass(t *testing.T) {
	// Pick a port that's almost certainly closed.
	t.Setenv("NSELF_AUDIT_PORT", "1") // port 1 = unreachable

	tmp := t.TempDir()
	results := CheckNSelfAuditScan(context.Background(), tmp)
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 skipped result, got %d: %+v", len(results), results)
	}
	r := results[0]
	if r.Status != "pass" {
		t.Errorf("expected pass (graceful skip), got %s: %s", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "skipped") {
		t.Errorf("expected message to say 'skipped', got %q", r.Message)
	}
}

// TestCheckNSelfAuditScan_ZeroFindings_EmitsSinglePass starts a fake plugin
// returning zero findings; doctor must emit a single "pass" check.
func TestCheckNSelfAuditScan_ZeroFindings_EmitsSinglePass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scan/run" || r.Method != http.MethodPost {
			http.Error(w, "wrong route", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"findings":[],"summary":{"critical":0,"high":0,"medium":0,"low":0,"total":0}}`))
	}))
	defer srv.Close()

	port := portFromURL(t, srv.URL)
	t.Setenv("NSELF_AUDIT_PORT", port)

	results := CheckNSelfAuditScan(context.Background(), t.TempDir())
	if len(results) != 1 {
		t.Fatalf("expected 1 result for zero findings, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s", results[0].Status)
	}
}

// TestCheckNSelfAuditScan_WithFindings_MapsEachToCheckResult starts a fake
// plugin returning two findings (one critical, one medium) and verifies the
// dispatch emits a header row + 2 per-finding rows with correct severities.
func TestCheckNSelfAuditScan_WithFindings_MapsEachToCheckResult(t *testing.T) {
	payload := nselfAuditScanResponse{
		Findings: []nselfAuditScanFinding{
			{
				RuleID:      "NSCAN-001",
				Severity:    "critical",
				Title:       "Weak HASURA_GRAPHQL_ADMIN_SECRET",
				Description: "set to a default",
				Remediation: "rotate it",
				Evidence:    "HASURA_GRAPHQL_ADMIN_SECRET=admi***",
			},
			{
				RuleID:      "NSCAN-006",
				Severity:    "medium",
				Title:       "nginx missing CSP",
				Description: "no Content-Security-Policy",
				Remediation: "add CSP header",
				Evidence:    "no CSP directive",
			},
		},
		Summary: nselfAuditScanSummary{
			Critical: 1, Medium: 1, Total: 2,
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUDIT_PORT", portFromURL(t, srv.URL))

	results := CheckNSelfAuditScan(context.Background(), t.TempDir())
	// Expect 1 header + 2 findings = 3
	if len(results) != 3 {
		t.Fatalf("expected 3 results (header+2 findings), got %d: %+v", len(results), results)
	}
	if results[0].Status != "fail" {
		t.Errorf("header status: expected fail (1 critical), got %s", results[0].Status)
	}
	// Per-finding rows
	if results[1].Name != "NSCAN-001" || results[1].Status != "fail" {
		t.Errorf("expected NSCAN-001/fail, got %s/%s", results[1].Name, results[1].Status)
	}
	if results[2].Name != "NSCAN-006" || results[2].Status != "warn" {
		t.Errorf("expected NSCAN-006/warn, got %s/%s", results[2].Name, results[2].Status)
	}
	if !strings.Contains(results[1].FixCmd, "rotate") {
		t.Errorf("expected remediation in FixCmd, got %q", results[1].FixCmd)
	}
}

// TestCheckNSelfAuditScan_HTTPError_ReturnsWarn covers the path where the
// plugin responds with a non-200 (e.g. 500).
func TestCheckNSelfAuditScan_HTTPError_ReturnsWarn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUDIT_PORT", portFromURL(t, srv.URL))

	results := CheckNSelfAuditScan(context.Background(), t.TempDir())
	if len(results) != 1 || results[0].Status != "warn" {
		t.Fatalf("expected 1 warn, got %+v", results)
	}
}

// TestCheckNSelfAuditScan_MalformedJSON_ReturnsWarn covers decode-failure path.
func TestCheckNSelfAuditScan_MalformedJSON_ReturnsWarn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid`))
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUDIT_PORT", portFromURL(t, srv.URL))

	results := CheckNSelfAuditScan(context.Background(), t.TempDir())
	if len(results) != 1 || results[0].Status != "warn" {
		t.Fatalf("expected 1 warn for bad JSON, got %+v", results)
	}
}

// TestCheckNSelfAuditScan_SyntheticProjectFiles_PassesPaths verifies that the
// dispatcher correctly discovers .env.dev, nginx/conf.d, and docker-compose.yml
// under projectDir and forwards them to the plugin in the request body.
func TestCheckNSelfAuditScan_SyntheticProjectFiles_PassesPaths(t *testing.T) {
	projectDir := t.TempDir()
	mustWrite(t, filepath.Join(projectDir, ".env.dev"), "HASURA_GRAPHQL_ADMIN_SECRET=devsecret\n")
	mustMkdir(t, filepath.Join(projectDir, "nginx", "conf.d"))
	mustWrite(t, filepath.Join(projectDir, "nginx", "conf.d", "default.conf"), "server { listen 80; }\n")
	mustWrite(t, filepath.Join(projectDir, "docker-compose.yml"), "services: {}\n")

	captured := make(chan nselfAuditScanRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req nselfAuditScanRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		captured <- req
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"findings":[],"summary":{"total":0}}`))
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUDIT_PORT", portFromURL(t, srv.URL))

	_ = CheckNSelfAuditScan(context.Background(), projectDir)

	select {
	case req := <-captured:
		if !strings.HasSuffix(req.EnvFilePath, ".env.dev") {
			t.Errorf("expected env_file_path ending .env.dev, got %q", req.EnvFilePath)
		}
		if !strings.HasSuffix(req.NginxConfigDir, filepath.Join("nginx", "conf.d")) {
			t.Errorf("expected nginx_config_dir, got %q", req.NginxConfigDir)
		}
		if !strings.HasSuffix(req.DockerComposePath, "docker-compose.yml") {
			t.Errorf("expected docker_compose_path, got %q", req.DockerComposePath)
		}
	default:
		t.Fatalf("plugin handler was never invoked")
	}
}

// TestScanStatusFromSeverity covers the severity → status mapping table.
func TestScanStatusFromSeverity(t *testing.T) {
	cases := map[string]string{
		"critical": "fail",
		"CRITICAL": "fail",
		"high":     "fail",
		"medium":   "warn",
		"low":      "warn",
		"unknown":  "warn",
		"":         "warn",
	}
	for in, want := range cases {
		if got := scanStatusFromSeverity(in); got != want {
			t.Errorf("scanStatusFromSeverity(%q)=%q want %q", in, got, want)
		}
	}
}

// TestScanStatusFromSummary covers the summary → header-status mapping.
func TestScanStatusFromSummary(t *testing.T) {
	if got := scanStatusFromSummary(nselfAuditScanSummary{Critical: 1}); got != "fail" {
		t.Errorf("critical>0 → fail, got %s", got)
	}
	if got := scanStatusFromSummary(nselfAuditScanSummary{Medium: 1}); got != "warn" {
		t.Errorf("medium>0 → warn, got %s", got)
	}
	if got := scanStatusFromSummary(nselfAuditScanSummary{}); got != "pass" {
		t.Errorf("empty → pass, got %s", got)
	}
}

// TestFirstExistingFile_PicksFirstHit verifies precedence order.
func TestFirstExistingFile_PicksFirstHit(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "b"), "x")
	mustWrite(t, filepath.Join(dir, "c"), "x")
	if got := firstExistingFile(dir, "a", "b", "c"); !strings.HasSuffix(got, "b") {
		t.Errorf("expected b, got %s", got)
	}
	if got := firstExistingFile(dir, "z"); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

// TestFirstExistingDir_PicksFirstHit verifies precedence order.
func TestFirstExistingDir_PicksFirstHit(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "p"))
	if got := firstExistingDir(dir, "q", "p"); !strings.HasSuffix(got, "p") {
		t.Errorf("expected p, got %s", got)
	}
	if got := firstExistingDir(dir, "z"); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
	// File-as-dir lookup must return empty.
	mustWrite(t, filepath.Join(dir, "f"), "")
	if got := firstExistingDir(dir, "f"); got != "" {
		t.Errorf("expected empty for file-not-dir, got %s", got)
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func portFromURL(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u.Port()
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}
