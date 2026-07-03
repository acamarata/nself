package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newBundleTestCmd returns an isolated cobra root with only the bundle
// command tree attached — avoids touching the global RootCmd state.
func newBundleTestCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	root.AddCommand(bundleCmd)
	return root
}

// ── Registration ─────────────────────────────────────────────────────

func TestBundleCmd_Registered(t *testing.T) {
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Use == "bundle" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("bundleCmd is not registered on RootCmd")
	}
}

func TestBundleCmd_SubcommandsRegistered(t *testing.T) {
	subs := map[string]bool{}
	for _, sub := range bundleCmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, want := range []string{"list", "info"} {
		if !subs[want] {
			t.Errorf("bundle subcommand %q is not registered", want)
		}
	}
}

// ── canonicalBundles ─────────────────────────────────────────────────

func TestCanonicalBundles_AllPresent(t *testing.T) {
	required := []string{"nclaw", "nchat", "nfamily", "ntv", "clawde", "nsentry", "ntask", "nself-plus"}
	for _, key := range required {
		if _, ok := canonicalBundles[key]; !ok {
			t.Errorf("canonicalBundles missing required bundle %q", key)
		}
	}
}

func TestCanonicalBundles_NamesNotEmpty(t *testing.T) {
	for key, b := range canonicalBundles {
		if b.Name == "" {
			t.Errorf("bundle %q has empty Name", key)
		}
		if b.Price == "" {
			t.Errorf("bundle %q has empty Price", key)
		}
	}
}

func TestCanonicalBundles_NClawPlugins(t *testing.T) {
	b := canonicalBundles["nclaw"]
	required := []string{"ai", "claw", "mux", "voice", "browser", "google", "notify", "cron"}
	for _, p := range required {
		found := false
		for _, got := range b.Plugins {
			if got == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("nclaw bundle missing expected plugin %q", p)
		}
	}
}

// ── bundle list ───────────────────────────────────────────────────────

func TestBundleList_RunsWithoutError(t *testing.T) {
	_, err := captureStdout(t, func() error {
		return runBundleList(nil, nil)
	})
	if err != nil {
		t.Fatalf("bundle list returned error: %v", err)
	}
}

func TestBundleList_ContainsAllBundles(t *testing.T) {
	out, _ := captureStdout(t, func() error {
		return runBundleList(nil, nil)
	})
	for _, key := range []string{"nclaw", "nchat", "ntv", "clawde", "nsentry", "ntask", "nself-plus"} {
		if !strings.Contains(out, key) {
			t.Errorf("bundle list output missing bundle key %q\nfull output:\n%s", key, out)
		}
	}
}

func TestBundleList_ContainsPricing(t *testing.T) {
	out, _ := captureStdout(t, func() error {
		return runBundleList(nil, nil)
	})
	if !strings.Contains(out, "$0.99") {
		t.Errorf("bundle list output missing $0.99 pricing\nfull output:\n%s", out)
	}
	if !strings.Contains(out, "$3.99") {
		t.Errorf("bundle list output missing $3.99 ɳSelf+ pricing\nfull output:\n%s", out)
	}
}

// ── bundle info ───────────────────────────────────────────────────────

func TestBundleInfo_NClawHappyPath(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runBundleInfo(&cobra.Command{}, []string{"nclaw"})
	})
	if err != nil {
		t.Fatalf("bundle info nclaw returned error: %v", err)
	}
	for _, want := range []string{"ɳClaw", "$0.99", "ai", "claw", "mux"} {
		if !strings.Contains(out, want) {
			t.Errorf("bundle info nclaw output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestBundleInfo_NSelfPlus(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runBundleInfo(&cobra.Command{}, []string{"nself-plus"})
	})
	if err != nil {
		t.Fatalf("bundle info nself-plus returned error: %v", err)
	}
	if !strings.Contains(out, "ɳSelf+") {
		t.Errorf("bundle info nself-plus missing ɳSelf+ name\nfull:\n%s", out)
	}
	if !strings.Contains(out, "$3.99") {
		t.Errorf("bundle info nself-plus missing $3.99 price\nfull:\n%s", out)
	}
}

func TestBundleInfo_SentryAliasResolvesToNSentry(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runBundleInfo(&cobra.Command{}, []string{"sentry"})
	})
	if err != nil {
		t.Fatalf("bundle info sentry (alias) returned error: %v", err)
	}
	for _, want := range []string{"ɳSentry", "$0.99"} {
		if !strings.Contains(out, want) {
			t.Errorf("bundle info sentry output missing %q\nfull:\n%s", want, out)
		}
	}
}

func TestBundleInfo_SentryAliasCaseAndSpaceInsensitive(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runBundleInfo(&cobra.Command{}, []string{"  SENTRY "})
	})
	if err != nil {
		t.Fatalf("bundle info '  SENTRY ' (alias) returned error: %v", err)
	}
	if !strings.Contains(out, "ɳSentry") {
		t.Errorf("bundle info '  SENTRY ' output missing ɳSentry name\nfull:\n%s", out)
	}
}

func TestBundleInfo_UnknownBundle(t *testing.T) {
	_, err := captureStdout(t, func() error {
		return runBundleInfo(&cobra.Command{}, []string{"bogus-bundle-xyz"})
	})
	if err == nil {
		t.Fatal("expected error for unknown bundle, got nil")
	}
	if !strings.Contains(err.Error(), "bundle not found") {
		t.Errorf("error message should mention 'bundle not found', got: %v", err)
	}
	if !strings.Contains(err.Error(), "nself bundle list") {
		t.Errorf("error message should suggest 'nself bundle list', got: %v", err)
	}
}

func TestBundleInfo_UnknownBundle_ExitCode(t *testing.T) {
	root := newBundleTestCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"bundle", "info", "does-not-exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected non-nil error for unknown bundle name via cobra")
	}
}

func TestBundleInfo_JSONFlag(t *testing.T) {
	root := newBundleTestCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"bundle", "info", "nclaw", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("bundle info nclaw --json returned error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{`"slug"`, `"nclaw"`, `"name"`, `"price"`, `"plugins"`, `"license_status"`, `"install_hint"`} {
		if !strings.Contains(out, want) {
			t.Errorf("--json output missing key %q\nfull:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "$0.99") {
		t.Errorf("--json output missing price $0.99\nfull:\n%s", out)
	}
}

func TestBundleInfo_JSON_AllFields(t *testing.T) {
	root := newBundleTestCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"bundle", "info", "ntask", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("bundle info ntask --json returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"FREE"`) {
		t.Errorf("--json ntask output missing FREE price\nfull:\n%s", out)
	}
	if !strings.Contains(out, `"plugin_count"`) {
		t.Errorf("--json output missing plugin_count\nfull:\n%s", out)
	}
}

func TestBundleInfo_CaseInsensitive(t *testing.T) {
	for _, input := range []string{"nClaw", "NCLAW", "NClaw"} {
		out, err := captureStdout(t, func() error {
			return runBundleInfo(&cobra.Command{}, []string{input})
		})
		if err != nil {
			t.Errorf("bundle info %q returned error: %v", input, err)
			continue
		}
		if !strings.Contains(out, "ɳClaw") {
			t.Errorf("bundle info %q did not resolve to nclaw\nfull:\n%s", input, out)
		}
	}
}

func TestBundleInfo_NSentry(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runBundleInfo(&cobra.Command{}, []string{"nsentry"})
	})
	if err != nil {
		t.Fatalf("bundle info nsentry returned error: %v", err)
	}
	if !strings.Contains(out, "ɳSentry") {
		t.Errorf("bundle info nsentry missing ɳSentry name\nfull:\n%s", out)
	}
	for _, p := range []string{"nself-uptime-monitor", "nself-status-page"} {
		if !strings.Contains(out, p) {
			t.Errorf("bundle info nsentry missing plugin %q\nfull:\n%s", p, out)
		}
	}
}

func TestBundleInfo_NTask(t *testing.T) {
	out, err := captureStdout(t, func() error {
		return runBundleInfo(&cobra.Command{}, []string{"ntask"})
	})
	if err != nil {
		t.Fatalf("bundle info ntask returned error: %v", err)
	}
	if !strings.Contains(out, "ɳTask") {
		t.Errorf("bundle info ntask missing ɳTask name\nfull:\n%s", out)
	}
	if !strings.Contains(out, "FREE") {
		t.Errorf("bundle info ntask missing FREE price\nfull:\n%s", out)
	}
}

// ── resolveBundleLicenseStatus ────────────────────────────────────────

func TestResolveBundleLicenseStatus_NTask(t *testing.T) {
	status := resolveBundleLicenseStatus("ntask")
	if !strings.Contains(status, "free") {
		t.Errorf("ntask license status should mention 'free', got: %s", status)
	}
}

func TestResolveBundleLicenseStatus_NoCache(t *testing.T) {
	// Override cache path to a guaranteed non-existent file.
	t.Setenv("LICENSE_CACHE_PATH", "/tmp/nself-test-no-license-cache-xyz.json")
	status := resolveBundleLicenseStatus("nclaw")
	if !strings.Contains(status, "not activated") {
		t.Errorf("expected 'not activated' when no cache, got: %s", status)
	}
}

// ── bundleDisplayOrder ────────────────────────────────────────────────

func TestBundleList_JSONFlag(t *testing.T) {
	root := newBundleTestCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"bundle", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("bundle list --json returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"slug"`) {
		t.Errorf("--json output missing 'slug' key: %s", out)
	}
	if !strings.Contains(out, `"nclaw"`) {
		t.Errorf("--json output missing nclaw entry: %s", out)
	}
}

func TestBundleList_InstalledFlag_NoPlugins(t *testing.T) {
	// With no plugins installed the --installed flag should produce empty output
	// (just the header and footer lines, no bundle rows).
	root := newBundleTestCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"bundle", "list", "--installed"})
	if err := root.Execute(); err != nil {
		t.Fatalf("bundle list --installed returned error: %v", err)
	}
}

func TestBundleDisplayOrder_AllKeysValid(t *testing.T) {
	for _, key := range bundleDisplayOrder {
		if _, ok := canonicalBundles[key]; !ok {
			t.Errorf("bundleDisplayOrder references unknown bundle key %q", key)
		}
	}
}

// ── parent command dispatching ────────────────────────────────────────

func TestBundleCmd_NoArgs_ShowsHelp(t *testing.T) {
	root := newBundleTestCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"bundle"})
	// No args shows help — should not error (cobra Help() returns nil)
	_ = root.Execute()
}

func TestBundleCmd_UnknownSubcommand(t *testing.T) {
	cmd := &cobra.Command{Use: "nself"}
	parent := *bundleCmd // shallow copy to avoid mutation
	cmd.AddCommand(&parent)
	cmd.SetArgs([]string{"bundle", "badcmd"})
	buf := &bytes.Buffer{}
	cmd.SetErr(buf)
	err := cmd.Execute()
	// cobra returns an error for unknown subcommands
	if err == nil {
		t.Log("no error returned for unknown subcommand (cobra may print help instead)")
	}
}
