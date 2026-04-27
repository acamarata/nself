package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"
	"github.com/nself-org/cli/internal/version"

	"github.com/spf13/cobra"
)

// executablePath is a package-level hook so tests can override which binary
// path selfUpdateFromURL (and the upgrade command's binary-url path) targets.
// In production it always delegates to os.Executable.
var executablePath = os.Executable

// channelConfig persists the user's chosen release channel.
type channelConfig struct {
	Channel string `json:"channel"` // "stable" or "canary"
}

// channelConfigPath returns the path to the channel config file.
func channelConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".nself", "channel.json")
	}
	return filepath.Join(home, ".config", "nself", "channel.json")
}

// loadChannel reads the persisted channel preference. Returns "stable" if none set.
func loadChannel() string {
	data, err := os.ReadFile(channelConfigPath())
	if err != nil {
		return "stable"
	}
	var cfg channelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "stable"
	}
	if cfg.Channel == "" {
		return "stable"
	}
	return cfg.Channel
}

// saveChannel persists the channel preference.
func saveChannel(channel string) error {
	path := channelConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.Marshal(channelConfig{Channel: channel})
	if err != nil {
		return fmt.Errorf("marshaling channel config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// detectInstallMethod returns how nself was installed: "homebrew", "direct", or "system".
func detectInstallMethod() string {
	exePath, err := os.Executable()
	if err != nil {
		return "direct"
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	// Homebrew detection: binary lives under a Homebrew prefix
	homebrewPrefixes := []string{
		"/usr/local/Cellar",
		"/opt/homebrew/Cellar",
		"/home/linuxbrew/.linuxbrew/Cellar",
	}
	for _, prefix := range homebrewPrefixes {
		if strings.HasPrefix(exePath, prefix) {
			return "homebrew"
		}
	}

	// System package manager detection (apt, rpm, etc.)
	systemPaths := []string{
		"/usr/bin/nself",
		"/usr/sbin/nself",
	}
	for _, sp := range systemPaths {
		if exePath == sp {
			return "system"
		}
	}

	return "direct"
}

// backupBinary copies the current binary to ~/.nself/bin/nself.prev for rollback.
func backupBinary() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving executable: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("resolving symlinks: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}

	backupDir := filepath.Join(home, ".nself", "bin")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("creating backup dir: %w", err)
	}

	backupPath := filepath.Join(backupDir, "nself.prev")
	src, err := os.ReadFile(exePath)
	if err != nil {
		return "", fmt.Errorf("reading current binary: %w", err)
	}
	if err := os.WriteFile(backupPath, src, 0755); err != nil {
		return "", fmt.Errorf("writing backup: %w", err)
	}

	return backupPath, nil
}

// rollbackBinary restores the previous binary from backup.
func rollbackBinary() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	backupPath := filepath.Join(home, ".nself", "bin", "nself.prev")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found at %s", backupPath)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable: %w", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	src, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}
	if err := os.WriteFile(exePath, src, 0755); err != nil {
		return fmt.Errorf("writing restored binary: %w", err)
	}

	ui.Success("Rolled back to previous version")
	return nil
}

// validateBinaryURL returns an error if url is not safe to use as a binary
// download target.  Only HTTPS is accepted (plain HTTP is forbidden — all
// downloads must be encrypted in transit).  The host must be on the operator
// allowlist so that an accidental or injected URL cannot redirect the binary
// swap to an arbitrary server.  The path must end in .tar.gz or .tgz so the
// extractor knows how to handle it; raw-binary downloads are rejected here
// to keep the operator-facing contract narrow.
func validateBinaryURL(url string) error {
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("--binary-url must use HTTPS (got %q)", url)
	}
	allowed := []string{
		"github.com",
		"objects.githubusercontent.com",
		"install.nself.org",
		"ping.nself.org",
	}
	// Strip the scheme, then extract the host (everything up to the first /).
	rest := strings.TrimPrefix(url, "https://")
	host := rest
	pathPart := ""
	if idx := strings.IndexByte(rest, '/'); idx != -1 {
		host = rest[:idx]
		pathPart = rest[idx:]
	}
	// Strip port if present.
	if idx := strings.LastIndexByte(host, ':'); idx != -1 {
		host = host[:idx]
	}
	hostOK := false
	for _, a := range allowed {
		if host == a || strings.HasSuffix(host, "."+a) {
			hostOK = true
			break
		}
	}
	if !hostOK {
		return fmt.Errorf(
			"--binary-url host %q is not on the allowed list; accepted hosts: %s",
			host,
			strings.Join(allowed, ", "),
		)
	}
	// Strip query string before extension check so URLs with auth tokens or
	// signed-URL params still pass.
	cleanPath := pathPart
	if idx := strings.IndexAny(cleanPath, "?#"); idx != -1 {
		cleanPath = cleanPath[:idx]
	}
	lower := strings.ToLower(cleanPath)
	if !strings.HasSuffix(lower, ".tar.gz") && !strings.HasSuffix(lower, ".tgz") {
		return fmt.Errorf(
			"--binary-url path must end in .tar.gz or .tgz (got %q)",
			pathPart,
		)
	}
	return nil
}

// validateBinarySHA256 returns nil when sum is a 64-char lowercase hex
// SHA-256 digest. Operators may pass it with or without surrounding
// whitespace; uppercase is normalized at the call site.
func validateBinarySHA256(sum string) error {
	if len(sum) != 64 {
		return fmt.Errorf("--binary-sha256 must be a 64-char hex digest (got length %d)", len(sum))
	}
	for _, r := range sum {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
		if !isHex {
			return fmt.Errorf("--binary-sha256 must be lowercase hex (got %q)", sum)
		}
	}
	return nil
}

// checksumURLFromBinaryURL derives the expected checksums.txt URL from a
// binary or archive URL.  The convention follows the goreleaser layout:
//
//	https://example.com/path/to/nself-linux-amd64          -> .../checksums.txt
//	https://example.com/path/to/v1.0.9/nself-1.0.9-linux.tar.gz -> .../checksums.txt
//
// The checksum file is expected to live in the same directory as the binary.
func checksumURLFromBinaryURL(binaryURL string) string {
	idx := strings.LastIndexByte(binaryURL, '/')
	if idx == -1 {
		return binaryURL + "/checksums.txt"
	}
	return binaryURL[:idx+1] + "checksums.txt"
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the nSelf CLI (detects install method)",
	Long: `Upgrade the nSelf CLI binary. Detects install method (Homebrew, direct download,
system package) and uses the appropriate upgrade strategy.

  nself upgrade                          # Upgrade to latest stable
  nself upgrade --check                  # Check without installing
  nself upgrade --channel canary         # Switch to canary channel
  nself upgrade --version 1.2.3         # Pin to specific version
  nself upgrade --rollback               # Revert to previous version
  nself upgrade --binary-url <url>       # Install from a specific URL (operators)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		channel, _ := cmd.Flags().GetString("channel")
		targetVersion, _ := cmd.Flags().GetString("version")
		doRollback, _ := cmd.Flags().GetBool("rollback")
		binaryURL, _ := cmd.Flags().GetString("binary-url")
		binarySHA, _ := cmd.Flags().GetString("binary-sha256")

		if doRollback {
			return rollbackBinary()
		}

		// --binary-url: operator-pinned direct download path.
		// Skips the GitHub releases API entirely and downloads from the
		// provided URL.  All other upgrade guards (backup, checksum,
		// atomic swap) still apply.
		if binaryURL != "" {
			if err := validateBinaryURL(binaryURL); err != nil {
				return err
			}

			binarySHA = strings.TrimSpace(strings.ToLower(binarySHA))
			if binarySHA != "" {
				if err := validateBinarySHA256(binarySHA); err != nil {
					return err
				}
			}

			method := detectInstallMethod()
			if method == "homebrew" {
				ui.Warn("This nSelf binary was installed via Homebrew.")
				ui.Warn("Using --binary-url with a Homebrew install may break Homebrew's tracking of the formula.")
				ui.Warn("Consider upgrading through Homebrew instead unless you have a specific reason to bypass it.")
			}

			// Backup before swapping.
			if backupPath, err := backupBinary(); err != nil {
				ui.Warn(fmt.Sprintf("Could not backup current binary: %v", err))
			} else {
				ui.Dimmed(fmt.Sprintf("Backup saved to %s", backupPath))
			}

			ui.Info(fmt.Sprintf("Downloading binary from %s", binaryURL))
			var upgradeErr error
			if binarySHA != "" {
				ui.Info("Verifying with operator-supplied --binary-sha256")
				upgradeErr = selfUpdateFromURLWithSHA(binaryURL, binarySHA)
			} else {
				checksumURL := checksumURLFromBinaryURL(binaryURL)
				ui.Info(fmt.Sprintf("Fetching checksum from %s", checksumURL))
				upgradeErr = selfUpdateFromURL(binaryURL, checksumURL)
			}
			if upgradeErr != nil {
				ui.Error(fmt.Sprintf("Upgrade failed: %v", upgradeErr))
				ui.Info("Run 'nself upgrade --rollback' to restore the previous version")
				return upgradeErr
			}

			ui.Success("Binary installed from custom URL")
			return nil
		}

		// --binary-sha256 without --binary-url is a misuse; reject early.
		if binarySHA != "" {
			return fmt.Errorf("--binary-sha256 requires --binary-url")
		}

		// Persist channel choice if provided
		if channel != "" {
			if channel != "stable" && channel != "canary" {
				return fmt.Errorf("invalid channel %q: must be 'stable' or 'canary'", channel)
			}
			if err := saveChannel(channel); err != nil {
				ui.Warn(fmt.Sprintf("Could not save channel preference: %v", err))
			}
		} else {
			channel = loadChannel()
		}

		method := detectInstallMethod()
		current := version.GetVersion()

		if check {
			ui.Info(fmt.Sprintf("Current version: %s", current))
			ui.Info(fmt.Sprintf("Install method: %s", method))
			ui.Info(fmt.Sprintf("Channel: %s", channel))
			ui.Info(fmt.Sprintf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH))
			return nil
		}

		switch method {
		case "homebrew":
			formula := "nself"
			if channel == "canary" {
				formula = "nself-canary"
			}
			ui.Info(fmt.Sprintf("Upgrading via Homebrew (%s)...", formula))
			ui.Info(fmt.Sprintf("Run: brew upgrade %s", formula))
			return nil

		case "system":
			ui.Warn("nSelf was installed via a system package manager.")
			ui.Info("Use your system package manager to upgrade (e.g. apt upgrade nself, dnf upgrade nself)")
			return nil

		case "direct":
			// Backup current binary before upgrade
			backupPath, err := backupBinary()
			if err != nil {
				ui.Warn(fmt.Sprintf("Could not backup current binary: %v", err))
			} else {
				ui.Dimmed(fmt.Sprintf("Backup saved to %s", backupPath))
			}

			// Delegate to the existing selfUpdate logic from update.go
			tag := ""
			if targetVersion != "" {
				tag = targetVersion
				if !strings.HasPrefix(tag, "v") {
					tag = "v" + tag
				}
			} else {
				latest, _, err := fetchLatestRelease()
				if err != nil {
					return fmt.Errorf("checking for updates: %w", err)
				}
				tag = latest
			}

			if normalizeVersion(current) == normalizeVersion(tag) {
				ui.Success(fmt.Sprintf("Already on latest version (%s)", current))
				return nil
			}

			// Warn about plugins that will be incompatible with the new version.
			pluginDir := resolvePluginDir()
			if incompat := plugin.IncompatiblePlugins(pluginDir, tag); len(incompat) > 0 {
				ui.Warn("The following plugins may be incompatible with " + tag + ":")
				for _, p := range incompat {
					ui.Warn("  - " + p)
				}
				ui.Warn("You may need to update these plugins after upgrading.")
			}

			ui.Info(fmt.Sprintf("Upgrading: %s -> %s", current, tag))
			if err := selfUpdate(tag); err != nil {
				ui.Error(fmt.Sprintf("Upgrade failed: %v", err))
				ui.Info("Run 'nself upgrade --rollback' to restore the previous version")
				return err
			}

			ui.Success(fmt.Sprintf("Upgraded to %s", tag))
			return nil

		default:
			return fmt.Errorf("unknown install method: %s", method)
		}
	},
}

func init() {
	upgradeCmd.Flags().Bool("check", false, "Check for updates without installing")
	upgradeCmd.Flags().String("channel", "", "Release channel: stable or canary")
	upgradeCmd.Flags().String("version", "", "Upgrade to a specific version (e.g. 1.2.3)")
	upgradeCmd.Flags().Bool("rollback", false, "Revert to the previously installed version")
	upgradeCmd.Flags().String("binary-url", "", "Install from a specific binary URL (HTTPS only; .tar.gz/.tgz; operator use)")
	upgradeCmd.Flags().String("binary-sha256", "", "Operator-supplied SHA-256 digest for --binary-url (skips checksums.txt fetch)")
	RootCmd.AddCommand(upgradeCmd)
}
