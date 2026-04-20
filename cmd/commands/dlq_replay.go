package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/dlq"

	"github.com/spf13/cobra"
)

var dlqCmd = &cobra.Command{
	Use:   "dlq",
	Short: "Manage dead-letter queues for nSelf plugins",
	Long: `Manage dead-letter queues (DLQ) for nSelf plugins.

Dead-letter queues accumulate rows that failed processing. Use 'nself dlq replay'
to re-enqueue rows after fixing the upstream bug that caused the failures.

Subcommands:
  replay   Re-enqueue DLQ rows for a plugin back to the work queue`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var dlqReplayCmd = &cobra.Command{
	Use:   "replay <plugin>",
	Short: "Re-enqueue DLQ rows for a plugin",
	Long: `Re-enqueue dead-letter queue rows for the named plugin back to the work queue.

WARNING: Only replay after the upstream bug causing the failures is fixed.
         Replaying before fixing the bug causes rows to re-DLQ immediately.

Safe defaults:
  --max-rows 100  Prevents accidental DLQ floods
  --dry-run       Preview without executing

Failures are reported per-row; the command exits non-zero if any row fails.
Operator-level authentication is required (set NSELF_API_TOKEN or NSELF_API_URL).

Examples:
  nself dlq replay mux
  nself dlq replay mux --dry-run
  nself dlq replay mux --max-rows 50
  nself dlq replay mux --filter status=quarantined
  nself dlq replay mux --filter status=quarantined --max-rows 10`,
	Args: cobra.ExactArgs(1),
	RunE: runDLQReplay,
}

func init() {
	f := dlqReplayCmd.Flags()
	f.Int("max-rows", 100, "Maximum rows to replay (prevents floods)")
	f.StringArray("filter", nil, "Filter as field=value (e.g. --filter status=quarantined)")
	f.Bool("dry-run", false, "Preview replay without executing")
	f.String("base-url", "", "nSelf API base URL (overrides NSELF_API_URL)")

	dlqCmd.AddCommand(dlqReplayCmd)
	RootCmd.AddCommand(dlqCmd)
}

func runDLQReplay(cmd *cobra.Command, args []string) error {
	plugin := args[0]
	maxRows, _ := cmd.Flags().GetInt("max-rows")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	filterArgs, _ := cmd.Flags().GetStringArray("filter")
	baseURL, _ := cmd.Flags().GetString("base-url")

	// Parse filter flags into map.
	filter := make(map[string]string)
	for _, f := range filterArgs {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid filter format %q: expected field=value", f)
		}
		filter[parts[0]] = parts[1]
	}

	_, err := dlq.Replay(cmd.Context(), dlq.ReplayOptions{
		Plugin:  plugin,
		MaxRows: maxRows,
		Filter:  filter,
		DryRun:  dryRun,
		BaseURL: baseURL,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	return err
}
