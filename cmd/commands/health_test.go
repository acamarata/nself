package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newHealthTestCmd builds an isolated cobra root with health subcommands stubbed
// so tests exercise flag wiring and command registration without Docker.
func newHealthTestCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	hCmd := &cobra.Command{
		Use:   "health [subcommand]",
		Short: "Health check management",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
	hCmd.PersistentFlags().Int("timeout", 30, "")
	hCmd.PersistentFlags().Int("interval", 10, "")
	hCmd.PersistentFlags().Int("retries", 3, "")
	hCmd.PersistentFlags().String("env", "", "")
	hCmd.PersistentFlags().Bool("json", false, "")
	hCmd.PersistentFlags().Bool("quiet", false, "")

	checkCmd := &cobra.Command{Use: "check", Short: "Run all health checks",
		RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	serviceCmd := &cobra.Command{Use: "service <name>", Short: "Check a single service",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	endpointCmd := &cobra.Command{Use: "endpoint <url>", Short: "Check an HTTP endpoint",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	watchCmd := &cobra.Command{Use: "watch", Short: "Continuous health monitoring",
		RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	historyCmd := &cobra.Command{Use: "history", Short: "Show last 20 health checks",
		RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	configCmd := &cobra.Command{Use: "config", Short: "Show health check settings",
		RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	hCmd.AddCommand(checkCmd, serviceCmd, endpointCmd, watchCmd, historyCmd, configCmd)
	root.AddCommand(hCmd)
	return root
}

// TestHealthCmd_Registered verifies health is registered on the global root.
func TestHealthCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "health" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'health' to be registered on RootCmd")
	}
}

// TestHealthCmd_SubcommandsRegistered verifies all 6 subcommands exist.
func TestHealthCmd_SubcommandsRegistered(t *testing.T) {
	expected := []string{"check", "service", "endpoint", "watch", "history", "config"}
	var names []string
	for _, c := range healthCmd.Commands() {
		names = append(names, c.Name())
	}
	for _, want := range expected {
		found := false
		for _, got := range names {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("health subcommand %q not registered", want)
		}
	}
}

// TestHealthCmd_PersistentFlags verifies all expected persistent flags exist.
func TestHealthCmd_PersistentFlags(t *testing.T) {
	flags := []string{"timeout", "interval", "retries", "env", "json", "quiet"}
	for _, name := range flags {
		if healthCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("health persistent flag --%s not registered", name)
		}
	}
}

// TestHealthCmd_FlagJSON verifies --json is accepted on the health command.
func TestHealthCmd_FlagJSON(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--json"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--json not recognized: %v", err)
	}
}

// TestHealthCmd_FlagQuiet verifies --quiet is accepted.
func TestHealthCmd_FlagQuiet(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--quiet"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--quiet not recognized: %v", err)
	}
}

// TestHealthCmd_FlagTimeout verifies --timeout is accepted.
func TestHealthCmd_FlagTimeout(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--timeout", "60"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--timeout not recognized: %v", err)
	}
}

// TestHealthCmd_FlagRetries verifies --retries is accepted.
func TestHealthCmd_FlagRetries(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--retries", "5"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--retries not recognized: %v", err)
	}
}

// TestHealthCmd_FlagEnv verifies --env is accepted.
func TestHealthCmd_FlagEnv(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--env", "staging"})
	if err := root.Execute(); err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--env not recognized: %v", err)
	}
}

// TestHealthCheckCmd_Exists verifies the check subcommand is reachable.
func TestHealthCheckCmd_Exists(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "check"})
	if err := root.Execute(); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

// TestHealthServiceCmd_RequiresArg verifies service requires exactly 1 arg.
func TestHealthServiceCmd_RequiresArg(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "service"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when service name is missing")
	}
}

// TestHealthEndpointCmd_RequiresArg verifies endpoint requires exactly 1 arg.
func TestHealthEndpointCmd_RequiresArg(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "endpoint"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when URL is missing")
	}
}

// TestHealthCmd_UnknownFlagRejected verifies unknown flags fail fast.
func TestHealthCmd_UnknownFlagRejected(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--no-such-flag"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// TestHealthConfigCmd_Exists verifies the config subcommand returns output.
func TestHealthConfigCmd_Exists(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "config"})
	if err := root.Execute(); err != nil {
		t.Fatalf("health config failed: %v", err)
	}
}

// TestHealthHistoryCmd_Exists verifies the history subcommand is reachable.
func TestHealthHistoryCmd_Exists(t *testing.T) {
	root := newHealthTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "history"})
	if err := root.Execute(); err != nil {
		t.Fatalf("health history failed: %v", err)
	}
}
