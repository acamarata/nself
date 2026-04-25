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
		"a",               // too short (min 2 chars after first letter)
		"MyPlugin",        // uppercase
		"my_plugin",       // underscore
		"1plugin",         // starts with digit
		"-plugin",         // starts with dash
		"plugin-",         // ends with dash (no rule against it but a safeguard)
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
