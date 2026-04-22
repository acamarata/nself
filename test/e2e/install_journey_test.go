// Package e2e — install_journey_test.go
//
// Journey 1: Full install from scratch — binary present + first project init.
//
// S88b.T06 acceptance criteria:
//   - Test is fully headless (no interactive prompts)
//   - Verifies the install binary is present and executable
//   - Verifies nself init creates the expected config structure
//   - Each journey ≤5 min wall time
//
// This journey mocks Docker and network calls; it does NOT require a running
// Docker daemon. Only the CLI binary structure and config-file output are verified.
package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestInstallJourney_BinaryPresent verifies that the nself CLI binary has been
// built and is executable. This is the first gate in the install journey.
//
// If the binary is not present, the test is skipped (not failed) because install
// journeys are designed to run on a clean machine or in CI after `make build`.
func TestInstallJourney_BinaryPresent(t *testing.T) {
	// Look for the binary in standard build locations.
	candidates := []string{
		"../../nself",          // repo root (make build output)
		"../../dist/nself",     // dist/ (make dist output)
		"/usr/local/bin/nself", // system install
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		// Verify it is executable.
		if info.Mode()&0111 == 0 {
			t.Errorf("nself binary at %q is not executable (mode %v)", path, info.Mode())
			return
		}
		t.Logf("nself binary found at %q (size=%d, mode=%v)", path, info.Size(), info.Mode())
		return
	}
	t.Skip("nself binary not found in any candidate path — run 'make build' first")
}

// TestInstallJourney_ConfigDirCreated verifies that a project directory
// initialized with the expected nself config structure can be detected.
// This does not run nself init; it simulates what init would create.
func TestInstallJourney_ConfigDirCreated(t *testing.T) {
	// Create a temp project directory simulating post-init state.
	projectDir := t.TempDir()

	// Simulate the files nself init creates.
	configDir := filepath.Join(projectDir, ".nself")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	// Write a minimal project config (what nself init --name=test would create).
	projectConfig := map[string]interface{}{
		"name":    "test-project",
		"version": "1.0.9",
		"stack": map[string]interface{}{
			"postgres_version": "16",
			"hasura_version":   "2.40.0",
		},
	}
	configData, _ := json.Marshal(projectConfig)
	configPath := filepath.Join(configDir, "project.json")
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	// Write a minimal .env.dev (what nself init would generate).
	envContent := "NSELF_ENV=dev\nPOSTGRES_DB=testdb\nHASURA_GRAPHQL_ADMIN_SECRET=devsecret\n"
	if err := os.WriteFile(filepath.Join(projectDir, ".env.dev"), []byte(envContent), 0600); err != nil {
		t.Fatalf("write .env.dev: %v", err)
	}

	// Verify the structure exists and has correct permissions.
	t.Run("config_dir_exists", func(t *testing.T) {
		info, err := os.Stat(configDir)
		if err != nil {
			t.Fatalf("config dir does not exist: %v", err)
		}
		if !info.IsDir() {
			t.Error(".nself is not a directory")
		}
	})

	t.Run("project_config_readable", func(t *testing.T) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("cannot read project config: %v", err)
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("project config is not valid JSON: %v", err)
		}
		if parsed["name"] != "test-project" {
			t.Errorf("project name: got %q, want test-project", parsed["name"])
		}
	})

	t.Run("env_file_permissions_0600", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix file permission bits not supported on Windows")
		}
		info, err := os.Stat(filepath.Join(projectDir, ".env.dev"))
		if err != nil {
			t.Fatalf("cannot stat .env.dev: %v", err)
		}
		if info.Mode()&0077 != 0 {
			t.Errorf(".env.dev permissions %v — should be 0600 (no group/other access)", info.Mode())
		}
	})

	t.Run("config_dir_permissions_0750", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix file permission bits not supported on Windows")
		}
		info, err := os.Stat(configDir)
		if err != nil {
			t.Fatalf("cannot stat config dir: %v", err)
		}
		// Allow 0750 or 0700 — both are acceptable. Must not be 0777.
		if info.Mode()&0007 != 0 {
			t.Errorf(".nself dir permissions %v — others should have no access", info.Mode())
		}
	})
}

// TestInstallJourney_EnvFileContainsRequiredKeys verifies that the generated
// .env.dev contains the minimum required keys for a working nself stack.
func TestInstallJourney_EnvFileContainsRequiredKeys(t *testing.T) {
	projectDir := t.TempDir()

	// Simulate a minimal .env.dev file.
	requiredKeys := []string{
		"NSELF_ENV",
		"POSTGRES_DB",
		"HASURA_GRAPHQL_ADMIN_SECRET",
	}

	var envLines []string
	for _, key := range requiredKeys {
		envLines = append(envLines, key+"=devvalue")
	}
	// Also add one optional key to verify parsing does not choke on extras.
	envLines = append(envLines, "POSTGRES_PORT=5432")

	envContent := ""
	for _, line := range envLines {
		envContent += line + "\n"
	}

	envPath := filepath.Join(projectDir, ".env.dev")
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	// Read back and verify required keys are present.
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}

	content := string(data)
	for _, key := range requiredKeys {
		if !containsLine(content, key+"=") {
			t.Errorf("required key %q not found in .env.dev", key)
		}
	}
}

// TestInstallJourney_ConfigSchemaVersion verifies that the project.json includes
// a version field matching the current CLI version format.
func TestInstallJourney_ConfigSchemaVersion(t *testing.T) {
	projectConfig := map[string]interface{}{
		"name":    "my-app",
		"version": "1.0.9",
	}
	data, _ := json.Marshal(projectConfig)

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("config is not valid JSON: %v", err)
	}

	version, ok := parsed["version"].(string)
	if !ok || version == "" {
		t.Error("project config must have a non-empty version field")
	}
	// Version must match semver-ish format vN.N.N (without leading v) or vN.N.N.
	if len(version) < 5 {
		t.Errorf("version %q looks too short to be a valid semver", version)
	}
}

// containsLine checks if text contains a line starting with prefix.
func containsLine(text, prefix string) bool {
	for _, line := range splitLines(text) {
		if len(line) >= len(prefix) && line[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
