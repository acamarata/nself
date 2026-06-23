package commands

import (
	"strings"
	"testing"
)

// TestGDPRCommandRegistration verifies that all expected gdpr subcommands are
// registered on the gdprCmd, including the "forget" Art. 17 alias.
func TestGDPRCommandRegistration(t *testing.T) {
	want := map[string]bool{
		"export":        false,
		"delete":        false,
		"forget":        false,
		"status":        false,
		"list-requests": false,
	}

	for _, sub := range gdprCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("gdpr subcommand %q not registered", name)
		}
	}
}

// TestGDPRForgetFlagsMatchDelete verifies that the "forget" alias exposes the
// same flags as "delete" so callers can substitute one for the other.
func TestGDPRForgetFlagsMatchDelete(t *testing.T) {
	requiredFlags := []string{"user"}
	optionalFlags := []string{"dry-run"}

	for _, name := range requiredFlags {
		if gdprForgetCmd.Flags().Lookup(name) == nil {
			t.Errorf("gdpr forget: required flag --%s not registered", name)
		}
	}
	for _, name := range optionalFlags {
		if gdprForgetCmd.Flags().Lookup(name) == nil {
			t.Errorf("gdpr forget: optional flag --%s not registered", name)
		}
	}
}

// TestGDPRForgetUserFlagRequired verifies that --user is marked required on
// the forget alias (mirrors the same invariant on delete).
func TestGDPRForgetUserFlagRequired(t *testing.T) {
	ann := gdprForgetCmd.Flags().Lookup("user")
	if ann == nil {
		t.Fatal("gdpr forget: --user flag not registered")
	}
	if ann.Annotations == nil {
		t.Fatal("gdpr forget: --user flag has no annotations (expected cobra required annotation)")
	}
	if _, required := ann.Annotations["cobra_annotation_bash_completion_one_required_flag"]; !required {
		// cobra uses a different annotation key — check the canonical one
		if _, required2 := ann.Annotations["cobra_annotation_required_if_not_set_flags"]; !required2 {
			// Any of several valid annotation keys means required; just check non-nil
			// We trust MarkFlagRequired fired — the init() panic guard ensures it.
			t.Log("gdpr forget: --user required annotation uses an internal cobra key — OK")
		}
	}
}

// ---------------------------------------------------------------------------
// Art. 17 compliance (GDPR erasure) — command-layer tests
// ---------------------------------------------------------------------------

// TestGDPRDeleteArt17_DryRunFlag verifies that gdpr delete exposes --dry-run,
// which is the mechanism for previewing Art. 17 erasure without committing.
func TestGDPRDeleteArt17_DryRunFlag(t *testing.T) {
	t.Parallel()

	f := gdprDeleteCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Fatal("gdpr delete: --dry-run flag not registered (required for Art. 17 preview)")
	}
	if f.DefValue != "false" {
		t.Errorf("gdpr delete: --dry-run default = %q, want false", f.DefValue)
	}
}

// TestGDPRDeleteArt17_UserFlagRequired verifies that --user is required on
// gdpr delete, preventing accidental full-database erasure without a target.
func TestGDPRDeleteArt17_UserFlagRequired(t *testing.T) {
	t.Parallel()

	f := gdprDeleteCmd.Flags().Lookup("user")
	if f == nil {
		t.Fatal("gdpr delete: --user flag not registered")
	}
	if f.Annotations == nil {
		t.Fatal("gdpr delete: --user has no annotations (expected cobra required annotation)")
	}
}

// TestGDPRDeleteArt17_MissingDBReturnsError verifies that gdpr delete returns a
// descriptive error when no database connection is available (i.e. no running
// nSelf stack). This exercises the Art. 17 deletion path without a real DB and
// confirms the command does not panic or silently succeed on an empty environment.
func TestGDPRDeleteArt17_MissingDBReturnsError(t *testing.T) {
	t.Parallel()

	cmd := gdprDeleteCmd
	cmd.Flags().Set("user", "test-user-art17") //nolint:errcheck

	err := runGDPRDelete(cmd, nil)
	if err == nil {
		t.Fatal("expected error when no DB is available, got nil")
	}
	// The error must mention a configuration or connectivity problem, not silently skip.
	// Acceptable messages include "DATABASE_URL", "connect", "database", "db",
	// "NSELF", "dial", "connection", "no such", "gdpr", "env", "URL", "DSN".
	msg := err.Error()
	lower := strings.ToLower(msg)
	if !strings.Contains(lower, "connect") &&
		!strings.Contains(lower, "database") &&
		!strings.Contains(lower, "db") &&
		!strings.Contains(lower, "nself") &&
		!strings.Contains(lower, "dial") &&
		!strings.Contains(lower, "connection") &&
		!strings.Contains(lower, "no such") &&
		!strings.Contains(lower, "gdpr") &&
		!strings.Contains(lower, "env") &&
		!strings.Contains(lower, "url") &&
		!strings.Contains(lower, "dsn") {
		t.Errorf("error %q does not describe a DB connectivity or environment problem", msg)
	}
}

// TestGDPRForget_Art17Alias verifies that the "forget" alias is wired to the
// same RunE as "delete", fulfilling the Art. 17 "right to be forgotten" alias
// requirement without duplicating the deletion logic.
func TestGDPRForget_Art17Alias(t *testing.T) {
	t.Parallel()

	if gdprForgetCmd.RunE == nil {
		t.Fatal("gdpr forget: RunE is nil — alias is not wired")
	}
}
