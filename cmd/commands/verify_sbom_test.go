package commands

// verify_sbom_test.go — unit tests for `nself verify-sbom`.
//
// Purpose: Verify flag declarations, command registration, and the SBOM
// signature-chain verification path (offline gate: missing --version or
// missing cosign surfaces the correct error before any network call is made).
//
// Constraints: No network. No cosign binary required. Tests confirm the
// command layer enforces prerequisites before attempting the signature chain.

import (
	"strings"
	"testing"
)

// TestVerifySBOMCmdRegistered verifies that verify-sbom is registered on RootCmd.
func TestVerifySBOMCmdRegistered(t *testing.T) {
	for _, c := range RootCmd.Commands() {
		if c.Name() == "verify-sbom" {
			return
		}
	}
	t.Fatal("verify-sbom not registered on RootCmd")
}

// TestVerifySBOMFlagsPresent verifies that --version and --repo flags are declared.
func TestVerifySBOMFlagsPresent(t *testing.T) {
	t.Parallel()

	flags := []string{"version", "repo"}
	for _, name := range flags {
		if verifySBOMCmd.Flags().Lookup(name) == nil {
			t.Errorf("flag --%s not declared on verify-sbom", name)
		}
	}
}

// TestVerifySBOMRepoDefault verifies that --repo defaults to the canonical repo.
func TestVerifySBOMRepoDefault(t *testing.T) {
	t.Parallel()

	f := verifySBOMCmd.Flags().Lookup("repo")
	if f == nil {
		t.Fatal("--repo flag not declared")
	}
	if f.DefValue != sbomGitHubRepo {
		t.Errorf("--repo default = %q, want %q", f.DefValue, sbomGitHubRepo)
	}
}

// TestVerifySBOMChain_MissingVersion verifies that the signature chain
// verification is gated on a non-empty --version: the command must return an
// error before attempting any cosign or network call.
func TestVerifySBOMChain_MissingVersion(t *testing.T) {
	t.Parallel()

	cmd := verifySBOMCmd
	cmd.Flags().Set("version", "") //nolint:errcheck

	err := runVerifySBOM(cmd, nil)
	if err == nil {
		t.Fatal("expected error when --version is empty, got nil")
	}
	if !strings.Contains(err.Error(), "--version") {
		t.Errorf("error %q should mention --version flag", err.Error())
	}
}

// TestVerifySBOMChain_CosignGate verifies that when --version is set but cosign
// is absent from PATH, the command returns an error describing the missing tool
// before attempting any network download. This confirms the pre-flight check
// in the signature chain (step 1 of 3: cosign present → download SBOM →
// download bundle → verify signature).
func TestVerifySBOMChain_CosignGate(t *testing.T) {
	// t.Setenv requires sequential execution (cannot combine with t.Parallel).

	// Temporarily empty PATH so cosign cannot be found.
	t.Setenv("PATH", "")

	cmd := verifySBOMCmd
	cmd.Flags().Set("version", "v1.0.0") //nolint:errcheck

	err := runVerifySBOM(cmd, nil)
	if err == nil {
		t.Fatal("expected error when cosign not in PATH, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "cosign") {
		t.Errorf("error %q should mention cosign", msg)
	}
}
