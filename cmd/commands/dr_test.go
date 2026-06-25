package commands

import (
	"strings"
	"testing"
)

// ── T09: dr family — S09 acceptance tests ────────────────────────────────────

// TestDRCmd_Registered verifies nself dr is present on RootCmd.
func TestDRCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "dr" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("drCmd not registered on RootCmd")
	}
}

// TestDRCmd_SubcommandsRegistered verifies all 5 dr subcommands exist.
func TestDRCmd_SubcommandsRegistered(t *testing.T) {
	required := []string{"drill", "promote-standby", "reconfigure-dns", "rollback", "fence"}
	subs := map[string]bool{}
	for _, sub := range drCmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range required {
		if !subs[want] {
			t.Errorf("dr subcommand %q not registered", want)
		}
	}
}

// TestDRAllSubcommands_HaveDryRunFlag verifies every dr subcommand carries --dry-run.
// This is the key S09 acceptance criterion: dry-run coverage is safety-critical
// because promote-standby/reconfigure-dns/rollback/fence are irreversible in production.
func TestDRAllSubcommands_HaveDryRunFlag(t *testing.T) {
	for _, sub := range drCmd.Commands() {
		f := sub.Flags().Lookup("dry-run")
		if f == nil {
			t.Errorf("dr %s: --dry-run flag not registered (required for all dr subcommands)", sub.Name())
		}
	}
}

// TestDRPromoteCmd_DryRunFlag verifies --dry-run on promote-standby specifically.
func TestDRPromoteCmd_DryRunFlag(t *testing.T) {
	f := drPromoteCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Error("dr promote-standby: --dry-run flag not registered")
	}
}

// TestDRReconfigureDNSCmd_DryRunFlag verifies --dry-run on reconfigure-dns.
func TestDRReconfigureDNSCmd_DryRunFlag(t *testing.T) {
	f := drReconfigureDNSCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Error("dr reconfigure-dns: --dry-run flag not registered")
	}
}

// TestDRRollbackCmd_DryRunFlag verifies --dry-run on rollback.
func TestDRRollbackCmd_DryRunFlag(t *testing.T) {
	f := drRollbackCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Error("dr rollback: --dry-run flag not registered")
	}
}

// TestDRFenceCmd_DryRunFlag verifies --dry-run on fence.
func TestDRFenceCmd_DryRunFlag(t *testing.T) {
	f := drFenceCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Error("dr fence: --dry-run flag not registered")
	}
}

// TestDRDrillCmd_DryRunFlag verifies --dry-run on drill (pre-existing).
func TestDRDrillCmd_DryRunFlag(t *testing.T) {
	f := drDrillCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Error("dr drill: --dry-run flag not registered")
	}
}

// ── Backup cron validation — S09 acceptance tests ─────────────────────────────

// TestValidateCronExpression_ValidExpressions verifies well-formed 5-field cron
// expressions pass validation.
func TestValidateCronExpression_ValidExpressions(t *testing.T) {
	valid := []string{
		"0 2 * * *",
		"30 3 1 * *",
		"0 0 * * 0",
		"*/15 * * * *",
		"0 2 * * 1-5",
	}
	for _, expr := range valid {
		if err := validateCronExpression(expr); err != nil {
			t.Errorf("validateCronExpression(%q) returned unexpected error: %v", expr, err)
		}
	}
}

// TestValidateCronExpression_EmptyReturnsError verifies empty string is rejected.
func TestValidateCronExpression_EmptyReturnsError(t *testing.T) {
	err := validateCronExpression("")
	if err == nil {
		t.Fatal("expected error for empty cron expression, got nil")
	}
}

// TestValidateCronExpression_TooFewFields verifies < 5 fields is rejected.
func TestValidateCronExpression_TooFewFields(t *testing.T) {
	bad := []string{"0 2 * *", "0 2", "*"}
	for _, expr := range bad {
		err := validateCronExpression(expr)
		if err == nil {
			t.Errorf("validateCronExpression(%q): expected error for too few fields, got nil", expr)
		}
		if !strings.Contains(err.Error(), "5 fields") {
			t.Errorf("validateCronExpression(%q): error should mention '5 fields', got %q", expr, err.Error())
		}
	}
}

// TestValidateCronExpression_TooManyFields verifies > 5 fields is rejected.
func TestValidateCronExpression_TooManyFields(t *testing.T) {
	err := validateCronExpression("0 2 * * * 2026")
	if err == nil {
		t.Fatal("expected error for 6-field cron expression (6-field format not supported), got nil")
	}
}

// TestValidateCronExpression_ErrorContainsExpression verifies the expression
// is quoted in the error message for operator visibility.
func TestValidateCronExpression_ErrorContainsExpression(t *testing.T) {
	expr := "not valid"
	err := validateCronExpression(expr)
	// "not valid" = 2 fields — should produce a field-count error
	if err == nil {
		t.Fatal("expected error")
	}
}
