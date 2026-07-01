package commands

// sentry_cloud.go — shared plumbing for the 'nself sentry' cloud subcommands.
//
// Purpose: One place for the --api-url/--api-key/--json flag trio, client
//   construction (flag → env → ~/.nself/sentry.json → default), and JSON output.
// Inputs:  cobra flags + NSELF_SENTRY_API_URL / NSELF_SENTRY_API_KEY env vars.
// Outputs: configured *sentryapi.Client; JSON printing helper.
// Constraints: parent sentryCmd carries no flags (see sentry.go) — every cloud
//   subcommand registers this trio itself via addSentryCloudFlags.
// SPORT: CLI-CMD-SENTRY-003

import (
	"encoding/json"
	"fmt"

	"github.com/nself-org/cli/internal/sentryapi"
	"github.com/spf13/cobra"
)

// addSentryCloudFlags registers the standard cloud-API flag trio on a subcommand.
func addSentryCloudFlags(cmd *cobra.Command) {
	cmd.Flags().String("api-url", "", "ɳSentry API base URL (env: "+sentryapi.EnvAPIURL+"; default: "+sentryapi.DefaultAPIURL+")")
	cmd.Flags().String("api-key", "", "ɳSentry API key nsk_* (env: "+sentryapi.EnvAPIKey+"; default: ~/.nself/sentry.json)")
	cmd.Flags().Bool("json", false, "Output JSON (AI/script-friendly)")
}

// sentryClientFromCmd builds a sentryapi.Client from the resolved config.
func sentryClientFromCmd(cmd *cobra.Command) *sentryapi.Client {
	flagURL, _ := cmd.Flags().GetString("api-url")
	flagKey, _ := cmd.Flags().GetString("api-key")
	apiURL, apiKey := sentryapi.Resolve(flagURL, flagKey)
	return sentryapi.New(apiURL, apiKey)
}

// sentryJSONFlag reads the --json flag.
func sentryJSONFlag(cmd *cobra.Command) bool {
	j, _ := cmd.Flags().GetBool("json")
	return j
}

// printSentryJSON marshals v as indented JSON to stdout.
func printSentryJSON(cmd *cobra.Command, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json output: %w", err)
	}
	cmd.Println(string(data))
	return nil
}

// dashIfEmpty returns "-" for empty table cells.
func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
