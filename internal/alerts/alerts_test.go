package alerts

import (
	"strings"
	"testing"
)

// TestDefaultRules_Count verifies 16+ default rules are returned.
func TestDefaultRules_Count(t *testing.T) {
	rules := DefaultRules()
	if len(rules) < 16 {
		t.Errorf("expected ≥16 default rules, got %d", len(rules))
	}
}

// TestDefaultRules_AllFieldsPresent verifies every rule has Name, Severity,
// Expr, and Summary set.
func TestDefaultRules_AllFieldsPresent(t *testing.T) {
	for _, r := range DefaultRules() {
		if r.Name == "" {
			t.Errorf("rule has empty Name: %+v", r)
		}
		if r.Severity == "" {
			t.Errorf("rule %q has empty Severity", r.Name)
		}
		if r.Expr == "" {
			t.Errorf("rule %q has empty Expr", r.Name)
		}
		if r.Summary == "" {
			t.Errorf("rule %q has empty Summary", r.Name)
		}
	}
}

// TestDefaultRules_SeverityValues verifies all rules use valid severity levels.
func TestDefaultRules_SeverityValues(t *testing.T) {
	valid := map[string]bool{
		SeverityP1: true,
		SeverityP2: true,
		SeverityP3: true,
	}
	for _, r := range DefaultRules() {
		if !valid[r.Severity] {
			t.Errorf("rule %q has unknown severity %q", r.Name, r.Severity)
		}
	}
}

// TestSeverityConstants_NonEmpty verifies P1/P2/P3 constants are non-empty and distinct.
func TestSeverityConstants_NonEmpty(t *testing.T) {
	if SeverityP1 == "" {
		t.Error("SeverityP1 is empty")
	}
	if SeverityP2 == "" {
		t.Error("SeverityP2 is empty")
	}
	if SeverityP3 == "" {
		t.Error("SeverityP3 is empty")
	}
	if SeverityP1 == SeverityP2 || SeverityP1 == SeverityP3 || SeverityP2 == SeverityP3 {
		t.Error("severity constants are not distinct")
	}
}

// TestList_NoFilter verifies that List("") returns all default rules.
func TestList_NoFilter(t *testing.T) {
	all := DefaultRules()
	got := List("")
	if len(got) != len(all) {
		t.Errorf("List(\"\") returned %d rules, want %d", len(got), len(all))
	}
}

// TestList_FilterByP1 verifies that filtering by P1 returns only P1 rules.
func TestList_FilterByP1(t *testing.T) {
	rules := List(SeverityP1)
	if len(rules) == 0 {
		t.Error("List(P1) returned no rules — expected at least one P1 rule")
	}
	for _, r := range rules {
		if r.Severity != SeverityP1 {
			t.Errorf("List(P1) returned non-P1 rule: %q severity=%q", r.Name, r.Severity)
		}
	}
}

// TestList_FilterByP2 verifies P2 filtering.
func TestList_FilterByP2(t *testing.T) {
	rules := List(SeverityP2)
	if len(rules) == 0 {
		t.Error("List(P2) returned no rules")
	}
	for _, r := range rules {
		if r.Severity != SeverityP2 {
			t.Errorf("List(P2) returned non-P2 rule: %q severity=%q", r.Name, r.Severity)
		}
	}
}

// TestList_UnknownSeverity verifies that filtering by an unknown severity
// returns an empty list (not an error, not all rules).
func TestList_UnknownSeverity(t *testing.T) {
	rules := List("P99")
	if len(rules) != 0 {
		t.Errorf("List(P99) should return empty list, got %d rules", len(rules))
	}
}

// TestDefaultRules_BackupFailed verifies the critical BackupFailed rule exists.
func TestDefaultRules_BackupFailed(t *testing.T) {
	found := false
	for _, r := range DefaultRules() {
		if r.Name == "BackupFailed" {
			found = true
			if r.Severity != SeverityP1 {
				t.Errorf("BackupFailed should be P1, got %q", r.Severity)
			}
			if !strings.Contains(r.Expr, "nself_backup") {
				t.Errorf("BackupFailed Expr should reference nself_backup: %q", r.Expr)
			}
		}
	}
	if !found {
		t.Error("BackupFailed rule not found in DefaultRules()")
	}
}
