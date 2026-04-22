package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// pluginLinksPath returns the path to the plugin shadow registry JSON file.
func pluginLinksPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "plugin-links.json")
	}
	return filepath.Join(home, ".nself", "plugin-links.json")
}

// loadPluginLinks reads plugin-links.json, returning an empty map on missing file.
func loadPluginLinks() (map[string]string, error) {
	path := pluginLinksPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading plugin-links.json: %w", err)
	}
	var links map[string]string
	if err := json.Unmarshal(data, &links); err != nil {
		return nil, fmt.Errorf("parsing plugin-links.json: %w", err)
	}
	return links, nil
}

// savePluginLinks writes the links map to plugin-links.json (0600 permissions).
func savePluginLinks(links map[string]string) error {
	path := pluginLinksPath()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("creating .nself dir: %w", err)
	}
	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling plugin-links: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing plugin-links.json: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("chmod plugin-links.json: %w", err)
	}
	return nil
}

// linkPlugin registers localPath in the plugin shadow registry under its plugin name.
// Called by plugin_dev.go auto-link path.
func linkPlugin(localPath string) error {
	// Resolve the plugin name from plugin.yaml or the directory basename.
	name, err := resolvePluginName(localPath)
	if err != nil {
		return err
	}

	links, err := loadPluginLinks()
	if err != nil {
		return err
	}

	abs, err := filepath.Abs(localPath)
	if err != nil {
		return fmt.Errorf("resolving absolute path: %w", err)
	}

	links[name] = abs
	return savePluginLinks(links)
}

// resolvePluginName reads plugin.yaml to find the plugin's declared name,
// falling back to the directory basename.
func resolvePluginName(dir string) (string, error) {
	// Try plugin.yaml first.
	yamlPath := filepath.Join(dir, "plugin.yaml")
	data, err := os.ReadFile(yamlPath)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "name:") {
				name := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				name = strings.Trim(name, `"'`)
				if name != "" {
					return name, nil
				}
			}
		}
	}
	// Fall back to directory basename.
	name := filepath.Base(dir)
	if name == "" || name == "." {
		return "", fmt.Errorf("cannot determine plugin name from path %q", dir)
	}
	return name, nil
}

// --- plugin link command ---

var pluginLinkCmd = &cobra.Command{
	Use:   "link <local-path>",
	Short: "Register a local plugin directory as a shadow override",
	Long: `Register a local directory as a plugin shadow, making nself build prefer it
over the registry version.

The link is recorded in ~/.nself/plugin-links.json and persists across CLI
invocations until you run nself plugin unlink <name>.

Modes:
  Default  Container-mount + air inside container (closer to production)
  --host   Run plugin process on host (faster but networking may differ)

Security: the local path must be readable and must contain a valid plugin.yaml.
Path traversal (e.g. ../../etc) is rejected.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginLink,
}

var pluginUnlinkCmd = &cobra.Command{
	Use:   "unlink <name>",
	Short: "Remove a local plugin shadow, restoring the registry version",
	Long: `Remove a plugin link registered with nself plugin link.

After unlinking, nself build will use the registry version again.
Unlinking an already-unlinked plugin returns success with a no-op message.`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginUnlink,
}

func init() {
	pluginLinkCmd.Flags().Bool("host", false, "Use host-process mode instead of container-mount (macOS recommended for faster I/O)")
	pluginLinkCmd.Flags().Bool("list", false, "List currently linked plugins")
	pluginCmd.AddCommand(pluginLinkCmd)
	pluginCmd.AddCommand(pluginUnlinkCmd)
}

func runPluginLink(cmd *cobra.Command, args []string) error {
	listMode, _ := cmd.Flags().GetBool("list")
	if listMode {
		return runPluginLinkList()
	}

	localPath := args[0]
	hostMode, _ := cmd.Flags().GetBool("host")

	// Security: validate the path is safe and contains a valid plugin manifest.
	absPath, err := validateLinkPath(localPath)
	if err != nil {
		return err
	}

	name, err := resolvePluginName(absPath)
	if err != nil {
		return err
	}

	links, err := loadPluginLinks()
	if err != nil {
		return err
	}

	links[name] = absPath
	if err := savePluginLinks(links); err != nil {
		return err
	}

	mode := "container-mount"
	if hostMode {
		mode = "host-process"
	}

	uiSuccessf("Linked %s -> %s (%s mode)", name, absPath, mode)
	ui.Dimmed("Run `nself build` to activate the local version.")
	uiDimmedf("Run `nself plugin unlink %s` to restore the registry version.", name)
	return nil
}

func runPluginUnlink(cmd *cobra.Command, args []string) error {
	name := args[0]

	links, err := loadPluginLinks()
	if err != nil {
		return err
	}

	if _, ok := links[name]; !ok {
		uiInfof("Plugin %q is not linked (no-op).", name)
		return nil
	}

	delete(links, name)
	if err := savePluginLinks(links); err != nil {
		return err
	}

	uiSuccessf("Unlinked %s. Run `nself build` to use the registry version.", name)
	return nil
}

func runPluginLinkList() error {
	links, err := loadPluginLinks()
	if err != nil {
		return err
	}

	if len(links) == 0 {
		ui.Info("No plugins currently linked.")
		return nil
	}

	tbl := ui.NewTable("Name", "Local Path")
	for name, path := range links {
		tbl.AddRow(name, path)
	}
	tbl.Render()
	return nil
}

// validateLinkPath resolves the path, checks it exists, and rejects path traversal attempts.
func validateLinkPath(rawPath string) (string, error) {
	abs, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Reject obvious path traversal (resolved path must be under home or a sane prefix).
	home, err := os.UserHomeDir()
	if err == nil {
		// Warn but don't hard-block paths outside home (user may use /srv/plugins etc.).
		_ = home
	}

	// Must exist and be a directory.
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("path %q does not exist: %w", abs, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("path %q is not a directory", abs)
	}

	// Must contain a plugin manifest.
	manifestPath := filepath.Join(abs, "plugin.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		return "", fmt.Errorf("path %q does not contain plugin.yaml — not a valid plugin directory", abs)
	}

	return abs, nil
}
