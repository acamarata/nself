package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/license"

	"github.com/spf13/cobra"
)

func init() {
	f := licenseTailCmd.Flags()
	f.StringArray("filter", nil, "Filter as field=value (e.g. --filter result=denied --filter key=nself_pro_xxx)")
	f.String("ping-url", "", "ping_api base URL (overrides NSELF_PING_URL)")

	licenseCmd.AddCommand(licenseTailCmd)
}

var licenseTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Stream live license validation events from ping_api",
	Long: `Stream live license validation events from ping.nself.org in real time.

Events are color-coded:
  green   — allow
  red     — deny
  yellow  — rate_limit

Use --filter to narrow the stream:
  --filter result=denied      Show only denied validations
  --filter key=nself_pro_xxx  Show only events for a specific key (prefix match)
  --filter plugin=ai          Show only events for a specific plugin

Press Ctrl-C to exit cleanly. The command reconnects automatically on
connection loss (exponential backoff up to 30s).

License key values are never shown in full — only the first 12 characters
(key prefix) are displayed.

Examples:
  nself license tail
  nself license tail --filter result=denied
  nself license tail --filter key=nself_pro_abc --filter plugin=ai`,
	RunE: runLicenseTail,
}

func runLicenseTail(cmd *cobra.Command, _ []string) error {
	filterArgs, _ := cmd.Flags().GetStringArray("filter")
	pingURL, _ := cmd.Flags().GetString("ping-url")

	filters := make(map[string]string)
	for _, f := range filterArgs {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid filter format %q: expected field=value", f)
		}
		filters[parts[0]] = parts[1]
	}

	return license.TailStream(cmd.Context(), license.TailOptions{
		Filters: filters,
		PingURL: pingURL,
		Stdout:  os.Stdout,
	})
}
