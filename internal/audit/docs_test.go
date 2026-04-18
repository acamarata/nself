package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestQuarterOf(t *testing.T) {
	cases := []struct {
		when time.Time
		want string
	}{
		{time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC), "2026-Q1"},
		{time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), "2026-Q2"},
		{time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), "2026-Q3"},
		{time.Date(2026, time.October, 31, 23, 0, 0, 0, time.UTC), "2026-Q4"},
		{time.Date(2026, time.December, 31, 23, 0, 0, 0, time.UTC), "2026-Q4"},
	}
	for _, c := range cases {
		if got := quarterOf(c.when); got != c.want {
			t.Errorf("quarterOf(%v) = %q, want %q", c.when, got, c.want)
		}
	}
}

func TestRunDocs_BannedWord(t *testing.T) {
	root := t.TempDir()
	// Required anchors so we don't pollute the test with missing-file findings.
	for _, anchor := range RequiredAnchors {
		writeFile(t, filepath.Join(root, anchor), "# placeholder\n")
	}
	writeFile(t, filepath.Join(root, ".claude/docs/sample.md"),
		"# Sample\n\nThis is a powerful tool.\n")

	r, err := RunDocs(Options{Root: root, Quarter: "2026-Q3"})
	if err != nil {
		t.Fatalf("RunDocs: %v", err)
	}
	if r.Quarter != "2026-Q3" {
		t.Errorf("quarter: got %q, want 2026-Q3", r.Quarter)
	}
	found := false
	for _, f := range r.Findings {
		if f.Category == "banned_word" && f.Match == "powerful" {
			found = true
			if !f.AutoFix {
				t.Errorf("expected banned_word to be auto-fixable")
			}
		}
	}
	if !found {
		t.Errorf("expected banned_word finding for 'powerful'; got %+v", r.Findings)
	}
}

func TestRunDocs_BannedWordIgnoresCodeBlocks(t *testing.T) {
	root := t.TempDir()
	for _, anchor := range RequiredAnchors {
		writeFile(t, filepath.Join(root, anchor), "# placeholder\n")
	}
	writeFile(t, filepath.Join(root, ".claude/docs/code.md"),
		"# Code sample\n\n```bash\necho \"this is a robust example\"\n```\n")

	r, err := RunDocs(Options{Root: root})
	if err != nil {
		t.Fatalf("RunDocs: %v", err)
	}
	for _, f := range r.Findings {
		if f.Category == "banned_word" {
			t.Errorf("banned_word found inside code fence: %+v", f)
		}
	}
}

func TestRunDocs_DeadLink(t *testing.T) {
	root := t.TempDir()
	for _, anchor := range RequiredAnchors {
		writeFile(t, filepath.Join(root, anchor), "# placeholder\n")
	}
	writeFile(t, filepath.Join(root, ".claude/docs/link.md"),
		"# Link\nSee [missing](./does-not-exist.md).\n")

	r, err := RunDocs(Options{Root: root})
	if err != nil {
		t.Fatalf("RunDocs: %v", err)
	}
	hit := false
	for _, f := range r.Findings {
		if f.Category == "dead_link" && strings.Contains(f.Match, "does-not-exist") {
			hit = true
		}
	}
	if !hit {
		t.Errorf("expected dead_link finding; got %+v", r.Findings)
	}
}

func TestRunDocs_MissingAnchors(t *testing.T) {
	root := t.TempDir()
	// Only create one anchor → others should be flagged.
	writeFile(t, filepath.Join(root, ".claude/CLAUDE.md"), "# placeholder\n")

	r, err := RunDocs(Options{Root: root})
	if err != nil {
		t.Fatalf("RunDocs: %v", err)
	}
	missingCount := 0
	for _, f := range r.Findings {
		if f.Category == "missing_file" && f.Severity == "high" {
			missingCount++
		}
	}
	// 10 required anchors minus the one we created = 9 expected.
	if missingCount < 9 {
		t.Errorf("expected >=9 missing anchors, got %d", missingCount)
	}
}

func TestApplyAutoFix_LeverageToUse(t *testing.T) {
	root := t.TempDir()
	for _, anchor := range RequiredAnchors {
		writeFile(t, filepath.Join(root, anchor), "# placeholder\n")
	}
	target := filepath.Join(root, ".claude/docs/leverage.md")
	writeFile(t, target, "# Title\n\nWe leverage the API for this.\n")

	r, err := RunDocs(Options{Root: root})
	if err != nil {
		t.Fatalf("RunDocs: %v", err)
	}
	modified, err := ApplyAutoFix(root, r)
	if err != nil {
		t.Fatalf("ApplyAutoFix: %v", err)
	}
	if len(modified) != 1 {
		t.Fatalf("expected 1 modified file, got %v", modified)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	got := string(data)
	if strings.Contains(strings.ToLower(got), "leverage") {
		t.Errorf("'leverage' should have been replaced: %q", got)
	}
	if !strings.Contains(got, "use") {
		t.Errorf("'use' should be present: %q", got)
	}
}

func TestReplaceWholeWord_DropsFiller(t *testing.T) {
	got := replaceWholeWord("Moreover, we ship.", "moreover", safeReplacement("moreover"))
	// The replacement is empty; we accept either ", we ship." or " , we ship."
	// as long as the word is gone and the sentence remains readable.
	if strings.Contains(strings.ToLower(got), "moreover") {
		t.Errorf("moreover still present: %q", got)
	}
}

func TestWriteJSON(t *testing.T) {
	r := &DocReport{
		Timestamp: "2026-04-17T00:00:00Z",
		Quarter:   "2026-Q2",
		Findings:  []Finding{{Category: "banned_word", File: "x.md"}},
	}
	var sb strings.Builder
	if err := r.WriteJSON(&sb); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !strings.Contains(sb.String(), "2026-Q2") {
		t.Errorf("JSON missing quarter: %s", sb.String())
	}
}
