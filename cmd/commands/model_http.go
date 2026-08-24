package commands

// Purpose: Shared Ollama HTTP helpers for the `nself model` command group,
// split out of model.go (CLI-R12 Batch B mechanical file-size split).
// Mirrors the equivalent helpers in ollama.go so both command trees share
// the same env-var configuration.
// Inputs: NSELF_OLLAMA_HOST / PLUGIN_AI_OLLAMA_URL (base URL) and
// NSELF_OLLAMA_TIMEOUT_SECONDS (client timeout override).
// Outputs: a configured base URL, an *http.Client, or a decoded JSON GET
// response.
// Constraints: pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// modelOllamaURL returns the base URL for the Ollama API. Reads the same env
// vars as the `nself ollama` command tree so both share configuration.
func modelOllamaURL() string {
	if u := os.Getenv("NSELF_OLLAMA_HOST"); u != "" {
		return u
	}
	if u := os.Getenv("PLUGIN_AI_OLLAMA_URL"); u != "" {
		return u
	}
	return "http://localhost:11434"
}

func modelHTTPClient() *http.Client {
	timeout := 120 * time.Second
	if t := os.Getenv("NSELF_OLLAMA_TIMEOUT_SECONDS"); t != "" {
		var secs int
		if _, err := fmt.Sscanf(t, "%d", &secs); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	return &http.Client{Timeout: timeout}
}

// modelOllamaGet performs a GET against the Ollama API and decodes JSON.
func modelOllamaGet(path string, dst any) error {
	resp, err := modelHTTPClient().Get(modelOllamaURL() + path)
	if err != nil {
		return fmt.Errorf("ollama GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ollama GET %s HTTP %d: %s", path, resp.StatusCode, body)
	}
	return json.Unmarshal(body, dst)
}
