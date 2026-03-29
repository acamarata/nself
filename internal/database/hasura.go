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
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

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

// HasuraApplyMetadata applies Hasura metadata from projectDir.
//
// It supports two metadata formats:
//
//  1. YAML directory format (standard Hasura CLI): projectDir/hasura/metadata/
//     Tables use !include directives referencing individual YAML files.
//     This format is detected first. If the hasura CLI binary is in PATH it is
//     used directly (handles all YAML features). Otherwise nSelf resolves
//     !include directives in Go and sends the result to the metadata API.
//
//  2. JSON format (legacy nSelf): projectDir/hasura/metadata.json
//     Sent directly to the replace_metadata API.
//
// If neither exists, reload_metadata is called (safe/idempotent).
func HasuraApplyMetadata(ctx context.Context, cfg *config.Config, projectDir string) error {
	hasuraDir := filepath.Join(projectDir, "hasura")
	metadataDir := filepath.Join(hasuraDir, "metadata")
	metadataFile := filepath.Join(hasuraDir, "metadata.json")

	// --- Format 1: YAML directory with !include directives ---
	if info, err := os.Stat(metadataDir); err == nil && info.IsDir() {
		return applyYAMLMetadata(ctx, cfg, hasuraDir, metadataDir)
	}

	// --- Format 2: legacy JSON file ---
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			_, reloadErr := postMetadata(ctx, cfg, metadataRequest{
				Type: "reload_metadata",
				Args: struct{}{},
			})
			if reloadErr != nil {
				return fmt.Errorf("reload metadata (no metadata found): %w", reloadErr)
			}
			return nil
		}
		return fmt.Errorf("read metadata file: %w", err)
	}

	var metadata json.RawMessage
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("parse metadata file: %w", err)
	}
	_, err = postMetadata(ctx, cfg, metadataRequest{
		Type: "replace_metadata",
		Args: metadata,
	})
	return err
}

// applyYAMLMetadata handles the Hasura CLI YAML metadata directory format.
// It prefers shelling out to the hasura CLI binary (which natively handles
// !include), falling back to Go-based !include resolution.
func applyYAMLMetadata(ctx context.Context, cfg *config.Config, hasuraDir, metadataDir string) error {
	if hasuraBin, err := exec.LookPath("hasura"); err == nil {
		return applyViaHasuraCLI(ctx, cfg, hasuraBin, hasuraDir)
	}
	return applyViaIncludeResolver(ctx, cfg, metadataDir)
}

// applyViaHasuraCLI shells out to the hasura CLI binary. The binary handles
// !include resolution and all YAML features natively.
func applyViaHasuraCLI(ctx context.Context, cfg *config.Config, hasuraBin, hasuraDir string) error {
	port := cfg.Hasura.Port
	if port == 0 {
		port = 8080
	}
	endpoint := fmt.Sprintf("http://localhost:%d", port)

	cmd := exec.CommandContext(ctx, hasuraBin,
		"metadata", "apply",
		"--endpoint", endpoint,
		"--admin-secret", cfg.Hasura.AdminSecret,
		"--project", hasuraDir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hasura metadata apply: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// applyViaIncludeResolver resolves !include directives in Go, assembles the
// full metadata object, and sends it to the Hasura replace_metadata API.
//
// The Hasura CLI YAML metadata format uses a config.yaml at the metadata root
// that !include-s sub-files (tables.yaml, actions.yaml, etc.). Each of those
// can in turn !include individual table/action YAML files.
//
// gopkg.in/yaml.v3 does not natively handle !include. We implement a two-pass
// approach: raw bytes are preprocessed to inline !include references before
// being decoded as YAML, then the result is re-encoded as JSON for the API.
func applyViaIncludeResolver(ctx context.Context, cfg *config.Config, metadataDir string) error {
	configFile := filepath.Join(metadataDir, "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Some projects use tables.yaml directly at the metadata root.
		configFile = filepath.Join(metadataDir, "tables.yaml")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return fmt.Errorf("no config.yaml or tables.yaml found in %s", metadataDir)
		}
	}

	resolved, err := resolveIncludes(metadataDir, configFile)
	if err != nil {
		return fmt.Errorf("resolve !include directives: %w", err)
	}

	// Decode the resolved YAML into a generic structure, then re-encode as JSON.
	var doc interface{}
	if err := yaml.Unmarshal(resolved, &doc); err != nil {
		return fmt.Errorf("parse resolved metadata YAML: %w", err)
	}
	jsonBytes, err := json.Marshal(convertYAMLToJSON(doc))
	if err != nil {
		return fmt.Errorf("encode metadata as JSON: %w", err)
	}

	_, err = postMetadata(ctx, cfg, metadataRequest{
		Type: "replace_metadata",
		Args: json.RawMessage(jsonBytes),
	})
	return err
}

// resolveIncludes reads a YAML file and recursively inlines every
// `!include <filename>` tag, returning the fully expanded bytes.
func resolveIncludes(baseDir, filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	var out strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match:  - !include foo.yaml  or  key: !include foo.yaml
		if idx := strings.Index(trimmed, "!include "); idx != -1 {
			includedFile := strings.TrimSpace(trimmed[idx+len("!include "):])
			includedPath := filepath.Join(baseDir, includedFile)
			includedData, err := resolveIncludes(filepath.Dir(includedPath), includedPath)
			if err != nil {
				return nil, err
			}
			// Preserve the leading indent from the original line, but replace
			// the !include token with the inlined content (indented to match).
			leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
			indent := line[:leadingSpaces]
			// If line starts with "- !include", emit list item prefix + first
			// line of included content, then indent remaining lines.
			prefix := ""
			if strings.HasPrefix(trimmed, "- !include") {
				prefix = "- "
				indent += "  "
			}
			includedLines := strings.Split(strings.TrimRight(string(includedData), "\n"), "\n")
			for i, il := range includedLines {
				if i == 0 {
					out.WriteString(line[:leadingSpaces] + prefix + il + "\n")
				} else {
					out.WriteString(indent + il + "\n")
				}
			}
		} else {
			out.WriteString(line + "\n")
		}
	}

	return []byte(out.String()), nil
}

// convertYAMLToJSON recursively converts a yaml.v3-decoded value (which uses
// map[string]interface{} for mappings and []interface{} for sequences) into a
// JSON-safe structure. yaml.v3 uses map[string]interface{} so this is mostly
// a pass-through, but map keys decoded as interface{} need string conversion.
func convertYAMLToJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, mv := range val {
			out[k] = convertYAMLToJSON(mv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, mv := range val {
			out[fmt.Sprintf("%v", k)] = convertYAMLToJSON(mv)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			out[i] = convertYAMLToJSON(item)
		}
		return out
	default:
		return val
	}
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
