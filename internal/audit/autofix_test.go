package audit

import (
	"strings"
	"testing"
)

func TestPRDescription(t *testing.T) {
	r := &DocReport{
		Timestamp: "2026-04-17T00:00:00Z",
		Quarter:   "2026-Q2",
		FilesScan: 12,
		Summary: Summary{
			Total:       3,
			BannedWords: 2,
			DeadLinks:   1,
		},
	}
	modified := []string{"b.md", "a.md"}
	body := PRDescription(r, modified, ".claude/qa/findings/2026-Q2-doc-audit.json")

	if !strings.Contains(body, "2026-Q2") {
		t.Errorf("missing quarter: %s", body)
	}
	if !strings.Contains(body, "| banned_word | 2 |") {
		t.Errorf("missing banned_word row: %s", body)
	}
	if !strings.Contains(body, "a.md") || !strings.Contains(body, "b.md") {
		t.Errorf("missing modified files: %s", body)
	}
	// Files should be sorted.
	aIdx := strings.Index(body, "a.md")
	bIdx := strings.Index(body, "b.md")
	if aIdx <= 0 || bIdx <= 0 || aIdx > bIdx {
		t.Errorf("modified files not alphabetized: a=%d b=%d", aIdx, bIdx)
	}
}

func TestIsSafeAutoFix(t *testing.T) {
	if !IsSafeAutoFix(Finding{AutoFix: true, Category: "banned_word"}) {
		t.Error("banned_word auto-fix should be safe")
	}
	if IsSafeAutoFix(Finding{AutoFix: true, Category: "dead_link"}) {
		t.Error("dead_link auto-fix should not be silent-safe")
	}
	if IsSafeAutoFix(Finding{AutoFix: false, Category: "banned_word"}) {
		t.Error("non-autofix finding should not be safe")
	}
}
