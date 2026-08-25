package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Linking a CLI-type plugin's binary into the directory the command proxy reads.
//
// Purpose: close the gap between installation and invocation. Install extracts a
// plugin to ~/.nself/plugins/<name>/, but ProxyCommand only ever looks for
// ~/.nself/plugins/bin/nself-<name>. Nothing connected the two, so a plugin that
// exists purely to add a command — which is what every CLI-R11 extraction
// produces — installed successfully and then could not be run. The user saw
// "unknown command", having just installed it.
//
// Inputs: the extracted plugin directory and its manifest.
//
// Outputs: ~/.nself/plugins/bin/nself-<name>, and the same path removed on
// uninstall.
//
// Constraints: the bin directory is the ONLY place ProxyCommand searches, and
// deliberately so — resolving plugin binaries through $PATH would let anything
// earlier on the path impersonate a plugin (S-002). Nothing here widens that.

// PluginBinDir returns the directory the command proxy searches for plugin
// binaries. Exported so the installer and the router cannot drift apart on
// where that is.
func PluginBinDir() string { return pluginBinDir() }

// PublishedBinaryPath returns where a plugin's command binary is published,
// including the platform's executable suffix.
//
// Exported so tests assert against the same path the installer writes. Having
// the test compute the name itself is how the Windows job went red: the
// installer appended .exe and the test did not.
func PublishedBinaryPath(binName string) string {
	dst := filepath.Join(pluginBinDir(), binName)
	if runtime.GOOS == "windows" {
		dst += ".exe"
	}
	return dst
}

// cliBinaryNames returns every binary a CLI-type plugin installs.
//
// Most plugins provide one command and return a single name. A plugin may
// declare several in cliCommands, which is how `sentry` ships both
// `nself sentry` and `nself sentry-server`: the proxy resolves nself-<command>
// for each, so each needs its own published binary. Splitting them into two
// plugins would mean two copies of the same client code, and folding
// sentry-server in as a subcommand would change a command users already type.
func cliBinaryNames(name string, m *PluginManifest) []string {
	if m == nil {
		return nil
	}
	if m.PluginType != "" && m.PluginType != "cli" {
		return nil
	}
	if len(m.CLICommands) > 0 {
		out := make([]string, 0, len(m.CLICommands))
		for _, c := range m.CLICommands {
			if c.Name != "" {
				out = append(out, "nself-"+c.Name)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if single := cliBinaryName(name, m); single != "" {
		return []string{single}
	}
	return nil
}

// cliBinaryName returns the binary name a single-command CLI plugin installs,
// or "" when the plugin does not provide a command.
//
// A manifest may state binaryName explicitly; otherwise the convention is
// nself-<plugin name>, which is what ProxyCommand looks for.
func cliBinaryName(name string, m *PluginManifest) string {
	if m == nil {
		return ""
	}
	if m.PluginType != "" && m.PluginType != "cli" {
		return ""
	}
	if m.BinaryName != "" {
		return m.BinaryName
	}
	if m.PluginType == "cli" {
		return "nself-" + name
	}
	return ""
}

// linkCLIBinary publishes a freshly extracted CLI plugin's binary into the
// proxy's lookup directory.
//
// A plugin that declares no command is not a CLI plugin and there is nothing to
// do — cliBinaryName returns "" for those, which is most plugins.
//
// A plugin that DOES declare a command but ships no binary is a hard error, and
// this used to return nil instead. That combination is exactly what the release
// pipeline currently produces: release-tarballs.yml runs `tar -czf` over the
// plugin's source directory and never compiles anything, so a Go CLI plugin
// arrives as source. The install then reported success and the command stayed
// dead:
//
//	$ nself install infra          # "installed"
//	$ nself infra --help
//	unknown command "infra", and no plugin named "infra" is installed
//
// Failing here does not fix the pipeline, and is not meant to. It turns a
// silent, confusing break into one that names the cause at the moment it
// happens, rather than leaving the user to discover it later with no clue why.
func linkCLIBinary(destDir, name string, m *PluginManifest) error {
	binNames := cliBinaryNames(name, m)
	if len(binNames) == 0 {
		return nil
	}

	binDir := pluginBinDir()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("creating plugin bin dir: %w", err)
	}

	for _, binName := range binNames {
		src := findExtractedBinary(destDir, binName)
		if src == "" {
			return fmt.Errorf(
				"plugin %q provides the %q command but its package contains no %s binary — "+
					"the published archive is source-only, so the command could not be installed",
				name, strings.TrimPrefix(binName, "nself-"), binName)
		}

		dst := PublishedBinaryPath(binName)

		// Replace any previous version rather than failing on an existing file:
		// an upgrade must leave the proxy pointing at the new binary.
		if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("replacing plugin binary %s: %w", dst, err)
		}

		if err := copyExecutable(src, dst); err != nil {
			return fmt.Errorf("publishing plugin binary %s: %w", binName, err)
		}
	}
	return nil
}

// unlinkCLIBinary removes a plugin's published binary. A missing file is not an
// error — the plugin may never have shipped one.
func unlinkCLIBinary(name string, m *PluginManifest) error {
	binNames := cliBinaryNames(name, m)
	if len(binNames) == 0 {
		binNames = []string{"nself-" + name} // best effort on an unreadable manifest
	}

	for _, binName := range binNames {
		dst := PublishedBinaryPath(binName)
		if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("removing plugin binary %s: %w", dst, err)
		}
	}
	return nil
}

// findExtractedBinary locates the named binary inside an extracted plugin.
// Tarballs vary: some put it at the root, some under bin/, some under a
// platform directory. Searching is more robust than guessing one layout.
func findExtractedBinary(destDir, binName string) string {
	candidates := []string{
		filepath.Join(destDir, binName),
		filepath.Join(destDir, "bin", binName),
		filepath.Join(destDir, binName+".exe"),
		filepath.Join(destDir, "bin", binName+".exe"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}

	var found string
	_ = filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" || info.IsDir() {
			return nil //nolint:nilerr // a walk error just means "keep looking"
		}
		base := info.Name()
		if base == binName || base == binName+".exe" {
			found = path
		}
		return nil
	})
	return found
}

// copyExecutable copies src to dst and makes it executable. A copy rather than
// a symlink: the extracted plugin directory is removed on uninstall, and a
// dangling symlink in the proxy's lookup path would be worse than no entry.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return os.Chmod(dst, 0o755)
}

// readPluginManifest loads plugin.json from an installed plugin directory.
// Returns nil when it cannot be read; callers treat that as "assume the
// nself-<name> convention" rather than as a failure, because a plugin with a
// corrupt manifest must still be removable.
func readPluginManifest(destDir string) *PluginManifest {
	data, err := os.ReadFile(filepath.Join(destDir, "plugin.json"))
	if err != nil {
		return nil
	}
	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return &m
}

// IsCommandInstalled reports whether a plugin providing the named command is
// present, i.e. whether ProxyCommand would find a binary for it.
//
// Used by the CLI to decide whether a relocated command still needs its
// "moved to a plugin" notice. Once the plugin is installed the old spelling is
// the supported spelling, and repeating the install hint is noise.
func IsCommandInstalled(cmdName string) bool {
	candidate := PublishedBinaryPath("nself-" + cmdName)
	info, err := os.Stat(candidate)
	return err == nil && !info.IsDir()
}

// pluginOwnsTables reports whether a plugin declares any database tables.
//
// A plugin with none needs no schema, and every CLI-R11 extraction produces
// exactly that shape: a Go binary that adds a command, with "tables": [] in its
// manifest. Running the schema step for those meant `nself install <cmd>`
// required Docker and a running Postgres to create an empty schema, so a
// command-line tool could not be installed on a machine without a stack.
//
// A nil manifest is treated as owning tables: it means the manifest could not
// be read, and skipping schema creation on a guess is the more damaging
// mistake of the two.
func pluginOwnsTables(m *PluginManifest) bool {
	if m == nil {
		return true
	}
	return len(m.Tables) > 0
}
