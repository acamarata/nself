package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckJWTSecretPresent_MissingEverywhere verifies a hard fail when
// HASURA_GRAPHQL_JWT_SECRET is not defined in any env file.
func TestCheckJWTSecretPresent_MissingEverywhere(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FOO=bar\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got := CheckJWTSecretPresent(dir)
	if got.Status != "fail" {
		t.Fatalf("expected fail, got %s (msg=%q)", got.Status, got.Message)
	}
	if got.Name != "jwt-secret-present" {
		t.Fatalf("expected name 'jwt-secret-present', got %q", got.Name)
	}
	if got.FixCmd == "" {
		t.Fatal("expected a fix_cmd for the failure case")
	}
}

// TestCheckJWTSecretPresent_InEnv verifies a pass when the key is in .env.
func TestCheckJWTSecretPresent_InEnv(t *testing.T) {
	dir := t.TempDir()
	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got := CheckJWTSecretPresent(dir)
	if got.Status != "pass" {
		t.Fatalf("expected pass, got %s (msg=%q)", got.Status, got.Message)
	}
}

// TestCheckJWTSecretPresent_OnlyInSecrets verifies a warn when the key is
// only in .env.secrets (the nSelf-preferred, but still-ok, arrangement).
func TestCheckJWTSecretPresent_OnlyInSecrets(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FOO=bar\n"), 0600); err != nil {
		t.Fatal(err)
	}
	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env.secrets"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got := CheckJWTSecretPresent(dir)
	if got.Status != "warn" {
		t.Fatalf("expected warn, got %s (msg=%q)", got.Status, got.Message)
	}
}

// TestCheckJWTSecretPresent_InLocal verifies a warn when the key is in
// .env.local (falls under "other env file" — configured, just not in .env).
func TestCheckJWTSecretPresent_InLocal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FOO=bar\n"), 0600); err != nil {
		t.Fatal(err)
	}
	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"xyz"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env.local"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got := CheckJWTSecretPresent(dir)
	if got.Status != "warn" {
		t.Fatalf("expected warn, got %s (msg=%q)", got.Status, got.Message)
	}
}

// TestCheckJWTSecretPresent_CommentIgnored verifies that a commented-out
// JWT secret line does not satisfy the check.
func TestCheckJWTSecretPresent_CommentIgnored(t *testing.T) {
	dir := t.TempDir()
	content := `# HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got := CheckJWTSecretPresent(dir)
	if got.Status != "fail" {
		t.Fatalf("commented line must not satisfy the check; got %s", got.Status)
	}
}

// TestEnvFileHasKey_MissingFile verifies a non-existent file is treated as
// "key not found" without erroring.
func TestEnvFileHasKey_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if envFileHasKey(filepath.Join(dir, "does-not-exist"), "FOO") {
		t.Fatal("expected false for non-existent file")
	}
}
