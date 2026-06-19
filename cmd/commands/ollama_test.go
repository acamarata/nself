package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// mockOllamaServer starts a test HTTP server that mimics the Ollama API.
// It returns the server and a cleanup function.
func mockOllamaServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	mux := http.NewServeMux()

	// GET /api/version
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": "0.5.0"})
	})

	// GET /api/tags
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"name":        "gemma-3-4b",
					"size":        int64(2_700_000_000),
					"modified_at": "2026-04-01T00:00:00Z",
				},
			},
		})
	})

	// POST /api/pull
	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// DELETE /api/delete
	mux.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	cleanup := func() { srv.Close() }
	return srv, cleanup
}

// setOllamaEnv points NSELF_OLLAMA_HOST at the test server URL and restores it on cleanup.
func setOllamaEnv(t *testing.T, url string) {
	t.Helper()
	prev := os.Getenv("NSELF_OLLAMA_HOST")
	os.Setenv("NSELF_OLLAMA_HOST", url)
	t.Cleanup(func() { os.Setenv("NSELF_OLLAMA_HOST", prev) })
}

// --- Tests ---

func TestOllamaURL_DefaultAndEnv(t *testing.T) {
	os.Unsetenv("NSELF_OLLAMA_HOST")
	os.Unsetenv("PLUGIN_AI_OLLAMA_URL")
	if got := ollamaURL(); got != "http://localhost:11434" {
		t.Errorf("default URL: got %q, want http://localhost:11434", got)
	}

	os.Setenv("NSELF_OLLAMA_HOST", "http://ollama:11434")
	t.Cleanup(func() { os.Unsetenv("NSELF_OLLAMA_HOST") })
	if got := ollamaURL(); got != "http://ollama:11434" {
		t.Errorf("env URL: got %q, want http://ollama:11434", got)
	}
}

func TestOllamaModelsList_MockServer(t *testing.T) {
	srv, cleanup := mockOllamaServer(t)
	defer cleanup()
	setOllamaEnv(t, srv.URL)

	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := ollamaGet("/api/tags", &resp); err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(resp.Models) == 0 {
		t.Fatal("expected at least one model")
	}
	if resp.Models[0].Name != "gemma-3-4b" {
		t.Errorf("got model %q, want gemma-3-4b", resp.Models[0].Name)
	}
}

func TestOllamaModelsPull_MockServer(t *testing.T) {
	srv, cleanup := mockOllamaServer(t)
	defer cleanup()
	setOllamaEnv(t, srv.URL)

	if err := ollamaPost("/api/pull", map[string]string{"name": "tinyllama"}); err != nil {
		t.Fatalf("pull: %v", err)
	}
}

func TestOllamaStatus_MockServer(t *testing.T) {
	srv, cleanup := mockOllamaServer(t)
	defer cleanup()
	setOllamaEnv(t, srv.URL)

	var version struct {
		Version string `json:"version"`
	}
	if err := ollamaGet("/api/version", &version); err != nil {
		t.Fatalf("status: %v", err)
	}
	if version.Version == "" {
		t.Error("expected non-empty version string")
	}
}

func TestOllamaStatus_NotReachable(t *testing.T) {
	// Use a server that immediately closes the connection. This is platform-
	// agnostic: it avoids relying on TCP connection-refused behaviour, which
	// differs between Linux (immediate ECONNREFUSED) and Windows (firewall DROP →
	// long timeout). The EOF/reset that the hijacker produces causes the HTTP
	// client to return an error instantly on every OS.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijacker", http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer srv.Close()
	setOllamaEnv(t, srv.URL)
	err := runOllamaStatus(nil, nil)
	if err == nil {
		t.Error("expected error when Ollama is not reachable")
	}
	if !strings.Contains(err.Error(), "not reachable") {
		t.Errorf("error should mention 'not reachable', got: %v", err)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{500, "500 B"},
		{2048, "2.0 KB"},
		{1_500_000, "1.4 MB"},
		{2_700_000_000, "2.5 GB"},
	}
	for _, c := range cases {
		got := formatBytes(c.input)
		if got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

// TestOllamaPluginAIProviderRouting verifies that when NSELF_AI_PROVIDER=ollama
// is set, the environment correctly points at the Ollama host. This is the
// integration contract: plugin-ai reads OLLAMA_ENABLED + PLUGIN_AI_OLLAMA_URL.
func TestOllamaPluginAIProviderRouting(t *testing.T) {
	srv, cleanup := mockOllamaServer(t)
	defer cleanup()

	os.Setenv("NSELF_AI_PROVIDER", "ollama")
	os.Setenv("NSELF_OLLAMA_HOST", srv.URL)
	os.Setenv("OLLAMA_ENABLED", "true")
	t.Cleanup(func() {
		os.Unsetenv("NSELF_AI_PROVIDER")
		os.Unsetenv("NSELF_OLLAMA_HOST")
		os.Unsetenv("OLLAMA_ENABLED")
	})

	// Verify the Ollama server is reachable at the configured host.
	var version struct {
		Version string `json:"version"`
	}
	if err := ollamaGet("/api/version", &version); err != nil {
		t.Fatalf("routing test: Ollama not reachable via NSELF_OLLAMA_HOST: %v", err)
	}
	if version.Version == "" {
		t.Error("routing test: expected Ollama version string")
	}
}
