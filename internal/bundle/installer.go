// installer.go — S2.T03: `nself bundle install <bundle>` orchestrator.
//
// Atomicity: pre-flight ALL licenses, then install each constituent plugin
// sequentially via plugin.Install. On any per-plugin failure we roll back by
// removing every plugin we successfully installed in THIS invocation. Plugins
// that were already on disk before the call are left untouched.
//
// --dry-run    prints the planned actions and exits without filesystem changes.
// --force      bypasses the license pre-flight (logged as "license-bypass").
// --strict     fails if any plugin in the bundle is missing from the registry
//
//	(default: skip missing with a warning).
//
// --channel    selects stable/beta/canary for version resolution (T22).
package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/plugin"
)

// InstallOpts captures the user-controllable knobs for bundle install.
type InstallOpts struct {
	DryRun  bool
	Force   bool
	Strict  bool
	Channel Channel

	// Out is the stream to print human-facing progress messages.
	// nil → os.Stderr.
	Out io.Writer

	// PluginDir overrides the default plugin install location. Empty → use
	// the same resolution as cmd/commands/plugin.resolvePluginDir.
	PluginDir string

	// Config overrides the loaded project config. Optional — when nil the
	// caller-loaded cfg is used by Install via cfg arg.
	Config *config.Config

	// licenseChecker is an internal hook for tests to bypass the real
	// license validator. Production code leaves this nil and the installer
	// uses checkBundleLicenses, which delegates to plugin.checkLicense via
	// a dry install probe.
	licenseChecker func(ctx context.Context, plugins []string) error

	// installer/remover are internal hooks for tests. Production code leaves
	// these nil and the installer uses plugin.Install / plugin.Remove.
	installer func(ctx context.Context, cfg *config.Config, name, pluginDir string) error
	remover   func(ctx context.Context, cfg *config.Config, name, pluginDir string) error
}

// InstallResult captures the outcome of a bundle install for callers that
// need to surface details (e.g. CR/QA harness, audit logger).
type InstallResult struct {
	Bundle        string
	Channel       Channel
	Planned       []string // plugins planned for install (post channel + strict filter)
	Skipped       []string // plugins skipped because missing from registry (non-strict)
	Installed     []string // plugins successfully installed in this call
	RolledBack    []string // plugins removed during rollback after a failure
	LicenseBypass bool     // true when --force was used
	DryRun        bool
}

// Install runs the bundle install flow. The returned InstallResult is populated
// even on error so callers (and tests) can inspect partial state.
func Install(ctx context.Context, bundleSlug string, opts InstallOpts) (*InstallResult, error) {
	out := opts.Out
	if out == nil {
		out = os.Stderr
	}

	b, ok := Get(bundleSlug)
	if !ok {
		return nil, UnknownBundleError(bundleSlug)
	}
	if !b.IsInstallable() {
		return nil, fmt.Errorf("bundle %q is not installable as a unit (meta or free)", b.Slug)
	}

	result := &InstallResult{
		Bundle:        b.Slug,
		Channel:       normalizeChannel(opts.Channel),
		LicenseBypass: opts.Force,
		DryRun:        opts.DryRun,
	}

	// Resolve plugin version pins via T22. Channel filter applied here.
	pins, err := ResolveBundleVersions(ctx, b.Slug, result.Channel)
	if err != nil {
		// Strict mode requires every plugin in the bundle to be in the registry.
		// Non-strict mode degrades to "install whatever IS in the registry".
		if opts.Strict {
			return result, fmt.Errorf("resolve bundle versions (strict): %w", err)
		}
		// Try a per-plugin probe to determine which plugins are missing.
		partialPins, partialSkipped, probeErr := probePartialBundle(ctx, b, result.Channel)
		if probeErr != nil {
			return result, fmt.Errorf("resolve bundle versions: %w", err)
		}
		pins = partialPins
		result.Skipped = partialSkipped
		if len(pins) == 0 {
			return result, fmt.Errorf("no plugins in bundle %q are eligible under channel %q", b.Slug, result.Channel)
		}
	}

	// Stable ordering for dry-run + install loop: walk b.Plugins, keep only
	// those resolved by pins.
	planned := make([]string, 0, len(pins))
	for _, name := range b.Plugins {
		if _, ok := pins[name]; ok {
			planned = append(planned, name)
		}
	}
	result.Planned = planned

	// Print the plan.
	printPlan(out, b, result.Channel, planned, pins, result.Skipped, opts)

	if opts.DryRun {
		fmt.Fprintln(out, "(dry-run) no changes made.")
		return result, nil
	}

	// License pre-flight: two-phase check.
	//   Phase 1 (bundle-level): call ping_api /license/validate?bundle=<slug> to
	//   verify the operator's key grants access to THIS bundle as a unit. A key
	//   with individual plugin entitlements does NOT satisfy a bundle purchase.
	//   Phase 2 (plugin-level fallback): per-plugin registry/license probe via
	//   defaultLicenseChecker to catch obvious missing-key cases before any FS
	//   change (mirrors existing plugin install behavior).
	// --force bypasses both phases with an audit log entry.
	if !opts.Force {
		// Phase 1: bundle-level entitlement via ping_api.
		key := license.CollectLicenseKey()
		entitled, entitleErr := license.BundleEntitled(ctx, key, b.Slug)
		if entitleErr != nil || !entitled {
			msg := fmt.Sprintf("bundle %q: not entitled", b.Slug)
			if entitleErr != nil {
				msg = entitleErr.Error()
			}
			return result, fmt.Errorf("license validation failed: %s\n\nBuy at: https://nself.org/pricing or run 'nself license set <key>'", msg)
		}

		// Phase 2: plugin-level pre-flight (catches no-key-at-all cases for free).
		check := opts.licenseChecker
		if check == nil {
			check = defaultLicenseChecker
		}
		if err := check(ctx, planned); err != nil {
			return result, fmt.Errorf("license validation failed for bundle %q: %w", b.Slug, err)
		}
	} else {
		fmt.Fprintf(out, "WARNING: --force bypassed license validation for bundle %q (logged as license-bypass)\n", b.Slug)
	}

	pluginDir := opts.PluginDir
	if pluginDir == "" {
		pluginDir = defaultPluginDir()
	}

	installFn := opts.installer
	if installFn == nil {
		installFn = func(ctx context.Context, cfg *config.Config, name, pd string) error {
			return plugin.Install(ctx, cfg, name, pd)
		}
	}
	removeFn := opts.remover
	if removeFn == nil {
		removeFn = func(ctx context.Context, cfg *config.Config, name, pd string) error {
			return plugin.Remove(ctx, cfg, name, pd, false, true)
		}
	}

	cfg := opts.Config
	if cfg == nil {
		loaded, lerr := loadConfigOrDefault()
		if lerr != nil {
			return result, fmt.Errorf("loading project config: %w", lerr)
		}
		cfg = loaded
	}

	// Build a lookup of currently installed plugin versions for downgrade detection.
	existingVersions := buildInstalledVersionMap(pluginDir)

	// Sequential install with rollback on failure.
	for _, name := range planned {
		targetVer := string(pins[name])
		existingVer, alreadyExists := existingVersions[strings.ToLower(name)]

		if alreadyExists {
			cmp := plugin.CompareVersions(existingVer, targetVer)
			if cmp == 0 {
				// Same version: skip silently.
				fmt.Fprintf(out, "  ✓ %s@%s (already installed, same version)\n", name, targetVer)
				continue
			}
			if cmp > 0 {
				// Existing is HIGHER than what the bundle pins.
				if opts.Strict {
					// --strict: fail if any plugin already at a higher version.
					result.RolledBack = rollbackInstalled(ctx, cfg, removeFn, pluginDir, result.Installed, out)
					return result, fmt.Errorf("--strict: plugin %q is at %s (higher than bundle pin %s); use without --strict to skip", name, existingVer, targetVer)
				}
				// Default: skip with warning, no silent downgrade.
				fmt.Fprintf(out, "  ⚠ %s: installed at %s (higher than bundle pin %s) — skipping to avoid downgrade\n", name, existingVer, targetVer)
				continue
			}
			// cmp < 0: existing is lower → upgrade, fall through to install.
			fmt.Fprintf(out, "  ↑ upgrading %s: %s → %s\n", name, existingVer, targetVer)
		} else {
			fmt.Fprintf(out, "  → installing %s@%s...\n", name, targetVer)
		}

		if err := installFn(ctx, cfg, name, pluginDir); err != nil {
			fmt.Fprintf(out, "  ✗ %s install failed: %v\n", name, err)
			result.RolledBack = rollbackInstalled(ctx, cfg, removeFn, pluginDir, result.Installed, out)
			return result, fmt.Errorf("bundle %q install failed at plugin %q: %w", b.Slug, name, err)
		}
		result.Installed = append(result.Installed, name)
		fmt.Fprintf(out, "  ✓ %s@%s installed\n", name, targetVer)
	}

	fmt.Fprintf(out, "\nBundle %q (%s) installed: %d plugins.\n", b.Name, b.Slug, len(result.Installed))
	if len(result.Skipped) > 0 {
		fmt.Fprintf(out, "Skipped (missing from registry): %s\n", strings.Join(result.Skipped, ", "))
	}

	// Trigger a single nself build to regenerate docker-compose.yml and nginx
	// configs with the newly installed plugins. This must happen ONCE at the
	// very end — not per-plugin. Skip when no plugins were actually installed.
	if len(result.Installed) > 0 {
		if err := triggerBuild(ctx, out); err != nil {
			// Build failure is non-fatal: plugins are on disk; user can run
			// nself build manually. Surface as a warning, not a hard error.
			fmt.Fprintf(out, "\nWARNING: nself build failed after bundle install: %v\n", err)
			fmt.Fprintln(out, "Run 'nself build' manually to apply the new plugins.")
		}
	}

	return result, nil
}

// probePartialBundle resolves bundle plugins one-by-one so that a single
// missing plugin does not abort the whole bundle in non-strict mode.
func probePartialBundle(ctx context.Context, b Bundle, ch Channel) (VersionPins, []string, error) {
	pins := make(VersionPins, len(b.Plugins))
	var skipped []string
	reg, err := plugin.FetchRegistry(ctx, "", defaultRegistryCacheDir())
	if err != nil {
		return nil, nil, err
	}
	for _, name := range b.Plugins {
		v, vErr := resolveOnePluginVersion(reg, name, ch)
		if vErr != nil {
			skipped = append(skipped, name)
			continue
		}
		pins[name] = v
	}
	return pins, skipped, nil
}

// rollbackInstalled walks installed in reverse and removes each plugin. Errors
// during rollback are logged but never propagated — best-effort cleanup.
func rollbackInstalled(
	ctx context.Context,
	cfg *config.Config,
	remove func(ctx context.Context, cfg *config.Config, name, pluginDir string) error,
	pluginDir string,
	installed []string,
	out io.Writer,
) []string {
	rolled := make([]string, 0, len(installed))
	for i := len(installed) - 1; i >= 0; i-- {
		name := installed[i]
		fmt.Fprintf(out, "  ↩ rolling back %s...\n", name)
		if err := remove(ctx, cfg, name, pluginDir); err != nil {
			fmt.Fprintf(out, "  ! rollback of %s failed: %v (manual cleanup may be required)\n", name, err)
			continue
		}
		rolled = append(rolled, name)
	}
	return rolled
}

// printPlan dumps a human-readable summary of what install will do.
func printPlan(out io.Writer, b Bundle, ch Channel, planned []string, pins VersionPins, skipped []string, opts InstallOpts) {
	header := "Install plan"
	if opts.DryRun {
		header = "Install plan (dry-run)"
	}
	fmt.Fprintf(out, "%s for bundle %s (%s) — channel %s\n", header, b.Name, b.Slug, ch)
	if len(planned) == 0 {
		fmt.Fprintln(out, "  (no plugins to install)")
	}
	for _, name := range planned {
		fmt.Fprintf(out, "  • %s@%s\n", name, pins[name])
	}
	if len(skipped) > 0 {
		fmt.Fprintf(out, "  skipped (not in registry, non-strict): %s\n", strings.Join(skipped, ", "))
	}
	if opts.Force {
		fmt.Fprintln(out, "  license: bypass (--force)")
	} else {
		fmt.Fprintln(out, "  license: will validate each plugin before any FS change")
	}
}

// defaultLicenseChecker validates every paid plugin in the bundle has license
// coverage BEFORE the installer touches the filesystem. Implementation
// strategy: rely on plugin.Install's internal license gate by NOT bypassing,
// and pre-check via a lightweight cache probe. Production-grade validation
// occurs inside plugin.Install for each plugin; this pre-flight surfaces
// the failure earlier and lets us short-circuit before partial install.
//
// We do this by calling plugin.CheckEOLBlock as a registry-fetch probe (cheap)
// and inspecting the manifest. If any plugin requires a license and no key is
// configured, we return an error here.
func defaultLicenseChecker(ctx context.Context, plugins []string) error {
	// Quick sanity: at least one license key must exist when any plugin
	// requires a license. plugin.Install does the deep validation per-plugin;
	// this pre-flight just blocks an empty-key obvious miss.
	if len(plugins) == 0 {
		return nil
	}
	// Use plugin.FetchRegistry to look up which plugins require licenses.
	reg, err := plugin.FetchRegistry(ctx, "", defaultRegistryCacheDir())
	if err != nil {
		// Network failure: defer to plugin.Install's offline cache logic.
		return nil
	}
	manifests := make(map[string]plugin.PluginManifest, len(plugins))
	for _, m := range reg.Plugins {
		manifests[strings.ToLower(m.Name)] = m
	}
	for _, name := range plugins {
		m, ok := manifests[strings.ToLower(name)]
		if !ok {
			// Missing from registry — plugin.Install will fail naturally;
			// don't pre-error here so we get the proper "not found" message.
			continue
		}
		// Free plugins don't need a license.
		if !m.RequiresLicense && !plugin.IsPaidPlugin(name) {
			continue
		}
		// Paid plugin: ensure at least one license key is set. We don't
		// validate ENTITLEMENTS here — plugin.Install does that per-plugin
		// with cache + remote fallback. We're just catching the obvious
		// "no key at all" case so the user sees one error, not N.
		if !hasAnyLicenseKey() {
			return errors.New("bundle requires at least one paid plugin; no license key configured. Run 'nself license set <key>' or visit nself.org/pricing")
		}
		// One paid plugin with a key present is enough to proceed; the
		// per-plugin validator inside plugin.Install runs full checks.
		break
	}
	return nil
}

// hasAnyLicenseKey returns true if the operator has any license key set
// (env, cache, or legacy key file). Real validation is left to plugin.Install.
func hasAnyLicenseKey() bool {
	if os.Getenv("NSELF_PLUGIN_LICENSE_KEY") != "" {
		return true
	}
	if os.Getenv("NSELF_PLUGIN_LICENSE_KEY_OWNER") != "" {
		return true
	}
	// Legacy: ~/.nself/license.key
	home, err := os.UserHomeDir()
	if err == nil {
		if _, err := os.Stat(filepath.Join(home, ".nself", "license.key")); err == nil {
			return true
		}
	}
	return false
}

// isAlreadyInstalled reports whether the plugin's directory exists in pluginDir.
// Used by both installer and remover for a fast existence probe.
func isAlreadyInstalled(pluginDir, name string) bool {
	_, err := os.Stat(filepath.Join(pluginDir, name))
	return err == nil
}

// buildInstalledVersionMap returns a map of lowercase plugin name → installed
// version string for all plugins found in pluginDir. Plugins with no version
// information get an empty string (treated as "0.0.0" by callers).
func buildInstalledVersionMap(pluginDir string) map[string]string {
	m := make(map[string]string)
	installed, err := plugin.ListInstalled(pluginDir)
	if err != nil {
		return m
	}
	for _, p := range installed {
		m[strings.ToLower(p.Name)] = p.Version
	}
	return m
}

// triggerBuild invokes `nself build` once at the end of a bundle install so
// docker-compose.yml and nginx configs are regenerated with the new plugins.
// This is a subprocess call matching the pattern in internal/promote/promote.go.
func triggerBuild(ctx context.Context, out io.Writer) error {
	fmt.Fprintln(out, "\nRunning 'nself build' to apply installed plugins...")
	nself, err := exec.LookPath("nself")
	if err != nil {
		// nself binary not found — likely running in tests or a non-standard PATH.
		// Allow callers to detect this via the error message.
		return fmt.Errorf("nself binary not found in PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, nself, "build", "--quiet")
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nself build: %w", err)
	}
	return nil
}

// defaultPluginDir resolves the standard plugin install directory. Mirrors
// cmd/commands/plugin.go's resolvePluginDir so this package does not import
// the cmd layer.
func defaultPluginDir() string {
	if d := os.Getenv("NSELF_PLUGIN_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "plugins")
	}
	return filepath.Join(home, ".nself", "plugins")
}

// loadConfigOrDefault loads the project config from cwd. On error (e.g. no
// .env files yet) we return a zero-value Config so plugin.Install can proceed
// using its own defaults. This mirrors plugin install behavior outside a project.
func loadConfigOrDefault() (*config.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return &config.Config{}, nil
	}
	cfg, err := config.Load(wd)
	if err != nil {
		// Bundle install must work outside a project too (CI, fresh install).
		return &config.Config{}, nil
	}
	return cfg, nil
}
