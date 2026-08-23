package commands

// Purpose: small health-check probes used by the AI wizard (Ollama reachable,
// local chat/embed round-trip) plus the OAuth callback waiter. Inputs are a
// context and, for the OAuth waiter, an expected state and timeout; outputs
// are bools or an error.
// Constraints: split out of doctor_ai.go (CLI-R12) as a pure move, no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func ollamaHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", ollamaBaseURL()+"/api/tags", nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func testLocalChat(ctx context.Context) bool {
	payload := `{"model":"gemma2:2b","messages":[{"role":"user","content":"Say hello in one word."}],"stream":false}`
	req, err := http.NewRequestWithContext(ctx, "POST", ollamaBaseURL()+"/api/chat",
		strings.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func testLocalEmbed(ctx context.Context) bool {
	payload := `{"model":"nomic-embed-text","prompt":"test"}`
	req, err := http.NewRequestWithContext(ctx, "POST", ollamaBaseURL()+"/api/embeddings",
		strings.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func waitForOAuthCallback(ctx context.Context, state string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("OAuth callback not received within %s", timeout)
		case <-ticker.C:
			// Poll the plugin-ai for OAuth completion.
			path := fmt.Sprintf("/ai/pool/oauth/status?state=%s", state)
			body, status, err := aiPluginRequest(ctx, "GET", path, nil)
			if err != nil {
				continue
			}
			if status < 400 {
				var resp struct {
					Complete bool   `json:"complete"`
					Error    string `json:"error"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					return fmt.Errorf("unmarshal response: %w", err)
				}
				if resp.Complete {
					return nil
				}
				if resp.Error != "" {
					return fmt.Errorf("OAuth error: %s", resp.Error)
				}
			}
		}
	}
}
