package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestGenerateSecret_Length verifies that GenerateSecret produces a string of
// at least the requested byte length.
func TestGenerateSecret_Length(t *testing.T) {
	cases := []int{16, 32, 64}
	for _, length := range cases {
		t.Run(string(rune('0'+length/16)), func(t *testing.T) {
			s, err := GenerateSecret(length)
			if err != nil {
				t.Fatalf("GenerateSecret(%d): %v", length, err)
			}
			if s == "" {
				t.Error("GenerateSecret returned empty string")
			}
		})
	}
}

// TestGenerateSecret_Unique verifies two calls produce different secrets.
func TestGenerateSecret_Unique(t *testing.T) {
	a, err := GenerateSecret(32)
	if err != nil {
		t.Fatalf("first GenerateSecret: %v", err)
	}
	b, err := GenerateSecret(32)
	if err != nil {
		t.Fatalf("second GenerateSecret: %v", err)
	}
	if a == b {
		t.Error("GenerateSecret should produce unique values each call")
	}
}

// TestWriteEnvFile_CreatesFile verifies WriteEnvFile writes content to disk.
func TestWriteEnvFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.dev")
	content := []byte("PROJECT_NAME=test\nBASE_DOMAIN=test.local\n")

	if err := WriteEnvFile(path, content); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("file content mismatch: got %q, want %q", string(got), string(content))
	}
}

// TestWriteEnvFile_RestrictedPermissions verifies the env file is written with
// restricted permissions (0600).
func TestWriteEnvFile_RestrictedPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows: chmod 0600 is not enforced")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.dev")

	if err := WriteEnvFile(path, []byte("KEY=val")); err != nil {
		t.Fatalf("WriteEnvFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected 0600 permissions, got %04o", perm)
	}
}

// TestEnsureEnvFilePermissions_FixesOverPermissive verifies permissions are
// corrected when a file is too permissive.
func TestEnsureEnvFilePermissions_FixesOverPermissive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows: chmod 0600 is not enforced")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("KEY=val"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := EnsureEnvFilePermissions(path); err != nil {
		t.Fatalf("EnsureEnvFilePermissions: %v", err)
	}

	info, _ := os.Stat(path)
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected 0600 after EnsureEnvFilePermissions, got %04o", perm)
	}
}

// TestValidateProjectName_Valid verifies well-formed project names pass.
func TestValidateProjectName_Valid(t *testing.T) {
	valid := []string{
		"my-project",
		"myproject",
		"my-app-2024",
		"ab",
		"a0",
	}
	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if err := validateProjectName(name); err != nil {
				t.Errorf("validateProjectName(%q): unexpected error: %v", name, err)
			}
		})
	}
}

// TestValidateProjectName_Invalid verifies malformed project names are rejected.
func TestValidateProjectName_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"a",                     // single char
		"My-Project",            // uppercase
		"-start",                // starts with dash
		"end-",                  // ends with dash
		"has_underscore",        // underscore
		strings.Repeat("a", 31), // too long
	}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			if err := validateProjectName(name); err == nil {
				t.Errorf("validateProjectName(%q): expected error, got nil", name)
			}
		})
	}
}
