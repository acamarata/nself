package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// testWriteFile writes content at path and fails the test on error.
// Uses a distinct name to avoid colliding with postvalidate_test.go's
// writeFile helper (go test compiles the package together).
func testWriteFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// cfgWithJWT builds a minimal Config where Hasura has an existing JWTKey —
// enough for BuildJWTSecret to produce the full JSON without any env var.
func cfgWithJWT() *config.Config {
	return &config.Config{
		Hasura: config.HasuraConfig{
			JWTKey:  "test-key-for-build-orchestrator-persist",
			JWTType: "HS256",
		},
	}
}

// TestPersistGeneratedSecrets_JWTWrittenWhenMissing verifies that
// HASURA_GRAPHQL_JWT_SECRET is appended to .env.secrets when no env file on
// disk defines it.
func TestPersistGeneratedSecrets_JWTWrittenWhenMissing(t *testing.T) {
	// Isolate from the caller's environment so an ambient
	// HASURA_GRAPHQL_JWT_SECRET doesn't leak into BuildJWTSecret.
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", "")

	dir := t.TempDir()
	testWriteFile(t, filepath.Join(dir, ".env"), "PROJECT_NAME=test\n", 0600)

	cfg := cfgWithJWT()

	if err := persistGeneratedSecrets(dir, cfg); err != nil {
		t.Fatalf("persistGeneratedSecrets: %v", err)
	}

	secrets := filepath.Join(dir, ".env.secrets")
	data, err := os.ReadFile(secrets)
	if err != nil {
		t.Fatalf("expected .env.secrets to exist: %v", err)
	}
	if !strings.Contains(string(data), "HASURA_GRAPHQL_JWT_SECRET=") {
		t.Fatalf(".env.secrets missing JWT entry:\n%s", string(data))
	}
}

// TestPersistGeneratedSecrets_EnvSecretsIsChmod0600 verifies the file ends
// up owner-only (0600) even when it existed with looser perms beforehand.
func TestPersistGeneratedSecrets_EnvSecretsIsChmod0600(t *testing.T) {
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", "")

	dir := t.TempDir()
	testWriteFile(t, filepath.Join(dir, ".env"), "PROJECT_NAME=test\n", 0600)
	// Pre-create .env.secrets with loose world-readable perms to prove the
	// orchestrator tightens them.
	testWriteFile(t, filepath.Join(dir, ".env.secrets"), "# pre-existing\n", 0644)

	cfg := cfgWithJWT()

	if err := persistGeneratedSecrets(dir, cfg); err != nil {
		t.Fatalf("persistGeneratedSecrets: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	got := info.Mode().Perm()
	if got != 0600 {
		t.Fatalf("expected .env.secrets mode 0600, got %o", got)
	}
}

// TestPersistGeneratedSecrets_JWTSkippedWhenAlreadyInEnv verifies no JWT
// append happens when .env already defines the key.
func TestPersistGeneratedSecrets_JWTSkippedWhenAlreadyInEnv(t *testing.T) {
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", "")

	dir := t.TempDir()
	existingJSON := `{"type":"HS256","key":"already-defined"}`
	testWriteFile(t, filepath.Join(dir, ".env"),
		"HASURA_GRAPHQL_JWT_SECRET="+existingJSON+"\n", 0600)

	cfg := cfgWithJWT()

	if err := persistGeneratedSecrets(dir, cfg); err != nil {
		t.Fatalf("persistGeneratedSecrets: %v", err)
	}

	// .env.secrets should either not exist or not contain a JWT line,
	// because the key was already defined in .env.
	data, err := os.ReadFile(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		// File not created — that's fine, means no candidates needed writing.
		return
	}
	if strings.Contains(string(data), "HASURA_GRAPHQL_JWT_SECRET=") {
		t.Fatalf("did not expect JWT re-write; contents:\n%s", string(data))
	}
}

// TestPersistGeneratedSecrets_JWTSkippedWhenAlreadyInSecrets verifies no JWT
// append happens when .env.secrets already defines the key.
func TestPersistGeneratedSecrets_JWTSkippedWhenAlreadyInSecrets(t *testing.T) {
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", "")

	dir := t.TempDir()
	testWriteFile(t, filepath.Join(dir, ".env"), "PROJECT_NAME=test\n", 0600)
	existingJSON := `{"type":"HS256","key":"already-in-secrets"}`
	testWriteFile(t, filepath.Join(dir, ".env.secrets"),
		"HASURA_GRAPHQL_JWT_SECRET="+existingJSON+"\n", 0600)

	cfg := cfgWithJWT()

	if err := persistGeneratedSecrets(dir, cfg); err != nil {
		t.Fatalf("persistGeneratedSecrets: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("read .env.secrets: %v", err)
	}
	if strings.Count(string(data), "HASURA_GRAPHQL_JWT_SECRET=") != 1 {
		t.Fatalf("expected exactly one JWT line; contents:\n%s", string(data))
	}
}
