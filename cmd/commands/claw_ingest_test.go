// Purpose: Tests for claw ingest — source type detection, flag validation,
// happy-path POST, and status polling.
//
// Acceptance: claw ingest: flags audited, exit codes 0/1/2, integration tests pass.
// SPORT: F02-COMMAND-INVENTORY row: claw ingest

package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestClawIngestSourceType(t *testing.T) {
	cases := []struct {
		source    string
		flagValue string
		want      string
	}{
		{"https://example.com/article", "", "url"},
		{"http://example.com/page", "", "url"},
		{"notes.md", "", "text"},
		{"README.txt", "", "text"},
		{"report.pdf", "", "document"},
		{"binary.bin", "", "document"},
		{"notes.md", "document", "document"}, // flag overrides heuristic
		{"https://example.com", "text", "text"},
	}
	for _, tc := range cases {
		got := clawIngestSourceType(tc.source, tc.flagValue)
		if got != tc.want {
			t.Errorf("clawIngestSourceType(%q, %q) = %q, want %q", tc.source, tc.flagValue, got, tc.want)
		}
	}
}

func TestClawIngestCmd_FlagsRegistered(t *testing.T) {
	wantFlags := []string{"type", "topic", "tag", "status", "json"}
	for _, name := range wantFlags {
		if clawIngestCmd.Flags().Lookup(name) == nil {
			t.Errorf("claw ingest: expected flag --%s to be registered", name)
		}
	}
}

func TestClawIngest_InvalidType(t *testing.T) {
	orig := clawIngestType
	defer func() { clawIngestType = orig }()

	clawIngestType = "bogus"
	// Need a file arg to get past ExactArgs; provide a fake path since type
	// validation fires before the file read.
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Also set env so clawClient doesn't fail before type check.
	t.Setenv("NSELF_CLAW_SERVER", "http://127.0.0.1:19099")
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	err := runClawIngest(cmd, []string{"some-file.txt"})
	if err == nil {
		t.Fatal("expected error for invalid --type, got nil")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("expected error to mention 'bogus', got: %v", err)
	}
}

func TestClawIngest_HappyPath_URL(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/claw/v1/ingest" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "job-abc",
			"status": "queued",
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	origType := clawIngestType
	origTopic := clawIngestTopic
	origTag := clawIngestTag
	origStatus := clawIngestStatus
	origJSON := clawIngestJSON
	defer func() {
		clawIngestType = origType
		clawIngestTopic = origTopic
		clawIngestTag = origTag
		clawIngestStatus = origStatus
		clawIngestJSON = origJSON
	}()
	clawIngestType = ""
	clawIngestTopic = "tech"
	clawIngestTag = "article"
	clawIngestStatus = ""
	clawIngestJSON = false

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawIngest(cmd, []string{"https://example.com/article"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-abc") {
		t.Errorf("expected job ID 'job-abc' in output, got: %q", out)
	}
	if capturedBody["source_type"] != "url" {
		t.Errorf("expected source_type=url, got: %v", capturedBody["source_type"])
	}
	if capturedBody["topic"] != "tech" {
		t.Errorf("expected topic=tech, got: %v", capturedBody["topic"])
	}
}

func TestClawIngest_HappyPath_File(t *testing.T) {
	// Create a temp file to ingest
	dir := t.TempDir()
	fpath := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(fpath, []byte("# Meeting Notes\nKey point: ship it."), 0600); err != nil {
		t.Fatal(err)
	}

	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "job-xyz",
			"status": "queued",
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	origType := clawIngestType
	origStatus := clawIngestStatus
	origJSON := clawIngestJSON
	defer func() {
		clawIngestType = origType
		clawIngestStatus = origStatus
		clawIngestJSON = origJSON
	}()
	clawIngestType = ""
	clawIngestStatus = ""
	clawIngestJSON = false

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawIngest(cmd, []string{fpath}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "job-xyz") {
		t.Errorf("expected job ID 'job-xyz' in output, got: %q", out)
	}
	if capturedBody["source_type"] != "text" {
		t.Errorf("expected source_type=text for .md file, got: %v", capturedBody["source_type"])
	}
	content, _ := capturedBody["content"].(string)
	if !strings.Contains(content, "Meeting Notes") {
		t.Errorf("expected file content in payload, got content=%q", content)
	}
}

func TestClawIngest_Status_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/claw/v1/ingest/job-abc/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "job-abc",
			"status":   "complete",
			"progress": 100,
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	origStatus := clawIngestStatus
	origJSON := clawIngestJSON
	defer func() {
		clawIngestStatus = origStatus
		clawIngestJSON = origJSON
	}()
	clawIngestStatus = "job-abc"
	clawIngestJSON = false

	out := captureClawStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		// --status triggers status mode; positional arg is not used
		if err := runClawIngest(cmd, []string{"ignored"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(out, "complete") {
		t.Errorf("expected 'complete' status in output, got: %q", out)
	}
	if !strings.Contains(out, "100") {
		t.Errorf("expected progress 100 in output, got: %q", out)
	}
}

func TestClawIngest_MissingFile(t *testing.T) {
	t.Setenv("NSELF_CLAW_SERVER", "http://127.0.0.1:19099")
	t.Setenv("NSELF_CLAW_API_KEY", "test-key")

	origType := clawIngestType
	origStatus := clawIngestStatus
	defer func() {
		clawIngestType = origType
		clawIngestStatus = origStatus
	}()
	clawIngestType = ""
	clawIngestStatus = ""

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runClawIngest(cmd, []string{"/nonexistent/path/notes.md"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "reading file") {
		t.Errorf("expected 'reading file' in error, got: %v", err)
	}
}
