package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSlugRE_ValidNames verifies that well-formed plugin slugs are accepted.
func TestSlugRE_ValidNames(t *testing.T) {
	valid := []string{
		"myplugin",
		"my-plugin",
		"my-plugin-v2",
		"ab",
		"a1",
		"plugin-with-many-words-here",
	}
	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if !SlugRE.MatchString(name) {
				t.Errorf("slug %q should be valid", name)
			}
		})
	}
}

// TestSlugRE_InvalidNames verifies that malformed slugs are rejected.
func TestSlugRE_InvalidNames(t *testing.T) {
	invalid := []string{
		"",
		"a",                     // too short (min 2 chars after first letter)
		"MyPlugin",              // uppercase
		"my_plugin",             // underscore
		"1plugin",               // starts with digit
		"-plugin",               // starts with dash
		"plugin-",               // ends with dash (no rule against it but a safeguard)
		strings.Repeat("a", 43), // too long (>41 chars total)
	}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			if SlugRE.MatchString(name) {
				t.Errorf("slug %q should be invalid but matched", name)
			}
		})
	}
}

// TestRun_CreatesFiles verifies that Run creates the expected scaffold files
// in the output directory.
func TestRun_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		Name:        "mytest",
		Tier:        "free",
		Description: "A test plugin",
		Author:      "Tester",
		Category:    "utility",
		Language:    "go",
		OutDir:      filepath.Join(dir, "mytest"),
		Force:       true,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == nil {
		t.Fatal("Run returned nil result")
	}

	// Verify the output directory was created.
	if _, err := os.Stat(opts.OutDir); err != nil {
		t.Errorf("output directory not created: %v", err)
	}
	// Verify at least one file was scaffolded.
	if len(result.Files) == 0 {
		t.Error("Run created no files")
	}
}

// TestRun_InvalidName verifies that an invalid plugin name returns an error.
func TestRun_InvalidName(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		Name:   "InvalidName", // uppercase — invalid
		Tier:   "free",
		OutDir: filepath.Join(dir, "invalid"),
	}

	_, err := Run(opts)
	if err == nil {
		t.Fatal("expected error for invalid plugin name, got nil")
	}
}

// TestRun_GoPlugin verifies that a Go plugin scaffold produces output files.
func TestRun_GoPlugin(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "goplugin")
	opts := Options{
		Name:     "goplugin",
		Tier:     "free",
		Language: "go",
		OutDir:   outDir,
		Force:    true,
	}

	result, err := Run(opts)
	if err != nil {
		t.Fatalf("Run (go): %v", err)
	}

	// Go plugins should produce at least one file.
	if len(result.Files) == 0 {
		t.Error("Run (go) produced no files")
	}
}

// TestRun_DefaultValues verifies that zero-value optional fields are defaulted.
func TestRun_DefaultValues(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "defaults")
	opts := Options{
		Name:   "defaults",
		OutDir: outDir,
		Force:  true,
		// Tier, Language, Category, etc. all omitted — must use defaults.
	}

	_, err := Run(opts)
	if err != nil {
		t.Fatalf("Run with defaults: %v", err)
	}
}

// TestRun_InvalidTier verifies that an unrecognized tier returns an error.
func TestRun_InvalidTier(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		Name:   "mytier",
		Tier:   "enterprise", // invalid
		OutDir: filepath.Join(dir, "mytier"),
	}
	_, err := Run(opts)
	if err == nil {
		t.Fatal("expected error for invalid tier, got nil")
	}
}

// TestRun_InvalidTenancy verifies that an unrecognized tenancy mode returns an error.
func TestRun_InvalidTenancy(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		Name:    "mytenancy",
		Tenancy: TenancyMode("cloud-only"), // invalid
		OutDir:  filepath.Join(dir, "mytenancy"),
	}
	_, err := Run(opts)
	if err == nil {
		t.Fatal("expected error for invalid tenancy, got nil")
	}
}

// TestRun_TenancyNone verifies that TenancyNone emits a comment-only migration
// and a no-filter Hasura stub.
func TestRun_TenancyNone(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "tnone")
	result, err := Run(Options{
		Name:    "tnone",
		Tenancy: TenancyNone,
		OutDir:  outDir,
		Force:   true,
	})
	if err != nil {
		t.Fatalf("Run(TenancyNone): %v", err)
	}

	migration := readFile(t, filepath.Join(outDir, "migrations/001_init.sql"))
	if strings.Contains(migration, "source_account_id") {
		t.Error("TenancyNone: migration should not contain source_account_id")
	}
	if strings.Contains(migration, "tenant_id") {
		t.Error("TenancyNone: migration should not contain tenant_id")
	}

	hasura := readFile(t, filepath.Join(outDir, "hasura/metadata.json"))
	if strings.Contains(hasura, "X-Hasura-Tenant-Id") {
		t.Error("TenancyNone: Hasura metadata should not contain tenant row filter")
	}

	pluginJSON := readFile(t, filepath.Join(outDir, "plugin.json"))
	if !strings.Contains(pluginJSON, `"supported": false`) {
		t.Error("TenancyNone: plugin.json multiApp.supported should be false")
	}

	_ = result
}

// TestRun_TenancyAppIsolation verifies that TenancyAppIsolation emits
// source_account_id and no Hasura tenant filter.
func TestRun_TenancyAppIsolation(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "tapp")
	result, err := Run(Options{
		Name:    "tapp",
		Tenancy: TenancyAppIsolation,
		OutDir:  outDir,
		Force:   true,
	})
	if err != nil {
		t.Fatalf("Run(TenancyAppIsolation): %v", err)
	}

	migration := readFile(t, filepath.Join(outDir, "migrations/001_init.sql"))
	if !strings.Contains(migration, "source_account_id") {
		t.Error("TenancyAppIsolation: migration missing source_account_id")
	}
	if strings.Contains(migration, "tenant_id") {
		t.Error("TenancyAppIsolation: migration should not contain tenant_id")
	}

	hasura := readFile(t, filepath.Join(outDir, "hasura/metadata.json"))
	if strings.Contains(hasura, "X-Hasura-Tenant-Id") {
		t.Error("TenancyAppIsolation: Hasura metadata should not have tenant filter")
	}

	pluginJSON := readFile(t, filepath.Join(outDir, "plugin.json"))
	if !strings.Contains(pluginJSON, `"supported": true`) {
		t.Error("TenancyAppIsolation: plugin.json multiApp.supported should be true")
	}
	if !strings.Contains(pluginJSON, `"isolationColumn": "source_account_id"`) {
		t.Error("TenancyAppIsolation: plugin.json isolationColumn should be source_account_id")
	}

	_ = result
}

// TestRun_TenancyCloudTenant verifies that TenancyCloudTenant emits
// tenant_id and the Hasura X-Hasura-Tenant-Id row filter.
func TestRun_TenancyCloudTenant(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "tcloud")
	result, err := Run(Options{
		Name:    "tcloud",
		Tenancy: TenancyCloudTenant,
		OutDir:  outDir,
		Force:   true,
	})
	if err != nil {
		t.Fatalf("Run(TenancyCloudTenant): %v", err)
	}

	migration := readFile(t, filepath.Join(outDir, "migrations/001_init.sql"))
	if !strings.Contains(migration, "tenant_id") {
		t.Error("TenancyCloudTenant: migration missing tenant_id")
	}
	if strings.Contains(migration, "source_account_id") {
		t.Error("TenancyCloudTenant: migration should not contain source_account_id")
	}

	hasura := readFile(t, filepath.Join(outDir, "hasura/metadata.json"))
	if !strings.Contains(hasura, "X-Hasura-Tenant-Id") {
		t.Error("TenancyCloudTenant: Hasura metadata missing tenant row filter")
	}

	pluginJSON := readFile(t, filepath.Join(outDir, "plugin.json"))
	if !strings.Contains(pluginJSON, `"supported": false`) {
		t.Error("TenancyCloudTenant: plugin.json multiApp.supported should be false (cloud-tenant uses tenant_id, not source_account_id)")
	}

	_ = result
}

// TestRun_TenancyBoth verifies that TenancyBoth emits both columns and the
// Hasura cloud tenant filter.
func TestRun_TenancyBoth(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "tboth")
	result, err := Run(Options{
		Name:    "tboth",
		Tenancy: TenancyBoth,
		OutDir:  outDir,
		Force:   true,
	})
	if err != nil {
		t.Fatalf("Run(TenancyBoth): %v", err)
	}

	migration := readFile(t, filepath.Join(outDir, "migrations/001_init.sql"))
	if !strings.Contains(migration, "source_account_id") {
		t.Error("TenancyBoth: migration missing source_account_id")
	}
	if !strings.Contains(migration, "tenant_id") {
		t.Error("TenancyBoth: migration missing tenant_id")
	}

	hasura := readFile(t, filepath.Join(outDir, "hasura/metadata.json"))
	if !strings.Contains(hasura, "X-Hasura-Tenant-Id") {
		t.Error("TenancyBoth: Hasura metadata should have tenant filter (both mode uses cloud filter)")
	}

	pluginJSON := readFile(t, filepath.Join(outDir, "plugin.json"))
	if !strings.Contains(pluginJSON, `"supported": true`) {
		t.Error("TenancyBoth: plugin.json multiApp.supported should be true")
	}

	_ = result
}

// TestPluginJSON_TenancyFields verifies that plugin.json multiApp fields are
// correct for all four tenancy modes without creating a full scaffold.
func TestPluginJSON_TenancyFields(t *testing.T) {
	cases := []struct {
		tenancy          TenancyMode
		wantSupported    string
		wantIsolationCol string
	}{
		{TenancyNone, `"supported": false`, `"isolationColumn": ""`},
		{TenancyAppIsolation, `"supported": true`, `"isolationColumn": "source_account_id"`},
		{TenancyCloudTenant, `"supported": false`, `"isolationColumn": ""`},
		{TenancyBoth, `"supported": true`, `"isolationColumn": "source_account_id"`},
	}
	for _, tc := range cases {
		t.Run(string(tc.tenancy), func(t *testing.T) {
			p := Params{
				Name:       "myplugin",
				PascalName: "Myplugin",
				EnvPrefix:  "MYPLUGIN",
				Tier:       "free",
				RepoBucket: "free",
				MinCLI:     "1.0.9",
				MinSDK:     "0.1.0",
				Category:   "custom",
				Port:       8080,
				Year:       2026,
				Tenancy:    tc.tenancy,
			}
			got := renderPluginJSON(p)
			if !strings.Contains(got, tc.wantSupported) {
				t.Errorf("tenancy %s: want %q in plugin.json, got:\n%s", tc.tenancy, tc.wantSupported, got)
			}
			if !strings.Contains(got, tc.wantIsolationCol) {
				t.Errorf("tenancy %s: want %q in plugin.json, got:\n%s", tc.tenancy, tc.wantIsolationCol, got)
			}
		})
	}
}

// readFile is a test helper that reads a file and returns its content as string.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile(%s): %v", path, err)
	}
	return string(b)
}
