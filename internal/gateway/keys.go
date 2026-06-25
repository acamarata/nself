package gateway

// Purpose: Key CRUD operations against nself-ai-gateway /v1/keys.
// Inputs: JWT token, provider, label, key material (write-only — never returned).
// Outputs: KeyInfo structs with id/provider/label/is_active/created_at (no key material).
// Constraints: Key material never included in any return value or log output.
// SPORT: F02 — nself gateway keys list/add/remove.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nself-org/cli/internal/ports"
)

// KeyInfo is the public representation of a gateway key.
// Key material is intentionally absent — never returned from the gateway list endpoint.
type KeyInfo struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Label     string    `json:"label"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func gatewayURL(path string) string {
	return fmt.Sprintf("http://localhost:%d%s", ports.AIGatewayPort, path)
}

func gatewayRequest(ctx context.Context, method, path, token string, body io.Reader) (*http.Response, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, method, gatewayURL(path), body)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w\n\nHint: ensure nself-ai-gateway is running (`nself plugin status nself-ai-gateway`)\nExit: 3", err)
	}
	return resp, nil
}

// ListKeys returns all keys from nself-ai-gateway. Key material is not included.
func ListKeys(ctx context.Context, token string) ([]KeyInfo, error) {
	resp, err := gatewayRequest(ctx, http.MethodGet, "/v1/keys", token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s\n\nExit: 1", resp.StatusCode, string(raw))
	}

	var keys []KeyInfo
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return keys, nil
}

// AddKey registers a new provider key with nself-ai-gateway.
// keyMaterial is write-only — only sent to the gateway, never stored locally.
func AddKey(ctx context.Context, token, provider, label, keyMaterial string) (string, error) {
	if provider == "" {
		return "", fmt.Errorf("provider required\n\nHint: use --provider anthropic|openai|google|custom\nExit: 1")
	}
	validProviders := map[string]bool{"anthropic": true, "openai": true, "google": true, "custom": true}
	if !validProviders[provider] {
		return "", fmt.Errorf("unknown provider %q\n\nHint: valid providers: anthropic, openai, google, custom\nExit: 1", provider)
	}
	if keyMaterial == "" {
		return "", fmt.Errorf("key material required\n\nHint: use --key to provide the API key\nExit: 1")
	}

	payload, _ := json.Marshal(map[string]string{
		"provider": provider,
		"label":    label,
		"key":      keyMaterial,
	})

	resp, err := gatewayRequest(ctx, http.MethodPost, "/v1/keys", token, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s\n\nExit: 1", resp.StatusCode, string(raw))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode error: %w", err)
	}
	return result.ID, nil
}

// RemoveKey deletes a key by ID from nself-ai-gateway.
func RemoveKey(ctx context.Context, token, id string) error {
	if id == "" {
		return fmt.Errorf("key ID required\n\nHint: use `nself gateway keys list` to see key IDs\nExit: 1")
	}

	resp, err := gatewayRequest(ctx, http.MethodDelete, "/v1/keys/"+id, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s\n\nHint: key may not exist\nExit: 1", resp.StatusCode, string(raw))
	}
	return nil
}
