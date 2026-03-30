package compose

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// ── authPGAliasVars ───────────────────────────────────────────────────────────

// TestAuthPGAliasVars_Keys verifies that authPGAliasVars returns all six
// required AUTH_DB_* keys.
func TestAuthPGAliasVars_Keys(t *testing.T) {
	pg := config.PostgresConfig{
		Host:     "postgres",
		User:     "pguser",
		Password: "pgpass",
		DB:       "nself",
		Port:     5432,
	}

	vars := authPGAliasVars(pg)

	required := []string{
		"AUTH_DB_HOST",
		"AUTH_DB_PORT",
		"AUTH_DB_NAME",
		"AUTH_DB_USER",
		"AUTH_DB_PASSWORD",
		"AUTH_DB_URL",
	}
	for _, key := range required {
		val, ok := vars[key]
		if !ok {
			t.Errorf("authPGAliasVars missing key %q", key)
		} else if val == "" {
			t.Errorf("authPGAliasVars key %q is empty", key)
		}
	}
}

// TestAuthPGAliasVars_Values verifies that the returned values match the
// postgres config fields.
func TestAuthPGAliasVars_Values(t *testing.T) {
	pg := config.PostgresConfig{
		Host:     "postgres",
		User:     "pguser",
		Password: "pgpass",
		DB:       "mydb",
		Port:     5432,
	}

	vars := authPGAliasVars(pg)

	if vars["AUTH_DB_HOST"] != "postgres" {
		t.Errorf("AUTH_DB_HOST = %q, want %q", vars["AUTH_DB_HOST"], "postgres")
	}
	if vars["AUTH_DB_PORT"] != "5432" {
		t.Errorf("AUTH_DB_PORT = %q, want %q", vars["AUTH_DB_PORT"], "5432")
	}
	if vars["AUTH_DB_NAME"] != "mydb" {
		t.Errorf("AUTH_DB_NAME = %q, want %q", vars["AUTH_DB_NAME"], "mydb")
	}
	if vars["AUTH_DB_USER"] != "pguser" {
		t.Errorf("AUTH_DB_USER = %q, want %q", vars["AUTH_DB_USER"], "pguser")
	}
	if vars["AUTH_DB_PASSWORD"] != "pgpass" {
		t.Errorf("AUTH_DB_PASSWORD = %q, want %q", vars["AUTH_DB_PASSWORD"], "pgpass")
	}
}

// TestAuthPGAliasVars_DSN verifies that AUTH_DB_URL is a valid postgres DSN
// using the internal container address (postgres:5432).
func TestAuthPGAliasVars_DSN(t *testing.T) {
	pg := config.PostgresConfig{
		Host:     "postgres",
		User:     "admin",
		Password: "secret",
		DB:       "nself",
		Port:     5432,
	}

	vars := authPGAliasVars(pg)
	dsn := vars["AUTH_DB_URL"]

	if !strings.HasPrefix(dsn, "postgresql://") {
		t.Errorf("AUTH_DB_URL should start with 'postgresql://', got %q", dsn)
	}
	if !strings.Contains(dsn, "@postgres:5432/") {
		t.Errorf("AUTH_DB_URL should contain '@postgres:5432/', got %q", dsn)
	}
	if !strings.Contains(dsn, "/nself") {
		t.Errorf("AUTH_DB_URL should contain the database name 'nself', got %q", dsn)
	}
}

// TestAuthPGAliasVars_DSNAlwaysUsesInternalPort verifies that AUTH_DB_URL
// always uses port 5432 (the internal container port) regardless of
// the pg.Port field, because the compose network uses fixed service names.
func TestAuthPGAliasVars_DSNAlwaysUsesInternalPort(t *testing.T) {
	pg := config.PostgresConfig{
		Host:     "postgres",
		User:     "admin",
		Password: "secret",
		DB:       "nself",
		Port:     9999, // external/host port — should NOT appear in DSN
	}

	vars := authPGAliasVars(pg)
	dsn := vars["AUTH_DB_URL"]

	if !strings.Contains(dsn, ":5432/") {
		t.Errorf("AUTH_DB_URL must always use internal port 5432 in DSN, got %q", dsn)
	}
	if strings.Contains(dsn, "9999") {
		t.Errorf("AUTH_DB_URL must not use host port 9999 in DSN, got %q", dsn)
	}
}

// TestAuthPGAliasVars_PasswordURLEncoded verifies that special characters in
// the password are URL-encoded in the DSN (path-escaped) so the DSN is valid.
func TestAuthPGAliasVars_PasswordURLEncoded(t *testing.T) {
	pg := config.PostgresConfig{
		Host:     "postgres",
		User:     "admin",
		Password: "pass@word!",
		DB:       "nself",
		Port:     5432,
	}

	vars := authPGAliasVars(pg)
	dsn := vars["AUTH_DB_URL"]

	// The raw "@" character in the password would break DSN parsing.
	// After URL encoding it should appear as %40.
	if strings.Contains(dsn, ":pass@word!@") {
		t.Errorf("AUTH_DB_URL contains unescaped special chars in password: %q", dsn)
	}
}

// ── computeAllowedRedirectURLs ────────────────────────────────────────────────

// TestComputeAllowedRedirectURLs_BaseURLs verifies that the result always
// includes both the exact base domain and the wildcard subdomain variant.
func TestComputeAllowedRedirectURLs_BaseURLs(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		Auth: config.AuthConfig{
			ClientURL: "http://localhost:3000",
		},
	}

	result := computeAllowedRedirectURLs(cfg)

	want := []string{
		"https://example.com",
		"https://*.example.com",
	}
	for _, u := range want {
		if !strings.Contains(result, u) {
			t.Errorf("computeAllowedRedirectURLs missing %q; got: %s", u, result)
		}
	}
}

// TestComputeAllowedRedirectURLs_ClientURL verifies that AUTH_CLIENT_URL is
// included in the allowed list.
func TestComputeAllowedRedirectURLs_ClientURL(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		Auth: config.AuthConfig{
			ClientURL: "http://localhost:3000",
		},
	}

	result := computeAllowedRedirectURLs(cfg)

	if !strings.Contains(result, "http://localhost:3000") {
		t.Errorf("computeAllowedRedirectURLs missing ClientURL; got: %s", result)
	}
}

// TestComputeAllowedRedirectURLs_ExtraURLs verifies that
// AUTH_EXTRA_REDIRECT_URLS are appended to the result.
func TestComputeAllowedRedirectURLs_ExtraURLs(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		Auth: config.AuthConfig{
			ClientURL:         "http://localhost:3000",
			ExtraRedirectURLs: "https://myapp.io,https://partner.com",
		},
	}

	result := computeAllowedRedirectURLs(cfg)

	for _, u := range []string{"https://myapp.io", "https://partner.com"} {
		if !strings.Contains(result, u) {
			t.Errorf("computeAllowedRedirectURLs missing extra URL %q; got: %s", u, result)
		}
	}
}

// TestComputeAllowedRedirectURLs_NoDuplicates verifies that duplicate URLs
// are deduplicated in the final result.
func TestComputeAllowedRedirectURLs_NoDuplicates(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		Auth: config.AuthConfig{
			ClientURL:         "https://example.com",
			ExtraRedirectURLs: "https://example.com",
		},
	}

	result := computeAllowedRedirectURLs(cfg)
	parts := strings.Split(result, ",")

	seen := map[string]int{}
	for _, p := range parts {
		seen[strings.TrimSpace(p)]++
	}
	for u, count := range seen {
		if count > 1 {
			t.Errorf("computeAllowedRedirectURLs contains duplicate URL %q (%d times)", u, count)
		}
	}
}

// TestComputeAllowedRedirectURLs_FrontendApps verifies that frontend app routes
// are included as full HTTPS URLs.
func TestComputeAllowedRedirectURLs_FrontendApps(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		Auth: config.AuthConfig{
			ClientURL: "http://localhost:3000",
		},
		FrontendApps: []config.FrontendApp{
			{Route: "app"},
			{Route: "dashboard"},
		},
	}

	result := computeAllowedRedirectURLs(cfg)

	for _, u := range []string{"https://app.example.com", "https://dashboard.example.com"} {
		if !strings.Contains(result, u) {
			t.Errorf("computeAllowedRedirectURLs missing frontend app URL %q; got: %s", u, result)
		}
	}
}

// ── buildAuthService integration ─────────────────────────────────────────────

// TestBuildAuthService_Port4000 verifies that the auth service host mapping uses
// Auth.Port and maps to the container's internal port 4002.
func TestBuildAuthService_Port4000(t *testing.T) {
	cfg := minimalConfig() // Auth.Port = 4000

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	found := false
	for _, p := range svc.Ports {
		if strings.Contains(p, "4000") && strings.Contains(p, "4002") {
			found = true
		}
		if strings.Contains(p, "8080") {
			t.Errorf("auth service port mapping must not reference 8080 (Hasura port), got: %s", p)
		}
	}
	if !found {
		t.Errorf("auth service Ports must map host 4000 to container 4002; got: %v", svc.Ports)
	}
}

// TestBuildAuthService_HealthcheckPort4002 verifies that the healthcheck URL
// uses port 4002 (nhost/hasura-auth internal port) and not port 8080 (Hasura).
func TestBuildAuthService_HealthcheckPort4002(t *testing.T) {
	cfg := minimalConfig()

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	if svc.Healthcheck == nil {
		t.Fatal("auth service has no healthcheck")
	}

	hcStr := strings.Join(svc.Healthcheck.Test, " ")

	if !strings.Contains(hcStr, "4002") {
		t.Errorf("auth healthcheck must reference port 4002, got: %s", hcStr)
	}
	if strings.Contains(hcStr, "8080") {
		t.Errorf("auth healthcheck must not reference port 8080 (Hasura), got: %s", hcStr)
	}
}

// TestBuildAuthService_HealthcheckHardcoded4002 verifies that the healthcheck
// uses the hardcoded container port 4002 even when AUTH_PORT is set to a
// different value. nhost/hasura-auth always binds on 4002 internally — the
// AUTH_PORT env var controls only the host-side port mapping.
func TestBuildAuthService_HealthcheckHardcoded4002(t *testing.T) {
	cfg := minimalConfig()
	cfg.Auth.Port = 4001 // non-default host port

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	if svc.Healthcheck == nil {
		t.Fatal("auth service has no healthcheck")
	}

	hcStr := strings.Join(svc.Healthcheck.Test, " ")

	if !strings.Contains(hcStr, "4002") {
		t.Errorf("auth healthcheck must always use container port 4002, got: %s", hcStr)
	}
	if strings.Contains(hcStr, "4001") {
		t.Errorf("auth healthcheck must not use host AUTH_PORT 4001, got: %s", hcStr)
	}

	// Port mapping must be host:container = 4001:4002
	found := false
	for _, p := range svc.Ports {
		if strings.Contains(p, "4001") && strings.Contains(p, "4002") {
			found = true
		}
	}
	if !found {
		t.Errorf("auth port mapping must be host:4001->container:4002; got: %v", svc.Ports)
	}
}
