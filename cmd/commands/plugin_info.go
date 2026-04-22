package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var pluginInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed plugin information",
	Long: `Display detailed information about a plugin from the registry.
Use --open to open the plugin's marketplace page in your browser.

  nself plugin info ai
  nself plugin info ai --open`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInfo,
}

func init() {
	pluginInfoCmd.Flags().Bool("open", false, "Open the plugin marketplace page in a browser")
	pluginInfoCmd.Flags().Bool("json", false, "Output as JSON")
	pluginCmd.AddCommand(pluginInfoCmd)
}

func runPluginInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	openBrowser, _ := cmd.Flags().GetBool("open")

	marketplaceURL := fmt.Sprintf("https://plugins.nself.org/%s", name)

	if openBrowser {
		return openURL(marketplaceURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	registryURL := os.Getenv("NSELF_PLUGIN_REGISTRY")
	if err := plugin.ValidateNetworkAccess(ctx, registryURL); err != nil {
		return err
	}

	cacheDir := os.Getenv("NSELF_PLUGIN_CACHE")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			cacheDir = "/tmp/.nself/cache/plugins"
		} else {
			cacheDir = home + "/.nself/cache/plugins"
		}
	}

	reg, err := plugin.FetchRegistry(ctx, registryURL, cacheDir)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	var manifest *plugin.PluginManifest
	for i := range reg.Plugins {
		if strings.EqualFold(reg.Plugins[i].Name, name) {
			manifest = &reg.Plugins[i]
			break
		}
	}
	if manifest == nil {
		return fmt.Errorf("plugin %q not found in registry", name)
	}

	// Display plugin details.
	tbl := ui.NewTable("Field", "Value")
	tbl.AddRow("Name", manifest.Name)
	tbl.AddRow("Version", manifest.Version)
	tbl.AddRow("Description", manifest.Description)
	tbl.AddRow("Category", manifest.Category)
	tbl.AddRow("License", manifest.License)
	tbl.AddRow("Tier", manifest.Tier)

	if manifest.Author != "" {
		tbl.AddRow("Author", manifest.Author)
	}
	if manifest.Language != "" {
		tbl.AddRow("Language", manifest.Language)
	}
	if manifest.Port > 0 {
		tbl.AddRow("Port", fmt.Sprintf("%d", manifest.Port))
	}
	if len(manifest.Dependencies) > 0 {
		tbl.AddRow("Dependencies", strings.Join(manifest.Dependencies, ", "))
	}
	if len(manifest.Tags) > 0 {
		tbl.AddRow("Tags", strings.Join(manifest.Tags, ", "))
	}
	if manifest.Repository != "" {
		tbl.AddRow("Repository", manifest.Repository)
	}
	if manifest.Compat != nil && manifest.Compat.Nself != "" {
		tbl.AddRow("CLI Compat", manifest.Compat.Nself)
	}

	tbl.Render()

	// S71-T03: Display declared permissions (one per line, with risk tier prefix).
	if len(manifest.Permissions) > 0 {
		fmt.Println("\nPermissions:")
		for _, perm := range manifest.Permissions {
			fmt.Printf("  %s %s\n", permissionRiskPrefix(perm), perm)
		}
		fmt.Println("  (v1.0.9: informational only; v1.1.0 will require explicit confirmation)")
	}

	// Check if installed locally.
	pluginDir := resolvePluginDir()
	installed, err := plugin.ListInstalled(pluginDir)
	if err == nil {
		for _, p := range installed {
			if strings.EqualFold(p.Name, name) {
				fmt.Printf("\nInstalled: %s (%s)\n", p.Version, p.Status)
				break
			}
		}
	}

	fmt.Printf("\nMarketplace: %s\n", marketplaceURL)
	fmt.Printf("Install:     nself plugin install %s\n", name)

	return nil
}

// permissionRiskPrefix returns a short risk label for a permission string.
// Color coding mirrors the admin UI badge classification (S71-T03):
//   - [green]  safe / scoped: db:read, network:plugin:*
//   - [yellow] moderate risk: network:internet, fs:write:*
//   - [red]    high risk: system:exec
//
// Plain-text prefixes are used here (no ANSI) so the output is pipe-safe.
func permissionRiskPrefix(perm string) string {
	switch {
	case perm == "system:exec":
		return "[high]"
	case perm == "network:internet",
		strings.HasPrefix(perm, "fs:write:"):
		return "[med] "
	default:
		return "[low] "
	}
}

// openURL opens the given URL in the default browser.
func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform %q — open %s manually", runtime.GOOS, url)
	}
	return cmd.Start()
}
