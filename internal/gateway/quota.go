package gateway

// Purpose: Quota display from nself-ai-gateway /v1/quota.
// Inputs: Optional provider and model filters; JWT token.
// Outputs: QuotaRow slice for table rendering.
// Constraints: None.
// SPORT: F02 — nself gateway quota.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// QuotaRow represents one quota usage row.
type QuotaRow struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Used     int64  `json:"used"`
	Limit    int64  `json:"limit"`
	ResetAt  string `json:"reset_at"`
}

// GetQuota fetches quota usage from nself-ai-gateway.
// provider and model are optional filters (empty string = no filter).
func GetQuota(ctx context.Context, token, provider, model string) ([]QuotaRow, error) {
	path := "/v1/quota"
	sep := "?"
	if provider != "" {
		path += sep + "provider=" + provider
		sep = "&"
	}
	if model != "" {
		path += sep + "model=" + model
	}

	resp, err := gatewayRequest(ctx, http.MethodGet, path, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s\n\nExit: 1", resp.StatusCode, string(raw))
	}

	var rows []QuotaRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return rows, nil
}
