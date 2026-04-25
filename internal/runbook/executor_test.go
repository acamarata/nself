package runbook

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var _ = io.Discard // keep import

func TestExtractFrontMatter(t *testing.T) {
	input := []byte("---\nid: test\ntitle: Test\n---\n\n# Body\n")
	got, err := extractFrontMatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), "id: test") {
		t.Errorf("expected front matter to contain id: test, got %q", got)
	}
}

func TestExtractFrontMatter_NoDelimiter(t *testing.T) {
	_, err := extractFrontMatter([]byte("no front matter here"))
	if err == nil {
		t.Error("expected error for missing front matter")
	}
}

func TestExpandParams(t *testing.T) {
	params := map[string]string{
		"message": "Restarted {{ service }}",
		"channel": "#nself-alerts",
	}
	vars := map[string]string{"service": "ai"}
	out := expandParams(params, vars)
	if out["message"] != "Restarted ai" {
		t.Errorf("expected 'Restarted ai', got %q", out["message"])
	}
	if out["channel"] != "#nself-alerts" {
		t.Errorf("channel should be unchanged, got %q", out["channel"])
	}
}

func TestExecutor_DryRun(t *testing.T) {
	dir := t.TempDir()
	// Write a minimal runbook
	content := `---
id: test-runbook
title: Test Runbook
trigger:
  alert: TestAlert
  severity: critical
steps:
  - action: check_logs
    params:
      service: ai
      last: 5m
    requires_confirmation: false
  - action: restart_service
    params:
      service: ai
    requires_confirmation: true
---

# Test Runbook
`
	if err := os.WriteFile(filepath.Join(dir, "test-runbook.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	ex := &Executor{
		RunbookDir: dir,
		DryRun:     true,
		Out:        &buf,
	}

	if err := ex.Execute(context.Background(), "test-runbook", nil); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Error("expected DRY RUN in output")
	}
	if !strings.Contains(output, "test-runbook") {
		t.Error("expected runbook ID in output")
	}
	if !strings.Contains(output, "dry-run") {
		t.Error("expected dry-run step marker in output")
	}
}

func TestExecutor_List(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.md", "b.md", "c.txt"} {
		content := "---\nid: " + strings.TrimSuffix(name, ".md") + "\ntitle: T\n---\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	ex := &Executor{RunbookDir: dir, Out: bytes.NewBuffer(nil)}
	books, err := ex.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(books) != 2 {
		t.Errorf("expected 2 runbooks (.txt excluded), got %d", len(books))
	}
}

func TestExecutor_NotFound(t *testing.T) {
	dir := t.TempDir()
	ex := &Executor{RunbookDir: dir, Out: bytes.NewBuffer(nil)}
	err := ex.Execute(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent runbook ID")
	}
}
