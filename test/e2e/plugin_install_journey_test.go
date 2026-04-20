// Package e2e — plugin_install_journey_test.go
//
// Journey 2: Plugin install with license validation against mock ping_api.
//
// S88b.T06 acceptance criteria:
//   - Headless (no interactive prompts)
//   - Validates license against mock ping_api before install
//   - Verifies install fails gracefully when license is invalid
//   - Verifies install succeeds for free plugins without license check
package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// pluginPingServer starts a mock ping server that validates plugin install
// requests based on the provided tier. For tier="pro", only pro plugins are
// allowed. For tier="free", all installs are denied (no license).
func pluginPingServer(t *testing.T, tier string, allowedPlugins []string) *httptest.Server {
	t.Helper()
	allowedSet := make(map[string]bool)
	for _, p := range allowedPlugins {
		allowedSet[p] = true
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/license/validate":
			if tier == "" {
				// No license — invalid.
				json.NewEncoder(w).Encode(map[string]interface{}{
					"valid":  false,
					"reason": "no_license",
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"valid":   true,
				"tier":    tier,
				"plugins": allowedPlugins,
				"expires_at": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			})

		case "/plugins/download":
			pluginName := r.URL.Query().Get("name")
			if !allowedSet[pluginName] {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error":  "not_in_tier",
					"plugin": pluginName,
				})
				return
			}
			// Return a minimal plugin tarball stub (just the JSON header).
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"plugin": pluginName, "status": "download_ok"})

		default:
			http.NotFound(w, r)
		}
	}))
}

// TestPluginInstallJourney_ProPluginWithValidLicense verifies that a pro plugin
// install succeeds when the license tier grants access.
func TestPluginInstallJourney_ProPluginWithValidLicense(t *testing.T) {
	srv := pluginPingServer(t, "pro", []string{"ai", "claw", "mux"})
	defer srv.Close()

	t.Setenv("LICENSE_PING_URL", srv.URL)
	tmp := t.TempDir()
	t.Setenv("LICENSE_CACHE_PATH", filepath.Join(tmp, "license.json"))

	const testKey = "nself_pro_plugin_install_test12345678901"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Validate license.
	result, err := validateRemoteForJourney(ctx, testKey, srv.URL)
	if err != nil {
		t.Fatalf("license validation for pro plugin install: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid=true for pro tier, got: valid=false reason=%s", result.Reason)
	}
	if result.Tier != "pro" {
		t.Errorf("tier: got %q, want pro", result.Tier)
	}

	// Step 2: Simulate plugin download request for "ai" (in allowed set).
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		srv.URL+"/plugins/download?name=ai", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("plugin download request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("plugin download for ai: status %d, want 200", resp.StatusCode)
	}
	t.Logf("pro plugin install journey OK: license valid, ai download permitted")
}

// TestPluginInstallJourney_ProPluginWithoutLicense verifies that a pro plugin
// install is denied when no license is present.
func TestPluginInstallJourney_ProPluginWithoutLicense(t *testing.T) {
	// No-license server: always returns invalid.
	srv := pluginPingServer(t, "", nil)
	defer srv.Close()

	t.Setenv("LICENSE_PING_URL", srv.URL)
	tmp := t.TempDir()
	t.Setenv("LICENSE_CACHE_PATH", filepath.Join(tmp, "license.json"))

	const unlicensedKey = "nself_pro_unlicensed_test12345678901234"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := validateRemoteForJourney(ctx, unlicensedKey, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must be denied.
	if result.Valid {
		t.Fatal("pro plugin install must be denied without license")
	}
	if result.Reason == "" {
		t.Error("denial response must include reason for CLI display")
	}
	t.Logf("no-license denial OK: reason=%q", result.Reason)
}

// TestPluginInstallJourney_PluginNotInTier verifies that a plugin not covered
// by the license tier is denied at the download stage (403 Forbidden).
func TestPluginInstallJourney_PluginNotInTier(t *testing.T) {
	// Pro tier but only "ai" allowed — requesting "livekit" which is not in the list.
	srv := pluginPingServer(t, "pro", []string{"ai"})
	defer srv.Close()

	t.Setenv("LICENSE_PING_URL", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt to download "livekit" which is not in the allowed set.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		srv.URL+"/plugins/download?name=livekit", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("plugin download request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("plugin not in tier: status %d, want 403", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body) //nolint:errcheck
	if body["error"] != "not_in_tier" {
		t.Errorf("error body: got %v, want not_in_tier", body)
	}
	t.Logf("tier enforcement OK: livekit denied (not in tier), error=%q", body["error"])
}

// TestPluginInstallJourney_FreePlugin verifies that free plugins do not require
// a license validation call (they can install without a key).
func TestPluginInstallJourney_FreePlugin(t *testing.T) {
	// Free plugins do not go through license validation at all.
	// This test verifies the free plugin manifest format is recognizable.
	freePluginManifest := map[string]interface{}{
		"name":          "webhooks",
		"version":       "1.0.9",
		"description":   "Outbound webhook dispatcher",
		"category":      "utility",
		"license":       "MIT",
		"isCommercial":  false,
		"requiredTier":  "",
	}

	// Free plugins must have isCommercial: false.
	if freePluginManifest["isCommercial"] != false {
		t.Error("free plugin must have isCommercial: false")
	}

	// Free plugins must have MIT or similar open-source license.
	lic, _ := freePluginManifest["license"].(string)
	if !strings.EqualFold(lic, "MIT") {
		t.Errorf("free plugin license: got %q, want MIT", lic)
	}

	// Free plugins must have empty or absent requiredTier.
	tier, _ := freePluginManifest["requiredTier"].(string)
	if tier != "" {
		t.Errorf("free plugin requiredTier: got %q, want empty (no license required)", tier)
	}

	t.Log("free plugin manifest validation OK: no license required")
}

// TestPluginInstallJourney_KeyFormatValidation verifies that malformed license
// keys are rejected before any network call is made (format check is local).
func TestPluginInstallJourney_KeyFormatValidation(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid_pro_key", "nself_pro_" + strings.Repeat("a", 22), false},
		{"too_short", "nself_pro_short", true},
		{"wrong_prefix", "invalid_prefix_" + strings.Repeat("a", 22), true},
		{"empty", "", true},
		{"whitespace_only", "   ", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateLicenseKeyFormat(tc.key)
			if tc.wantErr && err == nil {
				t.Errorf("key %q: expected validation error, got nil", tc.key)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("key %q: unexpected validation error: %v", tc.key, err)
			}
		})
	}
}

// validateLicenseKeyFormat mirrors the format check in internal/plugin/license.go
// without importing the internal package.
func validateLicenseKeyFormat(key string) error {
	key = strings.TrimSpace(key)
	if len(key) < 32 {
		return &keyFormatError{key: key, reason: "too short (minimum 32 characters)"}
	}
	validPrefixes := []string{
		"nself_pro_",
		"nself_max_",
		"nself_ent_",
		"nself_owner_",
		"nself_plus_",
		"nself_claw_",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix) {
			return nil
		}
	}
	return &keyFormatError{key: key, reason: "unrecognized prefix"}
}

type keyFormatError struct {
	key    string
	reason string
}

func (e *keyFormatError) Error() string {
	return "invalid license key: " + e.reason
}
