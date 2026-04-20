package dlq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"bytes"
)

// buildTestServer creates an httptest server with 5 mock DLQ rows for pluginName.
func buildTestServer(t *testing.T, pluginName string, replayFails map[string]bool) *httptest.Server {
	t.Helper()
	rows := []map[string]interface{}{
		{"id": "row-1", "status": "quarantined"},
		{"id": "row-2", "status": "quarantined"},
		{"id": "row-3", "status": "quarantined"},
		{"id": "row-4", "status": "quarantined"},
		{"id": "row-5", "status": "quarantined"},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// DLQ list endpoint.
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/dlq") && !strings.Contains(r.URL.Path, "/replay") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(rows)
			return
		}

		// Per-row replay endpoint.
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/replay") {
			parts := strings.Split(r.URL.Path, "/")
			rowID := parts[len(parts)-2] // .../dlq/{rowID}/replay
			if replayFails[rowID] {
				http.Error(w, "upstream error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
}

func TestReplay(t *testing.T) {
	srv := buildTestServer(t, "mux", nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	results, err := Replay(context.Background(), ReplayOptions{
		Plugin:     "mux",
		MaxRows:    100,
		BaseURL:    srv.URL,
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("row %s should have succeeded", r.ID)
		}
	}
}

func TestReplayDryRun(t *testing.T) {
	srv := buildTestServer(t, "mux", nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	results, err := Replay(context.Background(), ReplayOptions{
		Plugin:     "mux",
		MaxRows:    100,
		DryRun:     true,
		BaseURL:    srv.URL,
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("dry-run should not error, got: %v", err)
	}
	if results != nil {
		t.Error("dry-run should return nil results")
	}
	if !strings.Contains(stdout.String(), "DRY RUN") {
		t.Error("expected DRY RUN in output")
	}
	if !strings.Contains(stdout.String(), "No changes made") {
		t.Error("expected 'No changes made' in output")
	}
}

func TestReplayMaxRows(t *testing.T) {
	// MaxRows=2 should cap at 2 even though 5 rows exist.
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Return only 2 rows (server respects max param).
			rows := []map[string]interface{}{
				{"id": "row-1", "status": "quarantined"},
				{"id": "row-2", "status": "quarantined"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(rows)
			return
		}
		if r.Method == http.MethodPost {
			callCount++
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	results, err := Replay(context.Background(), ReplayOptions{
		Plugin:     "mux",
		MaxRows:    2,
		BaseURL:    srv.URL,
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (max-rows=2), got %d", len(results))
	}
	if callCount != 2 {
		t.Errorf("expected 2 replay calls (max-rows=2), got %d", callCount)
	}
}

func TestReplayWarning(t *testing.T) {
	srv := buildTestServer(t, "mux", nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	_, _ = Replay(context.Background(), ReplayOptions{
		Plugin:     "mux",
		MaxRows:    100,
		BaseURL:    srv.URL,
		Stdout:     &stdout,
		Stderr:     &stderr,
		HTTPClient: srv.Client(),
	})
	if !strings.Contains(stderr.String(), "upstream bug") {
		t.Error("expected upstream-bug warning in stderr")
	}
}
