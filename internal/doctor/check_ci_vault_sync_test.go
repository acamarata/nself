package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

// CI-VAULT-SYNC-01 reads ~/.claude/vault.env, which cannot exist on a CI runner,
// so it returned CRITICAL on every single run — a gate that could never pass.
func TestCIVaultSync_SkipsOnCIWithoutVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("CI", "true")

	got := CheckCIVaultSync(dir)
	if got.Status != "skip" {
		t.Fatalf("status = %q, want \"skip\" (a CI runner has no vault.env by design)", got.Status)
	}
}

// Guards the other direction so this cannot become a gate that never fails:
// when a vault file is present the check still runs, even on CI.
func TestCIVaultSync_StillChecksWhenVaultPresent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("GITHUB_ACTIONS", "true")

	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o750); err != nil {
		t.Fatal(err)
	}
	// Vault exists but contains none of the critical PATs.
	vault := filepath.Join(dir, ".claude", "vault.env")
	if err := os.WriteFile(vault, []byte("UNRELATED=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := CheckCIVaultSync(dir); got.Status != "fail" {
		t.Fatalf("status = %q, want \"fail\" — a present-but-empty vault must still fail", got.Status)
	}
}
