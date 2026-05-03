// Package backup — Hasura metadata export.
//
// BackupHasuraMetadata calls the Hasura v1/metadata export_metadata API and
// writes the result to <backupDir>/hasura-metadata-<YYYY-MM-DD>.json.
// The file is mode 0600.
package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// HasuraMetadataOptions holds parameters for BackupHasuraMetadata.
type HasuraMetadataOptions struct {
	// HasuraURL is the base URL of the Hasura instance, e.g.
	// "http://localhost:8080". Defaults to "http://localhost:8080".
	HasuraURL string
	// AdminSecret is the Hasura admin secret used for authentication.
	AdminSecret string
	// BackupDir is the directory where the JSON file will be written.
	// Defaults to "./backups".
	BackupDir string
}

// BackupHasuraMetadata exports Hasura metadata and saves it to BackupDir.
// Returns the path of the written file.
func BackupHasuraMetadata(ctx context.Context, opts HasuraMetadataOptions) (string, error) {
	hasuraURL := opts.HasuraURL
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080"
	}

	backupDir := opts.BackupDir
	if backupDir == "" {
		backupDir = "./backups"
	}

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}

	data, err := exportMetadata(ctx, hasuraURL, opts.AdminSecret)
	if err != nil {
		return "", fmt.Errorf("export hasura metadata: %w", err)
	}

	date := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("hasura-metadata-%s.json", date)
	dest := filepath.Join(backupDir, filename)

	// Pretty-print so diffs are human-readable.
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err == nil {
		data = pretty.Bytes()
	}

	if err := os.WriteFile(dest, data, 0600); err != nil {
		return "", fmt.Errorf("write hasura metadata backup: %w", err)
	}

	return dest, nil
}

// exportMetadata performs the HTTP POST to /v1/metadata export_metadata.
func exportMetadata(ctx context.Context, hasuraURL, adminSecret string) ([]byte, error) {
	payload := map[string]interface{}{
		"type": "export_metadata",
		"args": struct{}{},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := hasuraURL + "/v1/metadata"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if adminSecret != "" {
		req.Header.Set("X-Hasura-Admin-Secret", adminSecret)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hasura returned %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
