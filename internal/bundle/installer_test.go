package bundle

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// stubPlanReturn forces ResolveBundleVersions failure to a stable path by
// using a tiny bundle slug and overriding internal hooks. We need to bypass
// ResolveBundleVersions which hits the real network/cache; the test strategy
// is to set NSELF_PLUGIN_REGISTRY to a non-existent file and verify dry-run
// short-circuits before the install loop runs.

func TestInstall_UnknownBundle(t *testing.T) {
	ctx := context.Background()
	_, err := Install(ctx, "bogus", InstallOpts{DryRun: true, Out: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected error for unknown bundle")
	}
	if !strings.Contains(err.Error(), "unknown bundle") {
		t.Errorf("error %v does not mention 'unknown bundle'", err)
	}
}

func TestInstall_MetaBundleNotInstallable(t *testing.T) {
	ctx := context.Background()
	_, err := Install(ctx, "nself-plus", InstallOpts{DryRun: true, Out: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected error: nself-plus is not installable")
	}
	if !strings.Contains(err.Error(), "not installable") {
		t.Errorf("error %v missing 'not installable'", err)
	}
}

func TestInstall_DryRunWithMockedRegistry(t *testing.T) {
	dir := setupOfflineRegistry(t, mockNsentryRegistry())
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	res, err := Install(context.Background(), "nsentry", InstallOpts{
		DryRun:  true,
		Out:     &buf,
		Channel: ChannelStable,
	})
	if err != nil {
		t.Fatalf("dry-run Install failed: %v", err)
	}
	if !res.DryRun {
		t.Error("result.DryRun = false; want true")
	}
	if len(res.Installed) != 0 {
		t.Errorf("dry-run installed %d plugins; want 0", len(res.Installed))
	}
	if !strings.Contains(buf.String(), "Install plan (dry-run)") {
		t.Errorf("dry-run output missing header: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "nself-uptime-monitor@") {
		t.Errorf("dry-run output missing version pin: %s", buf.String())
	}
}

func TestInstall_AtomicRollbackOnFailure(t *testing.T) {
	dir := setupOfflineRegistry(t, mockNsentryRegistry())
	defer os.RemoveAll(dir)

	pluginDir := t.TempDir()
	var buf bytes.Buffer

	// Install hook: succeed for the first two plugins, fail on the third.
	failedAt := "nself-incident-mgmt"
	var installed []string
	installFn := func(ctx context.Context, cfg *config.Config, name, pd string) error {
		if name == failedAt {
			return errors.New("synthetic failure")
		}
		// Create the plugin dir to satisfy isAlreadyInstalled in rollback.
		if err := os.MkdirAll(filepath.Join(pd, name), 0700); err != nil {
			return err
		}
		installed = append(installed, name)
		return nil
	}
	var removed []string
	removeFn := func(ctx context.Context, cfg *config.Config, name, pd string) error {
		removed = append(removed, name)
		return os.RemoveAll(filepath.Join(pd, name))
	}

	res, err := Install(context.Background(), "nsentry", InstallOpts{
		Out:       &buf,
		Channel:   ChannelStable,
		Force:     true, // bypass license pre-flight (no real key in test env)
		PluginDir: pluginDir,
		installer: installFn,
		remover:   removeFn,
	})
	if err == nil {
		t.Fatalf("expected install failure; got nil err. output:\n%s", buf.String())
	}
	if len(res.Installed) == 0 {
		t.Error("expected some plugins installed before failure")
	}
	if len(res.RolledBack) != len(res.Installed) {
		t.Errorf("rolled back %d but %d were installed: %v vs %v", len(res.RolledBack), len(res.Installed), res.RolledBack, res.Installed)
	}
	// Rollback order should be reverse of install order.
	for i := range res.RolledBack {
		want := res.Installed[len(res.Installed)-1-i]
		if res.RolledBack[i] != want {
			t.Errorf("rollback order: got %q at index %d, want %q", res.RolledBack[i], i, want)
		}
	}
}

func TestInstall_LicenseBypassWithForce(t *testing.T) {
	dir := setupOfflineRegistry(t, mockNsentryRegistry())
	defer os.RemoveAll(dir)

	pluginDir := t.TempDir()
	var buf bytes.Buffer

	// Ensure license check is NOT called when --force is set.
	licenseCalled := false
	licenseFn := func(ctx context.Context, plugins []string) error {
		licenseCalled = true
		return errors.New("would have failed")
	}
	installFn := func(ctx context.Context, cfg *config.Config, name, pd string) error {
		return os.MkdirAll(filepath.Join(pd, name), 0700)
	}
	removeFn := func(ctx context.Context, cfg *config.Config, name, pd string) error {
		return nil
	}

	res, err := Install(context.Background(), "nsentry", InstallOpts{
		Force:          true,
		Out:            &buf,
		Channel:        ChannelStable,
		PluginDir:      pluginDir,
		licenseChecker: licenseFn,
		installer:      installFn,
		remover:        removeFn,
	})
	if err != nil {
		t.Fatalf("install with --force failed: %v\noutput:\n%s", err, buf.String())
	}
	if licenseCalled {
		t.Error("license checker was called despite --force")
	}
	if !res.LicenseBypass {
		t.Error("result.LicenseBypass = false; want true")
	}
	if !strings.Contains(buf.String(), "bypassed license validation") {
		t.Errorf("output missing license-bypass warning: %s", buf.String())
	}
}

func TestInstall_LicenseFailWithoutForce(t *testing.T) {
	dir := setupOfflineRegistry(t, mockNsentryRegistry())
	defer os.RemoveAll(dir)

	pluginDir := t.TempDir()
	var buf bytes.Buffer

	licenseFn := func(ctx context.Context, plugins []string) error {
		return errors.New("no license key configured")
	}

	res, err := Install(context.Background(), "nsentry", InstallOpts{
		Out:            &buf,
		Channel:        ChannelStable,
		PluginDir:      pluginDir,
		licenseChecker: licenseFn,
	})
	if err == nil {
		t.Fatal("expected license validation failure")
	}
	if len(res.Installed) != 0 {
		t.Errorf("license pre-flight should block install; got %d installed", len(res.Installed))
	}
}

func TestInstall_StrictModeMissingPlugin(t *testing.T) {
	// Mock registry has only one plugin; bundle expects 13. Strict mode → error.
	dir := setupOfflineRegistry(t, mockPartialNsentryRegistry())
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	_, err := Install(context.Background(), "nsentry", InstallOpts{
		DryRun:  true,
		Strict:  true,
		Out:     &buf,
		Channel: ChannelStable,
	})
	if err == nil {
		t.Fatal("expected strict-mode failure when plugins are missing")
	}
}

// --- helpers ---

// setupOfflineRegistry writes the given registry JSON to a temp file and
// points NSELF_PLUGIN_REGISTRY at the file:// URL so the registry fetcher
// resolves entirely offline.
//
// NOTE: file:// is not supported by the HTTPS-enforcing fetcher. Instead we
// pre-seed the cache directory with the registry JSON and let the fetcher
// pick it up via its stale-cache path. We set NSELF_PLUGIN_CACHE to a temp
// dir so the cache is isolated per-test.
func setupOfflineRegistry(t *testing.T, body []byte) string {
	t.Helper()
	cacheDir := t.TempDir()
	cachePath := filepath.Join(cacheDir, "registry.json")
	if err := os.WriteFile(cachePath, body, 0600); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	// Set cache TTL via env to make the seeded cache appear fresh.
	t.Setenv("NSELF_PLUGIN_CACHE", cacheDir)
	// Point registry at localhost (which will fail) so we fall through to cache.
	// The cache path is checked first by Fetch().
	t.Setenv("NSELF_PLUGIN_REGISTRY", "http://127.0.0.1:1") // non-routable, forces fallback
	return cacheDir
}

// mockNsentryRegistry returns a minimal registry JSON containing every plugin
// in the nsentry bundle, all at version 1.0.0 and status "stable".
func mockNsentryRegistry() []byte {
	return []byte(`{
  "version": "test",
  "tier": "pro",
  "plugins": {
    "nself-uptime-monitor": {"name":"nself-uptime-monitor","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-status-page":   {"name":"nself-status-page","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-incident-mgmt": {"name":"nself-incident-mgmt","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-alert-router":  {"name":"nself-alert-router","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-slo-tracker":   {"name":"nself-slo-tracker","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-synthetic-monitor":{"name":"nself-synthetic-monitor","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-rum":           {"name":"nself-rum","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-errors":        {"name":"nself-errors","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-cron-monitor":  {"name":"nself-cron-monitor","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-oncall":        {"name":"nself-oncall","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-crash":         {"name":"nself-crash","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-anomaly":       {"name":"nself-anomaly","version":"1.0.0","status":"stable","tier":"pro","requires_license":true},
    "nself-audit":         {"name":"nself-audit","version":"1.0.0","status":"stable","tier":"pro","requires_license":true}
  }
}`)
}

// mockPartialNsentryRegistry returns a registry with only one nsentry plugin
// to exercise strict-mode failure.
func mockPartialNsentryRegistry() []byte {
	return []byte(`{
  "version": "test",
  "tier": "pro",
  "plugins": {
    "nself-uptime-monitor": {"name":"nself-uptime-monitor","version":"1.0.0","status":"stable","tier":"pro","requires_license":true}
  }
}`)
}
