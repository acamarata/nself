// Package gateway provides CLI client functions for the nself-ai-gateway service (port 3761).
//
// Purpose: Wrap the nself-ai-gateway HTTP API (key vault, quota, routes) so cobra
// commands stay thin. All key material is write-only — never returned or displayed.
//
// Inputs:  nSelf JWT from auth.ReadAuthFile(); provider/label/id arguments from CLI flags
// Outputs: typed result structs or errors with nSelf error-UX format hints
// Constraints: key material (plaintext) must NEVER appear in any return value or log output
// SPORT: F02-COMMAND-INVENTORY.md (keys list/add/remove commands)
package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nself-org/cli/internal/auth"
	"github.com/nself-org/cli/internal/ports"
)

// ValidProviders is the list of supported AI provider identifiers.
var ValidProviders = []string{"anthropic", "openai", "google", "custom"}

// KeyEntry represents one row returned by GET /v1/keys.
// The encrypted_key field is never returned by the API in list responses.
type KeyEntry struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Label     string `json:"label"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

// gatewayHTTPClient returns an http.Client with Bearer auth from the local auth store.
// Returns (client, baseURL, error).
func gatewayHTTPClient() (*http.Client, string, error) {
	af, err := auth.ReadAuthFile()
	if err != nil {
		return nil, "", fmt.Errorf("not logged in — run 'nself login' first: %w", err)
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &bearerTransport{
			token: af.AccessToken,
			base:  http.DefaultTransport,
		},
	}
	return client, ports.AIGatewayBaseURL(), nil
}

// bearerTransport injects a Bearer token into every outgoing request.
type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}

// ListKeys calls GET /v1/keys and returns the list of provider key entries.
// Key material is never included in the response — only metadata fields.
func ListKeys() ([]KeyEntry, error) {
	client, baseURL, err := gatewayHTTPClient()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", baseURL+"/v1/keys", nil)
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
		Keys []KeyEntry `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return result.Keys, nil
}

// AddKeyRequest is the POST /v1/keys request body.
type AddKeyRequest struct {
	Provider string `json:"provider"`
	Key      string `json:"key"` // plaintext; encrypted server-side; never logged
	Label    string `json:"label,omitempty"`
}

// AddKeyResponse is the POST /v1/keys response body.
type AddKeyResponse struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Label     string `json:"label"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}

// AddKey calls POST /v1/keys with the given provider, plaintext key, and optional label.
// The plaintext key is sent over TLS and immediately discarded — it never appears in logs.
func AddKey(provider, key, label string) (*AddKeyResponse, error) {
	if !isValidProvider(provider) {
		return nil, fmt.Errorf("unsupported provider %q\nHint: use one of: anthropic, openai, google, custom\nExit: 1", provider)
	}
	if key == "" {
		return nil, fmt.Errorf("provider key not found for %q\nHint: run `nself gateway keys add --provider %s --key sk-...` first\nExit: 1", provider, provider)
	}

	client, baseURL, err := gatewayHTTPClient()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(AddKeyRequest{Provider: provider, Key: key, Label: label})
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/keys", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nself-ai-gateway not running (port %d unreachable)\nHint: run `nself start` or `nself gateway status` to diagnose\nExit: 3", ports.AIGatewayPort)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result AddKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

// RemoveKey calls DELETE /v1/keys/:id.
func RemoveKey(id string) error {
	if id == "" {
		return fmt.Errorf("key id is required\nHint: run `nself gateway keys list` to see available key IDs\nExit: 1")
	}

	client, baseURL, err := gatewayHTTPClient()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", baseURL+"/v1/keys/"+id, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("nself-ai-gateway not running (port %d unreachable)\nHint: run `nself start` or `nself gateway status` to diagnose\nExit: 3", ports.AIGatewayPort)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("key %q not found\nHint: run `nself gateway keys list` to see available key IDs\nExit: 1", id)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// isValidProvider returns true if the given provider is in ValidProviders.
func isValidProvider(provider string) bool {
	for _, v := range ValidProviders {
		if v == provider {
			return true
		}
	}
	return false
}
