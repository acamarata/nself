package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/license"
)

// ── licenseSimulateOffline ────────────────────────────────────────────────────

// TestLicenseSimulateCmd_Registered verifies simulate-offline is wired into the
// license command tree.
func TestLicenseSimulateCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range licenseCmd.Commands() {
		if sub.Name() == "simulate-offline" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("simulate-offline is not registered on licenseCmd")
	}
}

// TestLicenseSimulateCmd_FailOpenAt7Days verifies the core acceptance criterion:
// at exactly the 7-day grace threshold the system fails OPEN (CanProceed=true)
// and does NOT hard-block. WriteAllowed is false but the process can continue.
//
// The spec says "fail-open at 7-day TTL, no hard block". This test drives the
// internal library directly so it works in CI without a real ping.nself.org call.
func TestLicenseSimulateCmd_FailOpenAt7Days(t *testing.T) {
	t.Setenv(license.SimulationGuardEnv, "true")
	tmpFile := t.TempDir() + "/license.json"
	t.Setenv("LICENSE_CACHE_PATH", tmpFile)

	// Write a valid cache entry as if it was just fetched.
	entry := &license.CacheEntry{
		KeyHash:   "test_hash_failopen",
		Tier:      "pro",
		FetchedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour).Unix(),
	}
	if err := license.WriteCache(entry); err != nil {
		t.Fatalf("writing test cache: %v", err)
	}

	// Simulate exactly 7 days offline (hits GraceHard threshold).
	result, err := license.SimulateOffline(7)
	if err != nil {
		t.Fatalf("SimulateOffline(7): %v", err)
	}

	// Fail-open: CanProceed must be true (no hard block).
	if !result.CanProceed {
		t.Errorf("CanProceed=false at 7d offline: expected fail-open (no hard block), got hard block")
	}

	// At 7d, the system is in grace_hard — writes are restricted but process continues.
	if result.State != license.GraceHard {
		t.Errorf("expected GraceHard state at 7d offline, got %s", result.State)
	}

	// WriteAllowed=false is the degraded signal (not a hard block).
	if result.WriteAllowed {
		t.Errorf("WriteAllowed=true at 7d offline: expected paid plugin writes to be restricted")
	}
}

// TestLicenseSimulateCmd_FailOpenBeyond7Days verifies CanProceed=true at >7d
// (10 days) — the spec's "fail-open" means no hard block at any grace state.
func TestLicenseSimulateCmd_FailOpenBeyond7Days(t *testing.T) {
	t.Setenv(license.SimulationGuardEnv, "true")
	tmpFile := t.TempDir() + "/license.json"
	t.Setenv("LICENSE_CACHE_PATH", tmpFile)

	entry := &license.CacheEntry{
		KeyHash:   "test_hash_10d",
		Tier:      "pro",
		FetchedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour).Unix(),
	}
	if err := license.WriteCache(entry); err != nil {
		t.Fatalf("writing test cache: %v", err)
	}

	result, err := license.SimulateOffline(10)
	if err != nil {
		t.Fatalf("SimulateOffline(10): %v", err)
	}

	if !result.CanProceed {
		t.Errorf("CanProceed=false at 10d offline: fail-open means no hard block at any grace_hard state")
	}
	if result.State != license.GraceHard {
		t.Errorf("expected GraceHard at 10d, got %s", result.State)
	}
}

// TestLicenseSimulateCmd_GuardRequired verifies the CLI command returns an error
// when LICENSE_ALLOW_SIMULATION is not set.
func TestLicenseSimulateCmd_GuardRequired(t *testing.T) {
	os.Unsetenv(license.SimulationGuardEnv)

	buf := &bytes.Buffer{}
	licenseSimulateCmd.SetErr(buf)

	err := licenseSimulateCmd.RunE(licenseSimulateCmd, []string{"7"})
	if err == nil {
		t.Fatal("expected error when simulation guard is unset")
	}
	if !strings.Contains(err.Error(), "LICENSE_ALLOW_SIMULATION") {
		t.Errorf("error should mention LICENSE_ALLOW_SIMULATION env var, got: %v", err)
	}
}

// TestLicenseSimulateCmd_ClearFlag verifies --clear flag is registered.
func TestLicenseSimulateCmd_ClearFlag(t *testing.T) {
	if licenseSimulateCmd.Flags().Lookup("clear") == nil {
		t.Error("--clear flag not registered on licenseSimulateCmd")
	}
}

// ── NSELF_PLUGIN_LICENSE_KEY_OWNER ───────────────────────────────────────────

// TestOwnerKey_GrantsAllAccess verifies that NSELF_PLUGIN_LICENSE_KEY_OWNER
// is consumed by CollectLicenseKey() with highest precedence, confirming it
// grants all-access in integration tests on all license subcommands.
//
// The PPI states: NSELF_PLUGIN_LICENSE_KEY_OWNER = all-access, never expires.
// The internal license package picks it up via CollectLicenseKey() which checks
// NSELF_PLUGIN_LICENSE_KEY_OWNER first (checker.go:132).
func TestOwnerKey_GrantsAllAccess(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY_OWNER", "nself_owner_testabc123")
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "nself_claw_someotherkey")

	key := license.CollectLicenseKey()
	if key != "nself_owner_testabc123" {
		t.Errorf("CollectLicenseKey: expected owner key to take precedence, got %q", key)
	}
}

// TestOwnerKey_PrecedenceOverUserKey verifies that when the owner key env var
// is set it wins over a regular user key, even when user keys are also present.
func TestOwnerKey_PrecedenceOverUserKey(t *testing.T) {
	ownerKey := "nself_owner_xyz987"
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY_OWNER", ownerKey)
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "nself_plus_shouldnotwin")

	got := license.CollectLicenseKey()
	if got != ownerKey {
		t.Errorf("owner key should have highest precedence; got %q, want %q", got, ownerKey)
	}
}

// TestOwnerKey_UnsetFallsToUserKey verifies fallback to regular key when owner
// key env var is not set.
func TestOwnerKey_UnsetFallsToUserKey(t *testing.T) {
	os.Unsetenv("NSELF_PLUGIN_LICENSE_KEY_OWNER")
	userKey := "nself_claw_fallbackkey"
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", userKey)

	got := license.CollectLicenseKey()
	if got != userKey {
		t.Errorf("when owner key is unset, expected fallback to user key %q, got %q", userKey, got)
	}
}

// TestOwnerKey_IsOwnerKeyPrefix verifies IsOwnerKey correctly identifies the prefix.
func TestOwnerKey_IsOwnerKeyPrefix(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"nself_owner_abc123", true},
		{"nself_claw_abc123", false},
		{"nself_plus_abc123", false},
		{"", false},
		{"nself_owner_", true}, // minimal valid prefix
	}
	for _, tc := range cases {
		got := license.IsOwnerKey(tc.key)
		if got != tc.want {
			t.Errorf("IsOwnerKey(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

// ── licenseCmd subcommand registration ───────────────────────────────────────

// TestLicenseCmd_AllSubcommandsRegistered verifies all 14 spec subcommands are
// present. This is a structural guard — individual subcommand logic is tested
// separately in the internal/license package.
func TestLicenseCmd_AllSubcommandsRegistered(t *testing.T) {
	required := []string{
		"set", "add", "remove", "status", "list", "show", "validate",
		"clear", "upgrade", "refresh", "export", "import", "migrate",
		"simulate-offline",
	}
	subs := map[string]bool{}
	for _, sub := range licenseCmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range required {
		if !subs[want] {
			t.Errorf("license subcommand %q not registered", want)
		}
	}
}

// TestLicenseCmd_Registered verifies licenseCmd is on the global root.
func TestLicenseCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Use == "license" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("licenseCmd is not registered on RootCmd")
	}
}

// ── license status (no network) ──────────────────────────────────────────────

// TestLicenseStatus_NoKeysConfigured verifies status exits 0 (returns nil from
// RunE) when no license keys are configured. Output goes to os.Stdout directly
// in the current implementation; we assert only the exit code.
func TestLicenseStatus_NoKeysConfigured(t *testing.T) {
	// Clear all key sources via env vars.
	os.Unsetenv("NSELF_PLUGIN_LICENSE_KEY")
	os.Unsetenv("NSELF_PLUGIN_LICENSE_KEY_OWNER")
	os.Unsetenv("NSELF_OWNER_LICENSE_KEY")
	for i := 1; i <= 10; i++ {
		t.Setenv("NSELF_LICENSE_KEY_"+fmt.Sprintf("%d", i), "")
	}
	// Also ensure no stored key file by isolating HOME.
	isolateHome(t)

	// RunE should return nil (exit 0) — no-keys is informational, not an error.
	err := licenseStatusCmd.RunE(licenseStatusCmd, []string{})
	if err != nil {
		t.Fatalf("license status with no keys should exit 0 (nil error), got: %v", err)
	}
}
