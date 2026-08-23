package commands

// Purpose: `nself plugin outdated` — compares each installed plugin's version
//          against the live registry and reports which ones have a newer
//          version available (CLI-R16).
// Inputs:  cobra flags (--json); the local plugin directory; network access
//          to the plugin registry (same fallback chain as `plugin list`).
// Outputs: a table (or JSON array) of {name, installed, latest} to stdout.
// Constraints: Never calls os.Exit — signals the "something is outdated"
//              case via errs.Exit(1) so main() controls the process exit
//              status (internal/repoqa/os_exit_test.go enforces this).
//              Plugins not present in the registry (e.g. a linked/local dev
//              plugin, or one installed from a third-party URL) are skipped
//              rather than reported as outdated — there is nothing to compare
//              them against.
// SPORT: F02-COMMAND-INVENTORY.md (plugin subcommands); callers: none (leaf command)

import (
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var pluginOutdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "List installed plugins with a newer version available",
	Long: `Compares each installed plugin's version against the registry and reports
any that are behind. Exits 0 when every installed plugin is current, and
exits 1 when at least one is outdated (so it composes in CI/scripts).`,
	RunE: runPluginOutdated,
}

func init() {
	pluginOutdatedCmd.Flags().Bool("json", false, "Emit results as a JSON array")
	pluginCmd.AddCommand(pluginOutdatedCmd)
}

// outdatedPluginRow is one entry in `nself plugin outdated`'s output.
type outdatedPluginRow struct {
	Name      string `json:"name"`
	Installed string `json:"installed"`
	Latest    string `json:"latest"`
}

func runPluginOutdated(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	registryURL := os.Getenv("NSELF_PLUGIN_REGISTRY")
	if err := plugin.ValidateNetworkAccess(cmd.Context(), registryURL); err != nil {
		return err
	}

	pluginDir := resolvePluginDir()

	installed, err := plugin.List(pluginDir, true)
	if err != nil {
		return fmt.Errorf("listing installed plugins: %w", err)
	}
	if len(installed) == 0 {
		if jsonOut {
			return ui.PrintJSON([]outdatedPluginRow{})
		}
		fmt.Println("No plugins installed.")
		return nil
	}

	registry, err := plugin.List(pluginDir, false)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}
	latestVersions := make(map[string]string, len(registry))
	for _, p := range registry {
		latestVersions[p.Name] = p.Version
	}

	var rows []outdatedPluginRow
	for _, p := range installed {
		latest, found := latestVersions[p.Name]
		if !found {
			// Not in the registry — a linked/local dev plugin or a
			// third-party URL install. Nothing to compare it against.
			continue
		}
		if plugin.CompareVersions(p.Version, latest) < 0 {
			rows = append(rows, outdatedPluginRow{Name: p.Name, Installed: p.Version, Latest: latest})
		}
	}

	if jsonOut {
		if err := ui.PrintJSON(rows); err != nil {
			return err
		}
	} else if len(rows) == 0 {
		fmt.Println("All plugins are up to date.")
	} else {
		tbl := ui.NewTable("Name", "Installed", "Latest")
		for _, r := range rows {
			tbl.AddRow(r.Name, r.Installed, r.Latest)
		}
		tbl.Render()
	}

	if len(rows) > 0 {
		// Output already written above — Exit(nil-err) is silent by design
		// (internal/errs.ExitError.Silent), so main() prints nothing further.
		return errs.Exit(1)
	}
	return nil
}
