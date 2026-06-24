// Purpose: GetQuota retrieves per-provider/model daily usage from nself-ai-gateway (port 3761).
//
// Inputs:  optional provider and model filter strings (empty = return all)
// Outputs: []QuotaEntry with token/request counts; error with nSelf UX hint on failure
// Constraints: read-only; does not mutate quota state
// SPORT: F02-COMMAND-INVENTORY.md (nself gateway quota)
package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/nself-org/cli/internal/ports"
)

// QuotaEntry represents one row in the per-user/per-provider/per-model daily quota.
type QuotaEntry struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Date         string `json:"date"`
	TokensIn     int    `json:"tokens_in"`
	TokensOut    int    `json:"tokens_out"`
	RequestCount int    `json:"request_count"`
	DailyLimit   *int   `json:"daily_token_limit,omitempty"` // nil = no limit configured
}

// GetQuota calls GET /v1/quota with optional provider and model filters.
// Returns all quota rows if both filters are empty.
func GetQuota(provider, model string) ([]QuotaEntry, error) {
	client, baseURL, err := gatewayHTTPClient()
	if err != nil {
		return nil, err
	}

	reqURL, err := url.Parse(baseURL + "/v1/quota")
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	q := reqURL.Query()
	if provider != "" {
		q.Set("provider", provider)
	}
	if model != "" {
		q.Set("model", model)
	}
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", reqURL.String(), nil)
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
		Quota []QuotaEntry `json:"quota"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return result.Quota, nil
}
