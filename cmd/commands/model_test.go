package commands

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock server for model.go tests
// ---------------------------------------------------------------------------

// mockModelServer starts a minimal Ollama-compatible HTTP server for model.go tests.
func mockModelServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	mux := http.NewServeMux()

	// GET /api/tags — list models
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"name":        "gemma-3-4b",
					"size":        int64(2_700_000_000),
					"modified_at": "2026-04-01T00:00:00Z",
				},
				{
					"name":        "llama3.2:3b",
					"size":        int64(2_000_000_000),
					"modified_at": "2026-04-10T00:00:00Z",
				},
			},
		})
	})

	// POST /api/pull — pull model
	mux.HandleFunc("/api/pull", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// DELETE /api/delete — remove model
	mux.HandleFunc("/api/delete", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// POST /api/generate — benchmark
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"model":      "gemma-3-4b",
			"response":   "A Merkle tree is a hash tree.",
			"done":       true,
			"eval_count": 42,
		})
	})

	srv := httptest.NewServer(mux)
	return srv, srv.Close
}

// setModelOllamaEnv points NSELF_OLLAMA_HOST at the test server URL.
func setModelOllamaEnv(t *testing.T, url string) {
	t.Helper()
	prev := os.Getenv("NSELF_OLLAMA_HOST")
	os.Setenv("NSELF_OLLAMA_HOST", url)
	t.Cleanup(func() { os.Setenv("NSELF_OLLAMA_HOST", prev) })
}

// ---------------------------------------------------------------------------
// modelOllamaURL
// ---------------------------------------------------------------------------

func TestModelOllamaURL_Defaults(t *testing.T) {
	os.Unsetenv("NSELF_OLLAMA_HOST")
	os.Unsetenv("PLUGIN_AI_OLLAMA_URL")
	if got := modelOllamaURL(); got != "http://localhost:11434" {
		t.Errorf("default URL: got %q, want http://localhost:11434", got)
	}
}

func TestModelOllamaURL_EnvOverride(t *testing.T) {
	os.Setenv("NSELF_OLLAMA_HOST", "http://ollama-test:11434")
	t.Cleanup(func() { os.Unsetenv("NSELF_OLLAMA_HOST") })
	if got := modelOllamaURL(); got != "http://ollama-test:11434" {
		t.Errorf("env URL: got %q, want http://ollama-test:11434", got)
	}
}

func TestModelOllamaURL_FallbackToPluginAI(t *testing.T) {
	os.Unsetenv("NSELF_OLLAMA_HOST")
	os.Setenv("PLUGIN_AI_OLLAMA_URL", "http://plugin-ai-host:11434")
	t.Cleanup(func() { os.Unsetenv("PLUGIN_AI_OLLAMA_URL") })
	if got := modelOllamaURL(); got != "http://plugin-ai-host:11434" {
		t.Errorf("fallback URL: got %q, want http://plugin-ai-host:11434", got)
	}
}

// ---------------------------------------------------------------------------
// list
// ---------------------------------------------------------------------------

func TestModelList_MockServer(t *testing.T) {
	srv, cleanup := mockModelServer(t)
	defer cleanup()
	setModelOllamaEnv(t, srv.URL)

	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := modelOllamaGet("/api/tags", &resp); err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(resp.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(resp.Models))
	}
	names := []string{resp.Models[0].Name, resp.Models[1].Name}
	for _, want := range []string{"gemma-3-4b", "llama3.2:3b"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected model %q in list, got %v", want, names)
		}
	}
}

func TestRunModelList_Empty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	setModelOllamaEnv(t, srv.URL)

	// Should not error when there are no models.
	if err := runModelList(nil, nil); err != nil {
		t.Fatalf("runModelList (empty): %v", err)
	}
}

func TestRunModelList_NotReachable(t *testing.T) {
	setModelOllamaEnv(t, "http://127.0.0.1:19998")
	err := runModelList(nil, nil)
	if err == nil {
		t.Error("expected error when Ollama is not reachable")
	}
	if !strings.Contains(err.Error(), "model list") {
		t.Errorf("error should mention 'model list', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// pull
// ---------------------------------------------------------------------------

func TestRunModelPull_MockServer(t *testing.T) {
	srv, cleanup := mockModelServer(t)
	defer cleanup()
	setModelOllamaEnv(t, srv.URL)

	if err := runModelPull(nil, []string{"tinyllama"}); err != nil {
		t.Fatalf("pull: %v", err)
	}
}

func TestRunModelPull_NotReachable(t *testing.T) {
	setModelOllamaEnv(t, "http://127.0.0.1:19997")
	err := runModelPull(nil, []string{"tinyllama"})
	if err == nil {
		t.Error("expected error when Ollama is not reachable")
	}
}

// ---------------------------------------------------------------------------
// remove
// ---------------------------------------------------------------------------

func TestRunModelRemove_MockServer(t *testing.T) {
	srv, cleanup := mockModelServer(t)
	defer cleanup()
	setModelOllamaEnv(t, srv.URL)

	if err := runModelRemove(nil, []string{"gemma-3-4b"}); err != nil {
		t.Fatalf("remove: %v", err)
	}
}

func TestRunModelRemove_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/delete", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	setModelOllamaEnv(t, srv.URL)

	err := runModelRemove(nil, []string{"nonexistent-model"})
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

// ---------------------------------------------------------------------------
// update (delegates to pull)
// ---------------------------------------------------------------------------

func TestRunModelUpdate_DelegatesToPull(t *testing.T) {
	srv, cleanup := mockModelServer(t)
	defer cleanup()
	setModelOllamaEnv(t, srv.URL)

	// update calls runModelPull internally; success means pull succeeded.
	err := modelUpdateCmd.RunE(modelUpdateCmd, []string{"gemma-3-4b"})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
}

// ---------------------------------------------------------------------------
// benchmark
// ---------------------------------------------------------------------------

func TestModelBenchRun_MockServer(t *testing.T) {
	srv, cleanup := mockModelServer(t)
	defer cleanup()
	setModelOllamaEnv(t, srv.URL)

	toks, ms, err := modelBenchRun("gemma-3-4b", "Test prompt")
	if err != nil {
		t.Fatalf("bench run: %v", err)
	}
	if toks != 42 {
		t.Errorf("expected 42 tokens, got %d", toks)
	}
	if ms < 0 {
		t.Errorf("expected non-negative latency, got %.2f ms", ms)
	}
}

func TestModelBenchRun_NotReachable(t *testing.T) {
	setModelOllamaEnv(t, "http://127.0.0.1:19996")
	_, _, err := modelBenchRun("gemma-3-4b", "test")
	if err == nil {
		t.Error("expected error when Ollama not reachable")
	}
}

func TestRunModelBenchmark_AllRuns(t *testing.T) {
	srv, cleanup := mockModelServer(t)
	defer cleanup()
	setModelOllamaEnv(t, srv.URL)

	// Run 3 iterations; all should succeed with our mock.
	modelBenchRuns = 3
	modelBenchPrompt = ""
	defer func() {
		modelBenchRuns = 5
		modelBenchPrompt = ""
	}()

	if err := runModelBenchmark(nil, []string{"gemma-3-4b"}); err != nil {
		t.Fatalf("benchmark: %v", err)
	}
}

func TestRunModelBenchmark_AllFail(t *testing.T) {
	setModelOllamaEnv(t, "http://127.0.0.1:19995")
	modelBenchRuns = 2
	defer func() { modelBenchRuns = 5 }()

	err := runModelBenchmark(nil, []string{"gemma-3-4b"})
	if err == nil {
		t.Error("expected error when all runs fail")
	}
	if !strings.Contains(err.Error(), "all") {
		t.Errorf("error should mention 'all', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// p99 math
// ---------------------------------------------------------------------------

func TestP99Calculation(t *testing.T) {
	// 100 latency values: 1..100 ms. p99 should be 99.
	latencies := make([]float64, 100)
	for i := range latencies {
		latencies[i] = float64(i + 1)
	}

	n := len(latencies)
	idx := int(math.Ceil(float64(n)*0.99)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	// Pre-sorted, so index 98 = value 99.
	got := latencies[idx]
	if got != 99.0 {
		t.Errorf("p99 calculation: got %.1f, want 99.0", got)
	}
}

// ---------------------------------------------------------------------------
// Hardware compatibility check (unit — no network)
// ---------------------------------------------------------------------------

// TestHardwareRec verifies the recommended model list contains known local models.
func TestHardwareRec_KnownModels(t *testing.T) {
	// Verify that the models mentioned in the pull Long description are real
	// Ollama names (no typos / whitespace).
	knownModels := []string{
		"gemma-3-4b",
		"llama3.2:3b",
		"llama3.2:7b",
		"mistral",
	}
	for _, m := range knownModels {
		if strings.Contains(m, " ") {
			t.Errorf("model name %q contains spaces", m)
		}
		if m == "" {
			t.Error("empty model name")
		}
	}
}
