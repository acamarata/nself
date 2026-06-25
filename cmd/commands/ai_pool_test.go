package commands

import (
	"testing"
)

// ── T07: ai pool — S07 acceptance tests ──────────────────────────────────────

// TestAiPoolMaxKeys_Is30 verifies the pool cap constant is exactly 30 per ADR-006.
func TestAiPoolMaxKeys_Is30(t *testing.T) {
	if aiPoolMaxKeys != 30 {
		t.Errorf("aiPoolMaxKeys = %d, want 30 (ADR-006 / F-PLUGIN:ai-pool-keys)", aiPoolMaxKeys)
	}
}

// TestAiPoolCmd_Registered verifies nself ai pool is registered.
func TestAiPoolCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range aiCmd.Commands() {
		if sub.Name() == "pool" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("aiPoolCmd not registered on aiCmd")
	}
}

// TestAiPoolCmd_SubcommandsRegistered verifies all 8 pool subcommands exist.
func TestAiPoolCmd_SubcommandsRegistered(t *testing.T) {
	required := []string{"init", "status", "provision", "add", "remove", "rotate", "test", "daily-reset"}
	subs := map[string]bool{}
	for _, sub := range aiPoolCmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range required {
		if !subs[want] {
			t.Errorf("ai pool subcommand %q not registered", want)
		}
	}
}

// TestAiPoolStatusCmd_JSONFlag verifies --json flag is registered.
func TestAiPoolStatusCmd_JSONFlag(t *testing.T) {
	f := aiPoolStatusCmd.Flags().Lookup("json")
	if f == nil {
		t.Error("ai pool status: --json flag not registered")
	}
}

// TestAiPoolStatusCmd_VerboseFlag verifies --verbose flag is registered.
func TestAiPoolStatusCmd_VerboseFlag(t *testing.T) {
	f := aiPoolStatusCmd.Flags().Lookup("verbose")
	if f == nil {
		t.Error("ai pool status: --verbose flag not registered")
	}
}

// TestAiPoolAddCmd_AccountFlag verifies --account flag on pool add.
func TestAiPoolAddCmd_AccountFlag(t *testing.T) {
	f := aiPoolAddCmd.Flags().Lookup("account")
	if f == nil {
		t.Error("ai pool add: --account flag not registered")
	}
}

// TestAiPoolRemoveCmd_Flags verifies --key-id and --account flags on pool remove.
func TestAiPoolRemoveCmd_Flags(t *testing.T) {
	for _, flag := range []string{"key-id", "account"} {
		if f := aiPoolRemoveCmd.Flags().Lookup(flag); f == nil {
			t.Errorf("ai pool remove: --%s flag not registered", flag)
		}
	}
}

// TestAiPoolRotateCmd_KeyIDFlag verifies --key-id flag on pool rotate.
func TestAiPoolRotateCmd_KeyIDFlag(t *testing.T) {
	f := aiPoolRotateCmd.Flags().Lookup("key-id")
	if f == nil {
		t.Error("ai pool rotate: --key-id flag not registered")
	}
}

// TestAiPoolDailyResetCmd_DryRunFlag verifies --dry-run on pool daily-reset.
func TestAiPoolDailyResetCmd_DryRunFlag(t *testing.T) {
	f := aiPoolDailyResetCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Error("ai pool daily-reset: --dry-run flag not registered")
	}
}

// TestAiPoolProvisionCmd_AccountFlag verifies --account flag on pool provision.
func TestAiPoolProvisionCmd_AccountFlag(t *testing.T) {
	f := aiPoolProvisionCmd.Flags().Lookup("account")
	if f == nil {
		t.Error("ai pool provision: --account flag not registered")
	}
}
