package doctor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// metadataResponse is a minimal Hasura metadata export response for tests.
func metadataResponse(schemas []map[string]interface{}) []byte {
	payload := map[string]interface{}{
		"version":        3,
		"remote_schemas": schemas,
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestCheckOrphanRemoteSchemas_NoHasuraURL(t *testing.T) {
	os.Unsetenv("NSELF_HASURA_GRAPHQL_URL")
	os.Unsetenv("HASURA_GRAPHQL_URL")
	os.Unsetenv("HASURA_GRAPHQL_ADMIN_SECRET")

	result := CheckOrphanRemoteSchemas(context.Background())
	if result.Status != "skip" {
		t.Errorf("expected skip when HASURA_GRAPHQL_URL unset, got %q: %s", result.Status, result.Message)
	}
}

func TestCheckOrphanRemoteSchemas_NoAdminSecret(t *testing.T) {
	os.Setenv("HASURA_GRAPHQL_URL", "http://localhost:8080")
	defer os.Unsetenv("HASURA_GRAPHQL_URL")
	os.Unsetenv("HASURA_GRAPHQL_ADMIN_SECRET")

	result := CheckOrphanRemoteSchemas(context.Background())
	if result.Status != "skip" {
		t.Errorf("expected skip when admin secret unset, got %q: %s", result.Status, result.Message)
	}
}

func TestCheckOrphanRemoteSchemas_HasuraUnreachable(t *testing.T) {
	os.Setenv("HASURA_GRAPHQL_URL", "http://127.0.0.1:19999") // nothing listening
	defer os.Unsetenv("HASURA_GRAPHQL_URL")
	os.Setenv("HASURA_GRAPHQL_ADMIN_SECRET", "test-secret")
	defer os.Unsetenv("HASURA_GRAPHQL_ADMIN_SECRET")

	result := CheckOrphanRemoteSchemas(context.Background())
	if result.Status != "warn" {
		t.Errorf("expected warn when Hasura unreachable, got %q: %s", result.Status, result.Message)
	}
}

func TestCheckOrphanRemoteSchemas_NoRemoteSchemas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(metadataResponse(nil))
	}))
	defer srv.Close()

	os.Setenv("HASURA_GRAPHQL_URL", srv.URL)
	defer os.Unsetenv("HASURA_GRAPHQL_URL")
	os.Setenv("HASURA_GRAPHQL_ADMIN_SECRET", "test-secret")
	defer os.Unsetenv("HASURA_GRAPHQL_ADMIN_SECRET")

	result := CheckOrphanRemoteSchemas(context.Background())
	if result.Status != "pass" {
		t.Errorf("expected pass with no remote schemas, got %q: %s", result.Status, result.Message)
	}
}

func TestCheckOrphanRemoteSchemas_AllLoaded(t *testing.T) {
	schemas := []map[string]interface{}{
		{
			"name":       "ai-schema",
			"definition": map[string]interface{}{"url": "http://plugin-ai:4000/graphql"},
		},
		{
			"name":       "stripe-schema",
			"definition": map[string]interface{}{"url": "http://plugin-stripe:4001/graphql"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(metadataResponse(schemas))
	}))
	defer srv.Close()

	os.Setenv("HASURA_GRAPHQL_URL", srv.URL)
	defer os.Unsetenv("HASURA_GRAPHQL_URL")
	os.Setenv("HASURA_GRAPHQL_ADMIN_SECRET", "test-secret")
	defer os.Unsetenv("HASURA_GRAPHQL_ADMIN_SECRET")

	// Simulate both plugins loaded via NSELF_PLUGINS_LOADED.
	os.Setenv("NSELF_PLUGINS_LOADED", "ai,stripe")
	defer os.Unsetenv("NSELF_PLUGINS_LOADED")

	result := CheckOrphanRemoteSchemas(context.Background())
	if result.Status != "pass" {
		t.Errorf("expected pass when all remote schemas have matching plugins, got %q: %s", result.Status, result.Message)
	}
}

func TestCheckOrphanRemoteSchemas_OrphanDetected(t *testing.T) {
	schemas := []map[string]interface{}{
		{
			"name":       "ai-schema",
			"definition": map[string]interface{}{"url": "http://plugin-ai:4000/graphql"},
		},
		{
			// stripe was uninstalled but remote schema was left behind.
			"name":       "stripe-schema",
			"definition": map[string]interface{}{"url": "http://plugin-stripe:4001/graphql"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(metadataResponse(schemas))
	}))
	defer srv.Close()

	os.Setenv("HASURA_GRAPHQL_URL", srv.URL)
	defer os.Unsetenv("HASURA_GRAPHQL_URL")
	os.Setenv("HASURA_GRAPHQL_ADMIN_SECRET", "test-secret")
	defer os.Unsetenv("HASURA_GRAPHQL_ADMIN_SECRET")

	// Only ai is loaded; stripe was uninstalled.
	os.Setenv("NSELF_PLUGINS_LOADED", "ai")
	defer os.Unsetenv("NSELF_PLUGINS_LOADED")

	result := CheckOrphanRemoteSchemas(context.Background())
	if result.Status != "warn" {
		t.Errorf("expected warn for orphan, got %q: %s", result.Status, result.Message)
	}
	if !strings.Contains(result.Message, "stripe-schema") {
		t.Errorf("expected orphan name 'stripe-schema' in message, got: %s", result.Message)
	}
	if !strings.Contains(result.FixCmd, "untrack") {
		t.Errorf("expected fix command to suggest untrack, got: %s", result.FixCmd)
	}
}

func TestCheckOrphanRemoteSchemas_URLBasedMatch(t *testing.T) {
	pluginURL := "http://plugin-geocoding:4005"
	schemas := []map[string]interface{}{
		{
			"name":       "geo-schema", // name doesn't match "geocoding" exactly
			"definition": map[string]interface{}{"url": pluginURL + "/graphql"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(metadataResponse(schemas))
	}))
	defer srv.Close()

	os.Setenv("HASURA_GRAPHQL_URL", srv.URL)
	defer os.Unsetenv("HASURA_GRAPHQL_URL")
	os.Setenv("HASURA_GRAPHQL_ADMIN_SECRET", "test-secret")
	defer os.Unsetenv("HASURA_GRAPHQL_ADMIN_SECRET")

	// Plugin URL env var present → plugin is loaded.
	os.Setenv("PLUGIN_GEOCODING_INTERNAL_URL", pluginURL)
	defer os.Unsetenv("PLUGIN_GEOCODING_INTERNAL_URL")

	result := CheckOrphanRemoteSchemas(context.Background())
	if result.Status != "pass" {
		t.Errorf("expected pass when URL matches PLUGIN_*_INTERNAL_URL, got %q: %s", result.Status, result.Message)
	}
}

func TestResolveLoadedPlugins_MultipleSourcess(t *testing.T) {
	os.Unsetenv("NSELF_PLUGINS_LOADED")
	os.Setenv("NSELF_AI_LOADED", "1")
	defer os.Unsetenv("NSELF_AI_LOADED")
	os.Setenv("PLUGIN_STRIPE_INTERNAL_URL", "http://plugin-stripe:4001")
	defer os.Unsetenv("PLUGIN_STRIPE_INTERNAL_URL")

	loaded := resolveLoadedPlugins()
	if !loaded["ai"] {
		t.Error("expected 'ai' in loaded set from NSELF_AI_LOADED=1")
	}
	if !loaded["stripe"] {
		t.Error("expected 'stripe' in loaded set from PLUGIN_STRIPE_INTERNAL_URL")
	}
}
