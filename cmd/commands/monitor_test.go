package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newMonitorTestCmd builds an isolated cobra root with monitor subcommands.
func newMonitorTestCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	mCmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitoring stack management",
		RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}

	upgradeCmd := &cobra.Command{Use: "upgrade-dashboards", Short: "Upgrade Grafana dashboards",
		RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	upgradeCmd.Flags().Bool("force", false, "")

	mCmd.AddCommand(upgradeCmd)
	root.AddCommand(mCmd)
	return root
}

// TestMonitorCmd_Registered verifies monitor is on the global root.
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

// TestMonitorCmd_SubcommandsRegistered verifies upgrade-dashboards is present.
func TestMonitorCmd_SubcommandsRegistered(t *testing.T) {
	found := false
	for _, c := range monitorCmd.Commands() {
		if c.Name() == "upgrade-dashboards" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("monitor command missing 'upgrade-dashboards' subcommand")
	}
}

// TestMonitorUpgradeDashboards_ForceFlag verifies --force is accepted.
func TestMonitorUpgradeDashboards_ForceFlag(t *testing.T) {
	root := newMonitorTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"monitor", "upgrade-dashboards", "--force"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--force not recognized: %v", err)
	}
}

// TestMonitorUpgradeDashboards_NoForce verifies upgrade-dashboards runs without --force.
func TestMonitorUpgradeDashboards_NoForce(t *testing.T) {
	root := newMonitorTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"monitor", "upgrade-dashboards"})
	if err := root.Execute(); err != nil {
		t.Fatalf("monitor upgrade-dashboards failed: %v", err)
	}
}

// TestMonitorCmd_UnknownSubcommandHelp verifies bare monitor shows help.
func TestMonitorCmd_UnknownSubcommandHelp(t *testing.T) {
	root := newMonitorTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"monitor"})
	// Should not error; returns help text.
	if err := root.Execute(); err != nil {
		t.Fatalf("bare monitor command failed: %v", err)
	}
}

// TestMonitorUpgradeDashboards_RealCmd verifies the real command is registered
// with --force flag on the global command tree.
func TestMonitorUpgradeDashboards_RealCmd(t *testing.T) {
	var found *cobra.Command
	for _, c := range monitorCmd.Commands() {
		if c.Name() == "upgrade-dashboards" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("upgrade-dashboards not found in monitorCmd")
	}
	if found.Flags().Lookup("force") == nil {
		t.Error("upgrade-dashboards missing --force flag")
	}
}
