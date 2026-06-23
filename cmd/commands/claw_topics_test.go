// Purpose: Unit tests for claw topics — list/search paths against a fake server.
//
// Acceptance: claw topics: CRUD integration tests pass.
// Production embedding-pg fixture tests run under INTEGRATION=1.
// SPORT: F02-COMMAND-INVENTORY row: claw topics

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

func TestClawTopicsList_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/claw/topics" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"topics": []map[string]interface{}{
				{"id": "t1", "name": "Work", "path": "/work", "message_count": 42, "last_active_at": "2026-06-01T00:00:00Z"},
				{"id": "t2", "name": "Travel", "path": "/travel", "message_count": 7, "last_active_at": "2026-05-15T00:00:00Z"},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawTopicsList(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Work") {
		t.Errorf("expected topic 'Work' in output, got: %q", out)
	}
	if !strings.Contains(out, "Travel") {
		t.Errorf("expected topic 'Travel' in output, got: %q", out)
	}
}

func TestClawTopicsList_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"topics": []interface{}{},
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawTopicsList(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "No topics") {
		t.Errorf("expected 'No topics' in output, got: %q", out)
	}
}

func TestClawTopicsSearch_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/claw/topics/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if q := r.URL.Query().Get("q"); q != "travel" {
			t.Errorf("expected q=travel, got q=%s", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"topics": []map[string]interface{}{
				{"id": "t2", "name": "Travel", "path": "/travel", "message_count": 7, "last_active_at": ""},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawTopicsSearch(cmd, []string{"travel"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if !strings.Contains(out, "Travel") {
		t.Errorf("expected 'Travel' in output, got: %q", out)
	}
}

func TestClawTopicsList_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runClawTopicsList(cmd, nil)
	if err == nil {
		t.Fatal("expected error from server 504, got nil")
	}
	if !strings.Contains(err.Error(), "504") {
		t.Errorf("expected error to mention HTTP 504, got: %v", err)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"short", 100, "short"},
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.want)
		}
	}
}

func TestTrimLines(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello\nworld", "hello world"},
		{"  hello  ", "hello"},
		{"line1\nline2\nline3", "line1 line2 line3"},
		{"no newlines", "no newlines"},
	}
	for _, tc := range cases {
		got := trimLines(tc.input)
		if got != tc.want {
			t.Errorf("trimLines(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
