package plugin

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Bug #64 — nself plugin install fails with registry format error
//
// Root cause (v0.x): install command failed when the live registry returned
// plugins as a JSON object map ({"plugin_name": {...}}) instead of an array
// ([{...}]). Only the array format was handled.
//
// Go rewrite fix: parseRegistryJSON detects the first non-whitespace byte of
// the plugins field. '[' → array format (pro registry), '{' → object format
// (free / live registry). Both formats are parsed and normalised into
// []PluginManifest.
//
// These tests verify that both formats parse correctly and that unknown formats
// return a descriptive error — not a silent empty list.
// ---------------------------------------------------------------------------

// TestParseRegistryJSON_ArrayFormat verifies that the pro-registry array format
// is parsed correctly and all plugin fields are preserved.
func TestParseRegistryJSON_ArrayFormat(t *testing.T) {
	input := `{
		"version": "1.0.0",
		"plugins": [
			{
				"name": "notify",
				"version": "1.0.0",
				"description": "Push notifications",
				"category": "messaging",
				"tier": "basic",
				"license": "source-available",
				"repository": "https://github.com/nself-org/plugins-pro",
				"checksum": "abc123"
			}
		]
	}`

	reg, err := parseRegistryJSON([]byte(input))
	if err != nil {
		t.Fatalf("parseRegistryJSON (array format): %v", err)
	}
	if len(reg.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(reg.Plugins))
	}
	if reg.Plugins[0].Name != "notify" {
		t.Errorf("expected plugin name %q, got %q", "notify", reg.Plugins[0].Name)
	}
	if reg.Plugins[0].Tier != "basic" {
		t.Errorf("expected tier %q, got %q", "basic", reg.Plugins[0].Tier)
	}
}

// TestParseRegistryJSON_ObjectFormat verifies that the free/live-registry object
// format is parsed correctly. This is the format returned by plugins.nself.org
// (Cloudflare Worker). A missing "name" field in the object value must be
// backfilled from the map key.
func TestParseRegistryJSON_ObjectFormat(t *testing.T) {
	input := `{
		"version": "1.0.0",
		"plugins": {
			"redis": {
				"version": "1.0.0",
				"description": "Redis caching",
				"category": "caching",
				"tier": "free",
				"license": "MIT",
				"repository": "https://github.com/nself-org/plugins",
				"checksum": "def456"
			},
			"search": {
				"name": "search",
				"version": "1.1.0",
				"description": "MeiliSearch full-text search",
				"category": "search",
				"tier": "free",
				"license": "MIT",
				"repository": "https://github.com/nself-org/plugins",
				"checksum": "ghi789"
			}
		}
	}`

	reg, err := parseRegistryJSON([]byte(input))
	if err != nil {
		t.Fatalf("Bug #64 regression: parseRegistryJSON (object format) failed: %v", err)
	}
	if len(reg.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(reg.Plugins))
	}

	// Build a name → plugin map for deterministic lookup (map iteration order varies).
	byName := make(map[string]PluginManifest, len(reg.Plugins))
	for _, p := range reg.Plugins {
		byName[p.Name] = p
	}

	redisPlugin, ok := byName["redis"]
	if !ok {
		t.Fatalf("expected plugin %q (backfilled from key), not found in: %v", "redis", byName)
	}
	if redisPlugin.Tier != "free" {
		t.Errorf("expected tier %q for redis, got %q", "free", redisPlugin.Tier)
	}

	searchPlugin, ok := byName["search"]
	if !ok {
		t.Fatalf("expected plugin %q not found", "search")
	}
	if searchPlugin.Version != "1.1.0" {
		t.Errorf("expected version %q for search, got %q", "1.1.0", searchPlugin.Version)
	}
}

// TestParseRegistryJSON_ObjectFormat_NameBackfill verifies that when the plugin
// value object does not include a "name" field, it is backfilled from the map key.
func TestParseRegistryJSON_ObjectFormat_NameBackfill(t *testing.T) {
	input := `{
		"plugins": {
			"myplugin": {
				"version": "2.0.0",
				"description": "A plugin without name in body",
				"category": "utility",
				"tier": "free",
				"license": "MIT"
			}
		}
	}`

	reg, err := parseRegistryJSON([]byte(input))
	if err != nil {
		t.Fatalf("parseRegistryJSON: %v", err)
	}
	if len(reg.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(reg.Plugins))
	}
	if reg.Plugins[0].Name != "myplugin" {
		t.Errorf("name not backfilled from key: got %q, want %q", reg.Plugins[0].Name, "myplugin")
	}
}

// TestParseRegistryJSON_EmptyPlugins verifies that an empty plugins field
// (null or absent) returns an empty Registry without error.
func TestParseRegistryJSON_EmptyPlugins(t *testing.T) {
	inputs := []string{
		`{"version":"1.0.0","plugins":[]}`,
		`{"version":"1.0.0","plugins":null}`,
		`{"version":"1.0.0"}`,
	}
	for _, input := range inputs {
		reg, err := parseRegistryJSON([]byte(input))
		if err != nil {
			t.Errorf("parseRegistryJSON(%q): unexpected error: %v", input, err)
			continue
		}
		if reg == nil {
			t.Errorf("parseRegistryJSON(%q): returned nil registry", input)
		}
	}
}

// TestParseRegistryJSON_UnknownFormat verifies that an unrecognised plugins
// field format (e.g. a bare number or string) returns a descriptive error
// rather than silently returning an empty registry or panicking.
func TestParseRegistryJSON_UnknownFormat(t *testing.T) {
	input := `{"plugins": 42}`
	_, err := parseRegistryJSON([]byte(input))
	if err == nil {
		t.Fatal("expected error for unknown registry format, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected registry plugins format") {
		t.Errorf("expected 'unexpected registry plugins format' in error, got: %v", err)
	}
}

// TestParseAPIEndpoints_StringArray verifies that the legacy string-array
// endpoint format is parsed without error.
func TestParseAPIEndpoints_StringArray(t *testing.T) {
	raw := mustMarshalJSON([]string{"/api/v1/foo", "/api/v1/bar"})
	endpoints := parseAPIEndpoints(raw)
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d: %v", len(endpoints), endpoints)
	}
	if endpoints[0] != "/api/v1/foo" {
		t.Errorf("endpoint[0]: got %q, want %q", endpoints[0], "/api/v1/foo")
	}
}

// TestParseAPIEndpoints_ObjectArray verifies that the live-registry
// object-array endpoint format is normalised to []string of paths.
func TestParseAPIEndpoints_ObjectArray(t *testing.T) {
	raw, _ := json.Marshal([]pluginEndpointEntry{
		{Method: "GET", Path: "/v1/health", Description: "health check"},
		{Method: "POST", Path: "/v1/send", Description: "send notification"},
	})
	endpoints := parseAPIEndpoints(json.RawMessage(raw))
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d: %v", len(endpoints), endpoints)
	}
	if endpoints[0] != "/v1/health" {
		t.Errorf("endpoint[0]: got %q, want %q", endpoints[0], "/v1/health")
	}
	if endpoints[1] != "/v1/send" {
		t.Errorf("endpoint[1]: got %q, want %q", endpoints[1], "/v1/send")
	}
}

// mustMarshalJSON is a test helper that marshals v to json.RawMessage, panicking
// on error (should never happen with well-formed test data).
func mustMarshalJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestEntryToManifestLicenseType(t *testing.T) {
	e := pluginEntry{
		Name:        "test",
		Version:     "1.0.0",
		Description: "desc",
		Category:    "cat",
		License:     "MIT",
		LicenseType: "pro",
		Tier:        "pro",
	}
	m := entryToManifest(e)
	if m.LicenseType != "pro" {
		t.Errorf("expected LicenseType=pro, got %q", m.LicenseType)
	}
}

func TestEntryToManifestPopulatesRegistryFields(t *testing.T) {
	// Only checks that the fields pluginEntry CAN populate are actually populated.
	e := pluginEntry{
		Name:            "myplugin",
		Version:         "2.0.0",
		Description:     "A plugin",
		Category:        "auth",
		Tier:            "basic",
		License:         "MIT",
		LicenseType:     "free",
		Repository:      "https://github.com/test/test",
		Checksum:        "abc123",
		RequiresLicense: true,
		Tags:            []string{"test"},
		Tables:          []string{"np_test_foo"},
		Port:            9000,
		Dependencies:    []string{"other"},
		APIEndpoints:    mustMarshalJSON([]string{"/api/v1"}),
	}
	m := entryToManifest(e)

	checks := map[string]bool{
		"Name":            m.Name == e.Name,
		"Version":         m.Version == e.Version,
		"Description":     m.Description == e.Description,
		"Category":        m.Category == e.Category,
		"License":         m.License == e.License,
		"LicenseType":     m.LicenseType == e.LicenseType,
		"Repository":      m.Repository == e.Repository,
		"Checksum":        m.Checksum == e.Checksum,
		"RequiresLicense": m.RequiresLicense == e.RequiresLicense,
		"Port":            m.Port == e.Port,
	}
	for field, ok := range checks {
		if !ok {
			t.Errorf("field %s not correctly mapped", field)
		}
	}

	// Ensure reflect is used (satisfies import).
	_ = reflect.TypeOf(m)
}

// TestEntryToManifest_AllFieldsCopied verifies that entryToManifest copies all
// extended fields (Tables, Port, Dependencies, APIEndpoints) from a pluginEntry
// into the returned PluginManifest without loss.
func TestEntryToManifest_AllFieldsCopied(t *testing.T) {
	e := pluginEntry{
		Name:         "myplugin",
		Version:      "1.0.0",
		Description:  "A test plugin",
		Category:     "utility",
		Tier:         "free",
		License:      "MIT",
		Repository:   "https://github.com/example/myplugin",
		Checksum:     "abc123",
		Tables:       []string{"np_myplugin_items"},
		Port:         8080,
		Dependencies: []string{"redis"},
		APIEndpoints: mustMarshalJSON([]string{"/api/v1"}),
		Tags:         []string{"test"},
	}

	m := entryToManifest(e)

	if len(m.Tables) != 1 || m.Tables[0] != "np_myplugin_items" {
		t.Errorf("Tables not copied correctly: got %v, want [np_myplugin_items]", m.Tables)
	}

	if m.Port != 8080 {
		t.Errorf("Port not copied correctly: got %d, want 8080", m.Port)
	}

	if len(m.Dependencies) != 1 || m.Dependencies[0] != "redis" {
		t.Errorf("Dependencies not copied correctly: got %v, want [redis]", m.Dependencies)
	}

	if len(m.APIEndpoints) != 1 || m.APIEndpoints[0] != "/api/v1" {
		t.Errorf("APIEndpoints not copied correctly: got %v, want [/api/v1]", m.APIEndpoints)
	}

	// Verify core fields are also present.
	if m.Name != "myplugin" {
		t.Errorf("Name not copied correctly: got %q, want %q", m.Name, "myplugin")
	}
	if m.Tier != "free" {
		t.Errorf("Tier not set correctly: got %q, want %q", m.Tier, "free")
	}
}

// TestRegistryMarshalJSON_CacheRoundTrip verifies that MarshalJSON preserves
// Category, License, LicenseType, and RequiresLicense so that a cache write
// followed by a read-back returns the same values. Previously these fields
// were omitted from the cache envelope, causing nself plugin list to show
// empty categories and incorrect tier derivation after the first cache write.
func TestRegistryMarshalJSON_CacheRoundTrip(t *testing.T) {
	original := Registry{
		Plugins: []PluginManifest{
			{
				Name:            "stripe",
				Version:         "1.0.0",
				Description:     "Stripe payments",
				Category:        "commerce",
				License:         "MIT",
				LicenseType:     "free",
				RequiresLicense: false,
				Tier:            "free",
				Repository:      "https://github.com/nself-org/plugins",
				Checksum:        "abc123",
				Tags:            []string{"payments"},
				Tables:          []string{"np_stripe_customers"},
				Port:            3100,
				Dependencies:    []string{"webhooks"},
			},
		},
	}

	// Serialize (simulates writeCache).
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Deserialize (simulates readCache via parseRegistryJSON).
	got, err := parseRegistryJSON(data)
	if err != nil {
		t.Fatalf("parseRegistryJSON after marshal: %v", err)
	}
	if len(got.Plugins) != 1 {
		t.Fatalf("expected 1 plugin after round-trip, got %d", len(got.Plugins))
	}

	p := got.Plugins[0]

	if p.Category != "commerce" {
		t.Errorf("Category lost in cache round-trip: got %q, want %q", p.Category, "commerce")
	}
	if p.License != "MIT" {
		t.Errorf("License lost in cache round-trip: got %q, want %q", p.License, "MIT")
	}
	if p.LicenseType != "free" {
		t.Errorf("LicenseType lost in cache round-trip: got %q, want %q", p.LicenseType, "free")
	}
	if p.RequiresLicense != false {
		t.Errorf("RequiresLicense lost in cache round-trip: got %v, want false", p.RequiresLicense)
	}
	if p.Tier != "free" {
		t.Errorf("Tier lost in cache round-trip: got %q, want %q", p.Tier, "free")
	}
}
