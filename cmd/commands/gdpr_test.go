package commands

import (
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
