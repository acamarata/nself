package eval

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleResult() EvalRunResult {
	return EvalRunResult{
		SchemaVer:  expectedSchemaVer,
		ID:         "run-test-123",
		SuiteSlug:  "recall-quality-v1",
		Status:     "passed",
		PassRate:   0.9,
		SuiteScore: 0.88,
		Passed:     true,
		Tasks: []EvalTaskResult{
			{TaskID: "t1", Input: "query", Output: "answer", Score: 0.95, Passed: true, ScoringMode: "semantic"},
			{TaskID: "t2", Input: "q2", Output: "a2", Score: 0.60, Passed: false, ScoringMode: "exact"},
		},
	}
}

func TestWriteResultJSON(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatJSON, dir); err != nil {
		t.Fatalf("WriteResult JSON error: %v", err)
	}
	// Verify JSON output is valid.
	var decoded EvalRunResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output not valid: %v\noutput: %s", err, buf.String())
	}
	if decoded.SuiteSlug != "recall-quality-v1" {
		t.Errorf("unexpected suite slug: %q", decoded.SuiteSlug)
	}
}

func TestWriteResultTable(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatTable, dir); err != nil {
		t.Fatalf("WriteResult table error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "recall-quality-v1") {
		t.Error("table output missing suite slug")
	}
	if !strings.Contains(out, "PASSED") {
		t.Error("table output missing PASSED status")
	}
}

func TestWriteResultSummary(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatSummary, dir); err != nil {
		t.Fatalf("WriteResult summary error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "recall-quality-v1") {
		t.Error("summary output missing suite slug")
	}
	if !strings.Contains(out, "90.0%") {
		t.Error("summary output missing pass rate")
	}
}

func TestArtifactWrittenToDisk(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleResult(), FormatSummary, dir); err != nil {
		t.Fatalf("WriteResult error: %v", err)
	}
	artifactPath := filepath.Join(dir, artifactsDir, artifactFile)
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("artifact not written at %s: %v", artifactPath, err)
	}
	var decoded EvalRunResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("artifact JSON invalid: %v", err)
	}
	if decoded.ID != "run-test-123" {
		t.Errorf("unexpected artifact run ID: %q", decoded.ID)
	}
}

func TestWriteGateStatusCleared(t *testing.T) {
	var buf bytes.Buffer
	WriteGateStatus(&buf, GateStatus{Tier: "semi-auto", Cleared: true, Enforced: true}, true)
	out := buf.String()
	if !strings.Contains(out, "CLEARED") {
		t.Error("expected CLEARED in output")
	}
	if !strings.Contains(out, "semi-auto") {
		t.Error("expected tier in output")
	}
}

func TestWriteGateStatusBlocked(t *testing.T) {
	var buf bytes.Buffer
	WriteGateStatus(&buf, GateStatus{
		Tier:           "full-auto",
		Cleared:        false,
		Enforced:       true,
		BlockingSuites: []string{"recall-quality-v1", "generation-v1"},
	}, true)
	out := buf.String()
	if !strings.Contains(out, "BLOCKED") {
		t.Error("expected BLOCKED in output")
	}
	if !strings.Contains(out, "recall-quality-v1") {
		t.Error("expected blocking suite in output")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 10) != "hello" {
		t.Error("expected no truncation for short string")
	}
	result := truncate("123456789012345678901234567890", 10)
	if len([]rune(result)) != 10 {
		t.Errorf("expected length 10, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("expected ... suffix on truncated string")
	}
}
