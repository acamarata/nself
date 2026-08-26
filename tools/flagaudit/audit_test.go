// Purpose:     prove the audit logic itself still catches drift, independent
//
//	of the real cobra tree — a fixture doc with one known-good and
//	one known-bad invocation, checked against a small fixture
//	tree, so a regression in scan.go or audit.go fails a test
//	instead of silently passing every future CI run.
//
// Inputs:      none (self-contained fixtures).
// Outputs:     t.Fatal on any mismatch.
// Constraints: does not import cmd/commands — the fixture tree is built by
//
//	hand so this test has no dependency on any real command ever
//	existing or keeping its current flags.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// fixtureTree mirrors the shape buildTree produces, by hand:
//
//	nself start            --timeout, --skip-health-checks
//	nself ssl               (no local flags)
//	nself ssl setup         --wildcard
func fixtureTree() *commandNode {
	root := &commandNode{Path: "", Flags: map[string]bool{"help": true}, Children: map[string]*commandNode{}}

	start := &commandNode{
		Path:     "start",
		Flags:    map[string]bool{"help": true, "timeout": true, "skip-health-checks": true},
		Children: map[string]*commandNode{},
	}
	root.Children["start"] = start

	ssl := &commandNode{Path: "ssl", Flags: map[string]bool{"help": true}, Children: map[string]*commandNode{}}
	sslSetup := &commandNode{Path: "ssl setup", Flags: map[string]bool{"help": true, "wildcard": true}, Children: map[string]*commandNode{}}
	ssl.Children["setup"] = sslSetup
	root.Children["ssl"] = ssl

	return root
}

func TestAudit_KnownGoodAndKnownBad(t *testing.T) {
	root := fixtureTree()

	dir := t.TempDir()
	fixture := filepath.Join(dir, "fixture.md")
	content := "" +
		"Run `nself ssl setup --wildcard` to provision a wildcard cert.\n" +
		"Then `nself start --wait-healthy` to bring the stack up.\n"
	if err := os.WriteFile(fixture, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	invs, err := scanFile(fixture)
	if err != nil {
		t.Fatalf("scanFile: %v", err)
	}
	if len(invs) != 2 {
		t.Fatalf("expected 2 invocations, got %d: %+v", len(invs), invs)
	}

	result := audit(root, invs)

	if len(result.Findings) != 1 {
		t.Fatalf("expected exactly 1 finding (--wait-healthy), got %d: %+v", len(result.Findings), result.Findings)
	}
	got := result.Findings[0]
	if got.Flag != "wait-healthy" || got.Command != "nself start" {
		t.Fatalf("unexpected finding: %+v", got)
	}
	if got.Line != 2 {
		t.Fatalf("expected finding on line 2, got line %d", got.Line)
	}

	if len(result.Skips) != 0 {
		t.Fatalf("expected no skips, got %+v", result.Skips)
	}
}

func TestAudit_UnknownTopLevelCommandIsSkippedNotFailed(t *testing.T) {
	root := fixtureTree()

	invs := []invocation{
		{File: "doc.md", Line: 5, Raw: "nself region list --format json", PathTokens: []string{"region", "list"}, Flags: []string{"format"}},
	}

	result := audit(root, invs)

	if len(result.Findings) != 0 {
		t.Fatalf("plugin-provided command must never produce a finding, got %+v", result.Findings)
	}
	if len(result.Skips) != 1 || result.Skips[0].Command != "region" {
		t.Fatalf("expected one skip for 'region', got %+v", result.Skips)
	}
}

func TestAudit_HelpAndVersionAlwaysIgnored(t *testing.T) {
	root := fixtureTree()

	invs := []invocation{
		{File: "doc.md", Line: 1, Raw: "nself start --help", PathTokens: []string{"start"}, Flags: []string{"help"}},
		{File: "doc.md", Line: 2, Raw: "nself --version", PathTokens: nil, Flags: []string{"version"}},
	}

	result := audit(root, invs)

	if len(result.Findings) != 0 {
		t.Fatalf("--help/--version must never be flagged, got %+v", result.Findings)
	}
}

func TestParseInvocation_IgnoresBareDomainMentions(t *testing.T) {
	cases := []string{
		"curl -fsSL https://install.nself.org | bash",
		"See https://cloud.nself.org/account/offline-license for details.",
		"`nself.org` is the marketing site.",
	}
	for _, line := range cases {
		if _, ok := parseInvocation("doc.md", 1, line); ok {
			t.Fatalf("expected no invocation parsed from %q", line)
		}
	}
}

func TestParseInvocation_NestedInSSHHeredoc(t *testing.T) {
	inv, ok := parseInvocation("script.sh", 60, "nself start --wait-healthy")
	if !ok {
		t.Fatalf("expected an invocation to be parsed")
	}
	if len(inv.PathTokens) != 1 || inv.PathTokens[0] != "start" {
		t.Fatalf("unexpected path tokens: %+v", inv.PathTokens)
	}
	if len(inv.Flags) != 1 || inv.Flags[0] != "wait-healthy" {
		t.Fatalf("unexpected flags: %+v", inv.Flags)
	}
}
