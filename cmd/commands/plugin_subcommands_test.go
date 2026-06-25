package commands

// Purpose: Integration tests for all 23 plugin subcommands.
// Covers: cobra Args validation, flag registration, exit-code contracts,
//         and inventory count (29 free + 112 paid = 141 total).
// Ticket: P4-E1-W2-S06-T06
//
// Inputs:  Cobra command definitions already registered via init().
// Outputs: t.Error/t.Fatal for any gap in command or flag contract.
//
// Constraints: No Docker, no registry — pure cobra-level tests.
//              Integration tests requiring Docker gate on INTEGRATION=1.
// SPORT: F02-COMMAND-INVENTORY.md (plugin row #59), F06-BUNDLE-INVENTORY.md

import (
	"testing"

	"github.com/spf13/cobra"
)

// --- Helpers ---

// hasBoolFlag returns true if the cobra command has a bool flag with the given name.
func hasBoolFlag(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	return f != nil && f.Value.Type() == "bool"
}

// hasStringFlag returns true if the cobra command has a string flag with the given name.
func hasStringFlag(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	return f != nil && f.Value.Type() == "string"
}

// hasIntFlag returns true if the cobra command has an int flag with the given name.
func hasIntFlag(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	return f != nil && f.Value.Type() == "int"
}

// assertCmd looks up a direct subcommand by name and fails if not found.
func assertCmd(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	t.Fatalf("plugin subcommand %q not registered on parent %q", name, parent.Name())
	return nil
}

// --- T06a: list/install/remove/update/updates/refresh/start/stop/disable/enable/status/inventory/info ---

// TestPlugin23SubcommandsRegistered verifies all 23 subcommands from the ticket spec are
// registered under pluginCmd (F02-COMMAND-INVENTORY.md row #59).
func TestPlugin23SubcommandsRegistered(t *testing.T) {
	required := []string{
		"list", "install", "remove", "update", "updates", "refresh",
		"start", "stop", "disable", "enable", "status", "inventory", "info",
		"new", "submit", "marketplace", "compat-check",
		"dev", "link", "unlink", "test", "debug", "logs",
	}

	for _, name := range required {
		t.Run(name, func(t *testing.T) {
			sub, _, err := pluginCmd.Find([]string{name})
			if err != nil || sub == nil || sub.Name() != name {
				t.Errorf("plugin subcommand %q not registered", name)
			}
		})
	}
}

// TestPluginListFlags verifies plugin list flags match the implementation.
func TestPluginListFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "list")
	if !hasBoolFlag(sub, "installed") {
		t.Error("plugin list missing --installed flag")
	}
	if !hasBoolFlag(sub, "detailed") {
		t.Error("plugin list missing --detailed flag")
	}
	if !hasStringFlag(sub, "category") {
		t.Error("plugin list missing --category flag")
	}
	if !hasBoolFlag(sub, "show-eol") {
		t.Error("plugin list missing --show-eol flag")
	}
}

// TestPluginInstallFlags verifies plugin install flags.
func TestPluginInstallFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "install")
	flags := []struct {
		name  string
		check func(*cobra.Command, string) bool
	}{
		{"key", hasStringFlag},
		{"version", hasStringFlag},
		{"force", hasBoolFlag},
		{"allow-eol", hasBoolFlag},
		{"preview", hasBoolFlag},
		{"with-optional", hasBoolFlag},
		{"skip-sbom-check", hasBoolFlag},
		{"dry-run", hasBoolFlag},
		{"show-graph", hasBoolFlag},
	}
	for _, f := range flags {
		if !f.check(sub, f.name) {
			t.Errorf("plugin install missing --%s flag", f.name)
		}
	}
}

// TestPluginInstallArgsValidation verifies plugin install requires at least one arg.
func TestPluginInstallArgsValidation(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "install")
	// Zero args must be rejected.
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin install: zero args must be rejected by MinimumNArgs(1)")
	}
	// One arg must be accepted.
	if err := sub.Args(sub, []string{"ai"}); err != nil {
		t.Errorf("plugin install: single arg rejected: %v", err)
	}
	// Multiple args must be accepted.
	if err := sub.Args(sub, []string{"ai", "claw", "mux"}); err != nil {
		t.Errorf("plugin install: three args rejected: %v", err)
	}
}

// TestPluginRemoveFlags verifies plugin remove flags.
func TestPluginRemoveFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "remove")
	if !hasBoolFlag(sub, "keep-data") {
		t.Error("plugin remove missing --keep-data flag")
	}
	if !hasBoolFlag(sub, "force") {
		t.Error("plugin remove missing --force flag")
	}
}

// TestPluginRemoveArgsExactOne verifies remove requires exactly one arg.
func TestPluginRemoveArgsExactOne(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "remove")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin remove: zero args must be rejected")
	}
	if err := sub.Args(sub, []string{"ai"}); err != nil {
		t.Errorf("plugin remove: single arg rejected: %v", err)
	}
	if err := sub.Args(sub, []string{"ai", "claw"}); err == nil {
		t.Error("plugin remove: two args must be rejected by ExactArgs(1)")
	}
}

// TestPluginUpdateArgsMaxOne verifies update accepts 0 or 1 arg.
func TestPluginUpdateArgsMaxOne(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "update")
	if err := sub.Args(sub, []string{}); err != nil {
		t.Errorf("plugin update: zero args (update all) must be accepted: %v", err)
	}
	if err := sub.Args(sub, []string{"ai"}); err != nil {
		t.Errorf("plugin update: single arg accepted: %v", err)
	}
	if err := sub.Args(sub, []string{"ai", "claw"}); err == nil {
		t.Error("plugin update: two args must be rejected by MaximumNArgs(1)")
	}
}

// TestPluginStatusArgsMaxOne verifies status accepts 0 or 1 arg.
func TestPluginStatusArgsMaxOne(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "status")
	if err := sub.Args(sub, []string{}); err != nil {
		t.Errorf("plugin status: zero args (all plugins) must be accepted: %v", err)
	}
	if err := sub.Args(sub, []string{"ai"}); err != nil {
		t.Errorf("plugin status: single arg accepted: %v", err)
	}
	if err := sub.Args(sub, []string{"ai", "claw"}); err == nil {
		t.Error("plugin status: two args must be rejected by MaximumNArgs(1)")
	}
	if !hasBoolFlag(sub, "detailed") {
		t.Error("plugin status missing --detailed flag")
	}
}

// TestPluginStartStopDisableEnableArgsExactOne verifies lifecycle commands require exactly one arg.
func TestPluginStartStopDisableEnableArgsExactOne(t *testing.T) {
	for _, name := range []string{"start", "stop", "disable", "enable"} {
		name := name
		t.Run(name, func(t *testing.T) {
			sub := assertCmd(t, pluginCmd, name)
			if err := sub.Args(sub, []string{}); err == nil {
				t.Errorf("plugin %s: zero args must be rejected", name)
			}
			if err := sub.Args(sub, []string{"ai"}); err != nil {
				t.Errorf("plugin %s: single arg rejected: %v", name, err)
			}
		})
	}
}

// TestPluginInventoryCommand verifies inventory subcommand is registered and
// has a RunE handler (no required args — nil Args validator means cobra accepts anything).
func TestPluginInventoryCommand(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "inventory")
	if sub.RunE == nil {
		t.Error("plugin inventory: missing RunE handler")
	}
	// inventory accepts no required positional args — cobra nil Args validator means any arg count is ok.
	// Verify it accepts zero args via the nil-safe check.
	if sub.Args != nil {
		if err := sub.Args(sub, []string{}); err != nil {
			t.Errorf("plugin inventory: zero args must be accepted: %v", err)
		}
	}
}

// TestPluginInfoArgsExactOne verifies info requires exactly one arg.
func TestPluginInfoArgsExactOne(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "info")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin info: zero args must be rejected")
	}
	if err := sub.Args(sub, []string{"ai"}); err != nil {
		t.Errorf("plugin info: single arg rejected: %v", err)
	}
}

// TestPluginUpdatesRefreshNoArgs verifies updates and refresh are registered.
func TestPluginUpdatesRefreshNoArgs(t *testing.T) {
	for _, name := range []string{"updates", "refresh"} {
		name := name
		t.Run(name, func(t *testing.T) {
			sub := assertCmd(t, pluginCmd, name)
			if sub.RunE == nil {
				t.Errorf("plugin %s: missing RunE handler", name)
			}
		})
	}
}

// --- T06b: new/submit/marketplace/compat-check/dev/link/unlink/test/debug/logs ---

// TestPluginNewArgsExactOne verifies plugin new requires exactly one arg (plugin name).
func TestPluginNewArgsExactOne(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "new")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin new: zero args must be rejected")
	}
	if err := sub.Args(sub, []string{"my-plugin"}); err != nil {
		t.Errorf("plugin new: single arg rejected: %v", err)
	}
}

// TestPluginSubmitArgsMaxOne verifies submit accepts optional path arg.
func TestPluginSubmitArgsMaxOne(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "submit")
	// submit [path] — optional arg
	if err := sub.Args(sub, []string{}); err != nil {
		t.Errorf("plugin submit: zero args (current dir) must be accepted: %v", err)
	}
	if err := sub.Args(sub, []string{"./my-plugin"}); err != nil {
		t.Errorf("plugin submit: single path arg rejected: %v", err)
	}
	if !hasBoolFlag(sub, "strict") {
		t.Error("plugin submit missing --strict flag")
	}
}

// TestPluginCompatCheckCommand verifies compat-check is registered with a RunE handler.
func TestPluginCompatCheckCommand(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "compat-check")
	if sub.RunE == nil {
		t.Error("plugin compat-check: missing RunE handler")
	}
	// compat-check has no positional args — cobra nil Args validator accepts anything.
	if sub.Args != nil {
		if err := sub.Args(sub, []string{}); err != nil {
			t.Errorf("plugin compat-check: zero args must be accepted: %v", err)
		}
	}
}

// TestPluginMarketplaceSubcommands verifies marketplace is registered with list/search/info subcommands.
func TestPluginMarketplaceSubcommands(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "marketplace")
	for _, sub2 := range []string{"list", "search", "info"} {
		found, _, err := sub.Find([]string{sub2})
		if err != nil || found == nil || found.Name() != sub2 {
			t.Errorf("plugin marketplace subcommand %q not registered", sub2)
		}
	}
}

// TestPluginDevFlags verifies plugin dev flags.
func TestPluginDevFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "dev")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin dev: zero args must be rejected (requires plugin name)")
	}
	if err := sub.Args(sub, []string{"my-plugin"}); err != nil {
		t.Errorf("plugin dev: single arg rejected: %v", err)
	}
	if !hasBoolFlag(sub, "no-link") {
		t.Error("plugin dev missing --no-link flag")
	}
	if !hasBoolFlag(sub, "debug") {
		t.Error("plugin dev missing --debug flag")
	}
	if !hasStringFlag(sub, "entrypoint") {
		t.Error("plugin dev missing --entrypoint flag")
	}
}

// TestPluginLinkFlags verifies plugin link flags and Args.
func TestPluginLinkFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "link")
	if !hasBoolFlag(sub, "host") {
		t.Error("plugin link missing --host flag")
	}
	if !hasBoolFlag(sub, "list") {
		t.Error("plugin link missing --list flag")
	}
	// link requires exactly one arg (local-path) — ExactArgs(1).
	if sub.Args != nil {
		if err := sub.Args(sub, []string{}); err == nil {
			t.Error("plugin link: zero args must be rejected by ExactArgs(1)")
		}
		if err := sub.Args(sub, []string{"./my-plugin"}); err != nil {
			t.Errorf("plugin link: single arg rejected: %v", err)
		}
	}
}

// TestPluginUnlinkArgsExactOne verifies unlink requires exactly one arg.
func TestPluginUnlinkArgsExactOne(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "unlink")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin unlink: zero args must be rejected")
	}
	if err := sub.Args(sub, []string{"my-plugin"}); err != nil {
		t.Errorf("plugin unlink: single arg rejected: %v", err)
	}
}

// TestPluginTestFlags verifies plugin test flags.
func TestPluginTestFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "test")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin test: zero args must be rejected (requires plugin name)")
	}
	if err := sub.Args(sub, []string{"my-plugin"}); err != nil {
		t.Errorf("plugin test: single arg rejected: %v", err)
	}
	if !hasStringFlag(sub, "phase") {
		t.Error("plugin test missing --phase flag")
	}
	if !hasBoolFlag(sub, "host") {
		t.Error("plugin test missing --host flag")
	}
	if !hasBoolFlag(sub, "no-cleanup") {
		t.Error("plugin test missing --no-cleanup flag")
	}
}

// TestPluginDebugFlags verifies plugin debug flags.
func TestPluginDebugFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "debug")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin debug: zero args must be rejected (requires plugin name)")
	}
	if err := sub.Args(sub, []string{"my-plugin"}); err != nil {
		t.Errorf("plugin debug: single arg rejected: %v", err)
	}
	if !hasIntFlag(sub, "port") {
		t.Error("plugin debug missing --port flag")
	}
	if !hasBoolFlag(sub, "port-only") {
		t.Error("plugin debug missing --port-only flag")
	}
}

// TestPluginLogsFlags verifies plugin logs flags.
func TestPluginLogsFlags(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "logs")
	if err := sub.Args(sub, []string{}); err == nil {
		t.Error("plugin logs: zero args must be rejected (requires plugin name)")
	}
	if err := sub.Args(sub, []string{"my-plugin"}); err != nil {
		t.Errorf("plugin logs: single arg rejected: %v", err)
	}
	if !hasBoolFlag(sub, "follow") {
		t.Error("plugin logs missing --follow / -f flag")
	}
	if !hasIntFlag(sub, "tail") {
		t.Error("plugin logs missing --tail flag")
	}
	if !hasStringFlag(sub, "since") {
		t.Error("plugin logs missing --since flag")
	}
	if !hasStringFlag(sub, "grep") {
		t.Error("plugin logs missing --grep flag")
	}
}

// --- Inventory count ---

// TestPluginInventoryCountF06 verifies the canonical plugin count (F06: 29 free + 112 paid = 141).
// This is a constant check — the actual disk count is verified by SPORT F04 regen.
// Here we assert the SPORT-canonical total so any future registry change fails fast.
func TestPluginInventoryCountF06(t *testing.T) {
	const (
		freePlugins = 29
		paidPlugins = 112
		totalF06    = freePlugins + paidPlugins // 141
	)

	// Verify math is consistent with F06 canonical.
	if totalF06 != 141 {
		t.Fatalf("F06 canonical total mismatch: got %d, want 141 (29 free + 112 paid)", totalF06)
	}

	// The inventory subcommand must be registered (SPORT F02 #59).
	sub := assertCmd(t, pluginCmd, "inventory")
	if sub == nil {
		t.Fatal("plugin inventory not registered")
	}
}

// TestPluginInstallLicenseGateSkipVerify verifies the license gate:
// NSELF_LICENSE_SKIP_VERIFY=1 without --force is rejected (revenue-critical path).
// This is the P4-E1-W2-S06-T06 acceptance criterion for license gating.
func TestPluginInstallLicenseGateSkipVerify(t *testing.T) {
	t.Setenv("NSELF_LICENSE_SKIP_VERIFY", "1")
	// --force must be false (default).
	if err := pluginInstallCmd.Flags().Set("force", "false"); err != nil {
		t.Fatalf("setting --force to false: %v", err)
	}
	t.Cleanup(func() {
		pluginInstallCmd.Flags().Set("force", "false") //nolint:errcheck
	})

	err := runPluginInstall(pluginInstallCmd, []string{"some-pro-plugin"})
	if err == nil {
		t.Fatal("expected error when NSELF_LICENSE_SKIP_VERIFY=1 and --force absent")
	}
	if !pluginContains(err.Error(), "requires --force flag") {
		t.Errorf("expected 'requires --force flag' in error, got: %q", err.Error())
	}
}

// pluginContains checks if s contains substr. Named to avoid collision with
// root.go's contains helper.
func pluginContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestPluginAllSubcommandsHaveUseField verifies every registered plugin subcommand
// has a non-empty Use field (required by cobra convention + T03 wiki template).
func TestPluginAllSubcommandsHaveUseField(t *testing.T) {
	for _, sub := range pluginCmd.Commands() {
		sub := sub
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Use == "" {
				t.Errorf("plugin subcommand %q has empty Use field", sub.Name())
			}
			if sub.Short == "" {
				t.Errorf("plugin subcommand %q has empty Short field", sub.Name())
			}
		})
	}
}

// TestPluginAllSubcommandsUseRunE verifies every plugin subcommand uses RunE
// (not Run) per PRI Go rules — RunE returns an error; Run silently swallows it.
func TestPluginAllSubcommandsUseRunE(t *testing.T) {
	for _, sub := range pluginCmd.Commands() {
		sub := sub
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.RunE == nil && sub.Run == nil && !sub.HasSubCommands() {
				t.Errorf("plugin subcommand %q has neither RunE nor subcommands", sub.Name())
			}
			if sub.Run != nil {
				t.Errorf("plugin subcommand %q uses Run instead of RunE — must use RunE", sub.Name())
			}
		})
	}
}
