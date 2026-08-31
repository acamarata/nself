package plugin

// S20: Free-plugin accounts + nself_free_ keys.
//
// Free plugins (25 MIT plugins) can now require an anonymous free-tier key for
// rate-limiting, telemetry, and conversion tracking.
//
// Flow:
//   nself plugin install <free-plugin>
//     → if no key in env → prompt "Create a free account? [Y/n]"
//     → POST /v1/account/free-register → receive nself_free_<uuid>
//     → store via nself license set nself_free_<uuid>
//     → count installs → after 3rd, surface upsell prompt
//     → POST /v1/telemetry/plugin-install (fire-and-forget)

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
	saferecover "github.com/nself-org/cli/internal/recover"
)

const freeLicensePrefix = "nself_free_"

// IsFreeKey returns true if the key is a nself_free_ key.
func IsFreeKey(key string) bool {
	return strings.HasPrefix(key, freeLicensePrefix)
}

// IsPaidPlugin is the exported wrapper around the package-private isPaidPlugin.
// Returns true if the named plugin requires a paid license key.
func IsPaidPlugin(name string) bool {
	return isPaidPlugin(name)
}

// freeRegisterResponse is the JSON response from POST /v1/account/free-register.
type freeRegisterResponse struct {
	Key       string `json:"key"`
	CreatedAt string `json:"created_at"`
}

// RegisterFreeAccount calls ping_api to create a new nself_free_ key.
// Returns the raw key string on success.
func RegisterFreeAccount(ctx context.Context, pingURL string) (string, error) {
	url := strings.TrimRight(pingURL, "/") + "/v1/account/free-register"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("free-register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("free-register network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("free-register failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out freeRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("free-register decode: %w", err)
	}
	if !strings.HasPrefix(out.Key, freeLicensePrefix) {
		return "", fmt.Errorf("free-register: unexpected key format")
	}
	return out.Key, nil
}

// pluginInstallTelemetryPayload is the body for POST /v1/telemetry/plugin-install.
type pluginInstallTelemetryPayload struct {
	LicenseKey string `json:"license_key"`
	PluginName string `json:"plugin_name"`
}

// SendFreeInstallTelemetry fires a non-blocking install event to ping_api.
// Errors are silently discarded — telemetry must never block installs.
func SendFreeInstallTelemetry(pingURL string, licenseKey string, pluginName string) {
	saferecover.SafeGo("plugin_free_install_telemetry", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		payload := pluginInstallTelemetryPayload{
			LicenseKey: licenseKey,
			PluginName: pluginName,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return
		}

		url := strings.TrimRight(pingURL, "/") + "/v1/telemetry/plugin-install"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// httptimeout.Health (5s) matches the prior client timeout for fire-and-forget telemetry.
		resp, err := httptimeout.Health.Do(req)
		if err != nil {
			return
		}
		defer func() { _ = resp.Body.Close() }()
	})
}
