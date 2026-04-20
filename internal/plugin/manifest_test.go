package plugin

import (
	"strings"
	"testing"
)

// TestValidateManifest_ConsumesValid verifies that valid plugin names in
// Consumes pass manifest validation. S43-T17.
func TestValidateManifest_ConsumesValid(t *testing.T) {
	m := &PluginManifest{
		Name:        "my-plugin",
		Version:     "1.0.0",
		Description: "test",
		Category:    "utility",
		License:     "MIT",
		Consumes:    []string{"ai", "cron", "notify"},
		Provides:    []string{"my-plugin"},
	}
	if err := validateManifest(m); err != nil {
		t.Errorf("validateManifest with valid Consumes/Provides: unexpected error: %v", err)
	}
}

// TestValidateManifest_ConsumesInvalidName verifies that an invalid plugin
// name in Consumes is rejected with a descriptive error. S43-T17.
func TestValidateManifest_ConsumesInvalidName(t *testing.T) {
	m := &PluginManifest{
		Name:        "my-plugin",
		Version:     "1.0.0",
		Description: "test",
		Category:    "utility",
		License:     "MIT",
		Consumes:    []string{"INVALID_NAME"},
	}
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error for invalid Consumes name, got nil")
	}
	if !strings.Contains(err.Error(), "consumes entry") {
		t.Errorf("expected 'consumes entry' in error, got: %v", err)
	}
}

// TestValidateManifest_ProvidesInvalidName verifies that an invalid plugin
// name in Provides is rejected. S43-T17.
func TestValidateManifest_ProvidesInvalidName(t *testing.T) {
	m := &PluginManifest{
		Name:        "my-plugin",
		Version:     "1.0.0",
		Description: "test",
		Category:    "utility",
		License:     "MIT",
		Provides:    []string{"Bad Name!"},
	}
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error for invalid Provides name, got nil")
	}
	if !strings.Contains(err.Error(), "provides entry") {
		t.Errorf("expected 'provides entry' in error, got: %v", err)
	}
}

// TestVersionPattern_Valid verifies that a well-formed semver string matches.
func TestVersionPattern_Valid(t *testing.T) {
	if !versionPattern.MatchString("1.2.3") {
		t.Error("expected \"1.2.3\" to match versionPattern, but it did not")
	}
}

// TestVersionPattern_PreRelease verifies that pre-release suffixes are accepted.
func TestVersionPattern_PreRelease(t *testing.T) {
	if !versionPattern.MatchString("1.2.3-alpha") {
		t.Error("expected \"1.2.3-alpha\" to match versionPattern, but it did not")
	}
}

// TestVersionPattern_ShellInjection verifies that shell metacharacters are rejected.
func TestVersionPattern_ShellInjection(t *testing.T) {
	if versionPattern.MatchString("1.2.3; rm -rf /") {
		t.Error("expected \"1.2.3; rm -rf /\" to NOT match versionPattern, but it did")
	}
}

// TestVersionPattern_FourParts verifies that four-component version strings are rejected.
func TestVersionPattern_FourParts(t *testing.T) {
	if versionPattern.MatchString("1.2.3.4") {
		t.Error("expected \"1.2.3.4\" to NOT match versionPattern, but it did")
	}
}

// TestVersionPattern_ZeroVersion verifies that 0.0.0 is accepted as a valid semver string.
func TestVersionPattern_ZeroVersion(t *testing.T) {
	if !versionPattern.MatchString("0.0.0") {
		t.Error("expected \"0.0.0\" to match versionPattern, but it did not")
	}
}

// TestVersionPattern_LargeNumbers verifies that large numeric components are accepted.
func TestVersionPattern_LargeNumbers(t *testing.T) {
	if !versionPattern.MatchString("100.200.300") {
		t.Error("expected \"100.200.300\" to match versionPattern, but it did not")
	}
}
