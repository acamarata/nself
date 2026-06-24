// Purpose: ListRoutes fetches active routing rules from nself-ai-gateway (port 3761).
//
// Inputs:  none
// Outputs: []RouteEntry; error with nSelf UX hint on failure
// Constraints: read-only; does not mutate routing state
// SPORT: F02-COMMAND-INVENTORY.md (nself gateway routes)
package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nself-org/cli/internal/ports"
)

// RouteEntry mirrors one row in np_aigateway_routes returned by GET /v1/routes.
type RouteEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// ListRoutes calls GET /v1/routes and returns all configured routing rules.
func ListRoutes() ([]RouteEntry, error) {
	client, baseURL, err := gatewayHTTPClient()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", baseURL+"/v1/routes", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nself-ai-gateway not running (port %d unreachable)\nHint: run `nself start` or `nself gateway status` to diagnose\nExit: 3", ports.AIGatewayPort)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Routes []RouteEntry `json:"routes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return result.Routes, nil
}
