package version

import (
	"testing"
)

// TestGetVersion_NonEmpty verifies that GetVersion returns a non-empty string.
func TestGetVersion_NonEmpty(t *testing.T) {
	v := GetVersion()
	if v == "" {
		t.Error("GetVersion returned empty string — Version var must be set")
	}
}

// TestGetCommit_NonEmpty verifies that GetCommit returns a non-empty string.
func TestGetCommit_NonEmpty(t *testing.T) {
	c := GetCommit()
	if c == "" {
		t.Error("GetCommit returned empty string — Commit var must be set")
	}
}

// TestGetBuildDate_NonEmpty verifies that GetBuildDate returns a non-empty string.
func TestGetBuildDate_NonEmpty(t *testing.T) {
	d := GetBuildDate()
	if d == "" {
		t.Error("GetBuildDate returned empty string — BuildDate var must be set")
	}
}

// TestVersion_Overridable verifies that the package-level vars can be overridden
// (simulates what ldflags does at build time).
func TestVersion_Overridable(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()

	Version = "9.9.9"
	if GetVersion() != "9.9.9" {
		t.Errorf("expected GetVersion() = 9.9.9 after override, got %q", GetVersion())
	}
}

// TestCommit_Overridable verifies Commit override.
func TestCommit_Overridable(t *testing.T) {
	orig := Commit
	defer func() { Commit = orig }()

	Commit = "deadbeef"
	if GetCommit() != "deadbeef" {
		t.Errorf("expected GetCommit() = deadbeef after override, got %q", GetCommit())
	}
}

// TestBuildDate_Overridable verifies BuildDate override.
func TestBuildDate_Overridable(t *testing.T) {
	orig := BuildDate
	defer func() { BuildDate = orig }()

	BuildDate = "2026-01-01"
	if GetBuildDate() != "2026-01-01" {
		t.Errorf("expected GetBuildDate() = 2026-01-01 after override, got %q", GetBuildDate())
	}
}
