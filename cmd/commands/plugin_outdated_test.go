package commands

// Purpose: Tests for `nself plugin outdated` (CLI-R16) — registration,
//          exit-code contract, and the version-comparison happy path against
//          an httptest.Server standing in for the registry.
// Constraints: No Docker, no real network — the registry is always an
//              httptest.Server; the plugin directory is always a t.TempDir().

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/errs"
)

// writeInstalledPlugin writes a minimal valid plugin.json for name/version
// under pluginDir/name/plugin.json, matching what plugin.List(dir, true)
// expects to find.
func writeInstalledPlugin(t *testing.T, pluginDir, name, version string) {
	t.Helper()
	dir := filepath.Join(pluginDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	manifest := `{"name":"` + name + `","version":"` + version + `","description":"test","category":"utility","license":"MIT"}`
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// newOutdatedTestCmd returns an isolated invocation of pluginOutdatedCmd with
// stdout/stderr captured and a background context set (RunE reads
// cmd.Context(), which is nil unless a test sets it explicitly).
func newOutdatedTestCmd(t *testing.T, jsonOut bool) (run func() error, stdout func() string) {
	t.Helper()
	cmd := pluginOutdatedCmd
	if err := cmd.Flags().Set("json", boolStr(jsonOut)); err != nil {
		t.Fatalf("setting --json: %v", err)
	}
	t.Cleanup(func() {
		cmd.Flags().Set("json", "false") //nolint:errcheck
	})
	cmd.SetContext(context.Background())

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	run = func() error {
		err := runPluginOutdated(cmd, nil)
		_ = w.Close()
		os.Stdout = origStdout
		return err
	}
	stdout = func() string {
		var buf strings.Builder
		tmp := make([]byte, 4096)
		n, _ := r.Read(tmp)
		buf.Write(tmp[:n])
		return buf.String()
	}
	return run, stdout
}

// TestPluginOutdatedRegistered verifies the outdated subcommand is wired up
// under `plugin` with the expected flags and RunE handler.
func TestPluginOutdatedRegistered(t *testing.T) {
	sub := assertCmd(t, pluginCmd, "outdated")
	if sub.RunE == nil {
		t.Error("plugin outdated: missing RunE handler")
	}
	if !hasBoolFlag(sub, "json") {
		t.Error("plugin outdated: missing --json flag")
	}
}

// TestPluginOutdated_NoneInstalled verifies the no-plugins-installed case
// exits 0 with a plain message (and an empty JSON array under --json).
func TestPluginOutdated_NoneInstalled(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_DIR", t.TempDir())
	setPluginTestHome(t, t.TempDir())

	run, stdout := newOutdatedTestCmd(t, false)
	if err := run(); err != nil {
		t.Fatalf("expected nil error with nothing installed, got: %v", err)
	}
	if !strings.Contains(stdout(), "No plugins installed") {
		t.Errorf("expected 'No plugins installed' message, got: %q", stdout())
	}
}

// TestPluginOutdated_NoneInstalledNeedsNoNetwork pins the reason
// TestPluginOutdated_NoneInstalled kept failing in CI.
//
// runPluginOutdated used to probe the registry for reachability BEFORE checking
// whether anything was installed. With nothing installed there is nothing to
// compare a registry against, so the probe was wasted work that could only ever
// turn a locally-answerable question into a failure. In CI it did exactly that:
//
//	cannot reach plugin registry https://plugins.nself.org/registry.json:
//	Head "...": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
//
// This points the registry at an address that cannot answer, so if the probe
// ever moves back above the short-circuit this test fails immediately instead
// of intermittently, and on a developer's machine rather than in CI.
func TestPluginOutdated_NoneInstalledNeedsNoNetwork(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_DIR", t.TempDir())
	setPluginTestHome(t, t.TempDir())

	// Reserved as invalid-for-any-use by RFC 6890; nothing can answer here.
	t.Setenv("NSELF_PLUGIN_REGISTRY", "http://192.0.2.0:1/registry.json")

	run, stdout := newOutdatedTestCmd(t, false)
	if err := run(); err != nil {
		t.Fatalf("nothing installed must be answerable without the network, got: %v", err)
	}
	if !strings.Contains(stdout(), "No plugins installed") {
		t.Errorf("expected 'No plugins installed', got: %q", stdout())
	}
}

// TestPluginOutdated_AllCurrent verifies exit 0 and "up to date" messaging
// when every installed plugin matches the registry's version.
func TestPluginOutdated_AllCurrent(t *testing.T) {
	pluginDir := t.TempDir()
	writeInstalledPlugin(t, pluginDir, "demo-plugin", "1.0.0")
	t.Setenv("NSELF_PLUGIN_DIR", pluginDir)
	setPluginTestHome(t, t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plugins":[{"name":"demo-plugin","version":"1.0.0","description":"test","category":"utility","license":"MIT"}]}`))
	}))
	defer srv.Close()
	t.Setenv("NSELF_PLUGIN_REGISTRY", srv.URL)

	run, stdout := newOutdatedTestCmd(t, false)
	if err := run(); err != nil {
		t.Fatalf("expected nil error when everything is current, got: %v", err)
	}
	if !strings.Contains(stdout(), "up to date") {
		t.Errorf("expected 'up to date' message, got: %q", stdout())
	}
}

// TestPluginOutdated_ReportsOutdatedAndExitsOne is the happy-path test: one
// installed plugin is behind the registry version. Table output must list
// it, and the command must signal exit code 1 via errs.Exit rather than
// os.Exit (internal/repoqa/os_exit_test.go forbids the latter outside main).
func TestPluginOutdated_ReportsOutdatedAndExitsOne(t *testing.T) {
	pluginDir := t.TempDir()
	writeInstalledPlugin(t, pluginDir, "demo-plugin", "1.0.0")
	t.Setenv("NSELF_PLUGIN_DIR", pluginDir)
	setPluginTestHome(t, t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plugins":[{"name":"demo-plugin","version":"2.0.0","description":"test","category":"utility","license":"MIT"}]}`))
	}))
	defer srv.Close()
	t.Setenv("NSELF_PLUGIN_REGISTRY", srv.URL)

	run, stdout := newOutdatedTestCmd(t, false)
	err := run()
	if err == nil {
		t.Fatal("expected an error signaling exit 1 when a plugin is outdated, got nil")
	}
	exitErr, ok := err.(errs.ExitCoder)
	if !ok {
		t.Fatalf("expected an errs.ExitCoder, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("ExitCode() = %d, want 1", exitErr.ExitCode())
	}

	out := stdout()
	if !strings.Contains(out, "demo-plugin") || !strings.Contains(out, "1.0.0") || !strings.Contains(out, "2.0.0") {
		t.Errorf("expected table listing demo-plugin 1.0.0 -> 2.0.0, got: %q", out)
	}
}

// TestPluginOutdated_JSONOutput verifies --json emits a parseable array with
// the expected fields, and still signals exit 1 when something is outdated.
func TestPluginOutdated_JSONOutput(t *testing.T) {
	pluginDir := t.TempDir()
	writeInstalledPlugin(t, pluginDir, "demo-plugin", "1.0.0")
	t.Setenv("NSELF_PLUGIN_DIR", pluginDir)
	setPluginTestHome(t, t.TempDir())

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plugins":[{"name":"demo-plugin","version":"2.0.0","description":"test","category":"utility","license":"MIT"}]}`))
	}))
	defer srv.Close()
	t.Setenv("NSELF_PLUGIN_REGISTRY", srv.URL)

	run, stdout := newOutdatedTestCmd(t, true)
	err := run()
	if err == nil {
		t.Fatal("expected exit-1 error, got nil")
	}

	var rows []outdatedPluginRow
	if jsonErr := json.Unmarshal([]byte(stdout()), &rows); jsonErr != nil {
		t.Fatalf("--json output did not parse as JSON: %v\noutput: %q", jsonErr, stdout())
	}
	if len(rows) != 1 || rows[0].Name != "demo-plugin" || rows[0].Installed != "1.0.0" || rows[0].Latest != "2.0.0" {
		t.Errorf("unexpected JSON rows: %+v", rows)
	}
}

// setPluginTestHome points os.UserHomeDir at dir on every platform.
//
// os.UserHomeDir reads $HOME on Unix but %USERPROFILE% on Windows, so a test
// that sets only HOME silently exercises the developer's real home directory
// on Windows — which is how these tests passed locally and failed on the
// windows-2022 runner.
func setPluginTestHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}
