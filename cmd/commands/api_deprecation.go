package commands

// Purpose: the "nself api deprecation-check" subcommand. Inputs are the
// cobra command/args; outputs are printed deprecation findings or an error.
// Constraints: split out of api.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// apiDeprecationCheckCmd checks installed plugins for deprecated API usage (G6).
// --plugin <name> scopes the check to one plugin.
// --strict exits 1 if any BREAKING (no grace period) entry is found (G11 CI gate).
var apiDeprecationCheckCmd = &cobra.Command{
	Use:   "deprecation-check",
	Short: "Check for deprecated API usage in this install",
	Long: `Walk the plugin deprecation registry to find deprecated endpoints.

Cross-references internal/deprecation/registry.yaml. Use --plugin <name> to
check a specific plugin. Use --strict to fail CI when a BREAKING entry is found.

At v1.0.9 LTS baseline, the registry has no active endpoint deprecations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		pluginFilter, _ := cmd.Flags().GetString("plugin")
		strict, _ := cmd.Flags().GetBool("strict")

		items := scanDeprecations(pluginFilter)

		if len(items) == 0 {
			label := "install"
			if pluginFilter != "" {
				label = fmt.Sprintf("plugin '%s'", pluginFilter)
			}
			if jsonOut {
				out, _ := json.Marshal(map[string]interface{}{
					"deprecations_found": 0,
					"plugin_filter":      pluginFilter,
					"registry_version":   "v1.0.9-baseline",
					"status":             "clean",
				})
				fmt.Println(string(out))
			} else {
				fmt.Printf("0 deprecations found. Your %s is clean against the v1.0.9 LTS baseline.\n\n", label)
				fmt.Println("  Registry: internal/deprecation/registry.yaml")
				fmt.Println("  LTS window: 2026-04-17 → 2027-04-17")
				fmt.Println()
			}
			return nil
		}

		// Detect BREAKING entries — those with no deprecated_in grace period.
		hasBreaking := false
		for _, d := range items {
			if d["deprecated_in"] == "" {
				hasBreaking = true
				break
			}
		}

		if jsonOut {
			status := "warnings"
			if hasBreaking {
				status = "BREAKING"
			}
			out, _ := json.Marshal(map[string]interface{}{
				"deprecations_found": len(items),
				"status":             status,
				"items":              items,
			})
			fmt.Println(string(out))
		} else {
			fmt.Printf("Found %d deprecated API usage(s):\n\n", len(items))
			for _, d := range items {
				tag := "DEPRECATED"
				if d["deprecated_in"] == "" {
					tag = "BREAKING"
				}
				fmt.Printf("  [%s] %s: %s (deprecated in v%s, removed in v%s)\n",
					tag, d["plugin"], d["path"], d["deprecated_in"], d["removed_in"])
				if d["replacement"] != "" {
					fmt.Printf("    Replacement: %s\n", d["replacement"])
				}
				if d["sunset_header"] != "" {
					fmt.Printf("    Sunset: %s\n", d["sunset_header"])
				}
			}
			fmt.Println()
		}

		if strict && hasBreaking {
			return fmt.Errorf("BREAKING: %d endpoint(s) lack a deprecation grace period — add 'deprecated_in' before merging",
				countBreaking(items))
		}
		return nil
	},
}
