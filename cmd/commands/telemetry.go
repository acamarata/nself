package commands

// telemetry.go — `nself telemetry` command group (S54.T05)
//
// Ships a preference-management command at v1.0.9 so users can opt out NOW
// before the v1.1.0 telemetry client ships. No telemetry data is collected or
// transmitted in v1.0.9.
//
// Commands:
//   nself telemetry status   — show current opt-out state
//   nself telemetry on       — enable telemetry (writes to ~/.nself/config.toml)
//   nself telemetry off      — disable telemetry (writes to ~/.nself/config.toml)
//
// Environment override:
//   NSELF_TELEMETRY_OPT_OUT=1 always overrides the config file.
//
// Privacy policy: https://nself.org/legal/privacy
// v1.1.0 roadmap: PRIVACY.md § Future telemetry

import (
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Manage CLI telemetry preferences",
	Long: `Manage CLI telemetry opt-in/out preferences.

v1.0.9 ships no telemetry client. This command preserves your preference
for v1.1.0+ when an opt-in telemetry client is introduced.

The NSELF_TELEMETRY_OPT_OUT=1 environment variable always overrides the
stored preference.

Privacy policy: https://nself.org/legal/privacy`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var telemetryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current telemetry opt-out state",
	Long: `Print whether telemetry is currently enabled or disabled.

Sources checked in priority order:
  1. NSELF_TELEMETRY_OPT_OUT=1 environment variable (highest priority)
  2. ~/.nself/config.toml [telemetry] enabled = <bool>
  3. Default: enabled (no client ships until v1.1.0)

Note: v1.0.9 ships no telemetry client. Your preference is stored and will
be respected when v1.1.0 ships.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pref := config.GetTelemetryPreference()

		state := "enabled"
		if !pref.Enabled {
			state = "disabled (opt-out)"
		}

		fmt.Printf("Telemetry: %s\n", state)
		fmt.Printf("Source:    %s\n", pref.Source)

		if pref.Source == "env" {
			fmt.Println()
			fmt.Println("Note: NSELF_TELEMETRY_OPT_OUT=1 is set. This overrides any stored preference.")
		}

		fmt.Println()
		fmt.Println("v1.0.9: no telemetry client ships. This preference will take effect in v1.1.0.")
		fmt.Println("Privacy policy: https://nself.org/legal/privacy")
		return nil
	},
}

var telemetryOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Opt out of telemetry (writes to ~/.nself/config.toml)",
	Long: `Disable telemetry by writing enabled = false to ~/.nself/config.toml.

This preference is preserved for v1.1.0+. The NSELF_TELEMETRY_OPT_OUT=1
environment variable also achieves opt-out without modifying the config file.

v1.0.9 ships no telemetry client, so this command has no immediate effect
on data collection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Env opt-out already covers this; still persist to config for clarity.
		if os.Getenv("NSELF_TELEMETRY_OPT_OUT") == "1" {
			fmt.Println("NSELF_TELEMETRY_OPT_OUT=1 is already set. Writing preference to config too.")
		}
		if err := config.SetTelemetryEnabled(false); err != nil {
			return fmt.Errorf("saving telemetry preference: %w", err)
		}
		ui.Success("Telemetry disabled. Preference saved to ~/.nself/config.toml")
		fmt.Println("v1.0.9: no telemetry client ships. Your preference will take effect in v1.1.0.")
		return nil
	},
}

var telemetryOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Opt in to telemetry (writes to ~/.nself/config.toml)",
	Long: `Enable telemetry by writing enabled = true to ~/.nself/config.toml.

If NSELF_TELEMETRY_OPT_OUT=1 is set in the environment, it takes precedence
over this setting. Remove or unset the env var to fully re-enable.

v1.0.9 ships no telemetry client, so this command has no immediate effect
on data collection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("NSELF_TELEMETRY_OPT_OUT") == "1" {
			fmt.Fprintln(os.Stderr, "Warning: NSELF_TELEMETRY_OPT_OUT=1 is set in the environment.")
			fmt.Fprintln(os.Stderr, "Telemetry will remain disabled until that variable is removed.")
			fmt.Fprintln(os.Stderr, "Writing the preference to config anyway.")
		}
		if err := config.SetTelemetryEnabled(true); err != nil {
			return fmt.Errorf("saving telemetry preference: %w", err)
		}
		ui.Success("Telemetry enabled. Preference saved to ~/.nself/config.toml")
		fmt.Println("v1.0.9: no telemetry client ships. Your preference will take effect in v1.1.0.")
		return nil
	},
}

func init() {
	telemetryCmd.AddCommand(telemetryStatusCmd)
	telemetryCmd.AddCommand(telemetryOffCmd)
	telemetryCmd.AddCommand(telemetryOnCmd)
	RootCmd.AddCommand(telemetryCmd)
}
