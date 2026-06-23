// Purpose: Unit tests for claw memories — CRUD paths against a fake server.
//
// Acceptance: claw memories: CRUD integration tests pass.
// Production embedding-pg fixture tests run under INTEGRATION=1.
// SPORT: F02-COMMAND-INVENTORY row: claw memories

package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestClawMemoriesList_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/claw/memories" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"memories": []map[string]interface{}{
				{"id": "m1", "content": "First memory", "type": "fact", "created_at": "2026-01-01T00:00:00Z"},
				{"id": "m2", "content": "Second memory", "type": "note", "created_at": "2026-01-02T00:00:00Z"},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawMemoriesList(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "First memory") {
		t.Errorf("expected 'First memory' in output, got: %q", out)
	}
	if !strings.Contains(out, "Second memory") {
		t.Errorf("expected 'Second memory' in output, got: %q", out)
	}
}

func TestClawMemoriesList_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"memories": []interface{}{},
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawMemoriesList(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No memories") {
		t.Errorf("expected 'No memories' in output, got: %q", out)
	}
}

func TestClawMemoriesSearch_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/claw/memories/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if q := r.URL.Query().Get("q"); q != "Virginia" {
			t.Errorf("expected q=Virginia, got q=%s", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"memories": []map[string]interface{}{
				{"id": "m3", "content": "Virginia trip notes", "type": "note", "created_at": "2026-03-15T00:00:00Z"},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawMemoriesSearch(cmd, []string{"Virginia"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Virginia") {
		t.Errorf("expected 'Virginia' in output, got: %q", out)
	}
}

func TestClawMemoriesList_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runClawMemoriesList(cmd, nil)
	if err == nil {
		t.Fatal("expected error from server 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention HTTP 500, got: %v", err)
	}
}

// captureClawStdout captures os.Stdout within fn. Used by multiple claw test files.
// Reuse captureLocalStdout from claw_keys_test.go via the same package.
func captureClawStdout(t *testing.T, fn func()) string {
	t.Helper()
	return captureLocalStdout(t, fn)
}
