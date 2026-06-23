package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newStopCmd returns an isolated cobra.Command tree with just the stop command
// for flag-parsing and registration tests.
func newStopCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	sc := &cobra.Command{
		Use:   "stop [SERVICES...]",
		Short: "Gracefully shut down all services or specific services",
		RunE:  runStop,
	}
	sc.Flags().BoolP("volumes", "v", false, "Remove volumes (WARNING: deletes data)")
	sc.Flags().Bool("rmi", false, "Remove Docker images")
	sc.Flags().Bool("remove-images", false, "Alias for --rmi")
	sc.Flags().Bool("remove-orphans", false, "Remove orphaned containers")
	sc.Flags().Int("graceful", 30, "Graceful shutdown timeout in seconds")
	sc.Flags().Bool("verbose", false, "Show detailed output")
	sc.Flags().Bool("no-monorepo", false, "Disable automatic monorepo backend detection")

	root.AddCommand(sc)
	return root
}

// TestStopCmd_Registered verifies that the stop command is registered on RootCmd.
func TestStopCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "stop" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'stop' to be registered on RootCmd")
	}
}

// TestStopCmd_FlagVolumes verifies that --volumes/-v is accepted.
func TestStopCmd_FlagVolumes(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "--volumes"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--volumes flag was not recognised: %v", err)
	}
}

// TestStopCmd_FlagRmi verifies that --rmi is accepted.
func TestStopCmd_FlagRmi(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "--rmi"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--rmi flag was not recognised: %v", err)
	}
}

// TestStopCmd_FlagRemoveImages verifies that --remove-images (alias for --rmi) is accepted.
func TestStopCmd_FlagRemoveImages(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "--remove-images"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--remove-images flag was not recognised: %v", err)
	}
}

// TestStopCmd_FlagGraceful verifies that --graceful accepts an int value.
func TestStopCmd_FlagGraceful(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "--graceful=60"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--graceful flag was not recognised: %v", err)
	}
}

// TestStopCmd_FlagRemoveOrphans verifies that --remove-orphans is accepted.
func TestStopCmd_FlagRemoveOrphans(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "--remove-orphans"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--remove-orphans flag was not recognised: %v", err)
	}
}

// TestStopCmd_FlagVerbose verifies that --verbose is accepted.
func TestStopCmd_FlagVerbose(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "--verbose"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--verbose flag was not recognised: %v", err)
	}
}

// TestStopCmd_PositionalServices verifies that service names are accepted as
// positional arguments without a flag-parse error.
func TestStopCmd_PositionalServices(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "postgres", "redis"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("positional SERVICE args caused a flag error: %v", err)
	}
}

// TestStopCmd_UnknownFlagIsRejected verifies that an unrecognised flag is rejected.
func TestStopCmd_UnknownFlagIsRejected(t *testing.T) {
	root := newStopCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"stop", "--this-flag-does-not-exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for unknown flag, got nil")
	}
}

// TestDownAlias_Registered verifies that the hidden 'down' alias is registered on RootCmd.
func TestDownAlias_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "down" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'down' alias to be registered on RootCmd")
	}
}

// TestDownAlias_ShortMentionsStop verifies that the down alias help text
// references 'nself stop' so operators know it is an alias.
func TestDownAlias_ShortMentionsStop(t *testing.T) {
	for _, c := range RootCmd.Commands() {
		if c.Name() == "down" {
			if !strings.Contains(c.Short, "stop") && !strings.Contains(c.Short, "nself stop") {
				t.Errorf("'down' Short text should reference 'stop', got: %q", c.Short)
			}
			return
		}
	}
	t.Fatal("'down' command not found on RootCmd")
}
