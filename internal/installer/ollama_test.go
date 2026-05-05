package installer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
)

// TestInstallerHTTPTimeout verifies that installer HTTP calls respect a
// short client-level timeout. We construct a fresh client via
// httptimeout.WithTimeout(100ms) and point it at a slow test server that
// sleeps 400ms. The request must error within ~200ms (well below 400ms).
//
// This is a regression guard: prior to T02, the installer used
// http.DefaultClient (no timeout) and 30-minute hand-rolled clients,
// which let stalled downloads hang for the full deadline. After T02,
// every installer HTTP call goes through httptimeout.Installer (300s
// default, env-tunable). This test pins the contract that the package's
// timeout factory works as documented.
func TestInstallerHTTPTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := httptimeout.WithTimeout(100 * time.Millisecond)
	if client.Timeout != 100*time.Millisecond {
		t.Fatalf("WithTimeout: client.Timeout = %v, want 100ms", client.Timeout)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err == nil {
		resp.Body.Close()
		t.Fatalf("expected timeout error, got nil after %v", elapsed)
	}
	// Timeout should fire near 100ms, well below the server's 400ms sleep.
	// Allow generous slack (up to 250ms) for CI scheduler jitter.
	if elapsed >= 250*time.Millisecond {
		t.Fatalf("timeout fired too late: elapsed=%v, want < 250ms", elapsed)
	}
}

// TestInstallerScopedClientHasTimeout pins the contract that the
// httptimeout.Installer client used by the installer package always
// has a positive Timeout. This guards against an accidental shadow of
// the package-level var with an unbounded &http.Client{}.
func TestInstallerScopedClientHasTimeout(t *testing.T) {
	if httptimeout.Installer.Timeout <= 0 {
		t.Fatalf("httptimeout.Installer.Timeout = %v, want > 0",
			httptimeout.Installer.Timeout)
	}
}
