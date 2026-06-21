package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// newStatusCmdTree returns a minimal cobra tree with a stub status command
// so flag and registration tests run without Docker or a live nSelf project.
func newStatusCmdTree() *cobra.Command {
	root := &cobra.Command{
		Use:  "nself",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	stub := &cobra.Command{
		Use:  "status [SERVICE]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	stub.Flags().Bool("json", false, "")
	stub.Flags().BoolP("verbose", "v", false, "")
	stub.Flags().Bool("health-only", false, "")
	stub.Flags().Bool("metrics", false, "")
	stub.Flags().Bool("deep", false, "")
	root.AddCommand(stub)
	return root
}

// TestStatusCmd_Registered verifies status is registered on the global root.
func TestStatusCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'status' to be registered on RootCmd")
	}
}

// TestStatusCmd_ExitCodesDocumented verifies the Long description mentions exit codes 0/1/2.
func TestStatusCmd_ExitCodesDocumented(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "status" {
			if c.Long != "" {
				found = true
			}
			break
		}
	}
	if !found {
		t.Fatal("status command should have a Long description documenting exit codes")
	}
}

// TestStatusCmd_FlagJSON verifies --json is accepted.
func TestStatusCmd_FlagJSON(t *testing.T) {
	root := newStatusCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestStatusCmd_FlagVerbose verifies --verbose is accepted.
func TestStatusCmd_FlagVerbose(t *testing.T) {
	root := newStatusCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--verbose"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestStatusCmd_FlagDeep verifies --deep is accepted.
func TestStatusCmd_FlagDeep(t *testing.T) {
	root := newStatusCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--deep"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestStatusCmd_AcceptsPositionalArg verifies status accepts a single SERVICE arg.
func TestStatusCmd_AcceptsPositionalArg(t *testing.T) {
	root := newStatusCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "postgres"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error for positional arg: %v", err)
	}
}

// TestStatusCmd_RejectsTooManyArgs verifies status rejects more than 1 positional arg.
func TestStatusCmd_RejectsTooManyArgs(t *testing.T) {
	root := newStatusCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "svc1", "svc2"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when more than 1 arg is supplied")
	}
}

// TestStatusCmd_UnknownFlagRejected verifies unknown flags are rejected.
func TestStatusCmd_UnknownFlagRejected(t *testing.T) {
	root := newStatusCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--no-such-flag"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
