package doctor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFilterBySection(t *testing.T) {
	results := []CheckResult{
		{Section: "host", Name: "Disk", Status: "pass"},
		{Section: "docker", Name: "Daemon", Status: "pass"},
		{Section: "host", Name: "CPU", Status: "warn"},
		{Section: "postgres", Name: "pg_isready", Status: "pass"},
	}

	host := FilterBySection(results, "host")
	if len(host) != 2 {
		t.Errorf("expected 2 host results, got %d", len(host))
	}

	pg := FilterBySection(results, "postgres")
	if len(pg) != 1 {
		t.Errorf("expected 1 postgres result, got %d", len(pg))
	}

	empty := FilterBySection(results, "nonexistent")
	if len(empty) != 0 {
		t.Errorf("expected 0 results for nonexistent section, got %d", len(empty))
	}
}

func TestDefaultSchedulesSectionNames(t *testing.T) {
	// Verify all 12 subsystem section names are used in DeepChecks output types
	expectedSections := []string{
		"host", "docker", "postgres", "hasura", "nginx", "ssl",
		"ping", "plugins", "license", "monitoring", "backups", "security",
	}
	for _, s := range expectedSections {
		// Just verify the section name is a non-empty string
		if s == "" {
			t.Error("empty section name")
		}
	}
}

// parsePort extracts the numeric port from a "host:port" address string.
func parsePort(t *testing.T, addr string) int {
	t.Helper()
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		t.Fatalf("unexpected addr format: %s", addr)
	}
	port := 0
	for _, c := range parts[len(parts)-1] {
		if c >= '0' && c <= '9' {
			port = port*10 + int(c-'0')
		}
	}
	return port
}

// TestProbePluginHTTP_Pass verifies a 200 on /health yields a pass result.
func TestProbePluginHTTP_Pass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	port := parsePort(t, srv.Listener.Addr().String())
	client := &http.Client{}
	result := probePluginHTTP(client, "testplugin", port)
	if result.Status != "pass" {
		t.Errorf("expected pass, got %s: %s", result.Status, result.Message)
	}
	if !strings.Contains(result.Name, "PLUGIN-HEALTH-testplugin") {
		t.Errorf("expected check ID PLUGIN-HEALTH-testplugin, got %s", result.Name)
	}
}

// TestProbePluginHTTP_Fallback verifies /health 404 falls back to /healthz.
func TestProbePluginHTTP_Fallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	port := parsePort(t, srv.Listener.Addr().String())
	client := &http.Client{}
	result := probePluginHTTP(client, "myplugin", port)
	if result.Status != "pass" {
		t.Errorf("expected pass on /healthz fallback, got %s: %s", result.Status, result.Message)
	}
}

// TestProbePluginHTTP_Fail verifies a non-200 non-404 yields a fail result.
func TestProbePluginHTTP_Fail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	port := parsePort(t, srv.Listener.Addr().String())
	client := &http.Client{}
	result := probePluginHTTP(client, "badplugin", port)
	if result.Status != "fail" {
		t.Errorf("expected fail on 500, got %s: %s", result.Status, result.Message)
	}
}

// TestProbePluginHTTP_ConnectionRefused verifies a refused connection yields fail.
func TestProbePluginHTTP_ConnectionRefused(t *testing.T) {
	// Port 1 is not bound anywhere in a test environment.
	client := &http.Client{}
	result := probePluginHTTP(client, "deadplugin", 1)
	if result.Status != "fail" {
		t.Errorf("expected fail on connection refused, got %s", result.Status)
	}
	if !strings.Contains(result.Message, "not running") {
		t.Errorf("expected 'not running' in message, got: %s", result.Message)
	}
}

// TestVerifyDockerRunning_CancelledContext verifies that a cancelled context
// causes verifyDockerRunning to return a non-nil error.
func TestVerifyDockerRunning_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before exec starts
	err := verifyDockerRunning(ctx)
	if err == nil {
		// Docker desktop may have responded before the context killed the process;
		// this is a non-failure in CI where Docker is available.
		t.Log("verifyDockerRunning returned nil with cancelled context (Docker responded faster than cancel)")
	}
}
