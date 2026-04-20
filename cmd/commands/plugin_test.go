package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestPluginInstallSkipVerifyRequiresForce verifies that NSELF_LICENSE_SKIP_VERIFY=1
// without --force is rejected with a clear error message (S09.T-SEC-01).
func TestPluginInstallSkipVerifyRequiresForce(t *testing.T) {
	t.Setenv("NSELF_LICENSE_SKIP_VERIFY", "1")
	// Ensure --force is NOT set (default false).
	pluginInstallCmd.Flags().Set("force", "false") //nolint:errcheck

	var errMsg string
	// Invoke the RunE handler directly with a minimal cobra context.
	cmd := pluginInstallCmd
	cmd.SetArgs([]string{"ai"})
	var buf strings.Builder
	cmd.SetErr(&buf)

	err := runPluginInstall(cmd, []string{"ai"})
	if err == nil {
		t.Fatal("expected error when NSELF_LICENSE_SKIP_VERIFY=1 without --force, got nil")
	}
	errMsg = err.Error()
	if !strings.Contains(errMsg, "requires --force flag") {
		t.Errorf("expected 'requires --force flag' in error, got: %q", errMsg)
	}
}

// TestPluginInstallSkipVerifyWithForceEmitsWarning verifies that
// NSELF_LICENSE_SKIP_VERIFY=1 WITH --force proceeds and emits a warning to
// stderr (S09.T-SEC-01 acceptance criterion: warning emitted when --force used).
func TestPluginInstallSkipVerifyWithForceEmitsWarning(t *testing.T) {
	t.Setenv("NSELF_LICENSE_SKIP_VERIFY", "1")

	// Point to unreachable registry so Install fails fast after the gate check.
	t.Setenv("NSELF_PLUGIN_REGISTRY", "http://127.0.0.1:19999/unreachable")

	if err := pluginInstallCmd.Flags().Set("force", "true"); err != nil {
		t.Fatalf("setting --force flag: %v", err)
	}
	t.Cleanup(func() {
		pluginInstallCmd.Flags().Set("force", "false") //nolint:errcheck
	})

	// Capture stderr.
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// runPluginInstall will fail at network validation, but the SKIP_VERIFY warning
	// must have been emitted to stderr before that failure.
	runPluginInstall(pluginInstallCmd, []string{"ai"}) //nolint:errcheck

	w.Close()
	os.Stderr = origStderr
	var buf strings.Builder
	buf.Grow(512)
	tmp := make([]byte, 512)
	n, _ := r.Read(tmp)
	buf.Write(tmp[:n])

	stderrOutput := buf.String()
	if !strings.Contains(stderrOutput, "license verification bypassed") {
		t.Errorf("expected 'license verification bypassed' warning in stderr, got: %q", stderrOutput)
	}
}

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
