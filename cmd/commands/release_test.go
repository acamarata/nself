package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecWebVersionBumps_RejectsInjection verifies that a version string
// containing shell metacharacters is rejected before any exec.Command is called.
// The semver gate must fire — no shell output must result.
func TestExecWebVersionBumps_RejectsInjection(t *testing.T) {
	ctx := context.Background()
	injectionVers := []string{
		"1.0.0;echo INJECTED",
		"1.0.0 && echo INJECTED",
		"$(rm -rf /)",
		"`echo bad`",
		"1.0.0|cat /etc/passwd",
		"../../../etc/passwd",
		"",
		"notasemver",
		"1.2",
		"1.2.3.4",
	}
	for _, ver := range injectionVers {
		err := execWebVersionBumps(ctx, ver)
		if err == nil {
			t.Errorf("execWebVersionBumps(%q): expected error (semver rejection), got nil", ver)
		}
		if !strings.Contains(err.Error(), "invalid version") {
			t.Errorf("execWebVersionBumps(%q): expected 'invalid version' in error, got: %v", ver, err)
		}
	}
}

// TestExecWebVersionBumps_AcceptsValidSemver verifies that valid semver strings
// pass the semver gate. The function may fail later (node not found or web/ absent),
// but the semver gate itself must not reject valid input.
func TestExecWebVersionBumps_AcceptsValidSemver(t *testing.T) {
	ctx := context.Background()
	validVers := []string{
		"1.0.0",
		"1.2.3",
		"0.0.1",
		"10.20.30",
		"1.0.0-rc.1",
		"1.2.3-alpha",
		"1.2.3-beta.2",
	}
	for _, ver := range validVers {
		err := execWebVersionBumps(ctx, ver)
		// The semver gate must NOT reject these. A downstream error (no node, no web/ dir) is OK.
		if err != nil && strings.Contains(err.Error(), "invalid version") {
			t.Errorf("execWebVersionBumps(%q): valid semver incorrectly rejected: %v", ver, err)
		}
	}
}

// TestExecReadmeBadges_RejectsInjection verifies that malformed version strings
// are rejected by execReadmeBadges before any file operation.
func TestExecReadmeBadges_RejectsInjection(t *testing.T) {
	ctx := context.Background()
	injectionVers := []string{
		"1.0.0;echo INJECTED",
		"$(rm -rf /)",
		"notasemver",
		"",
		"1.2",
	}
	for _, ver := range injectionVers {
		err := execReadmeBadges(ctx, ver)
		if err == nil {
			t.Errorf("execReadmeBadges(%q): expected error (semver rejection), got nil", ver)
		}
		if !strings.Contains(err.Error(), "invalid version") {
			t.Errorf("execReadmeBadges(%q): expected 'invalid version' in error, got: %v", ver, err)
		}
	}
}

// TestExecReadmeBadges_UpdatesBadge verifies the Go-native badge replacement logic.
// Creates a temp README, runs execReadmeBadges against it, and checks the result.
func TestExecReadmeBadges_UpdatesBadge(t *testing.T) {
	// Create a temporary directory tree mirroring ../<repo>/README.md
	tmp := t.TempDir()
	// We'll place a "cli" README one level up from a fake working dir
	// The function uses fmt.Sprintf("../%s/README.md", r) from cwd.
	// For testing we need to cd into a subdir of tmp.
	repoDir := filepath.Join(tmp, "cli")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	readmePath := filepath.Join(repoDir, "README.md")
	content := "[![Version](https://img.shields.io/badge/version-1.0.0-blue)](https://nself.org)\n"
	if err := os.WriteFile(readmePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	// Change cwd into a "fake/child" so that "../cli/README.md" resolves correctly
	childDir := filepath.Join(tmp, "fakecwd")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("mkdir childDir: %v", err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(childDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	ctx := context.Background()
	if err := execReadmeBadges(ctx, "1.2.3"); err != nil {
		// May fail on repos other than "cli" that don't exist — that's fine; they are skipped.
		// A real failure would be on the "cli" README we created.
		t.Fatalf("execReadmeBadges: %v", err)
	}

	got, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if !strings.Contains(string(got), "version-1.2.3-") {
		t.Errorf("badge not updated: got %q", string(got))
	}
	if strings.Contains(string(got), "version-1.0.0-") {
		t.Errorf("old badge still present: got %q", string(got))
	}
}

// TestReleaseVerRe verifies the anchored semver regexp used by execWebVersionBumps
// and execReadmeBadges directly.
func TestReleaseVerRe(t *testing.T) {
	valid := []string{"0.0.0", "1.0.0", "10.20.30", "1.0.0-rc.1", "1.2.3-beta.2", "1.2.3-alpha"}
	for _, v := range valid {
		if !releaseVerRe.MatchString(v) {
			t.Errorf("releaseVerRe should accept %q", v)
		}
	}
	invalid := []string{"", "1", "1.2", "1.2.3.4", "v1.2.3", "1.2.3;echo", "$(x)", "1.2.3 "}
	for _, v := range invalid {
		if releaseVerRe.MatchString(v) {
			t.Errorf("releaseVerRe should reject %q", v)
		}
	}
}
