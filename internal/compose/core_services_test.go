package compose

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// minimalAuthConfig returns a Config with the minimum required fields set so
// buildAuthService (and the helpers it calls) won't produce zero-value
// surprises.  It does NOT call ApplyDefaults so tests that need specific
// values can control exactly what ends up in the struct.
func minimalConfig() *config.Config {
	return &config.Config{
		ProjectName:   "test",
		BaseDomain:    "localhost",
		Env:           "dev",
		DockerNetwork: "test_network",
		Postgres: config.PostgresConfig{
			Host:     "postgres",
			User:     "postgres",
			Password: "password",
			DB:       "nself",
			Port:     5432,
		},
		Hasura: config.HasuraConfig{
			AdminSecret: "secret",
			JWTKey:      "testsecretkey12345678901234567890",
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
			SMTPSender:         "noreply@localhost",
			MemLimit:           "256m",
			CPULimit:           "0.25",
		},
	}
}

// TestAuthHealthcheckPort verifies T02: the healthcheck URL must use Auth.Port
// (nhost/hasura-auth binds on AUTH_PORT), not 8080 (which belongs to Hasura).
func TestAuthHealthcheckPort(t *testing.T) {
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

	// minimalConfig sets Auth.Port=4000; healthcheck must match the listening port.
	if !strings.Contains(hcStr, "4000") {
		t.Errorf("expected healthcheck URL to contain '4000' (Auth.Port), got: %s", hcStr)
	}
	if strings.Contains(hcStr, "8080") {
		t.Errorf("healthcheck URL must NOT contain '8080' (that is Hasura's port), got: %s", hcStr)
	}
}

// TestAuthResourceLimits_Default verifies T03 default path: ApplyDefaults sets
// Auth.MemLimit = "256m" and Auth.CPULimit = "0.25", and buildAuthService uses them.
func TestAuthResourceLimits_Default(t *testing.T) {
	cfg := &config.Config{
		ProjectName:   "test",
		BaseDomain:    "localhost",
		Env:           "dev",
		DockerNetwork: "test_network",
		Postgres: config.PostgresConfig{
			Host:     "postgres",
			User:     "postgres",
			Password: "password",
			DB:       "nself",
			Port:     5432,
		},
		Hasura: config.HasuraConfig{
			AdminSecret: "secret",
			JWTKey:      "testsecretkey12345678901234567890",
			JWTType:     "HS256",
		},
	}
	var applyErr error
	cfg, applyErr = config.ApplyDefaults(cfg)
	if applyErr != nil {
		t.Fatalf("ApplyDefaults() error: %v", applyErr)
	}

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	if svc.Deploy == nil || svc.Deploy.Resources == nil || svc.Deploy.Resources.Limits == nil {
		t.Fatal("auth service has no deploy resource limits")
	}

	limits := svc.Deploy.Resources.Limits
	if limits.Memory != "256m" {
		t.Errorf("expected default Auth MemLimit '256m', got %q", limits.Memory)
	}
	if limits.CPUs != "0.25" {
		t.Errorf("expected default Auth CPULimit '0.25', got %q", limits.CPUs)
	}
}

// TestAuthResourceLimits_Override verifies T03 override path: setting
// cfg.Auth.MemLimit = "512m" results in that value in the compose service.
func TestAuthResourceLimits_Override(t *testing.T) {
	cfg := minimalConfig()
	cfg.Auth.MemLimit = "512m"

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	if svc.Deploy == nil || svc.Deploy.Resources == nil || svc.Deploy.Resources.Limits == nil {
		t.Fatal("auth service has no deploy resource limits")
	}

	if svc.Deploy.Resources.Limits.Memory != "512m" {
		t.Errorf("expected Auth MemLimit '512m', got %q", svc.Deploy.Resources.Limits.Memory)
	}
}

// TestAuthResourceLimits_IndependentFromHasura verifies T03 independence: Hasura
// and Auth resource limits are distinct values in the generated compose.
func TestAuthResourceLimits_IndependentFromHasura(t *testing.T) {
	cfg := minimalConfig()
	cfg.Hasura.MemLimit = "1g"
	cfg.Auth.MemLimit = "256m"

	g := NewGenerator(cfg)
	hasuraSvc, err := g.buildHasuraService()
	if err != nil {
		t.Fatalf("buildHasuraService() error: %v", err)
	}
	authSvc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	if hasuraSvc.Deploy == nil || hasuraSvc.Deploy.Resources == nil || hasuraSvc.Deploy.Resources.Limits == nil {
		t.Fatal("hasura service has no deploy resource limits")
	}
	if authSvc.Deploy == nil || authSvc.Deploy.Resources == nil || authSvc.Deploy.Resources.Limits == nil {
		t.Fatal("auth service has no deploy resource limits")
	}

	hasuraLimit := hasuraSvc.Deploy.Resources.Limits.Memory
	authLimit := authSvc.Deploy.Resources.Limits.Memory

	if hasuraLimit == authLimit {
		t.Errorf("Hasura and Auth have the same MemLimit (%q); they should be independent", authLimit)
	}

	if hasuraLimit != "1g" {
		t.Errorf("expected Hasura MemLimit '1g', got %q", hasuraLimit)
	}
	if authLimit != "256m" {
		t.Errorf("expected Auth MemLimit '256m', got %q", authLimit)
	}
}

// TestHasuraRemoteSchemaPassthrough verifies that REMOTE_SCHEMA_* and HASURA_EXTRA_*
// vars from cfg.Passthrough are forwarded to the Hasura container environment.
// These vars are required for Hasura's url_from_env Remote Schema feature.
func TestHasuraRemoteSchemaPassthrough(t *testing.T) {
	cfg := minimalConfig()
	cfg.Passthrough = map[string]string{
		"REMOTE_SCHEMA_UMMAPP_URL":    "http://host.docker.internal:3043/api/graphql",
		"REMOTE_SCHEMA_UMMPRO_URL":    "http://host.docker.internal:3044/api/graphql",
		"HASURA_EXTRA_FOO":            "bar",
		"AUTH_PROVIDER_GITHUB_SECRET": "should-not-appear",
	}

	g := NewGenerator(cfg)
	svc, err := g.buildHasuraService()
	if err != nil {
		t.Fatalf("buildHasuraService() error: %v", err)
	}

	for _, key := range []string{"REMOTE_SCHEMA_UMMAPP_URL", "REMOTE_SCHEMA_UMMPRO_URL", "HASURA_EXTRA_FOO"} {
		if _, ok := svc.Environment[key]; !ok {
			t.Errorf("expected %q in Hasura environment, not found", key)
		}
	}
	if _, ok := svc.Environment["AUTH_PROVIDER_GITHUB_SECRET"]; ok {
		t.Errorf("AUTH_PROVIDER_* keys must not leak into Hasura environment")
	}
}

// TestAuthPortFromConfig verifies T04: cfg.Auth.Port controls both host and container ports.
// nhost/hasura-auth binds on the port set by AUTH_PORT env var (= cfg.Auth.Port).
// Setting Auth.Port = 4001 must produce "127.0.0.1:4001:4001".
func TestAuthPortFromConfig(t *testing.T) {
	cfg := minimalConfig()
	cfg.Auth.Port = 4001

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	const wantPort = "127.0.0.1:4001:4001"
	found := false
	for _, p := range svc.Ports {
		if p == wantPort {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected port mapping %q in auth service Ports, got: %v", wantPort, svc.Ports)
	}
}

// TestAuthPortDefault verifies that the default host port 4000 produces the expected mapping.
// nhost/hasura-auth binds on AUTH_PORT, so default mapping is 127.0.0.1:4000:4000.
func TestAuthPortDefault(t *testing.T) {
	cfg := minimalConfig() // Auth.Port = 4000

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	const wantPort = "127.0.0.1:4000:4000"
	found := false
	for _, p := range svc.Ports {
		if p == wantPort {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected default port mapping %q in auth service Ports, got: %v", wantPort, svc.Ports)
	}
}

// TestAuthHealthcheckUsesConfigPort verifies that the healthcheck URL uses cfg.Auth.Port.
// nhost/hasura-auth binds on the AUTH_PORT env var — healthcheck must match.
func TestAuthHealthcheckUsesConfigPort(t *testing.T) {
	cfg := minimalConfig()
	cfg.Auth.Port = 4001

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	if svc.Healthcheck == nil {
		t.Fatal("auth service has no healthcheck")
	}

	hcStr := strings.Join(svc.Healthcheck.Test, " ")
	if !strings.Contains(hcStr, "4001") {
		t.Errorf("healthcheck URL must use Auth.Port 4001, got: %s", hcStr)
	}
	if strings.Contains(hcStr, "4000") {
		t.Errorf("healthcheck URL must NOT use default port 4000 when Auth.Port=4001, got: %s", hcStr)
	}
}

// TestAuthAllowedRedirectURLs_BaseDomain verifies T07: AUTH_ALLOWED_REDIRECT_URLS
// contains both the base domain and wildcard subdomain.
func TestAuthAllowedRedirectURLs_BaseDomain(t *testing.T) {
	cfg := minimalConfig()
	cfg.BaseDomain = "example.com"
	cfg.Auth.ClientURL = "http://localhost:3000"

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	redirectURLs, ok := svc.Environment["AUTH_ALLOWED_REDIRECT_URLS"]
	if !ok {
		t.Fatal("auth service env missing AUTH_ALLOWED_REDIRECT_URLS")
	}

	want := []string{"https://example.com", "https://*.example.com"}
	for _, u := range want {
		if !strings.Contains(redirectURLs, u) {
			t.Errorf("AUTH_ALLOWED_REDIRECT_URLS missing %q; got: %s", u, redirectURLs)
		}
	}
}

// TestAuthAllowedRedirectURLs_ExtraURLs verifies that AUTH_EXTRA_REDIRECT_URLS
// appends to the computed list.
func TestAuthAllowedRedirectURLs_ExtraURLs(t *testing.T) {
	cfg := minimalConfig()
	cfg.BaseDomain = "example.com"
	cfg.Auth.ExtraRedirectURLs = "https://myapp.com"

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	redirectURLs, ok := svc.Environment["AUTH_ALLOWED_REDIRECT_URLS"]
	if !ok {
		t.Fatal("auth service env missing AUTH_ALLOWED_REDIRECT_URLS")
	}

	if !strings.Contains(redirectURLs, "https://myapp.com") {
		t.Errorf("AUTH_ALLOWED_REDIRECT_URLS missing extra URL %q; got: %s", "https://myapp.com", redirectURLs)
	}
}

// TestAuthPGAliasVars verifies T06: AUTH_DB_* alias vars are injected into
// the auth service environment so Nhost Auth can connect to PostgreSQL.
func TestAuthPGAliasVars(t *testing.T) {
	cfg := minimalConfig()

	g := NewGenerator(cfg)
	svc, err := g.buildAuthService()
	if err != nil {
		t.Fatalf("buildAuthService() error: %v", err)
	}

	required := []string{
		"AUTH_DB_HOST",
		"AUTH_DB_PORT",
		"AUTH_DB_NAME",
		"AUTH_DB_USER",
		"AUTH_DB_PASSWORD",
		"AUTH_DB_URL",
	}
	for _, key := range required {
		val, ok := svc.Environment[key]
		if !ok {
			t.Errorf("auth service env missing %q", key)
		} else if val == "" {
			t.Errorf("auth service env %q is empty", key)
		}
	}

	if host := svc.Environment["AUTH_DB_HOST"]; host != "postgres" {
		t.Errorf("AUTH_DB_HOST = %q, want %q", host, "postgres")
	}
	if port := svc.Environment["AUTH_DB_PORT"]; port != "5432" {
		t.Errorf("AUTH_DB_PORT = %q, want %q", port, "5432")
	}
	if name := svc.Environment["AUTH_DB_NAME"]; name != cfg.Postgres.DB {
		t.Errorf("AUTH_DB_NAME = %q, want %q", name, cfg.Postgres.DB)
	}
}
