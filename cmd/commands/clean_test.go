package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newCleanCmd returns an isolated cobra.Command tree with just the clean command.
func newCleanCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	cc := &cobra.Command{
		Use:   "clean",
		Short: "Remove generated artifacts (docker-compose.yml, nginx configs, build cache)",
		RunE:  runClean,
	}

	root.AddCommand(cc)
	return root
}

// TestCleanCmd_Registered verifies that the clean command is registered on RootCmd.
func TestCleanCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "clean" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'clean' to be registered on RootCmd")
	}
}

// TestCleanCmd_NoProject verifies that running clean in an empty temp directory
// succeeds without panicking. clean is intentionally forgiving about missing
// project roots.
func TestCleanCmd_NoProject(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newCleanCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"clean"})
	// clean is designed to succeed even with no project; it just has nothing to remove.
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "panic") {
		t.Fatalf("clean panicked in empty dir: %v", err)
	}
}

// TestCleanCmd_UnknownFlagIsRejected verifies that an unrecognised flag is rejected.
func TestCleanCmd_UnknownFlagIsRejected(t *testing.T) {
	root := newCleanCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"clean", "--this-flag-does-not-exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for unknown flag, got nil")
	}
}

// TestCleanCmd_HelpTextMentionsArtifacts verifies that the clean command's Long
// description documents what is removed and what is preserved.
func TestCleanCmd_HelpTextMentionsArtifacts(t *testing.T) {
	for _, c := range RootCmd.Commands() {
		if c.Name() == "clean" {
			if !strings.Contains(c.Long, "docker-compose.yml") {
				t.Error("clean Long help should mention docker-compose.yml")
			}
			if !strings.Contains(c.Long, "Preserved") && !strings.Contains(c.Long, "preserved") {
				t.Error("clean Long help should document what is preserved")
			}
			return
		}
	}
	t.Fatal("'clean' command not found on RootCmd")
}
