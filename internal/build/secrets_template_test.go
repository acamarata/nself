package build

// Purpose: Repro + regression tests for PCI compose-secret-templating
//          (2026-07-03): the generated docker-compose.yml must contain NO
//          literal secret values — only ${VAR} references resolved from
//          .nself/compose.env at container-start time.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/compose"
	"github.com/nself-org/cli/internal/config"
)

// secretTestConfig returns a config with distinctive secret values so literal
// leaks are unambiguous in assertions.
func secretTestConfig() *config.Config {
	cfg := &config.Config{
		ProjectName:   "sectest",
		BaseDomain:    "localhost",
		Env:           "dev",
		DockerNetwork: "sectest_network",
		Postgres: config.PostgresConfig{
			Host:     "postgres",
			User:     "postgres",
			Password: "pg-secret-Zx9Qw8Er7T",
			DB:       "nself",
			Port:     5432,
			Version:  "16",
		},
		Hasura: config.HasuraConfig{
			AdminSecret: "hasura-admin-K3v9Bn2Mp5",
			JWTKey:      "jwt-key-A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6",
			JWTType:     "HS256",
			Port:        8080,
			Version:     "v2.36.0",
			MemLimit:    "1g",
			CPULimit:    "1.0",
		},
		Auth: config.AuthConfig{
			Version:            "0.36.0",
			Port:               4000,
			ClientURL:          "http://localhost:3000",
			AccessTokenExpiry:  900,
			RefreshTokenExpiry: 2592000,
			SMTPHost:           "mailpit",
			SMTPPort:           1025,
			SMTPPass:           "smtp-pass-Qw4Rt6Yu8I",
			SMTPSender:         "noreply@localhost",
			MemLimit:           "256m",
			CPULimit:           "0.25",
		},
	}
	cfg.Minio.Enabled = true
	cfg.Minio.Version = "latest"
	cfg.Minio.Port = 9000
	cfg.Minio.ConsolePort = 9001
	cfg.Minio.RootUser = "minio-user-F7g8H9j0"
	cfg.Minio.RootPassword = "minio-pass-L5m6N7b8V9"
	cfg.Minio.MemLimit = "1G"
	cfg.Minio.CPULimit = "0.5"
	return cfg
}

// TestTemplateSecrets_NoLiteralSecretsInGeneratedCompose is the PCI repro:
// run the real compose generator, apply secret templating, and assert the
// output contains only ${VAR} references — zero literal secret values.
func TestTemplateSecrets_NoLiteralSecretsInGeneratedCompose(t *testing.T) {
	cfg := secretTestConfig()
	raw, err := compose.NewGenerator(cfg).Generate()
	if err != nil {
		t.Fatalf("compose generation failed: %v", err)
	}

	secrets := SecretEnvMap(cfg)
	templated := TemplateSecrets(raw, secrets)
	doc := string(templated)

	// No secret literal may survive.
	for name, val := range secrets {
		if strings.Contains(doc, val) {
			t.Errorf("literal secret %s (%q) still present in generated compose", name, val)
		}
	}
	if leaks := LiteralSecretLeaks(templated, secrets); len(leaks) != 0 {
		t.Errorf("LiteralSecretLeaks reported %v, want none", leaks)
	}

	// The ${VAR} references must be present instead.
	for _, ref := range []string{
		"POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}",
		"HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}",
		"HASURA_GRAPHQL_JWT_SECRET: ${HASURA_GRAPHQL_JWT_SECRET}",
		"MINIO_ROOT_USER: ${MINIO_ROOT_USER}",
		"MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}",
		"AUTH_JWT_SECRET: ${HASURA_JWT_KEY}",
		":${POSTGRES_PASSWORD}@", // DATABASE_URL credential position
	} {
		if !strings.Contains(doc, ref) {
			t.Errorf("expected templated reference %q in generated compose", ref)
		}
	}
}

// TestTemplateSecrets_AliasKeyLeftAloneWhenValueDiffers ensures alias env keys
// carrying an independent value are never clobbered.
func TestTemplateSecrets_AliasKeyLeftAloneWhenValueDiffers(t *testing.T) {
	yaml := "services:\n  auth:\n    environment:\n      AUTH_JWT_SECRET: some-other-independent-value\n"
	secrets := map[string]string{"HASURA_JWT_KEY": "jwt-key-A1b2C3d4E5f6"}
	out := string(TemplateSecrets([]byte(yaml), secrets))
	if !strings.Contains(out, "AUTH_JWT_SECRET: some-other-independent-value") {
		t.Errorf("alias key with independent value was clobbered:\n%s", out)
	}
}

// TestTemplateSecrets_URLEscapedPasswordLeftLiteral: passwords that require
// percent-encoding must not be substituted into URL credential positions
// (compose interpolation would inject the raw, unescaped value).
func TestTemplateSecrets_URLEscapedPasswordLeftLiteral(t *testing.T) {
	pw := "has space%pw"
	yaml := "      DATABASE_URL: postgresql://postgres:has%20space%25pw@postgres:5432/db\n"
	secrets := map[string]string{"POSTGRES_PASSWORD": pw}
	out := string(TemplateSecrets([]byte(yaml), secrets))
	if strings.Contains(out, ":${POSTGRES_PASSWORD}@") {
		t.Errorf("URL-escaped password must not be substituted, got:\n%s", out)
	}
}

func TestWriteComposeEnv_ContentAndPermissions(t *testing.T) {
	workdir := t.TempDir()
	cfg := secretTestConfig()
	secrets := SecretEnvMap(cfg)
	plugins := map[string]string{"NSELF_PLUGIN_DIR": "/tmp/plugins"}

	if err := WriteComposeEnv(workdir, cfg, secrets, plugins); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(workdir, ".nself", "compose.env")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows os.Chmod cannot represent POSIX owner-only bits (NTFS ACLs, not
	// mode bits) — same exception as internal/build/hasura_config_test.go.
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm != 0o600 {
		t.Errorf("compose.env permissions = %o, want 0600", perm)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"POSTGRES_PASSWORD=" + cfg.Postgres.Password,
		"HASURA_GRAPHQL_ADMIN_SECRET=" + cfg.Hasura.AdminSecret,
		"DOCKER_NETWORK=sectest_network",
		"NSELF_PLUGIN_DIR=/tmp/plugins",
		"HASURA_GRAPHQL_ENABLE_CONSOLE=false",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("compose.env missing %q", want)
		}
	}
}

func TestComposeEnvFiles_OrderAndFallback(t *testing.T) {
	// Legacy project: no compose.env → nil (docker compose default discovery).
	legacy := t.TempDir()
	if err := os.WriteFile(filepath.Join(legacy, ".env"), []byte("A=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ComposeEnvFiles(legacy); got != nil {
		t.Errorf("legacy project: expected nil env files, got %v", got)
	}

	// Templated project: .env + .nself/compose.env, compose.env last (wins).
	templated := t.TempDir()
	if err := os.WriteFile(filepath.Join(templated, ".env"), []byte("A=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(templated, ".nself"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templated, ".nself", "compose.env"), []byte("B=2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := ComposeEnvFiles(templated)
	if len(got) != 2 {
		t.Fatalf("expected 2 env files, got %v", got)
	}
	if filepath.Base(got[0]) != ".env" || filepath.Base(got[1]) != "compose.env" {
		t.Errorf("env file order wrong: %v", got)
	}
}
