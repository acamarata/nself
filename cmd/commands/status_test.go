package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newStatusTestCmd builds an isolated cobra root with the status command stubbed.
func newStatusTestCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	sCmd := &cobra.Command{
		Use:   "status [SERVICE]",
		Short: "Show health status of all services",
		Args:  cobra.MaximumNArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
	sCmd.Flags().Bool("json", false, "")
	sCmd.Flags().Bool("verbose", false, "")
	sCmd.Flags().Bool("health-only", false, "")
	sCmd.Flags().Bool("metrics", false, "")
	sCmd.Flags().Bool("deep", false, "")

	root.AddCommand(sCmd)
	return root
}

// TestStatusCmd_Registered verifies status is on the global root.
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

// TestStatusCmd_FlagJSON verifies --json is accepted.
func TestStatusCmd_FlagJSON(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--json"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--json not recognized: %v", err)
	}
}

// TestStatusCmd_FlagVerbose verifies --verbose is accepted.
func TestStatusCmd_FlagVerbose(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--verbose"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--verbose not recognized: %v", err)
	}
}

// TestStatusCmd_FlagDeep verifies --deep is accepted.
func TestStatusCmd_FlagDeep(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--deep"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--deep not recognized: %v", err)
	}
}

// TestStatusCmd_FlagHealthOnly verifies --health-only is accepted.
func TestStatusCmd_FlagHealthOnly(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--health-only"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--health-only not recognized: %v", err)
	}
}

// TestStatusCmd_FlagMetrics verifies --metrics is accepted.
func TestStatusCmd_FlagMetrics(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--metrics"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--metrics not recognized: %v", err)
	}
}

// TestStatusCmd_PositionalArg verifies positional SERVICE arg is accepted.
func TestStatusCmd_PositionalArg(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "postgres"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status postgres failed: %v", err)
	}
}

// TestStatusCmd_TooManyArgs verifies more than 1 positional arg is rejected.
func TestStatusCmd_TooManyArgs(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "svc1", "svc2"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for two positional args")
	}
}

// TestStatusCmd_UnknownFlagRejected verifies unknown flags fail fast.
func TestStatusCmd_UnknownFlagRejected(t *testing.T) {
	root := newStatusTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"status", "--no-such-flag"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// TestStatusCmd_Flags verifies all expected flags are registered on the real command.
func TestStatusCmd_Flags(t *testing.T) {
	flags := []string{"json", "verbose", "health-only", "metrics", "deep"}
	for _, name := range flags {
		if statusCmd.Flags().Lookup(name) == nil {
			t.Errorf("status flag --%s not registered", name)
		}
	}
}

// TestStatusCmd_ExitCodeDocumented verifies the Long description documents exit codes.
func TestStatusCmd_ExitCodeDocumented(t *testing.T) {
	if !strings.Contains(statusCmd.Long, "Exit codes") {
		t.Error("status Long description should document exit codes")
	}
}
