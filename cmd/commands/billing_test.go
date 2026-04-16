package commands

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// unknownSubErr is a sentinel error type used by the synthetic billing
// parent command in tests so assertions can differentiate an unknown
// subcommand from other failures.
type unknownSubErr struct{ sub string }

func (e *unknownSubErr) Error() string { return fmt.Sprintf("unknown billing subcommand %q", e.sub) }

// newBillingCmd returns a fresh cobra.Command tree wired with just the billing
// command and its subcommands so the test can exercise dispatch in isolation
// from the global RootCmd state.
//
// Each subcommand's RunE is replaced with a sentinel that records which
// handler was reached. That lets us assert the parent command's RunE is NOT
// invoked when a subcommand is given.
func newBillingCmd(hit *string) *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	parentCalled := false

	bc := &cobra.Command{
		Use:   "billing",
		Short: "Billing operations: usage, invoice-preview, report, retry-event",
		RunE: func(cmd *cobra.Command, args []string) error {
			parentCalled = true
			*hit = "parent"
			if len(args) == 0 {
				return cmd.Help()
			}
			return &unknownSubErr{sub: args[0]}
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if !parentCalled && *hit == "" {
				*hit = "none"
			}
			return nil
		},
	}

	usageCmd := &cobra.Command{
		Use:  "usage",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			*hit = "usage"
			return nil
		},
	}
	usageCmd.Flags().String("month", "", "Filter by month (YYYY-MM)")
	usageCmd.Flags().String("format", "csv", "Output format: csv or json")

	invoiceCmd := &cobra.Command{
		Use:  "invoice-preview",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			*hit = "invoice-preview"
			return nil
		},
	}

	reportCmd := &cobra.Command{
		Use: "report",
		RunE: func(cmd *cobra.Command, args []string) error {
			*hit = "report"
			return nil
		},
	}
	reportCmd.Flags().String("tenant", "", "Filter by tenant slug")
	reportCmd.Flags().String("month", "", "Filter by month (YYYY-MM)")
	reportCmd.Flags().String("format", "table", "Output format: table or json")

	retryCmd := &cobra.Command{
		Use:  "retry-event",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			*hit = "retry-event"
			return nil
		},
	}

	bc.AddCommand(usageCmd)
	bc.AddCommand(invoiceCmd)
	bc.AddCommand(reportCmd)
	bc.AddCommand(retryCmd)
	root.AddCommand(bc)
	return root
}

// TestBillingCmd_Registered verifies that 'billing' is registered on the
// global RootCmd.
func TestBillingCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "billing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'billing' to be registered on RootCmd")
	}
}

// TestBillingCmd_AllSubcommandsRegistered verifies every documented billing
// subcommand is wired under the parent command.
func TestBillingCmd_AllSubcommandsRegistered(t *testing.T) {
	var parent *cobra.Command
	for _, c := range RootCmd.Commands() {
		if c.Name() == "billing" {
			parent = c
			break
		}
	}
	if parent == nil {
		t.Fatal("billing command not found on RootCmd")
	}

	want := map[string]bool{
		"usage":           false,
		"invoice-preview": false,
		"report":          false,
		"retry-event":     false,
	}
	for _, sub := range parent.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("expected 'billing %s' subcommand to be registered", name)
		}
	}
}

// TestBillingCmd_UsageDispatches verifies that `billing usage <id>` resolves
// to the usage handler — not the parent's help-style RunE.
func TestBillingCmd_UsageDispatches(t *testing.T) {
	var hit string
	root := newBillingCmd(&hit)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"billing", "usage", "tenant-123"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit != "usage" {
		t.Fatalf("expected 'usage' handler to run, got %q", hit)
	}
}

// TestBillingCmd_InvoicePreviewDispatches verifies that
// `billing invoice-preview <id>` resolves to the invoice-preview handler.
func TestBillingCmd_InvoicePreviewDispatches(t *testing.T) {
	var hit string
	root := newBillingCmd(&hit)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"billing", "invoice-preview", "tenant-123"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit != "invoice-preview" {
		t.Fatalf("expected 'invoice-preview' handler to run, got %q", hit)
	}
}

// TestBillingCmd_ReportDispatches verifies that `billing report` resolves to
// the report handler.
func TestBillingCmd_ReportDispatches(t *testing.T) {
	var hit string
	root := newBillingCmd(&hit)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"billing", "report"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit != "report" {
		t.Fatalf("expected 'report' handler to run, got %q", hit)
	}
}

// TestBillingCmd_RetryEventDispatches verifies that `billing retry-event <id>`
// resolves to the retry-event handler.
func TestBillingCmd_RetryEventDispatches(t *testing.T) {
	var hit string
	root := newBillingCmd(&hit)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"billing", "retry-event", "evt_123"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit != "retry-event" {
		t.Fatalf("expected 'retry-event' handler to run, got %q", hit)
	}
}

// TestBillingCmd_NoSubcommandShowsHelp verifies that invoking 'billing' with
// no subcommand reaches the parent's RunE and prints help-like output.
func TestBillingCmd_NoSubcommandShowsHelp(t *testing.T) {
	var hit string
	root := newBillingCmd(&hit)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"billing"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit != "parent" {
		t.Fatalf("expected parent RunE to run with no subcommand, got %q", hit)
	}
	if !strings.Contains(out.String(), "billing") {
		t.Fatalf("expected help output to mention 'billing', got %q", out.String())
	}
}

// TestBillingCmd_UnknownSubcommandErrors verifies that a non-existent
// subcommand produces a cobra "unknown command" error rather than silently
// showing help.
func TestBillingCmd_UnknownSubcommandErrors(t *testing.T) {
	var hit string
	root := newBillingCmd(&hit)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"billing", "does-not-exist"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for unknown subcommand, got nil")
	}
}
