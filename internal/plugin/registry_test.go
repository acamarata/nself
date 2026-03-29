package plugin

import (
	"reflect"
	"testing"
)

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
		APIEndpoints:    []string{"/api/v1"},
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
		APIEndpoints: []string{"/api/v1"},
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
