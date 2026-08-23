package commands

// Purpose: the "nself api changelog" subcommand. Inputs are the cobra
// command/args; outputs are a printed API changelog or an error.
// Constraints: split out of api.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/deprecation"

	"github.com/spf13/cobra"
)

// apiChangelogCmd prints the deprecation sunset calendar for a named plugin (G9).
var apiChangelogCmd = &cobra.Command{
	Use:   "changelog <plugin>",
	Short: "Print the deprecation sunset calendar for a plugin",
	Long: `Print a date-sorted list of deprecated endpoints for a plugin, including
sunset dates, replacements, and migration links.

Example:
  nself api changelog ai`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		jsonOut, _ := cmd.Flags().GetBool("json")

		reg, err := deprecation.LoadEmbeddedPluginRegistry()
		if err != nil {
			return fmt.Errorf("loading plugin registry: %w", err)
		}

		entry, ok := reg.LookupPlugin(pluginName)
		if !ok {
			return fmt.Errorf("plugin %q not found in deprecation registry", pluginName)
		}

		calEntries := reg.SunsetDate(pluginName)

		if jsonOut {
			type jsonEntry struct {
				Plugin       string `json:"plugin"`
				APIVersion   string `json:"api_version"`
				Path         string `json:"path"`
				DeprecatedIn string `json:"deprecated_in"`
				RemovedIn    string `json:"removed_in"`
				Replacement  string `json:"replacement"`
				Reason       string `json:"reason"`
				SunsetHeader string `json:"sunset_header"`
			}
			rows := make([]jsonEntry, 0, len(calEntries))
			for _, e := range calEntries {
				rows = append(rows, jsonEntry{
					Plugin:       pluginName,
					APIVersion:   entry.APIVersion,
					Path:         e.Path,
					DeprecatedIn: e.DeprecatedIn,
					RemovedIn:    e.RemovedIn,
					Replacement:  e.Replacement,
					Reason:       e.Reason,
					SunsetHeader: deprecation.HTTPSunsetHeader(e.RemovedIn),
				})
			}
			out, _ := json.Marshal(map[string]interface{}{
				"plugin":      pluginName,
				"api_version": entry.APIVersion,
				"changelog":   rows,
			})
			fmt.Println(string(out))
			return nil
		}

		fmt.Printf("\nAPI Deprecation Calendar — plugin: %s  (current API version: %s)\n\n",
			pluginName, entry.APIVersion)

		if len(calEntries) == 0 {
			fmt.Println("  No deprecated endpoints. All paths are current.")
			fmt.Println()
			return nil
		}

		fmt.Printf("  %-35s %-14s %-12s %s\n", "Path", "Deprecated In", "Removed In", "Replacement")
		fmt.Println("  " + strings.Repeat("-", 82))
		for _, e := range calEntries {
			fmt.Printf("  %-35s %-14s %-12s %s\n", e.Path, e.DeprecatedIn, e.RemovedIn, e.Replacement)
			if e.Reason != "" {
				fmt.Printf("  %s  Reason: %s\n", strings.Repeat(" ", 35), e.Reason)
			}
			sunset := deprecation.HTTPSunsetHeader(e.RemovedIn)
			if sunset != "" {
				fmt.Printf("  %s  Sunset: %s\n", strings.Repeat(" ", 35), sunset)
			}
		}
		fmt.Println()
		return nil
	},
}

// =============================================================================
// Helpers
// =============================================================================
