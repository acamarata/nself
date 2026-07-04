package commands

// sentry_status.go — 'nself sentry status' command.
//
// Purpose: Fetch and display the ɳSentry status page JSON, report component
//   health, and exit non-zero when any component is in outage/degraded state.
// Inputs:  --url, --token, --json, --timeout flags
// Outputs: human summary to stdout (or JSON when --json); errors to stderr.
// Constraints: token appended as ?t=<token> query param; exit 0 = fully
//   operational, exit 1 = any outage/degraded/unreachable condition.
// SPORT: CLI-CMD-SENTRY-002

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// statusPageResponse is the normalized shape returned by the status JSON endpoint.
type statusPageResponse struct {
	SiteName      string            `json:"site_name"`
	OverallStatus string            `json:"overall_status"`
	Components    []statusComponent `json:"components"`
}

// statusComponent represents a single monitored component.
type statusComponent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

// statusJSONOut is the normalized JSON output shape for --json mode.
type statusJSONOut struct {
	Overall     string             `json:"overall"`
	Total       int                `json:"total"`
	Operational int                `json:"operational"`
	Degraded    int                `json:"degraded"`
	Down        []string           `json:"down"`
	Components  []compactComponent `json:"components"`
}

// compactComponent is the per-component shape inside statusJSONOut.
type compactComponent struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// outageStatuses are component statuses that trigger exit 1.
var outageStatuses = map[string]bool{
	"partial_outage": true,
	"major_outage":   true,
}

// degradedStatuses count toward the degraded tally but do not alone trigger exit 1.
var degradedStatuses = map[string]bool{
	"degraded_performance": true,
	"under_maintenance":    true,
}

var sentryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Fetch and display the ɳSentry status page",
	Long: `Fetch the ɳSentry status page JSON and display component health.

Exits 0 when overall_status is operational and no component is in
partial_outage or major_outage. Exits 1 otherwise.

Examples:
  nself sentry status
  nself sentry status --json
  nself sentry status --url https://status.nself.org/status.json --timeout 15s
  nself sentry status --token mytoken --json`,
	RunE: runSentryStatus,
}

func init() {
	sentryStatusCmd.Flags().String("url", "https://status.nself.org/status.json", "Status JSON endpoint URL")
	sentryStatusCmd.Flags().String("token", "", "Auth token (env: NSENTRY_PRIVATE_TOKEN or STATUS_PAGE_PRIVATE_TOKEN)")
	sentryStatusCmd.Flags().Bool("json", false, "Output normalized JSON to stdout")
	sentryStatusCmd.Flags().Duration("timeout", 10*time.Second, "HTTP request timeout")

	sentryCmd.AddCommand(sentryStatusCmd)
}

// runSentryStatus implements 'nself sentry status'.
//
// Purpose:  GET the status JSON endpoint, parse component health, display
//
//	human or JSON output, and exit non-zero on any outage/unreachable state.
//
// Inputs:   --url, --token, --json, --timeout flags (cmd).
// Outputs:  Human table to stdout, or normalized JSON when --json; errors to stderr.
func runSentryStatus(cmd *cobra.Command, _ []string) error {
	rawURL, _ := cmd.Flags().GetString("url")
	token, _ := cmd.Flags().GetString("token")
	jsonOut, _ := cmd.Flags().GetBool("json")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// Resolve token: flag → NSENTRY_PRIVATE_TOKEN → STATUS_PAGE_PRIVATE_TOKEN.
	if token == "" {
		token = os.Getenv("NSENTRY_PRIVATE_TOKEN")
	}
	if token == "" {
		token = os.Getenv("STATUS_PAGE_PRIVATE_TOKEN")
	}

	// Append token as query param when provided.
	if token != "" {
		u, err := url.Parse(rawURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --url %q: %v\n", rawURL, err)
			os.Exit(1)
		}
		q := u.Query()
		q.Set("t", token)
		u.RawQuery = q.Encode()
		rawURL = u.String()
	}

	// Build HTTP client with the requested timeout.
	client := httptimeout.WithTimeout(timeout)

	page, err := fetchStatusPage(cmd.Context(), client, rawURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", ui.C(ui.Red, ui.IconFailure), err)
		os.Exit(1)
	}

	// Compute aggregates.
	totalComponents := len(page.Components)
	operationalCount := 0
	degradedCount := 0
	var downNames []string
	var compacts []compactComponent

	for _, c := range page.Components {
		compacts = append(compacts, compactComponent{Name: c.Name, Status: c.Status})
		if c.Status == "operational" {
			operationalCount++
		} else if degradedStatuses[c.Status] {
			degradedCount++
		} else if outageStatuses[c.Status] {
			degradedCount++
			downNames = append(downNames, c.Name)
		}
	}

	// Determine exit code: 1 if not operational overall OR any component is in outage.
	exitBad := page.OverallStatus != "operational" || len(downNames) > 0

	if jsonOut {
		out := statusJSONOut{
			Overall:     page.OverallStatus,
			Total:       totalComponents,
			Operational: operationalCount,
			Degraded:    degradedCount,
			Down:        downNames,
			Components:  compacts,
		}
		if out.Down == nil {
			out.Down = []string{}
		}
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		if exitBad {
			os.Exit(1)
		}
		return nil
	}

	// Human output.
	overallColor := ui.Green
	overallIcon := ui.IconSuccess
	if exitBad {
		overallColor = ui.Red
		overallIcon = ui.IconFailure
	} else if degradedCount > 0 {
		overallColor = ui.Yellow
		overallIcon = ui.IconWarning
	}

	ui.CommandHeader("ɳSentry Status", page.SiteName)
	fmt.Printf("\nStatus: %s\n\n", ui.C(overallColor, overallIcon+" "+page.OverallStatus))

	notOKCount := 0
	for _, c := range page.Components {
		prefix := "   "
		color := ui.Dim
		if outageStatuses[c.Status] || degradedStatuses[c.Status] {
			prefix = ui.C(ui.Yellow, "[!]")
			color = ui.Yellow
			if outageStatuses[c.Status] {
				color = ui.Red
				prefix = ui.C(ui.Red, "[!]")
			}
			notOKCount++
		}
		fmt.Printf("%s %s — %s\n", prefix, c.Name, ui.C(color, c.Status))
	}

	notOKLabel := "all operational"
	if notOKCount > 0 {
		notOKLabel = fmt.Sprintf("%d not operational", notOKCount)
	}
	fmt.Printf("\n%d components, %s\n", totalComponents, notOKLabel)

	if exitBad {
		os.Exit(1)
	}
	return nil
}

// fetchStatusPage performs the GET request and decodes the response JSON.
//
// Purpose:  Execute HTTP GET with context, enforce non-200 as error, decode JSON.
// Inputs:   ctx (cancellation), client (timeout-bounded), rawURL (fully formed).
// Outputs:  Parsed statusPageResponse or an error.
func fetchStatusPage(ctx context.Context, client *http.Client, rawURL string) (*statusPageResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching status page: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status page returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var page statusPageResponse
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing status JSON: %w", err)
	}
	return &page, nil
}
