package commands

// The small AI client `nself doctor` needs.
//
// Purpose: doctor probes a local Ollama install and the ai plugin daemon as
// part of its AI checks. Those probes used to share helpers with the `nself ai`
// command family, which moved to the ai-cli plugin under CLI-R11; doctor stayed
// in core, so the handful of helpers it actually calls stayed with it.
//
// Inputs: OLLAMA_BASE_URL / OLLAMA_HOST, PLUGIN_AI_INTERNAL_URL and
// PLUGIN_INTERNAL_SECRET from the environment.
//
// Outputs: a base URL, and the raw bytes plus status of a plugin request.
//
// Constraints: this is a deliberate duplicate of code the ai-cli plugin also
// carries, and the smaller half of it — doctor only reads. Merging them back
// would mean either dragging doctor into the plugin or dragging the plugin's
// command tree back into core. If the plugin's port or auth header changes,
// this has to change with it, which is why both sides name the same env vars
// rather than hardcoding anything.

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/httptimeout"
)

// ollamaBaseURL resolves where a local Ollama is listening.
// OLLAMA_HOST is accepted without a scheme because that is how Ollama's own
// tooling sets it.
func ollamaBaseURL() string {
	if u := os.Getenv("OLLAMA_BASE_URL"); u != "" {
		return u
	}
	if u := os.Getenv("OLLAMA_HOST"); u != "" {
		if !strings.HasPrefix(u, "http") {
			return "http://" + u
		}
		return u
	}
	return "http://127.0.0.1:11434"
}

// aiPluginURL resolves the ai plugin daemon's internal address.
//
// plugin-ai listens on 3709. The default hostname matches the docker-compose
// service name `nself build` generates; older templates used "ai:3680", so
// PLUGIN_AI_INTERNAL_URL overrides it.
func aiPluginURL() string {
	if u := os.Getenv("PLUGIN_AI_INTERNAL_URL"); u != "" {
		return u
	}
	return "http://plugin-ai:3709"
}

// aiPluginRequest performs a request against the ai plugin daemon, returning
// the body and status rather than an error for non-2xx: doctor reports what it
// found rather than failing on it.
func aiPluginRequest(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, aiPluginURL()+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok := os.Getenv("PLUGIN_INTERNAL_SECRET"); tok != "" {
		req.Header.Set("X-Internal-Token", tok)
	}
	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	// Capped read: this talks to a local daemon, but doctor must not be a way to
	// exhaust memory if that daemon misbehaves.
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return b, resp.StatusCode, nil
}
