package commands

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/plugin/scaffold"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// uiInfof formats and prints an info message.
func uiInfof(format string, args ...any) { ui.Info(fmt.Sprintf(format, args...)) }

// uiSuccessf formats and prints a success message.
func uiSuccessf(format string, args ...any) { ui.Success(fmt.Sprintf(format, args...)) }

// uiDimmedf formats and prints a dimmed message.
func uiDimmedf(format string, args ...any) { ui.Dimmed(fmt.Sprintf(format, args...)) }

// uiWarnf formats and prints a warning message.
func uiWarnf(format string, args ...any) { ui.Warn(fmt.Sprintf(format, args...)) }

var pluginNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new plugin project",
	Long: `Create a new plugin project with all required files.

This command uses the canonical shared scaffold package so that output is
identical to the standalone new-plugin binary shipped with plugin-sdk-go.

Templates: go (default), rust, node, static

  nself plugin new my-plugin                          # Go plugin (free tier)
  nself plugin new my-plugin --tier pro               # Pro Go plugin
  nself plugin new my-plugin --template rust          # Rust plugin
  nself plugin new my-plugin --template node          # Node.js plugin
  nself plugin new my-plugin --bundle nClaw --tier pro`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginNew,
}

func init() {
	pluginNewCmd.Flags().String("template", "go", "Plugin template: go, rust, node, static")
	pluginNewCmd.Flags().String("tier", "free", "Plugin tier: free or pro")
	pluginNewCmd.Flags().String("bundle", "", "Bundle display name (e.g. nClaw) — pro plugins only")
	pluginNewCmd.Flags().String("description", "", "Short description of the plugin")
	pluginNewCmd.Flags().String("author", "", "Plugin author name")
	pluginNewCmd.Flags().String("category", "custom", "Plugin category (see F04-PLUGIN-INVENTORY-PRO)")
	pluginNewCmd.Flags().String("min-cli", "1.0.9", "Minimum nSelf CLI version required")
	pluginNewCmd.Flags().String("min-sdk", "0.1.0", "Minimum plugin-sdk-go version required")
	pluginNewCmd.Flags().Int("port", 8080, "Default listen port")
	pluginNewCmd.Flags().String("out", "", "Output directory (default ./<name>)")
	pluginNewCmd.Flags().Bool("force", false, "Overwrite files if the output directory already exists")
	pluginCmd.AddCommand(pluginNewCmd)
}

func runPluginNew(cmd *cobra.Command, args []string) error {
	name := args[0]
	tmplType, _ := cmd.Flags().GetString("template")
	tier, _ := cmd.Flags().GetString("tier")
	bundle, _ := cmd.Flags().GetString("bundle")
	description, _ := cmd.Flags().GetString("description")
	author, _ := cmd.Flags().GetString("author")
	category, _ := cmd.Flags().GetString("category")
	minCLI, _ := cmd.Flags().GetString("min-cli")
	minSDK, _ := cmd.Flags().GetString("min-sdk")
	port, _ := cmd.Flags().GetInt("port")
	outDir, _ := cmd.Flags().GetString("out")
	force, _ := cmd.Flags().GetBool("force")

	// Validate template.
	switch tmplType {
	case "go", "rust", "node", "static":
	default:
		return fmt.Errorf("unknown template %q: must be go, rust, node, or static", tmplType)
	}

	uiInfof("Scaffolding %s plugin %q (tier: %s)...", tmplType, name, tier)

	result, err := scaffold.Run(scaffold.Options{
		Name:        name,
		Tier:        tier,
		Bundle:      bundle,
		Description: description,
		Author:      author,
		Category:    category,
		Language:    tmplType,
		MinCLI:      minCLI,
		MinSDK:      minSDK,
		Port:        port,
		OutDir:      outDir,
		Force:       force,
	})
	if err != nil {
		return fmt.Errorf("scaffold failed: %w", err)
	}

	uiSuccessf("Plugin %q scaffolded in %s/", name, result.Dir)
	uiDimmedf("Files created: %s", strings.Join(result.Files, ", "))
	ui.Dimmed("")
	ui.Dimmed("Next steps:")
	uiDimmedf("  cd %s", result.Dir)
	switch tmplType {
	case "go":
		ui.Dimmed("  go mod tidy && go test ./...")
		uiDimmedf("  nself plugin link %s    # link for development", result.Dir)
		uiDimmedf("  nself plugin dev %s     # start hot-reload watcher", name)
	case "rust":
		ui.Dimmed("  cargo build")
	case "node":
		ui.Dimmed("  pnpm install && pnpm build")
	case "static":
		ui.Dimmed("  # Edit static/ directory with your content")
	}

	return nil
}
