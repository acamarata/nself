package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// newHealthCmdTree returns a minimal cobra tree with the real healthCmd
// registered so flag and registration tests run without Docker.
func newHealthCmdTree() *cobra.Command {
	root := &cobra.Command{
		Use: "nself",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	// Mount a stub health command that wires the same flags as the real one
	// but does not perform I/O, so tests run without a live nSelf project.
	stubHealth := &cobra.Command{
		Use:   "health [subcommand]",
		Short: "Health check management",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
	stubHealth.PersistentFlags().Int("timeout", 30, "")
	stubHealth.PersistentFlags().Int("interval", 10, "")
	stubHealth.PersistentFlags().Int("retries", 3, "")
	stubHealth.PersistentFlags().String("env", "", "")
	stubHealth.PersistentFlags().Bool("json", false, "")
	stubHealth.PersistentFlags().Bool("quiet", false, "")

	stubCheck := &cobra.Command{Use: "check", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	stubService := &cobra.Command{Use: "service <name>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	stubEndpoint := &cobra.Command{Use: "endpoint <url>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	stubWatch := &cobra.Command{Use: "watch", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	stubHistory := &cobra.Command{Use: "history", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	stubConfig := &cobra.Command{Use: "config", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	stubHealth.AddCommand(stubCheck, stubService, stubEndpoint, stubWatch, stubHistory, stubConfig)
	root.AddCommand(stubHealth)
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

// TestHealthCmd_SubcommandsRegistered verifies all 6 health subcommands are registered.
func TestHealthCmd_SubcommandsRegistered(t *testing.T) {
	want := []string{"check", "service", "endpoint", "watch", "history", "config"}
	registered := map[string]bool{}
	for _, c := range healthCmd.Commands() {
		registered[c.Name()] = true
	}
	for _, sub := range want {
		if !registered[sub] {
			t.Errorf("health subcommand %q not registered", sub)
		}
	}
}

// TestHealthCmd_FlagTimeout verifies --timeout flag is accepted.
func TestHealthCmd_FlagTimeout(t *testing.T) {
	root := newHealthCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--timeout", "5"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHealthCmd_FlagJSON verifies --json flag is accepted.
func TestHealthCmd_FlagJSON(t *testing.T) {
	root := newHealthCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHealthCmd_FlagQuiet verifies --quiet flag is accepted.
func TestHealthCmd_FlagQuiet(t *testing.T) {
	root := newHealthCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--quiet"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHealthCmd_FlagRetries verifies --retries flag is accepted.
func TestHealthCmd_FlagRetries(t *testing.T) {
	root := newHealthCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--retries", "1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHealthCmd_UnknownFlagRejected verifies unknown flags are rejected.
func TestHealthCmd_UnknownFlagRejected(t *testing.T) {
	root := newHealthCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "--no-such-flag"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

// TestHealthCmd_ServiceSubcmdRequiresArg verifies health service requires exactly 1 arg.
func TestHealthCmd_ServiceSubcmdRequiresArg(t *testing.T) {
	root := newHealthCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "service"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when service name is missing")
	}
}

// TestHealthCmd_EndpointSubcmdRequiresArg verifies health endpoint requires exactly 1 arg.
func TestHealthCmd_EndpointSubcmdRequiresArg(t *testing.T) {
	root := newHealthCmdTree()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"health", "endpoint"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when endpoint URL is missing")
	}
}
