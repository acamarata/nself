package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newResetCmd returns an isolated cobra.Command tree with just the reset command
// for flag-parsing and registration tests.
func newResetCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	rc := &cobra.Command{
		Use:   "reset",
		Short: "Stop containers, remove all data volumes, and clean generated files",
		RunE:  runReset,
	}
	rc.Flags().Bool("yes", false, "Skip confirmation prompt (for CI/CD)")
	rc.Flags().Bool("no-monorepo", false, "Disable automatic monorepo backend detection")

	root.AddCommand(rc)
	return root
}

// TestResetCmd_Registered verifies that the reset command is registered on RootCmd.
func TestResetCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "reset" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'reset' to be registered on RootCmd")
	}
}

// TestResetCmd_FlagYes verifies that --yes (CI skip-confirm gate) is accepted.
func TestResetCmd_FlagYes(t *testing.T) {
	root := newResetCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"reset", "--yes"})
	err := root.Execute()
	// Command will fail on missing project; the flag itself must be accepted.
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--yes flag was not recognised: %v", err)
	}
}

// TestResetCmd_FlagNoMonorepo verifies that --no-monorepo is accepted.
func TestResetCmd_FlagNoMonorepo(t *testing.T) {
	root := newResetCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"reset", "--no-monorepo"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--no-monorepo flag was not recognised: %v", err)
	}
}

// TestResetCmd_NoProject verifies that reset without a project root returns an
// actionable error mentioning 'nself init', not a panic.
func TestResetCmd_NoProject(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newResetCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"reset", "--yes", "--no-monorepo"})
	err := root.Execute()

	if err == nil {
		t.Fatal("expected an error when running reset outside an nself project, got nil")
	}
	if !strings.Contains(err.Error(), "nself init") && !strings.Contains(err.Error(), "nself project") {
		t.Errorf("expected error to mention 'nself init' or 'nself project', got: %v", err)
	}
}

// TestResetCmd_LongTextMentionsDestructive verifies that the Long description
// documents the destructive nature of the operation (operator safety).
func TestResetCmd_LongTextMentionsDestructive(t *testing.T) {
	for _, c := range RootCmd.Commands() {
		if c.Name() == "reset" {
			longLower := strings.ToLower(c.Long)
			if !strings.Contains(longLower, "destructive") && !strings.Contains(longLower, "irreversible") {
				t.Error("reset Long help should warn that the operation is destructive or irreversible")
			}
			return
		}
	}
	t.Fatal("'reset' command not found on RootCmd")
}

// TestResetCmd_UnknownFlagIsRejected verifies that an unrecognised flag is rejected.
func TestResetCmd_UnknownFlagIsRejected(t *testing.T) {
	root := newResetCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"reset", "--this-flag-does-not-exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for unknown flag, got nil")
	}
}
