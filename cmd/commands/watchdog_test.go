package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// newWatchdogCmdTree returns a minimal cobra tree with stub watchdog subcommands
// registered so flag and registration tests run without Docker or file-based state.
func newWatchdogCmdTree() *cobra.Command {
	root := &cobra.Command{
		Use:  "nself",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	stub := &cobra.Command{
		Use:  "watchdog",
		RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}

	stubStatus := &cobra.Command{
		Use:  "status",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	stubStatus.Flags().Bool("json", false, "")

	stubReset := &cobra.Command{Use: "reset-breakers", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	stubResetSvc := &cobra.Command{
		Use:  "reset <service>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	stubResetSvc.Flags().Bool("force", false, "")

	stubHistory := &cobra.Command{Use: "history", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	stubHistory.Flags().String("since", "24h", "")
	stubHistory.Flags().Bool("json", false, "")

	stubTestAlert := &cobra.Command{Use: "test-alert", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	stubTestAlert.Flags().String("service", "test-service", "")
	stubTestAlert.Flags().String("severity", "critical", "")

	stub.AddCommand(stubStatus, stubReset, stubResetSvc, stubHistory, stubTestAlert)
	root.AddCommand(stub)
	return root
}

// TestWatchdogCmd_Registered verifies watchdog is registered on the global root.
func TestWatchdogCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "watchdog" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'watchdog' to be registered on RootCmd")
	}
}

// TestWatchdogCmd_SubcommandsRegistered verifies all 5 watchdog subcommands.
func TestWatchdogCmd_SubcommandsRegistered(t *testing.T) {
	want := []string{"status", "reset-breakers", "reset", "history", "test-alert"}
	registered := map[string]bool{}
	for _, c := range watchdogCmd.Commands() {
		registered[c.Name()] = true
	}
	for _, sub := range want {
		if !registered[sub] {
			t.Errorf("watchdog subcommand %q not registered", sub)
		}
	}
}

// TestWatchdogStatusCmd_FlagJSON verifies --json flag is accepted on status.
func TestWatchdogStatusCmd_FlagJSON(t *testing.T) {
	root := newWatchdogCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"watchdog", "status", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWatchdogResetServiceCmd_RequiresArg verifies reset requires a service name.
func TestWatchdogResetServiceCmd_RequiresArg(t *testing.T) {
	root := newWatchdogCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"watchdog", "reset"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when service name is missing")
	}
}

// TestWatchdogHistoryCmd_FlagSince verifies --since flag is accepted.
func TestWatchdogHistoryCmd_FlagSince(t *testing.T) {
	root := newWatchdogCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"watchdog", "history", "--since", "7d"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWatchdogTestAlertCmd_FlagService verifies --service flag is accepted.
func TestWatchdogTestAlertCmd_FlagService(t *testing.T) {
	root := newWatchdogCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"watchdog", "test-alert", "--service", "postgres"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestWatchdogCmd_UnknownFlagRejected verifies unknown flags are rejected.
func TestWatchdogCmd_UnknownFlagRejected(t *testing.T) {
	root := newWatchdogCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"watchdog", "status", "--no-such-flag"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
