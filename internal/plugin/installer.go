package plugin

// Purpose: Plugin install, remove, and update operations with file-lock serialization,
//          dependency resolution, rollback on failure, and license/security pre-checks.
// Inputs:  context.Context, *config.Config, plugin name string, pluginDir path string.
// Outputs: error on failure; side effects are plugin directory and database schema changes.
// Constraints: Acquires {pluginDir}/.install.lock (O_CREATE|O_EXCL) to prevent concurrent
//              installs. Calls installLocked for dependency recursion to avoid deadlock.
// SPORT: install/remove/update operations; callers: cmd/plugin/install.go, cmd/plugin/remove.go

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/audit"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/plugin/verify"
	"github.com/nself-org/cli/internal/version"
)

// dangerousPermissions lists permission strings that warrant a visible warning
// on install. system:exec and network:internet are the two highest-risk grants.
// S71-T02.
var dangerousPermissions = map[string]bool{
	"system:exec":      true,
	"network:internet": true,
}

// installLockPath returns the path to the install lock file for pluginDir.
func installLockPath(pluginDir string) string {
	return filepath.Join(pluginDir, ".install.lock")
}

// acquireInstallLock attempts to create an exclusive lock file at
// {pluginDir}/.install.lock using O_CREATE|O_EXCL (atomic on POSIX). It polls
// every 250 ms until the lock is acquired or the 5-second timeout elapses.
// Returns the open lock file on success so the caller can defer its cleanup.
func acquireInstallLock(pluginDir string) (*os.File, error) {
	// Ensure the plugin directory exists so the lock file can be created.
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating plugin directory: %w", err)
	}

	lockPath := installLockPath(pluginDir)
	deadline := time.Now().Add(5 * time.Second)

	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			// Lock acquired — write our PID for diagnostic purposes.
			fmt.Fprintf(f, "%d\n", os.Getpid())
			return f, nil
		}
		if !os.IsExist(err) {
			// Unexpected error (permissions, etc.).
			return nil, fmt.Errorf("acquiring install lock: %w", err)
		}
		// Lock file already exists — check timeout before polling again.
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("another plugin install is already running")
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// releaseInstallLock closes and removes the lock file returned by
// acquireInstallLock. Errors are silently ignored; a stale lock file will be
// cleaned up on the next acquire attempt.
func releaseInstallLock(f *os.File, pluginDir string) {
	f.Close()
	os.Remove(installLockPath(pluginDir))
}

// Install downloads, extracts, and configures a plugin. For paid plugins it
// checks the license first. Dependencies are resolved and recursively
// installed before the target plugin. If any step after extraction fails, the
// extracted directory and database schema are rolled back.
//
// A file lock on {pluginDir}/.install.lock is held for the duration of the
// operation to prevent concurrent installs from corrupting plugin state.
func Install(ctx context.Context, cfg *config.Config, name string, pluginDir string) error {
	lock, err := acquireInstallLock(pluginDir)
	if err != nil {
		return err
	}
	defer releaseInstallLock(lock, pluginDir)

	return installLocked(ctx, cfg, name, pluginDir)
}

// installLocked performs the actual install work after the caller has already
// acquired the install lock. Dependency installs call this directly to avoid
// attempting to re-acquire the lock (which would deadlock).
func installLocked(ctx context.Context, cfg *config.Config, name string, pluginDir string) error {
	// Step 1: License check for paid plugins.
	if isPaidPlugin(name) {
		if err := checkLicense(ctx, name); err != nil {
			return err
		}
	}

	// Step 2: Fetch registry and locate the plugin.
	cacheDir := defaultCacheDir()
	reg, err := FetchRegistry(ctx, "", cacheDir)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	manifest, found := findPlugin(reg, name)
	if !found {
		return errs.ErrPluginNotFound
	}

	// Status check: lifecycle policy enforcement (S58-T01, S58-T02, S58-T03).
	// "stable" and "" (legacy, no status field) proceed silently.
	switch manifest.PublishStatus {
	case "planned":
		return fmt.Errorf(
			"plugin %q is not yet available — coming soon.\nSee https://nself.org/plugins/%s for the release timeline.\nRun 'nself plugin list' to see available plugins.",
			name, name,
		)
	case "experimental":
		fmt.Fprintf(os.Stderr, "warning: %s is experimental — API and behavior may change without notice\n", name)
	case "beta":
		fmt.Fprintf(os.Stderr, "warning: %s is in beta — use in production at your own risk\n", name)
	case "deprecated":
		d := manifest.Deprecation
		if d != nil {
			fmt.Fprintf(os.Stderr, "warning: %s is deprecated (EOL: %s). %s\n", name, d.EOLDate, d.MigrationGuide)
			if d.ReplacedBy != "" {
				fmt.Fprintf(os.Stderr, "  A replacement is available: %s\n", d.ReplacedBy)
				fmt.Fprintf(os.Stderr, "  Install the replacement instead? Run: nself plugin install %s\n", d.ReplacedBy)
			}
		} else {
			fmt.Fprintf(os.Stderr, "warning: %s is deprecated\n", name)
		}
	case "eol":
		// EOL blocking is enforced at the command layer via --allow-eol.
		// By the time we reach installLocked the caller has already verified
		// the flag; emit a prominent warning so the risk acknowledgment is
		// logged even in non-interactive contexts.
		fmt.Fprintf(os.Stderr, "warning: %s has reached end-of-life and is no longer maintained\n", name)
	}

	// S58-T09: Author CRL check. Blocks install if the plugin's declared author
	// key appears in the revocation list at plugins.nself.org. Network errors
	// are non-fatal (logged to stderr) so installs are not bricked by a
	// transient CRL outage.
	if err := checkAuthorRevocation(ctx, manifest.Author); err != nil {
		return err
	}

	// Compat check: verify CLI version satisfies the plugin's declared range.
	if err := CheckCLICompat(manifest.Compat, version.GetVersion()); err != nil {
		return fmt.Errorf("compatibility check failed for %q: %w", name, err)
	}

	// T21: Check for table prefix conflicts with already-installed plugins.
	if err := checkTablePrefixConflict(pluginDir, name, manifest.Tables); err != nil {
		return err
	}

	// Check for exact table name conflicts with already-installed plugins.
	if err := checkTableConflicts(pluginDir, manifest); err != nil {
		return err
	}

	// T22: Check for nginx route conflicts with already-installed plugins.
	existingRoutes := collectInstalledPluginRoutes(pluginDir, name)
	if err := checkPluginRouteConflict(manifest, existingRoutes); err != nil {
		return err
	}

	// T23: Warn if this install would downgrade an already-installed plugin.
	existingManifest, err := parseManifest(filepath.Join(pluginDir, name, "plugin.json"))
	if err == nil {
		// Plugin exists — check for downgrade
		if compareSemver(existingManifest.Version, manifest.Version) > 0 {
			fmt.Fprintf(os.Stderr, "⚠ Downgrading %s from %s to %s — this may cause data loss\n",
				name, existingManifest.Version, manifest.Version)
		}
	}

	// S43-T17: Validate inter-plugin communication contract (Consumes).
	// Every plugin listed in manifest.Consumes must be installed (or will be
	// installed as a dependency below). A plugin declaring X in Consumes will
	// send X-Source-Plugin: <name> requests to X's HTTP API; if X is not
	// installed there is no service to receive those calls.
	if err := validateConsumes(pluginDir, name, manifest.Consumes); err != nil {
		return err
	}

	// Step 3: Resolve dependencies. The resolver reads manifests from
	// pluginDir, so already-installed plugins are picked up automatically.
	// For plugins not yet installed we install them first, which places
	// their manifest on disk for the resolver. We call installLocked here
	// (not Install) because the lock is already held by this goroutine.
	deps := manifest.Dependencies
	if len(deps) > 0 {
		fmt.Fprintf(os.Stderr, "Installing %s (requires: %s)\n", name, strings.Join(deps, ", "))
	}
	for _, dep := range deps {
		depDir := filepath.Join(pluginDir, dep)
		if _, err := os.Stat(depDir); err == nil {
			fmt.Fprintf(os.Stderr, "  ✓ %s (already installed)\n", dep)
			continue // dependency already installed
		}
		fmt.Fprintf(os.Stderr, "  → Installing dependency %s...\n", dep)
		if err := installLocked(ctx, cfg, dep, pluginDir); err != nil {
			return fmt.Errorf("installing dependency %q: %w", dep, err)
		}
		fmt.Fprintf(os.Stderr, "  ✓ %s installed\n", dep)
	}

	// Step 4: Download the plugin archive.
	archivePath, err := downloadPlugin(ctx, name, manifest.Version, manifest.Repository)
	if err != nil {
		return fmt.Errorf("downloading plugin %q: %w", name, err)
	}
	defer os.Remove(archivePath)

	// Step 5: Verify checksum before extraction.
	// For stable plugins a missing checksum is a hard error (V06-F2).
	// For non-stable plugins an absent checksum emits a warning and continues.
	if manifest.Checksum != "" {
		if err := verifyChecksum(archivePath, manifest.Checksum, manifest.PublishStatus); err != nil {
			os.Remove(archivePath)
			return fmt.Errorf("checksum verification for plugin %q: %w", name, err)
		}
	} else {
		if err := verifyChecksum(archivePath, "", manifest.PublishStatus); err != nil {
			// stable plugin — hard fail returned by verifyChecksum
			os.Remove(archivePath)
			return fmt.Errorf("checksum verification for plugin %q: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "warning: no checksum in registry for plugin %q, skipping verification\n", name)
	}

	// Step 5b: Verify Ed25519 signature (T09 — Security-Always-Free).
	// The signature is computed over the raw SHA-256 digest of the tarball.
	// Public key is pinned in the registry; never fetched at verify time (TOCTOU).
	// For stable plugins a missing signature is a hard error (V06-F2).
	// Skip requires BOTH NSELF_LICENSE_SKIP_VERIFY=1 AND NSELF_LICENSE_SKIP_VERIFY_FORCE=1.
	// Either var alone is insufficient — standalone skip is not permitted (matches license/validate.go).
	// In prod/staging these bypass vars are fatal — dev-only escapes must never reach production.
	bypassed, bypassErr := checkSigBypassAllowed(cfg.Env, name)
	if bypassErr != nil {
		os.Remove(archivePath)
		return bypassErr
	}
	if bypassed {
		fmt.Fprintf(os.Stderr, "WARNING: plugin signature verification skipped (NSELF_LICENSE_SKIP_VERIFY=1 + FORCE)\n")
		if werr := audit.Write("plugin-install-bypass", map[string]string{
			"plugin": name,
			"reason": "NSELF_LICENSE_SKIP_VERIFY=1 + NSELF_LICENSE_SKIP_VERIFY_FORCE=1",
			"uid":    os.Getenv("USER"),
			"env":    cfg.Env,
		}); werr != nil {
			slog.Warn("audit log write failed — bypass event not recorded", "error", werr)
		}
	} else {
		if err := verifyPluginSignature(archivePath, manifest.AuthorPublicKey, manifest.Signature, manifest.PublishStatus); err != nil {
			os.Remove(archivePath)
			return fmt.Errorf("signature verification for plugin %q: %w", name, err)
		}
	}

	// Step 5c: Verify SBOM (S2.T12). Downloads sbom-{version}.cdx.json from the
	// GitHub Release and validates CycloneDX schema. 404 = pre-SBOM release (warn,
	// don't fail). Skip via --skip-sbom-check for air-gapped installs only.
	sbomSkip := os.Getenv("NSELF_SKIP_SBOM_CHECK") == "1"
	if err := verify.VerifySBOM(ctx, name, manifest.Version, verify.SBOMCheckOptions{
		SkipCheck: sbomSkip,
		Version:   manifest.Version,
	}); err != nil {
		os.Remove(archivePath)
		return fmt.Errorf("sbom verification for plugin %q: %w", name, err)
	}

	// Step 6: Extract to pluginDir/{name}/.
	destDir := filepath.Join(pluginDir, name)
	if err := extractTarGz(archivePath, destDir); err != nil {
		return fmt.Errorf("extracting plugin %q: %w", name, err)
	}

	// Step 6b: Publish a CLI plugin's binary where ProxyCommand will find it.
	// Without this a command-providing plugin installs cleanly and then cannot
	// be run at all — the proxy only reads ~/.nself/plugins/bin/.
	if err := linkCLIBinary(destDir, name, manifest); err != nil {
		rollbackInstall(ctx, cfg, name, destDir)
		return fmt.Errorf("publishing plugin %q command: %w", name, err)
	}

	// Step 7: Create database schema. On failure, rollback extraction.
	if err := createPluginSchema(ctx, cfg, name); err != nil {
		rollbackInstall(ctx, cfg, name, destDir)
		return fmt.Errorf("creating schema for plugin %q: %w", name, err)
	}

	// Step 7b (Q01): Generate per-plugin Ed25519 identity keypair and register
	// the public key with ping_api. This is a best-effort step — a failure is
	// logged as a warning but does not roll back the install, because the plugin
	// will still function without JWT auth until Phase B-3 strict mode.
	// The key is only generated when PLUGIN_INTERNAL_SECRET is set (i.e., the
	// operator has opted into the inter-plugin JWT system).
	if os.Getenv("PLUGIN_INTERNAL_SECRET") != "" {
		// pluginDir doubles as the identity data root — each plugin's keypair
		// is stored at pluginDir/<name>/identity.key alongside its manifest.
		if !IdentityKeyExists(pluginDir, name) {
			pubKey, idErr := GenerateEd25519Keypair(pluginDir, name)
			if idErr != nil {
				slog.Warn("plugin identity key generation failed — inter-plugin JWT auth unavailable until resolved",
					"plugin", name, "error", idErr)
			} else {
				if regErr := RegisterIdentity(ctx, name, pubKey); regErr != nil {
					slog.Warn("plugin identity registration with ping_api failed — JWT auth unavailable until resolved",
						"plugin", name, "error", regErr)
				} else {
					slog.Info("plugin.identity.registered", "plugin", name)
				}
			}
		}
	}

	// S71-T02: Emit structured audit log for the granted permission set.
	// One line per install, consumable by Loki. Never logs secret values —
	// only the permission strings declared in the manifest.
	slog.Info("plugin.install.permissions",
		"plugin", name,
		"version", manifest.Version,
		"permissions", manifest.Permissions,
	)

	// S71-T02: Warn via doctor when dangerous permissions are present.
	logDangerousPermissions(name, manifest.Permissions)

	fmt.Fprintf(os.Stderr, "\nℹ Run 'nself build' to include %s in your stack.\n", name)

	// S68-T02: Fire-and-forget install-event to plugins.nself.org registry.
	// Silent, 1s timeout, never blocks the install. Sends only an opaque
	// SHA-256 hash of the machine fingerprint — no PII in the payload.
	go postInstallEvent(name)

	return nil
}

// logDangerousPermissions emits a stderr warning for any dangerous permissions
// held by the named plugin. Called immediately after schema creation so the
// warning appears before the "Run nself build" footer line. S71-T02.
func logDangerousPermissions(pluginName string, permissions []string) {
	for _, perm := range permissions {
		if dangerousPermissions[perm] {
			fmt.Fprintf(os.Stderr,
				"warning: plugin %q holds elevated permission %q — review with 'nself plugin info %s'\n",
				pluginName, perm, pluginName,
			)
		}
	}
}

// checkReverseDependencies scans all installed plugins in pluginDir and returns
// the names of any that declare name as a dependency. This prevents silently
// breaking dependent plugins during uninstall.
func checkReverseDependencies(pluginDir, name string) ([]string, error) {
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugin directory: %w", err)
	}

	var dependents []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == name {
			continue
		}
		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.json")
		m, err := parseManifest(manifestPath)
		if err != nil {
			continue // skip directories without valid manifests
		}
		for _, dep := range m.Dependencies {
			if strings.EqualFold(dep, name) {
				dependents = append(dependents, m.Name)
				break
			}
		}
	}
	return dependents, nil
}

// Remove stops a plugin (if running), optionally drops its database schema,
// and removes its directory from disk. When force is false and other installed
// plugins depend on the target, Remove returns an error listing them.
//
// A file lock on {pluginDir}/.install.lock is held for the duration of the
// operation so that concurrent install and remove calls serialize correctly.
func Remove(ctx context.Context, cfg *config.Config, name string, pluginDir string, keepData bool, force bool) error {
	lock, err := acquireInstallLock(pluginDir)
	if err != nil {
		return err
	}
	defer releaseInstallLock(lock, pluginDir)

	destDir := filepath.Join(pluginDir, name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	// Read the manifest before the directory goes away, so the published
	// binary can be un-published by the same name it was published under.
	// Best effort: an unreadable manifest falls back to the nself-<name>
	// convention rather than leaving a stale binary the proxy would still run.
	removedManifest := readPluginManifest(destDir)

	// Check for reverse dependencies before doing anything destructive.
	if !force {
		dependents, err := checkReverseDependencies(pluginDir, name)
		if err != nil {
			return fmt.Errorf("checking reverse dependencies: %w", err)
		}
		if len(dependents) > 0 {
			return fmt.Errorf("plugin %s depends on %q. Use --force to remove anyway",
				strings.Join(dependents, ", "), name)
		}
	}

	// Stop if running.
	st, err := Status(name)
	if err == nil && st.State == "running" {
		if stopErr := Stop(ctx, name); stopErr != nil {
			return fmt.Errorf("stopping plugin %q: %w", name, stopErr)
		}
	}

	// Drop schema unless the caller wants to keep data.
	if !keepData {
		if err := dropPluginSchema(ctx, cfg, name); err != nil {
			return fmt.Errorf("dropping schema for plugin %q: %w", name, err)
		}
	}

	// Un-publish the command before the directory goes away, so the proxy does
	// not keep resolving a plugin the user just removed.
	if err := unlinkCLIBinary(name, removedManifest); err != nil {
		return fmt.Errorf("removing plugin %q command: %w", name, err)
	}

	// Remove plugin directory.
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("removing plugin directory %q: %w", destDir, err)
	}

	fmt.Fprintf(os.Stderr, "\nℹ Run 'nself build' to update your stack.\n")
	return nil
}

// Update backs up the current installation, then reinstalls from the registry.
// If the new install fails, the previous version is restored automatically.
func Update(ctx context.Context, cfg *config.Config, name string, pluginDir string) error {
	currentDir := filepath.Join(pluginDir, name)
	backupDir := filepath.Join(pluginDir, name+".prev")

	// Rename current install to .prev so we can restore on failure.
	if _, err := os.Stat(currentDir); err == nil {
		// Remove any stale backup from a prior interrupted update.
		_ = os.RemoveAll(backupDir)
		if err := os.Rename(currentDir, backupDir); err != nil {
			return fmt.Errorf("backing up plugin %q for update: %w", name, err)
		}
	}

	// Install the new version.
	if err := Install(ctx, cfg, name, pluginDir); err != nil {
		// Restore the previous version from backup.
		if _, statErr := os.Stat(backupDir); statErr == nil {
			_ = os.RemoveAll(currentDir) // clean partial install if any
			if renameErr := os.Rename(backupDir, currentDir); renameErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to restore previous version of %q: %v\n", name, renameErr)
			}
		}
		return fmt.Errorf("installing updated plugin %q: %w", name, err)
	}

	// Success: remove the backup.
	_ = os.RemoveAll(backupDir)
	return nil
}

// checkSigBypassAllowed evaluates whether the NSELF_LICENSE_SKIP_VERIFY /
// NSELF_LICENSE_SKIP_VERIFY_FORCE bypass is allowed for the given env and
// plugin name.
//
// Returns (true, nil) when both vars are set and env is not prod/staging.
// Returns (false, err) with a SECURITY prefix when env is prod or staging.
// Returns (false, err) when only one var is set (standalone bypass rejected).
// Returns (false, nil) when neither var is set (normal verification path).
//
// Purpose: Extracted so unit tests can exercise the bypass-gate logic without
//
//	spinning up a full HTTP registry server.
//
// SPORT: security-audit; ticket P2-E2-W2-S3-T10
func checkSigBypassAllowed(env, pluginName string) (bypassed bool, err error) {
	skipVerify := os.Getenv("NSELF_LICENSE_SKIP_VERIFY") == "1"
	forceSkip := os.Getenv("NSELF_LICENSE_SKIP_VERIFY_FORCE") == "1"

	if !skipVerify && !forceSkip {
		// Normal path — no bypass requested.
		return false, nil
	}

	// Any bypass var set in prod/staging is a fatal security violation.
	if env == "prod" || env == "staging" {
		return false, fmt.Errorf(
			"SECURITY: NSELF_LICENSE_SKIP_VERIFY and NSELF_LICENSE_SKIP_VERIFY_FORCE are not permitted in %s — remove these env vars",
			env,
		)
	}

	// In dev, both vars must be set together.
	if !skipVerify || !forceSkip {
		return false, fmt.Errorf(
			"NSELF_LICENSE_SKIP_VERIFY requires NSELF_LICENSE_SKIP_VERIFY_FORCE=1; standalone skip is not permitted",
		)
	}

	return true, nil
}
