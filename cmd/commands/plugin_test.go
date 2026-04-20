package commands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestPluginInstallMultiArg verifies that the install command accepts multiple
// plugin name arguments (MinimumNArgs(1)) and that the cobra Args validator does
// not reject a call with 3 arguments. This is the S01.T01 regression test.
func TestPluginInstallMultiArg(t *testing.T) {
	// Verify the command Use string is updated to show multiple-arg syntax.
	if !strings.Contains(pluginInstallCmd.Use, "[plugin...]") {
		t.Errorf("S01.T01 regression: pluginInstallCmd.Use = %q, expected to contain %q",
			pluginInstallCmd.Use, "[plugin...]")
	}

	// Verify Args validator allows 1 argument.
	if err := pluginInstallCmd.Args(pluginInstallCmd, []string{"ai"}); err != nil {
		t.Errorf("single arg rejected: %v", err)
	}

	// Verify Args validator allows 3 arguments.
	if err := pluginInstallCmd.Args(pluginInstallCmd, []string{"ai", "claw", "mux"}); err != nil {
		t.Errorf("S01.T01 regression: three args rejected by cobra validator: %v", err)
	}

	// Verify Args validator rejects 0 arguments (MinimumNArgs(1) must still enforce minimum).
	err := pluginInstallCmd.Args(pluginInstallCmd, []string{})
	if err == nil {
		t.Error("S01.T01 regression: zero args must be rejected by MinimumNArgs(1), but got nil error")
	}

	// Verify the configured Args is MinimumNArgs(1) by checking its type via the
	// behaviour rather than reflection — cobra's ExactArgs returns an error for 2
	// args; MinimumNArgs should not.
	exactOne := cobra.ExactArgs(1)
	if exactOne(pluginInstallCmd, []string{"ai", "claw"}) == nil {
		t.Fatal("test setup error: ExactArgs(1) should reject 2 args")
	}
	// pluginInstallCmd.Args must accept 2 args (MinimumNArgs behaviour).
	if err := pluginInstallCmd.Args(pluginInstallCmd, []string{"ai", "claw"}); err != nil {
		t.Errorf("S01.T01 regression: two args must be accepted, got: %v", err)
	}
}
