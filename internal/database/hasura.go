// Package database provides database and Hasura metadata operations.
package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/config"
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

	resp, err := http.DefaultClient.Do(req)
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

// HasuraApplyMetadata reads project metadata from projectDir/hasura/metadata.json
// and sends it to Hasura as a replace_metadata request.
// If no metadata file exists, it falls back to reload_metadata (safe/idempotent).
func HasuraApplyMetadata(ctx context.Context, cfg *config.Config, projectDir string) error {
	metadataFile := filepath.Join(projectDir, "hasura", "metadata.json")
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No metadata file — use reload_metadata (safe, idempotent)
			_, reloadErr := postMetadata(ctx, cfg, metadataRequest{
				Type: "reload_metadata",
				Args: struct{}{},
			})
			if reloadErr != nil {
				return fmt.Errorf("reload metadata (no metadata.json found): %w", reloadErr)
			}
			return nil
		}
		return fmt.Errorf("read metadata file: %w", err)
	}

	// Parse the metadata JSON
	var metadata json.RawMessage
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("parse metadata file: %w", err)
	}

	_, err = postMetadata(ctx, cfg, metadataRequest{
		Type: "replace_metadata",
		Args: metadata,
	})
	if err != nil {
		return fmt.Errorf("apply metadata: %w", err)
	}
	return nil
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
