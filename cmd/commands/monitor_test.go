package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// newMonitorCmdTree returns a minimal cobra tree with stub monitor subcommands.
func newMonitorCmdTree() *cobra.Command {
	root := &cobra.Command{
		Use:  "nself",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	stub := &cobra.Command{
		Use:  "monitor",
		RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	stubUpgrade := &cobra.Command{
		Use:  "upgrade-dashboards",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	stubUpgrade.Flags().Bool("force", false, "")
	stub.AddCommand(stubUpgrade)
	root.AddCommand(stub)
	return root
}

// TestMonitorCmd_Registered verifies monitor is registered on the global root.
func TestMonitorCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "monitor" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'monitor' to be registered on RootCmd")
	}
}

// TestMonitorCmd_UpgradeDashboardsRegistered verifies the upgrade-dashboards subcommand exists.
func TestMonitorCmd_UpgradeDashboardsRegistered(t *testing.T) {
	found := false
	for _, c := range monitorCmd.Commands() {
		if c.Name() == "upgrade-dashboards" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'upgrade-dashboards' to be registered under monitor")
	}
}

// TestMonitorUpgradeDashboardsCmd_FlagForce verifies --force flag is accepted.
func TestMonitorUpgradeDashboardsCmd_FlagForce(t *testing.T) {
	root := newMonitorCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"monitor", "upgrade-dashboards", "--force"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestMonitorUpgradeDashboardsCmd_NoForce verifies upgrade-dashboards runs without --force.
func TestMonitorUpgradeDashboardsCmd_NoForce(t *testing.T) {
	root := newMonitorCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"monitor", "upgrade-dashboards"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestMonitorCmd_UnknownFlagRejected verifies unknown flags are rejected.
func TestMonitorCmd_UnknownFlagRejected(t *testing.T) {
	root := newMonitorCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"monitor", "upgrade-dashboards", "--no-such-flag"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}
