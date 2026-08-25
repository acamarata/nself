package plugin

// installer_locked.go — the install-lock-held plugin install sequence.
//
// Purpose: run the full plugin install sequence (license check, checksum verify, systemDependencies, schema, dep resolution, runtime env) while the install lock from installer.go is held, split out for file size.
// Inputs: the plugin manifest, target config and pluginDir, called by Install in installer.go.
// Outputs: an installed plugin on disk, or an error with any partial install rolled back.
// Constraints: pure move from installer.go (CLI-R12 Batch F); no behaviour change. This is the frozen manager.go load order (registry fetch -> license check -> checksum verify -> systemDependencies -> schema -> dep resolution -> runtime env) -- the sequence inside installLocked is untouched, only its file location moved.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/audit"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/plugin/verify"
	"github.com/nself-org/cli/internal/version"
)

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
	// A plugin that provides a command needs a package built for this platform;
	// one that does not is source and works anywhere. cliBinaryName returns ""
	// for the latter, which is most plugins.
	archivePath, err := downloadPluginPackage(ctx, name, manifest.Version, manifest.Repository,
		cliBinaryName(name, manifest))
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
	//
	// Skipped for a plugin that owns no tables. Every CLI-R11 extraction
	// produces exactly that: a plugin whose whole job is to add a command, with
	// "tables": [] in its manifest. Creating a schema for it meant reaching for
	// Postgres through Docker, so `nself install infra` failed outright on a
	// machine with no stack running:
	//
	//	error installing "infra": creating schema for plugin "infra":
	//	creating np_common.schema_versions: ... dial unix /var/run/docker.sock
	//
	// A command-line tool must be installable without a database. This is not a
	// reordering of the frozen load sequence — the step stays exactly where it
	// is and still runs for every plugin that has tables; it just does nothing
	// when there is nothing to create.
	if pluginOwnsTables(manifest) {
		if err := createPluginSchema(ctx, cfg, name); err != nil {
			rollbackInstall(ctx, cfg, name, destDir)
			return fmt.Errorf("creating schema for plugin %q: %w", name, err)
		}
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
		"permissions", manifest.Permissions.Strings(),
	)

	// S71-T02: Warn via doctor when dangerous permissions are present.
	logDangerousPermissions(name, manifest.Permissions.Strings())

	fmt.Fprintf(os.Stderr, "\nℹ Run 'nself build' to include %s in your stack.\n", name)

	// S68-T02: Fire-and-forget install-event to plugins.nself.org registry.
	// Silent, 1s timeout, never blocks the install. Sends only an opaque
	// SHA-256 hash of the machine fingerprint — no PII in the payload.
	go postInstallEvent(name)

	return nil
}
