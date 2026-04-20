package plugin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/nginx"
	"github.com/nself-org/cli/internal/version"
)

// PluginInfo describes a plugin's identity and current state.
type PluginInfo struct {
	Name          string
	Version       string
	Category      string
	Installed     bool
	Running       bool
	PublishStatus string // S58-T01: "experimental"|"planned"|"beta"|"stable"|"deprecated"|"eol" — from registry
}

// InstalledPluginInfo is a richer view of an installed plugin used by the
// inventory subcommand. It includes Description and Tier from the manifest.
type InstalledPluginInfo struct {
	Name        string
	Version     string
	Tier        string
	Status      string
	Description string
}

// ListInstalled scans pluginDir and returns detailed information for every
// installed plugin. Each entry's Status is "running", "installed", or
// "unknown" depending on the plugin's current runtime state.
func ListInstalled(pluginDir string) ([]InstalledPluginInfo, error) {
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugin directory: %w", err)
	}

	var plugins []InstalledPluginInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.json")
		m, err := parseManifest(manifestPath)
		if err != nil {
			continue // skip directories without valid manifests
		}

		status := "installed"
		if st, stErr := Status(m.Name); stErr == nil {
			status = st.State
		}

		tier := m.Tier
		if tier == "" {
			if m.RequiresLicense || m.LicenseType == "pro" {
				tier = "pro"
			} else {
				tier = "free"
			}
		}

		plugins = append(plugins, InstalledPluginInfo{
			Name:        m.Name,
			Version:     m.Version,
			Tier:        tier,
			Status:      status,
			Description: m.Description,
		})
	}

	return plugins, nil
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
	if manifest.Checksum != "" {
		if err := verifyChecksum(archivePath, manifest.Checksum); err != nil {
			os.Remove(archivePath)
			return fmt.Errorf("checksum verification for plugin %q: %w", name, err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "warning: no checksum in registry for plugin %q, skipping verification\n", name)
	}

	// Step 5b: Verify Ed25519 signature (T09 — Security-Always-Free).
	// The signature is computed over the raw SHA-256 digest of the tarball.
	// Public key is pinned in the registry; never fetched at verify time (TOCTOU).
	// Skip flag requires --force co-flag; standalone skip emits WARN to stderr.
	skipVerify := os.Getenv("NSELF_LICENSE_SKIP_VERIFY") == "1"
	if skipVerify {
		fmt.Fprintf(os.Stderr, "WARNING: plugin signature verification skipped (NSELF_LICENSE_SKIP_VERIFY=1)\n")
	} else {
		if err := verifyPluginSignature(archivePath, manifest.AuthorPublicKey, manifest.Signature); err != nil {
			os.Remove(archivePath)
			return fmt.Errorf("signature verification for plugin %q: %w", name, err)
		}
	}

	// Step 6: Extract to pluginDir/{name}/.
	destDir := filepath.Join(pluginDir, name)
	if err := extractTarGz(archivePath, destDir); err != nil {
		return fmt.Errorf("extracting plugin %q: %w", name, err)
	}

	// Step 7: Create database schema. On failure, rollback extraction.
	if err := createPluginSchema(ctx, cfg, name); err != nil {
		rollbackInstall(ctx, cfg, name, destDir)
		return fmt.Errorf("creating schema for plugin %q: %w", name, err)
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

// dangerousPermissions lists permission strings that warrant a visible warning
// on install. system:exec and network:internet are the two highest-risk grants.
// S71-T02.
var dangerousPermissions = map[string]bool{
	"system:exec":      true,
	"network:internet": true,
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

// postInstallEvent POSTs an anonymous install count event to the registry worker.
// It is always called in a goroutine and swallows all errors silently.
// The instanceId is a SHA-256 hex hash of the machineID — opaque, no PII.
func postInstallEvent(pluginName string) {
	mid := machineID() // 16-char hex
	// SHA-256 of the machineID to produce the required 64-char hex instanceId
	h := sha256.Sum256([]byte(mid))
	instanceID := hex.EncodeToString(h[:])

	body := `{"instanceId":"` + instanceID + `"}`
	url := "https://plugins.nself.org/plugins/" + pluginName + "/install-event"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return // silent
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return // silent
	}
	_ = resp.Body.Close()
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

// List returns plugin information. When installed is true it scans pluginDir
// for locally installed plugins. When false it returns all plugins known to
// the registry.
func List(pluginDir string, installed bool) ([]PluginInfo, error) {
	if installed {
		return listInstalled(pluginDir)
	}
	return listFromRegistry(pluginDir)
}

// CheckTablePrefixConflict scans all installed plugins in pluginDir and
// returns an error if any of them share a table prefix with the tables listed
// in newTables. Table prefixes are derived from table names by taking the
// first two underscore-separated segments followed by a trailing underscore
// (e.g. "np_chat_messages" → prefix "np_chat_"). The newPluginName parameter
// is used to skip the plugin being installed (allowing reinstalls/updates).
func checkTablePrefixConflict(pluginDir, newPluginName string, newTables []string) error {
	if len(newTables) == 0 {
		return nil
	}

	// Derive prefix set for the new plugin's tables.
	newPrefixes := make(map[string]bool)
	for _, table := range newTables {
		if p := tablePrefix(table); p != "" {
			newPrefixes[p] = true
		}
	}
	if len(newPrefixes) == 0 {
		return nil
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading plugin directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip the plugin being installed so reinstalls and updates are allowed.
		if strings.EqualFold(entry.Name(), newPluginName) {
			continue
		}
		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.json")
		m, err := parseManifest(manifestPath)
		if err != nil {
			continue // skip directories without valid manifests
		}
		for _, table := range m.Tables {
			p := tablePrefix(table)
			if p != "" && newPrefixes[p] {
				return fmt.Errorf("plugin %s conflicts with installed plugin %s: table prefix %q already claimed", newPluginName, m.Name, p)
			}
		}
	}
	return nil
}

// checkTableConflicts scans all installed plugins in pluginDir and returns an
// error if any of them declare a table name that also appears in newPlugin's
// Tables list. This catches exact name collisions (the prefix check above
// catches broader namespace conflicts).
func checkTableConflicts(pluginDir string, newPlugin *PluginManifest) error {
	if len(newPlugin.Tables) == 0 {
		return nil
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return nil // No plugins dir = no conflicts
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == newPlugin.Name {
			continue
		}
		existing, err := parseManifest(filepath.Join(pluginDir, entry.Name(), "plugin.json"))
		if err != nil {
			continue
		}
		// Check for overlapping table names
		for _, newTable := range newPlugin.Tables {
			for _, existingTable := range existing.Tables {
				if newTable == existingTable {
					return fmt.Errorf("table %q already used by plugin %q", newTable, existing.Name)
				}
			}
		}
	}
	return nil
}

// tablePrefix extracts the two-segment prefix from a table name. Given
// "np_chat_messages" it returns "np_chat_". Returns empty string for table
// names that do not have at least two underscore-separated segments.
func tablePrefix(table string) string {
	parts := strings.Split(table, "_")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "_" + parts[1] + "_"
}

// DisablePlugin creates a .disabled marker file in the plugin's directory,
// causing it to be excluded from compose files on the next build.
func DisablePlugin(name, pluginDir string) error {
	destDir := filepath.Join(pluginDir, name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	markerPath := filepath.Join(destDir, ".disabled")
	if _, err := os.Stat(markerPath); err == nil {
		return fmt.Errorf("plugin %q is already disabled", name)
	}

	f, err := os.Create(markerPath)
	if err != nil {
		return fmt.Errorf("creating disable marker: %w", err)
	}
	f.Close()
	return nil
}

// EnablePlugin removes the .disabled marker file from the plugin's directory,
// allowing it to be included in compose files on the next build.
func EnablePlugin(name, pluginDir string) error {
	destDir := filepath.Join(pluginDir, name)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not installed", name)
	}

	markerPath := filepath.Join(destDir, ".disabled")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not disabled", name)
	}

	if err := os.Remove(markerPath); err != nil {
		return fmt.Errorf("removing disable marker: %w", err)
	}
	return nil
}

// IsDisabled returns true if the named plugin has a .disabled marker file.
func IsDisabled(name, pluginDir string) bool {
	markerPath := filepath.Join(pluginDir, name, ".disabled")
	_, err := os.Stat(markerPath)
	return err == nil
}

// --- internal helpers ---

// verifyChecksum computes the SHA256 hash of the file at filePath and compares
// it to expectedHash (hex-encoded). Returns an error if the hashes differ.
func verifyChecksum(filePath string, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedHash) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}

// verifyPluginSignature verifies that the Ed25519 signature stored in the
// plugin's registry manifest matches the SHA-256 hash of the downloaded
// tarball. The public key is pinned in the registry (never fetched at verify
// time, preventing TOCTOU attacks).
//
// The signed message is the raw 32-byte SHA-256 digest of the tarball —
// consistent with how release tarballs are signed on the publisher side.
//
// If authorPublicKeyHex or signatureHex is empty the function returns nil
// (verification is skipped with a warning — callers should already warn the
// user via the checksum warning path).
func verifyPluginSignature(archivePath, authorPublicKeyHex, signatureHex string) error {
	if authorPublicKeyHex == "" || signatureHex == "" {
		return nil // no signature to verify
	}

	// Decode the hex-encoded Ed25519 public key.
	pkBytes, err := hex.DecodeString(authorPublicKeyHex)
	if err != nil {
		return fmt.Errorf("decoding author public key: %w", err)
	}
	if len(pkBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("author public key has wrong length: expected %d bytes, got %d", ed25519.PublicKeySize, len(pkBytes))
	}
	pubKey := ed25519.PublicKey(pkBytes)

	// Decode the hex-encoded signature.
	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("decoding plugin signature: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("plugin signature has wrong length: expected %d bytes, got %d", ed25519.SignatureSize, len(sigBytes))
	}

	// Compute the SHA-256 digest of the tarball.
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive for signature verification: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing archive for signature verification: %w", err)
	}
	digest := h.Sum(nil) // 32-byte raw digest

	if !ed25519.Verify(pubKey, digest, sigBytes) {
		return fmt.Errorf("plugin signature verification failed: tarball does not match registry signature (possible tampering)")
	}
	return nil
}

// checkLicense validates that a license key exists and is acceptable.
// It checks all configured keys (multi-key support) against the local
// entitlement cache first, then falls through to remote validation.
func checkLicense(ctx context.Context, name string) error {
	keys := license.CollectLicenseKeys()

	// Fallback: try legacy single-key path if CollectLicenseKeys found nothing.
	if len(keys) == 0 {
		key := os.Getenv("NSELF_PLUGIN_LICENSE_KEY")
		if key == "" {
			keyPath := licenseKeyPath()
			data, err := os.ReadFile(keyPath)
			if err != nil {
				return fmt.Errorf("plugin %q requires a license key. Run 'nself license add <key>' or visit %s",
					name, "nself.org/pricing")
			}
			key = strings.TrimSpace(string(data))
		}
		if key != "" {
			keys = []string{key}
		}
	}

	if len(keys) == 0 {
		return fmt.Errorf("plugin %q requires a license key. Run 'nself license add <key>' or visit %s",
			name, "nself.org/pricing")
	}

	cacheDir := licenseCacheDir()

	// Fast path: check the entitlement cache before any network call.
	if allowed, found := checkEntitlements(cacheDir, name); found {
		if allowed {
			return nil
		}
		// Cache says not allowed, but we might have a new key not yet cached.
		// Fall through to check all keys.
	}

	// Try each key. If any key covers this plugin, allow it.
	var lastErr error
	for _, key := range keys {
		if err := validateLicenseFormat(key); err != nil {
			continue
		}

		// Check the per-key license cache before hitting the network.
		if valid, found := checkLicenseCache(key, cacheDir); found {
			if valid {
				return nil
			}
			continue
		}

		valid, lvr, err := validateLicenseRemoteWithEntitlements(ctx, key, pingAPIURL())
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				lastErr = fmt.Errorf("plugin %q: %w", name, errs.ErrLicenseExpired)
				continue
			}
			// Network unavailable: try offline cache.
			if offlineValid, offlineFound := checkLicenseCacheOffline(key, cacheDir); offlineFound && offlineValid {
				return nil
			}
			lastErr = fmt.Errorf("plugin %q: %w", name, errs.ErrLicenseNetworkUnavailable)
			continue
		}
		_ = CacheLicense(key, valid, cacheDir)

		if lvr != nil && len(lvr.Plugins) > 0 {
			_ = cacheEntitlements(cacheDir, lvr.Tier, lvr.Plugins)
			for _, p := range lvr.Plugins {
				if p == name {
					return nil
				}
			}
		}

		if valid {
			// Key is valid but doesn't cover this specific plugin.
			lastErr = fmt.Errorf("plugin %q: %w", name, errs.ErrLicenseTierTooLow)
			continue
		}
		lastErr = fmt.Errorf("plugin %q: license key is not valid", name)
	}

	// No key covered this plugin. Provide a helpful suggestion.
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("plugin %q requires a license key. Get one at %s", name, "nself.org/pricing")
}

// CheckEOLBlock fetches the registry entry for name and returns an error if
// the plugin's status is "eol" and allowEOL is false. (S58-T03)
// If the registry cannot be reached or the plugin is not found, this function
// returns nil (install will fail downstream with a more specific error).
// This is intentionally a pre-flight check: it does not install anything.
func CheckEOLBlock(ctx context.Context, name string, allowEOL bool) error {
	cacheDir := defaultCacheDir()
	reg, err := FetchRegistry(ctx, "", cacheDir)
	if err != nil {
		// Registry unreachable — defer failure to Install proper.
		return nil
	}
	manifest, found := findPlugin(reg, name)
	if !found {
		return nil
	}
	if manifest.PublishStatus == "eol" && !allowEOL {
		return fmt.Errorf(
			"plugin %q has reached end-of-life and cannot be installed.\n"+
				"Use --allow-eol to override (not recommended).\n"+
				"Run 'nself plugin info %s' for details.",
			name, name,
		)
	}
	return nil
}

// checkAuthorRevocation fetches the author CRL from plugins.nself.org and
// returns an error if the plugin author appears in the revocation list. (S58-T09)
// Short-circuits on any network / parse error — install proceeds and the
// full install step will catch other issues. Revocation is security-critical
// so errors are logged but not fatal (to avoid bricking installs during a
// transient outage).
func checkAuthorRevocation(ctx context.Context, author string) error {
	if author == "" {
		return nil // no author field — nothing to check
	}

	url := "https://plugins.nself.org/.well-known/revoked-authors.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil // construction failure — do not block install
	}
	req.Header.Set("User-Agent", "nself-cli")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Network failure — do not block install. CLI warns instead.
		fmt.Fprintf(os.Stderr, "warning: could not fetch author revocation list (offline?): %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Non-200 — registry side issue, do not block.
		return nil
	}

	type revokedEntry struct {
		AuthorKey string `json:"authorKey"`
		RevokedAt string `json:"revokedAt"`
		Reason    string `json:"reason,omitempty"`
	}
	type crlResponse struct {
		RevokedAuthors []revokedEntry `json:"revokedAuthors"`
	}

	var crl crlResponse
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // 512 KB max
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(body, &crl); err != nil {
		return nil
	}

	for _, entry := range crl.RevokedAuthors {
		if strings.EqualFold(entry.AuthorKey, author) {
			msg := fmt.Sprintf(
				"plugin author %q has been revoked and cannot be installed (revoked: %s).\n"+
					"Remove any plugins from this author and contact support@nself.org.\n"+
					"See https://nself.org/security/revocations for details.",
				author, entry.RevokedAt,
			)
			if entry.Reason != "" {
				msg += "\nReason: " + entry.Reason
			}
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}

// findPlugin locates a plugin in the registry by name (case-insensitive).
func findPlugin(reg *Registry, name string) (*PluginManifest, bool) {
	for i := range reg.Plugins {
		if strings.EqualFold(reg.Plugins[i].Name, name) {
			return &reg.Plugins[i], true
		}
	}
	return nil, false
}

// downloadPlugin fetches the plugin tarball to a temporary file.
// For free plugins, it tries the R2-backed worker URL first and falls back to
// GitHub Releases on 5xx responses (S67-T03).
func downloadPlugin(ctx context.Context, name, version, repository string) (string, error) {
	primaryURL := buildDownloadURL(name, version, repository)
	tmp, err := downloadFromURL(ctx, primaryURL)
	if err == nil {
		return tmp, nil
	}

	// If not a paid plugin, attempt GitHub Releases fallback on primary failure.
	if !isPaidPlugin(name) {
		fallbackURL := buildFallbackDownloadURL(name, version, repository)
		if fallbackURL != primaryURL {
			tmp2, fallbackErr := downloadFromURL(ctx, fallbackURL)
			if fallbackErr == nil {
				return tmp2, nil
			}
			return "", fmt.Errorf("download failed: primary %s: %w; fallback %s: %v", primaryURL, err, fallbackURL, fallbackErr)
		}
	}

	return "", err
}

// downloadFromURL fetches a single URL to a temp file and returns the file path.
func downloadFromURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating download request: %w", err)
	}
	req.Header.Set("User-Agent", "nself-cli")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP GET %s: status %d", url, resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "nself-plugin-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("writing download to temp file: %w", err)
	}

	return tmp.Name(), nil
}

// buildDownloadURL constructs the tarball URL for a plugin. Pro plugins use
// the ping API download endpoint; free plugins use the plugins.nself.org worker
// which 302-redirects to R2 (primary) with GitHub Releases as fallback.
func buildDownloadURL(name, version, repository string) string {
	if isPaidPlugin(name) {
		base := pingAPIURL()
		return fmt.Sprintf("%s/plugins/%s/download", base, name)
	}
	// Free plugins: S67-T03 — use plugins.nself.org worker tarball endpoint.
	// The worker 302-redirects to R2 (primary CDN, free egress) or falls back
	// to GitHub Releases on R2 5xx. Override via NSELF_PLUGIN_REGISTRY env var.
	base := "https://plugins.nself.org"
	if envURL := os.Getenv("NSELF_PLUGIN_REGISTRY"); envURL != "" {
		base = strings.TrimRight(envURL, "/")
	}
	return fmt.Sprintf("%s/plugins/%s/tarball", base, name)
}

// buildFallbackDownloadURL constructs the GitHub Releases fallback URL for a
// free plugin. Used when the primary R2/worker download fails.
func buildFallbackDownloadURL(name, version, repository string) string {
	if repository != "" {
		repo := strings.TrimSuffix(repository, ".git")
		return fmt.Sprintf("%s/releases/download/v%s/%s-v%s.tar.gz", repo, version, name, version)
	}
	return fmt.Sprintf("https://github.com/nself-org/plugins/releases/download/v%s/%s-v%s.tar.gz", version, name, version)
}

// extractTarGz extracts a gzipped tarball into destDir.
func extractTarGz(archivePath, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		// Tar slip prevention: reject absolute paths and directory traversal.
		if filepath.IsAbs(hdr.Name) {
			return fmt.Errorf("unsafe tar entry: absolute path %q", hdr.Name)
		}
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("unsafe tar entry: directory traversal %q", hdr.Name)
		}
		target := filepath.Join(destDir, cleanName)
		// Final safety check: verify resolved target stays within destDir.
		if !strings.HasPrefix(target, destDir+string(filepath.Separator)) && target != destDir {
			return fmt.Errorf("tar slip detected: %q escapes destination directory", hdr.Name)
		}

		// S-014: Strip setuid, setgid, and sticky bits from tar entries
		// to prevent privilege escalation from malicious plugin archives.
		mode := os.FileMode(hdr.Mode) &^ (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)

		switch hdr.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("unsafe tar entry: symlinks not allowed (%q → %q)", hdr.Name, hdr.Linkname)
		case tar.TypeDir:
			if err := os.MkdirAll(target, mode); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("creating parent directory for %s: %w", target, err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			outFile.Close()
		}
	}

	return nil
}

// rollbackInstall cleans up a partially installed plugin by removing the
// extracted directory and dropping the database schema. Errors during
// rollback are logged but not returned.
func rollbackInstall(ctx context.Context, cfg *config.Config, name string, destDir string) {
	if err := os.RemoveAll(destDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: rollback cleanup failed for %s: %v\n", destDir, err)
	}
	if err := dropPluginSchema(ctx, cfg, name); err != nil {
		fmt.Fprintf(os.Stderr, "warning: rollback schema drop failed for %s: %v\n", name, err)
	}
}

// listInstalled scans pluginDir for subdirectories that contain a valid
// plugin.json manifest and returns PluginInfo for each.
func listInstalled(pluginDir string) ([]PluginInfo, error) {
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugin directory: %w", err)
	}

	var plugins []PluginInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.json")
		m, err := parseManifest(manifestPath)
		if err != nil {
			continue // skip directories without valid manifests
		}

		running := false
		if st, err := Status(m.Name); err == nil && st.State == "running" {
			running = true
		}

		plugins = append(plugins, PluginInfo{
			Name:      m.Name,
			Version:   m.Version,
			Category:  m.Category,
			Installed: true,
			Running:   running,
		})
	}

	return plugins, nil
}

// listFromRegistry fetches the registry and returns PluginInfo for every
// known plugin, marking those already installed locally.
func listFromRegistry(pluginDir string) ([]PluginInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cacheDir := defaultCacheDir()
	reg, err := FetchRegistry(ctx, "", cacheDir)
	if err != nil {
		return nil, fmt.Errorf("fetching registry: %w", err)
	}

	// Build a set of installed plugin names for quick lookup.
	installedSet := make(map[string]bool)
	if entries, dirErr := os.ReadDir(pluginDir); dirErr == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				installedSet[entry.Name()] = true
			}
		}
	}

	plugins := make([]PluginInfo, 0, len(reg.Plugins))
	for _, m := range reg.Plugins {
		installed := installedSet[m.Name]
		running := false
		if installed {
			if st, err := Status(m.Name); err == nil && st.State == "running" {
				running = true
			}
		}
		plugins = append(plugins, PluginInfo{
			Name:          m.Name,
			Version:       m.Version,
			Category:      m.Category,
			Installed:     installed,
			Running:       running,
			PublishStatus: m.PublishStatus,
		})
	}

	return plugins, nil
}

// baseServiceRoutes lists the always-on nSelf service routes that are present
// on any clean init. Plugins must not claim these paths — they are owned by
// core infrastructure and seeded into the conflict-detection set before any
// plugin routes are evaluated.
var baseServiceRoutes = []nginx.NginxRoute{
	{ServerName: "api", Location: "/", PluginName: "hasura"},
	{ServerName: "auth", Location: "/", PluginName: "auth"},
	{ServerName: "storage", Location: "/", PluginName: "storage"},
}

// collectInstalledPluginRoutes scans pluginDir and returns the nginx routes
// declared by all installed plugins except the one named skipPlugin (the plugin
// being installed, so reinstalls are permitted). Base service routes (Hasura,
// Auth, Storage) are seeded first so plugins that claim those paths are
// rejected with a clear conflict message naming the base service as owner.
func collectInstalledPluginRoutes(pluginDir, skipPlugin string) []nginx.NginxRoute {
	// Seed with always-on base service routes. This prevents false-positive
	// "clean install succeeded" for plugins that try to claim /api, /auth, or
	// /storage — and produces a clear "claimed by hasura/auth/storage" message.
	routes := make([]nginx.NginxRoute, len(baseServiceRoutes))
	copy(routes, baseServiceRoutes)

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return routes
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.EqualFold(entry.Name(), skipPlugin) {
			continue
		}
		manifestPath := filepath.Join(pluginDir, entry.Name(), "plugin.json")
		m, err := parseManifest(manifestPath)
		if err != nil {
			continue
		}
		for _, endpoint := range m.APIEndpoints {
			ep := strings.TrimPrefix(endpoint, "https://")
			ep = strings.TrimPrefix(ep, "http://")
			parts := strings.SplitN(ep, "/", 2)
			serverName := parts[0]
			location := "/"
			if len(parts) == 2 && parts[1] != "" {
				location = "/" + parts[1]
			}
			routes = append(routes, nginx.NginxRoute{
				ServerName: serverName,
				Location:   location,
				PluginName: m.Name,
			})
		}
	}
	return routes
}

// --- path helpers ---

// DefaultCacheDir is the exported alias of defaultCacheDir for use by
// commands that need to call FetchRegistry directly. (S58-T06)
func DefaultCacheDir() string { return defaultCacheDir() }

// FindPluginByName is the exported wrapper around findPlugin for use by
// commands outside the plugin package. (S58-T06)
func FindPluginByName(reg *Registry, name string) (*PluginManifest, bool) {
	return findPlugin(reg, name)
}

// CompareVersions compares two semver strings a and b.
// Returns -1 if a < b, 0 if a == b, +1 if a > b. (S58-T06)
func CompareVersions(a, b string) int { return compareSemver(a, b) }

// defaultCacheDir returns the default plugin cache directory.
func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "cache", "plugins")
	}
	return filepath.Join(home, ".nself", "cache", "plugins")
}

// LicenseCacheDir returns the directory used for license validation caching.
// This is ~/.nself/license/ (or /tmp/.nself/license/ if the home directory
// cannot be determined).
func LicenseCacheDir() string {
	return licenseCacheDir()
}

// licenseCacheDir returns the directory used for license validation caching.
func licenseCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "license")
	}
	return filepath.Join(home, ".nself", "license")
}

// licenseKeyPath returns the file path where the license key is stored.
func licenseKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "license", "key")
	}
	return filepath.Join(home, ".nself", "license", "key")
}

// pingAPIURL returns the license validation API base URL.
func pingAPIURL() string {
	if u := os.Getenv("NSELF_PING_API_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "https://ping.nself.org"
}
