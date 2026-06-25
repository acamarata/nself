package gateway

// Purpose: Route listing from nself-ai-gateway /v1/routes.
// Inputs: JWT token.
// Outputs: RouteRow slice for table rendering.
// Constraints: None.
// SPORT: F02 — nself gateway routes.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RouteRow represents one routing rule in the gateway.
type RouteRow struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Target   string `json:"target"`
	Active   bool   `json:"active"`
}

// ListRoutes fetches registered routing rules from nself-ai-gateway.
func ListRoutes(ctx context.Context, token string) ([]RouteRow, error) {
	resp, err := gatewayRequest(ctx, http.MethodGet, "/v1/routes", token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s\n\nExit: 1", resp.StatusCode, string(raw))
	}

	var rows []RouteRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return rows, nil
}
