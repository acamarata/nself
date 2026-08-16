package plugin

import (
	"strings"
	"testing"
)

// baseManifest returns a valid minimal PluginManifest for use in tests.
func baseManifest() *PluginManifest {
	return &PluginManifest{
		Name:        "my-plugin",
		Version:     "1.0.0",
		Description: "test plugin",
		Category:    "utility",
		License:     "MIT",
	}
}

// --- S58-T01: TestParseStatus — 6 valid values + 1 invalid ---

func TestParseStatus_Experimental(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "experimental"
	if err := validateManifest(m); err != nil {
		t.Errorf("expected experimental to be valid, got: %v", err)
	}
}

func TestParseStatus_Planned(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "planned"
	if err := validateManifest(m); err != nil {
		t.Errorf("expected planned to be valid, got: %v", err)
	}
}

func TestParseStatus_Beta(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "beta"
	if err := validateManifest(m); err != nil {
		t.Errorf("expected beta to be valid, got: %v", err)
	}
}

func TestParseStatus_Stable(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "stable"
	if err := validateManifest(m); err != nil {
		t.Errorf("expected stable to be valid, got: %v", err)
	}
}

func TestParseStatus_Deprecated(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "deprecated"
	m.Deprecation = &DeprecationBlock{
		AnnouncedDate:  "2026-04-01",
		EOLDate:        "2027-04-01",
		MigrationGuide: "https://nself.org/docs/migrate/my-plugin",
	}
	if err := validateManifest(m); err != nil {
		t.Errorf("expected deprecated with full block to be valid, got: %v", err)
	}
}

func TestParseStatus_EOL(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "eol"
	if err := validateManifest(m); err != nil {
		t.Errorf("expected eol to be valid, got: %v", err)
	}
}

func TestParseStatus_Invalid(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "removed" // 7th value — must be rejected
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error for invalid status 'removed', got nil")
	}
	if !strings.Contains(err.Error(), "invalid plugin status") {
		t.Errorf("expected 'invalid plugin status' in error, got: %v", err)
	}
}

// TestParseStatus_AbsentDefaultsToStable verifies that a missing status field
// does not trigger an error (backwards-compatible default). (S58-T01 CR-A)
func TestParseStatus_AbsentDefaultsToStable(t *testing.T) {
	m := baseManifest()
	// PublishStatus intentionally left empty
	if err := validateManifest(m); err != nil {
		t.Errorf("expected absent status to be valid (defaults to stable), got: %v", err)
	}
}

// --- S58-T02: Deprecation block validation tests ---

func TestDeprecationBlock_MissingWhenDeprecated(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "deprecated"
	// No deprecation block
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error when deprecated status has no deprecation block")
	}
	if !strings.Contains(err.Error(), "deprecation block") {
		t.Errorf("expected 'deprecation block' in error, got: %v", err)
	}
}

func TestDeprecationBlock_MissingRequiredFields(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "deprecated"
	m.Deprecation = &DeprecationBlock{
		// announcedDate and migrationGuide missing
		EOLDate: "2027-04-01",
	}
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error when deprecation block is missing required fields")
	}
}

func TestDeprecationBlock_ReplacedByPromptFires(t *testing.T) {
	m := baseManifest()
	m.PublishStatus = "deprecated"
	m.Deprecation = &DeprecationBlock{
		AnnouncedDate:  "2026-04-01",
		EOLDate:        "2027-04-01",
		MigrationGuide: "https://nself.org/docs/migrate/my-plugin",
		ReplacedBy:     "my-plugin-v2",
	}
	if err := validateManifest(m); err != nil {
		t.Errorf("valid deprecated manifest with replacedBy should pass: %v", err)
	}
	// Verify replacedBy is accessible for UI prompt logic.
	if m.Deprecation.ReplacedBy != "my-plugin-v2" {
		t.Errorf("expected replacedBy='my-plugin-v2', got %q", m.Deprecation.ReplacedBy)
	}
}

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

// --- S71-T01: Permission validation tests ---

// TestValidatePermissions_ValidExact verifies that exact canonical permissions pass.
func TestValidatePermissions_ValidExact(t *testing.T) {
	m := baseManifest()
	m.Permissions = []string{"network:internet", "db:read", "db:write", "system:exec"}
	if err := validateManifest(m); err != nil {
		t.Errorf("expected valid exact permissions to pass, got: %v", err)
	}
}

// TestValidatePermissions_ValidParameterized verifies that parameterized permission
// forms with valid params pass (e.g. network:plugin:ai, secrets:env:MY_KEY).
func TestValidatePermissions_ValidParameterized(t *testing.T) {
	m := baseManifest()
	m.Permissions = []string{
		"network:plugin:ai",
		"secrets:env:OPENAI_API_KEY",
		"fs:write:media-data",
		"ai:provider:openai",
	}
	if err := validateManifest(m); err != nil {
		t.Errorf("expected valid parameterized permissions to pass, got: %v", err)
	}
}

// TestValidatePermissions_UnknownPermission verifies that an unknown permission
// string is rejected with a descriptive error (fail-closed semantics). S71-T01.
func TestValidatePermissions_UnknownPermission(t *testing.T) {
	m := baseManifest()
	m.Permissions = []string{"network:internet", "made-up-perm"}
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error for unknown permission 'made-up-perm', got nil")
	}
	if !strings.Contains(err.Error(), "unknown permission") {
		t.Errorf("expected 'unknown permission' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "made-up-perm") {
		t.Errorf("expected permission name in error, got: %v", err)
	}
}

// TestValidatePermissions_MultipleUnknown verifies that all unknown permissions are
// reported in a single error (not just the first one). S71-T01.
func TestValidatePermissions_MultipleUnknown(t *testing.T) {
	m := baseManifest()
	m.Permissions = []string{"fake-a", "fake-b"}
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error for multiple unknown permissions, got nil")
	}
	if !strings.Contains(err.Error(), "fake-a") || !strings.Contains(err.Error(), "fake-b") {
		t.Errorf("expected both unknown perms in error, got: %v", err)
	}
}

// TestValidatePermissions_EmptyList verifies that a manifest with no permissions
// passes validation without error. S71-T01.
func TestValidatePermissions_EmptyList(t *testing.T) {
	m := baseManifest()
	m.Permissions = []string{}
	if err := validateManifest(m); err != nil {
		t.Errorf("expected empty permissions list to pass, got: %v", err)
	}
}

// TestValidatePermissions_NilList verifies that a manifest with nil permissions
// passes validation (field is optional). S71-T01.
func TestValidatePermissions_NilList(t *testing.T) {
	m := baseManifest()
	// Permissions nil by default from baseManifest
	if err := validateManifest(m); err != nil {
		t.Errorf("expected nil permissions to pass, got: %v", err)
	}
}

// TestValidatePermissions_ParameterizedMissingParam verifies that a parameterized
// permission with no param segment (e.g. "network:plugin" alone) is rejected.
func TestValidatePermissions_ParameterizedMissingParam(t *testing.T) {
	m := baseManifest()
	m.Permissions = []string{"network:plugin"} // no :<name> suffix
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error for 'network:plugin' without param, got nil")
	}
	if !strings.Contains(err.Error(), "unknown permission") {
		t.Errorf("expected 'unknown permission' in error, got: %v", err)
	}
}

// TestValidatePermissions_ShellInjectionInParam verifies that a permission param
// containing shell-dangerous characters is rejected. S71-T01 security gate.
func TestValidatePermissions_ShellInjectionInParam(t *testing.T) {
	m := baseManifest()
	m.Permissions = []string{"secrets:env:MY_KEY; rm -rf /"}
	err := validateManifest(m)
	if err == nil {
		t.Fatal("expected error for shell injection in permission param, got nil")
	}
}
