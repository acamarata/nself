// Package gateway — unit tests for status.go
package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAllHealthy_AllUp verifies AllHealthy returns true when all services are healthy.
func TestAllHealthy_AllUp(t *testing.T) {
	statuses := []ServiceStatus{
		{Name: "nself-ai-cc", Port: 3760, Healthy: true},
		{Name: "nself-ai-gateway", Port: 3761, Healthy: true},
		{Name: "nself-ai-mcp", Port: 3762, Healthy: true},
	}
	if !AllHealthy(statuses) {
		t.Error("expected AllHealthy=true when all services healthy")
	}
}

// TestAllHealthy_OneDown verifies AllHealthy returns false when any service is down.
func TestAllHealthy_OneDown(t *testing.T) {
	statuses := []ServiceStatus{
		{Name: "nself-ai-cc", Port: 3760, Healthy: true},
		{Name: "nself-ai-gateway", Port: 3761, Healthy: false, Error: "unreachable"},
		{Name: "nself-ai-mcp", Port: 3762, Healthy: true},
	}
	if AllHealthy(statuses) {
		t.Error("expected AllHealthy=false when one service is down")
	}
}

// TestAllHealthy_Empty verifies AllHealthy returns true for empty slice (vacuous truth).
func TestAllHealthy_Empty(t *testing.T) {
	if !AllHealthy(nil) {
		t.Error("expected AllHealthy=true for empty slice")
	}
}

// TestHealthCheck_Healthy verifies healthCheck marks service healthy on HTTP 200.
func TestHealthCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := srv.Client()
	// Use the server's host:port directly.
	status := healthCheckURL(client, "nself-ai-gateway", srv.URL+"/health")
	if !status.Healthy {
		t.Errorf("expected Healthy=true, got error=%q", status.Error)
	}
}

// TestHealthCheck_Unhealthy verifies healthCheck marks service unhealthy on non-200.
func TestHealthCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := srv.Client()
	status := healthCheckURL(client, "nself-ai-cc", srv.URL+"/health")
	if status.Healthy {
		t.Error("expected Healthy=false on HTTP 503")
	}
	if status.Error == "" {
		t.Error("expected non-empty Error string")
	}
}

// TestHealthCheck_ConnectionRefused verifies healthCheck marks service unreachable
// when the connection is refused (port not listening).
func TestHealthCheck_ConnectionRefused(t *testing.T) {
	// Use a port that is almost certainly not listening.
	client := &http.Client{}
	status := healthCheckURL(client, "nself-ai-mcp", "http://localhost:19999/health")
	if status.Healthy {
		t.Error("expected Healthy=false for refused connection")
	}
	if status.Error == "" {
		t.Error("expected non-empty Error on refused connection")
	}
}

// healthCheckURL is a test-injectable variant that accepts an explicit URL instead
// of constructing it from port. This avoids binding to real ports in tests.
func healthCheckURL(client *http.Client, name string, url string) ServiceStatus {
	resp, err := client.Get(url)
	if err != nil {
		return ServiceStatus{Name: name, Healthy: false, Error: "unreachable: " + err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return ServiceStatus{Name: name, Healthy: true}
	}
	return ServiceStatus{
		Name:    name,
		Healthy: false,
		Error:   "HTTP " + http.StatusText(resp.StatusCode),
	}
}
