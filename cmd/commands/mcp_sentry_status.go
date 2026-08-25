package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Reading an ɳSentry status page, for the MCP tool that exposes it.
//
// Purpose: `nself mcp` publishes an ɳSentry status tool, and mcp is a core
// command. The `sentry` command family moved to the sentry plugin under
// CLI-R11 and took this helper with it, so the core kept the part mcp calls.
//
// Inputs: a status page URL.
//
// Outputs: the parsed page.
//
// Constraints: this duplicates code the sentry plugin also carries, which is
// the smaller of two bad options. The alternative is either dragging mcp into
// the plugin — it is core, and covers far more than sentry — or having the CLI
// depend on a plugin being installed before `nself mcp` can start. Plugins
// cannot contribute MCP tools; when they can, this should go and the tool
// should come from the plugin that owns it.

// statusPageResponse is the shape an ɳSentry status page serves.
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

// fetchStatusPage retrieves and parses a status page.
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
