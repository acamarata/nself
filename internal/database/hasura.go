// Package database provides database and Hasura metadata operations.
package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/httptimeout"
)

// metadataRequest is the JSON body sent to the Hasura metadata API.
type metadataRequest struct {
	Type string      `json:"type"`
	Args interface{} `json:"args,omitempty"`
}

// hasuraMetadataURL returns the Hasura v1/metadata endpoint URL.
func hasuraMetadataURL(cfg *config.Config) string {
	port := cfg.Hasura.Port
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf("http://localhost:%d/v1/metadata", port)
}

// postMetadata sends a JSON request to the Hasura metadata API and returns
// the raw response body. It handles auth headers and HTTP error status codes.
func postMetadata(ctx context.Context, cfg *config.Config, payload metadataRequest) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hasuraMetadataURL(cfg), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create metadata request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", cfg.Hasura.AdminSecret)

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hasura metadata request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read metadata response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hasura metadata API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HasuraExportMetadata retrieves the current metadata from Hasura as raw JSON bytes.
func HasuraExportMetadata(ctx context.Context, cfg *config.Config) ([]byte, error) {
	respBody, err := postMetadata(ctx, cfg, metadataRequest{
		Type: "export_metadata",
		Args: struct{}{},
	})
	if err != nil {
		return nil, fmt.Errorf("export metadata: %w", err)
	}

	return respBody, nil
}

// HasuraReloadMetadata tells Hasura to reload its metadata from the internal catalog,
// picking up any changes to tracked tables, relationships, or permissions.
func HasuraReloadMetadata(ctx context.Context, cfg *config.Config) error {
	_, err := postMetadata(ctx, cfg, metadataRequest{
		Type: "reload_metadata",
		Args: struct{}{},
	})
	if err != nil {
		return fmt.Errorf("reload metadata: %w", err)
	}

	return nil
}
