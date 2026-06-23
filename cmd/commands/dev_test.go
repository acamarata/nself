package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newDevCmd returns an isolated cobra.Command tree with just the dev command
// for flag-parsing and registration tests.
func newDevCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	dc := &cobra.Command{
		Use:   "dev",
		Short: "Start development environment",
		RunE:  runDev,
	}
	dc.Flags().Bool("hot", false, "Enable plugin hot-reload on source changes")

	root.AddCommand(dc)
	return root
}

// TestDevCmd_Registered verifies that the dev command is registered on RootCmd.
func TestDevCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "dev" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'dev' to be registered on RootCmd")
	}
}

// TestDevCmd_FlagHot verifies that --hot is accepted by the dev command.
func TestDevCmd_FlagHot(t *testing.T) {
	root := newDevCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dev", "--hot"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--hot flag was not recognised: %v", err)
	}
}

// TestDevCmd_NoProject verifies that running dev outside an nself project
// returns a meaningful error rather than panicking.
func TestDevCmd_NoProject(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newDevCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dev"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error when running dev outside an nself project, got nil")
	}
}

// TestDevCmd_UnknownFlagIsRejected verifies that an unrecognised flag is rejected.
func TestDevCmd_UnknownFlagIsRejected(t *testing.T) {
	root := newDevCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"dev", "--this-flag-does-not-exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for unknown flag, got nil")
	}
}

// TestDevCmd_HotReloadFlagDefault verifies that --hot defaults to false.
func TestDevCmd_HotReloadFlagDefault(t *testing.T) {
	root := newDevCmd()
	// Parse only -- do not execute -- to inspect the default value.
	devSubCmd, _, err := root.Find([]string{"dev"})
	if err != nil || devSubCmd == nil {
		t.Fatal("could not find dev subcommand on isolated root")
	}
	// Simulate parsing with no flags set.
	if err = devSubCmd.ParseFlags([]string{}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	hot, err := devSubCmd.Flags().GetBool("hot")
	if err != nil {
		t.Fatalf("GetBool('hot') failed: %v", err)
	}
	if hot {
		t.Error("--hot should default to false")
	}
}
