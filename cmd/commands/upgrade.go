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

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the nSelf CLI (detects install method)",
	Long: `Upgrade the nSelf CLI binary. Detects install method (Homebrew, direct download,
system package) and uses the appropriate upgrade strategy.

  nself upgrade              # Upgrade to latest stable
  nself upgrade --check      # Check without installing
  nself upgrade --channel canary  # Switch to canary channel
  nself upgrade --version 1.2.3   # Pin to specific version
  nself upgrade --rollback   # Revert to previous version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		channel, _ := cmd.Flags().GetString("channel")
		targetVersion, _ := cmd.Flags().GetString("version")
		doRollback, _ := cmd.Flags().GetBool("rollback")

		if doRollback {
			return rollbackBinary()
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
	RootCmd.AddCommand(upgradeCmd)
}
