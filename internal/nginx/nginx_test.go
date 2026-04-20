package nginx

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// ── Route generation ─────────────────────────────────────────────────────────

// TestGenerateBasicConfig verifies that Generate() succeeds with a minimal
// configuration and that the output contains expected nginx server block tokens.
func TestGenerateBasicConfig(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Generate() returned no files")
	}

	// nginx.conf must be present and contain basic nginx tokens.
	mainConf, ok := files["nginx/nginx.conf"]
	if !ok {
		t.Fatal("nginx/nginx.conf not in output")
	}
	for _, want := range []string{"worker_processes", "http {", "server_names_hash_bucket_size"} {
		if !strings.Contains(mainConf, want) {
			t.Errorf("nginx.conf missing expected token %q", want)
		}
	}

	// Auth service route must exist.
	authConf, ok := files["nginx/sites/auth.conf"]
	if !ok {
		t.Fatal("nginx/sites/auth.conf not in output")
	}

	// server_name must follow the Route.BaseDomain pattern.
	if !strings.Contains(authConf, "server_name") {
		t.Errorf("auth.conf missing server_name directive")
	}
	if !strings.Contains(authConf, "server {") {
		t.Errorf("auth.conf missing server block")
	}

	t.Logf("Generate() produced %d files", len(files))
}

// ── Auth route server_name regression (S1-T01) ───────────────────────────────

// TestAuthRouteServerName verifies that the generated auth.conf contains
// "server_name auth.example.com" and NOT "auth.auth.example.com".
// This guards against the double-FQDN bug: AUTH_ROUTE stores only the
// subdomain prefix ("auth"), not the full FQDN.
func TestAuthRouteServerName(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	// Confirm the route is the bare subdomain.
	if cfg.Auth.Route != "auth" {
		t.Fatalf("Auth.Route = %q, expected bare subdomain %q", cfg.Auth.Route, "auth")
	}

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	authConf, ok := files["nginx/sites/auth.conf"]
	if !ok {
		t.Fatal("nginx/sites/auth.conf not generated")
	}

	want := "server_name auth.example.com"
	doubleFQDN := "server_name auth.auth.example.com"

	if !strings.Contains(authConf, want) {
		t.Errorf("auth.conf does not contain %q\ngot:\n%s", want, authConf)
	}
	if strings.Contains(authConf, doubleFQDN) {
		t.Errorf("double-FQDN bug detected: auth.conf contains %q — AUTH_ROUTE must be bare subdomain, not full FQDN", doubleFQDN)
	}

	t.Logf("server_name is correct: %s", want)
}

// ── Domain conflict detection ─────────────────────────────────────────────────

// TestHasDomainConflictDetectsMatch verifies HasDomainConflict returns true
// and conflict details when two routes share the same server_name.
func TestHasDomainConflictDetectsMatch(t *testing.T) {
	routes := []NginxRoute{
		{ServerName: "app.example.com", Location: "/"},
		{ServerName: "api.example.com", Location: "/"},
		{ServerName: "app.example.com", Location: "/"},
	}

	conflict, details := HasDomainConflict(routes)
	if !conflict {
		t.Fatal("HasDomainConflict() = false, expected true for duplicate domain")
	}
	if len(details) == 0 {
		t.Fatal("HasDomainConflict() returned no conflict details")
	}
	found := false
	for _, d := range details {
		if strings.Contains(d, "app.example.com") {
			found = true
		}
	}
	if !found {
		t.Errorf("conflict details do not mention the conflicting domain; got: %v", details)
	}

	t.Logf("conflict detected: %v", details)
}

// TestHasDomainConflictNoFalsePositive verifies HasDomainConflict returns false
// when all routes are unique.
func TestHasDomainConflictNoFalsePositive(t *testing.T) {
	routes := []NginxRoute{
		{ServerName: "app.example.com", Location: "/"},
		{ServerName: "api.example.com", Location: "/"},
		{ServerName: "auth.example.com", Location: "/"},
	}

	conflict, details := HasDomainConflict(routes)
	if conflict {
		t.Errorf("HasDomainConflict() = true on unique routes; details: %v", details)
	}
}

// TestFrontendAppDuplicateDomainConflict verifies that two frontend apps sharing
// the same Route value result in a domain conflict being detected at the
// HasDomainConflict level (the generator skips duplicates).
func TestFrontendAppDuplicateDomainConflict(t *testing.T) {
	routes := []NginxRoute{
		{ServerName: "app.example.com", Location: "/"},
		{ServerName: "app.example.com", Location: "/"},
	}

	conflict, _ := HasDomainConflict(routes)
	if !conflict {
		t.Fatal("expected domain conflict for two frontend apps with the same server_name")
	}
}

// ── Frontend app injection safety (S1-T07) ────────────────────────────────────

// TestFrontendAppServerNameSafe verifies that RenderServiceRoute produces
// a server_name that matches exactly the Route.BaseDomain combination and
// does not contain shell injection characters.
func TestFrontendAppServerNameSafe(t *testing.T) {
	// SystemName is used as filename; Route is used in server_name.
	// Only safe, sanitized route values should be accepted in the template.
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	gen := NewGenerator(cfg, t.TempDir())

	safeRoute := "myapp"
	data := ServiceRouteData{
		Route:      safeRoute,
		BaseDomain: "example.com",
		Upstream:   "host.docker.internal:3000",
		SSLDir:     "example-com",
		RateZone:   "static",
		WebSocket:  true,
		LazyResolve: true,
	}

	out, err := gen.RenderServiceRoute(data)
	if err != nil {
		t.Fatalf("RenderServiceRoute() error: %v", err)
	}

	wantServerName := "server_name myapp.example.com"
	if !strings.Contains(out, wantServerName) {
		t.Errorf("output does not contain %q\ngot:\n%s", wantServerName, out)
	}

	// Confirm no shell/HTTP injection characters appear in the domain portion
	// of the server_name line. Semicolons are valid nginx syntax terminators.
	// Dollar signs appear in other nginx variables ($host etc.) not in server_name values.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "server_name") {
			// Extract the domain value between "server_name " and ";"
			trimmed := strings.TrimSpace(line)
			trimmed = strings.TrimPrefix(trimmed, "server_name ")
			trimmed = strings.TrimSuffix(trimmed, ";")
			for _, bad := range []string{"`", "|", "&", "<", ">", "\n", "\r"} {
				if strings.Contains(trimmed, bad) {
					t.Errorf("server_name value contains injection character %q: %s", bad, trimmed)
				}
			}
			// Confirm the domain value is the expected safe value.
			if trimmed != "myapp.example.com" {
				t.Errorf("server_name value = %q, want %q", trimmed, "myapp.example.com")
			}
		}
	}

	t.Logf("server_name line is safe: %s", wantServerName)
}

// ── Template rendering ─────────────────────────────────────────────────────────

// TestAllTemplatesRenderWithMinimalConfig verifies that every embedded nginx
// template renders without error when given a minimal config struct.
func TestAllTemplatesRenderWithMinimalConfig(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	gen := NewGenerator(cfg, t.TempDir())
	if err := gen.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates() error: %v", err)
	}

	t.Run("nginx.conf.tmpl", func(t *testing.T) {
		out, err := gen.generateMainConf()
		if err != nil {
			t.Fatalf("generateMainConf() error: %v", err)
		}
		if len(out) == 0 {
			t.Error("nginx.conf.tmpl rendered empty output")
		}
	})

	t.Run("rate-limits.conf.tmpl", func(t *testing.T) {
		out, err := gen.generateRateLimits()
		if err != nil {
			t.Fatalf("generateRateLimits() error: %v", err)
		}
		if len(out) == 0 {
			t.Error("rate-limits.conf.tmpl rendered empty output")
		}
	})

	t.Run("default.conf.tmpl", func(t *testing.T) {
		out, err := gen.generateDefaultServer()
		if err != nil {
			t.Fatalf("generateDefaultServer() error: %v", err)
		}
		if len(out) == 0 {
			t.Error("default.conf.tmpl rendered empty output")
		}
	})

	t.Run("service.conf.tmpl", func(t *testing.T) {
		data := ServiceRouteData{
			Route:      "api",
			BaseDomain: "example.com",
			Upstream:   "hasura:8080",
			SSLDir:     "example-com",
			RateZone:   "graphql_api",
			Burst:      20,
			ConnLimit:  10,
			WebSocket:  true,
		}
		out, err := gen.RenderServiceRoute(data)
		if err != nil {
			t.Fatalf("RenderServiceRoute() error: %v", err)
		}
		if len(out) == 0 {
			t.Error("service.conf.tmpl rendered empty output")
		}
		_ = fmt.Sprintf("service route len=%d", len(out))
	})
}

// TestAuthRouteDefault verifies that ApplyDefaults sets Auth.Route to the bare
// subdomain "auth" (not a full FQDN), so the template expansion
// "{{.Route}}.{{.BaseDomain}}" correctly yields "auth.local.nself.org".
func TestAuthRouteDefault(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "local.nself.org",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if cfg.Auth.Route != "auth" {
		t.Errorf("Auth.Route = %q, want %q", cfg.Auth.Route, "auth")
	}

	// Simulate what the nginx template does: Route + "." + BaseDomain
	fqdn := cfg.Auth.Route + "." + cfg.BaseDomain
	want := "auth.local.nself.org"
	if fqdn != want {
		t.Errorf("constructed FQDN = %q, want %q", fqdn, want)
	}

	t.Logf("Auth.Route = %q, constructed FQDN = %q", cfg.Auth.Route, fqdn)
}

// TestAuthRouteNoDoubleDomain verifies the bug fix: auth route must NOT be the
// full FQDN, which would produce "auth.local.nself.org.local.nself.org" when
// combined with BaseDomain in the template.
func TestAuthRouteNoDoubleDomain(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "local.nself.org",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	// Correct: bare subdomain produces one level of domain.
	correct := cfg.Auth.Route + "." + cfg.BaseDomain
	doubled := cfg.Auth.Route + "." + cfg.BaseDomain + "." + cfg.BaseDomain

	if correct != "auth.local.nself.org" {
		t.Errorf("correct FQDN = %q, want %q", correct, "auth.local.nself.org")
	}
	if strings.Contains(correct, "local.nself.org.local.nself.org") {
		t.Errorf("double domain detected in %q — Auth.Route must be bare subdomain, got %q", correct, cfg.Auth.Route)
	}

	t.Logf("correct FQDN = %q", correct)
	t.Logf("would-be double = %q (not generated)", doubled)
}

// TestAuthRateLimitDefault verifies that without AUTH_RATE_LIMIT in the
// environment, the generated rate-limits.conf contains "30r/m" for the auth
// zone.
func TestAuthRateLimitDefault(t *testing.T) {
	// Ensure AUTH_RATE_LIMIT is not set in env.
	os.Unsetenv("AUTH_RATE_LIMIT")

	cfg := &config.Config{
		BaseDomain: "local.nself.org",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if cfg.Nginx.AuthRateLimit != "30r/m" {
		t.Fatalf("AuthRateLimit = %q, want %q", cfg.Nginx.AuthRateLimit, "30r/m")
	}

	gen := NewGenerator(cfg, t.TempDir())
	if err := gen.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	out, err := gen.generateRateLimits()
	if err != nil {
		t.Fatalf("generateRateLimits: %v", err)
	}

	if !strings.Contains(out, "30r/m") {
		t.Errorf("rate-limits.conf does not contain %q\ngot:\n%s", "30r/m", out)
	}

	t.Logf("auth zone rate = 30r/m confirmed")
}

// ── Optional routes ───────────────────────────────────────────────────────────

// TestOptionalRoutesGenerated verifies that optional service routes (Admin,
// Search, Mailpit, Functions) are included in the output when the services
// are enabled in the config.
func TestOptionalRoutesGenerated(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	cfg.Admin.Enabled = true
	cfg.Search.Enabled = true
	cfg.Mailpit.Enabled = true
	cfg.Functions.Enabled = true

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	for _, wantFile := range []string{
		"nginx/sites/admin.conf",
		"nginx/sites/search.conf",
		"nginx/sites/mail.conf",
		"nginx/sites/functions.conf",
	} {
		if _, ok := files[wantFile]; !ok {
			t.Errorf("expected %q in generated files, but it was missing", wantFile)
		}
	}
}

// TestCustomServiceRouteGenerated verifies that a custom service with a Route
// and Port produces a cs-{name}.conf in the nginx output.
func TestCustomServiceRouteGenerated(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	// Use a route that does not conflict with core routes (hasura defaults to "api").
	cfg.CustomServices = []config.CustomService{
		{Index: 1, Name: "myapi", Port: 8001, Route: "myapi"},
	}

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	wantFile := "nginx/sites/cs-myapi.conf"
	conf, ok := files[wantFile]
	if !ok {
		var keys []string
		for k := range files {
			keys = append(keys, k)
		}
		t.Fatalf("expected %q in generated files; keys: %v", wantFile, keys)
	}
	if !strings.Contains(conf, "server_name myapi.example.com") {
		t.Errorf("cs conf missing expected server_name; got:\n%s", conf)
	}
}

// TestCustomServiceRoute_SkippedWhenNoRoute verifies that a custom service
// without a Route (internal-only) does not produce an nginx conf file.
func TestCustomServiceRoute_SkippedWhenNoRoute(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.CustomServices = []config.CustomService{
		{Index: 1, Name: "internal-worker", Port: 9000, Route: ""},
	}

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	for k := range files {
		if strings.Contains(k, "internal-worker") {
			t.Errorf("expected no nginx conf for internal-only service, but found %q", k)
		}
	}
}

// TestInternalRouteGenerated verifies that an InternalRoute with Subdomain
// and Target produces an ir-{name}.conf in the nginx output.
func TestInternalRouteGenerated(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.InternalRoutes = []config.InternalRoute{
		{Index: 1, Name: "ping", Subdomain: "ping", Target: "http://ping-api:8001", RateZone: "general"},
	}

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	wantFile := "nginx/sites/ir-ping.conf"
	conf, ok := files[wantFile]
	if !ok {
		var keys []string
		for k := range files {
			keys = append(keys, k)
		}
		t.Fatalf("expected %q; keys: %v", wantFile, keys)
	}
	if !strings.Contains(conf, "server_name ping.example.com") {
		t.Errorf("ir conf missing expected server_name; got:\n%s", conf)
	}
}

// ---------------------------------------------------------------------------
// Bug #68 — nginx fails to start due to missing SSL certs on fresh install
//
// Root cause (v0.x): on fresh `nself init` + `nself build` + `nself start`,
// nginx crashed because SSL cert files didn't exist yet. The nginx config
// referenced ssl_certificate / ssl_certificate_key directives pointing to cert
// files that the SSL generator hadn't created yet.
//
// Go rewrite fix: NewGenerator sets hasSSL=true only when SSL_MODE is "local"
// (the default for local dev, where mkcert/OpenSSL generates certs as part of
// `nself build`). For all other SSL modes (letsencrypt, custom, none) hasSSL
// is false, so the generated nginx config omits ssl_certificate directives.
// Nginx can then start without cert files, and certs are provisioned after.
//
// These tests verify:
//  1. Local mode generates SSL directives (certs will be present from build).
//  2. Non-local modes omit SSL directives so nginx starts without cert files.
// ---------------------------------------------------------------------------

// TestNginxGenerator_LocalSSLMode_HasSSLTrue verifies that NewGenerator with
// SSL_MODE="" (empty, defaults to "local") sets hasSSL=true.
// In local mode, `nself build` always generates self-signed certs first, so
// it is safe for nginx to reference them.
func TestNginxGenerator_LocalSSLMode_HasSSLTrue(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		// SSLMode="" defaults to "local" in NewGenerator.
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.SSLMode = "" // explicit empty → local

	gen := NewGenerator(cfg, t.TempDir())
	if !gen.hasSSL {
		t.Error("Bug #68 regression: local SSL mode should set hasSSL=true")
	}
}

// TestNginxGenerator_LetsEncryptMode_HasSSLFalse verifies that when SSL_MODE
// is "letsencrypt", hasSSL is false so nginx does not reference cert files
// that certbot has not yet provisioned.
func TestNginxGenerator_LetsEncryptMode_HasSSLFalse(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		SSLMode:    "letsencrypt",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.SSLMode = "letsencrypt"

	gen := NewGenerator(cfg, t.TempDir())
	if gen.hasSSL {
		t.Error("Bug #68 regression: letsencrypt SSL mode must set hasSSL=false (no certs on first boot)")
	}
}

// TestNginxGenerator_CustomSSLMode_HasSSLFalse verifies that SSL_MODE="custom"
// does not inject ssl_certificate directives (user provides their own certs).
func TestNginxGenerator_CustomSSLMode_HasSSLFalse(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		SSLMode:    "custom",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.SSLMode = "custom"

	gen := NewGenerator(cfg, t.TempDir())
	if gen.hasSSL {
		t.Error("Bug #68 regression: custom SSL mode must set hasSSL=false")
	}
}

// TestNginxGenerator_NoneSSLMode_HasSSLFalse verifies that SSL_MODE="none"
// produces nginx configs without any SSL directives.
func TestNginxGenerator_NoneSSLMode_HasSSLFalse(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		SSLMode:    "none",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.SSLMode = "none"

	gen := NewGenerator(cfg, t.TempDir())
	if gen.hasSSL {
		t.Error("Bug #68 regression: SSL_MODE=none must set hasSSL=false")
	}
}

// TestNginxDefaultServer_LetsEncryptOmitsSSLCertDirs verifies that the
// generated default.conf does NOT contain ssl_certificate directives when
// SSL_MODE is letsencrypt — nginx must be able to start before certbot runs.
func TestNginxDefaultServer_LetsEncryptOmitsSSLCertDirs(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.SSLMode = "letsencrypt"

	gen := NewGenerator(cfg, t.TempDir())
	if err := gen.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}

	out, err := gen.generateDefaultServer()
	if err != nil {
		t.Fatalf("generateDefaultServer: %v", err)
	}

	if strings.Contains(out, "ssl_certificate ") {
		t.Errorf("Bug #68 regression: letsencrypt default.conf must not contain ssl_certificate directive\ngot:\n%s", out)
	}
}

// TestNginxDefaultServer_LocalModeIncludesSSLCertDirs verifies that the
// generated default.conf DOES contain ssl_certificate directives when
// SSL_MODE is "local" — because `nself build` always generates the certs
// before writing the compose stack.
func TestNginxDefaultServer_LocalModeIncludesSSLCertDirs(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	// Leave SSLMode="" to use local default.

	gen := NewGenerator(cfg, t.TempDir())
	if err := gen.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}

	out, err := gen.generateDefaultServer()
	if err != nil {
		t.Fatalf("generateDefaultServer: %v", err)
	}

	if !strings.Contains(out, "ssl_certificate") {
		t.Errorf("local SSL mode default.conf should contain ssl_certificate directive\ngot:\n%s", out)
	}
}

// ── TLS + HSTS verification (S43-T05) ────────────────────────────────────────

// TestNginxConf_TLSProtocols verifies that the generated nginx.conf contains
// ssl_protocols TLSv1.2 TLSv1.3 and explicitly excludes older protocols.
// Acceptance criterion for S43-T05.
func TestNginxConf_TLSProtocols(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	gen := NewGenerator(cfg, t.TempDir())
	if err := gen.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	out, err := gen.generateMainConf()
	if err != nil {
		t.Fatalf("generateMainConf: %v", err)
	}

	// Must contain TLS 1.2 and 1.3.
	if !strings.Contains(out, "TLSv1.2") {
		t.Errorf("nginx.conf missing TLSv1.2 in ssl_protocols directive\ngot:\n%s", out)
	}
	if !strings.Contains(out, "TLSv1.3") {
		t.Errorf("nginx.conf missing TLSv1.3 in ssl_protocols directive\ngot:\n%s", out)
	}

	// Must NOT enable legacy protocols.
	for _, bad := range []string{"SSLv3", "TLSv1.0", "TLSv1.1", "TLSv1 "} {
		if strings.Contains(out, bad) {
			t.Errorf("nginx.conf contains legacy protocol %q — must be excluded", bad)
		}
	}
}

// TestServiceConf_HSTS verifies that the generated service.conf (HTTPS vhost)
// contains a Strict-Transport-Security header with max-age >= 31536000.
// Uses local SSL mode (SSLMode="") so the generator's hasSSL=true, which
// causes the HSTS directive to be emitted. Acceptance criterion for S43-T05.
func TestServiceConf_HSTS(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
		// SSLMode="" defaults to "local" in NewGenerator → hasSSL=true.
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	// Leave SSLMode empty — local mode sets hasSSL=true, emitting HSTS.
	cfg.SSLMode = ""

	gen := NewGenerator(cfg, t.TempDir())
	if err := gen.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}

	data := ServiceRouteData{
		Route:      "api",
		BaseDomain: "example.com",
		Upstream:   "hasura:8080",
		SSLDir:     "example-com",
	}
	out, err := gen.RenderServiceRoute(data)
	if err != nil {
		t.Fatalf("RenderServiceRoute: %v", err)
	}

	if !strings.Contains(out, "Strict-Transport-Security") {
		t.Errorf("service.conf missing Strict-Transport-Security header when hasSSL=true\ngot:\n%s", out)
	}
	if !strings.Contains(out, "includeSubDomains") {
		t.Errorf("service.conf HSTS missing includeSubDomains\ngot:\n%s", out)
	}
}

// TestAuthRateLimitOverride verifies that setting AUTH_RATE_LIMIT=10r/m causes
// the generated rate-limits.conf to use "10r/m" for the auth zone.
func TestAuthRateLimitOverride(t *testing.T) {
	os.Setenv("AUTH_RATE_LIMIT", "10r/m")
	t.Cleanup(func() { os.Unsetenv("AUTH_RATE_LIMIT") })

	cfg := &config.Config{
		BaseDomain: "local.nself.org",
		Nginx: config.NginxConfig{
			AuthRateLimit: "10r/m", // simulates env var being loaded into config
		},
	}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}

	if cfg.Nginx.AuthRateLimit != "10r/m" {
		t.Fatalf("AuthRateLimit = %q, want %q", cfg.Nginx.AuthRateLimit, "10r/m")
	}

	gen := NewGenerator(cfg, t.TempDir())
	if err := gen.parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	out, err := gen.generateRateLimits()
	if err != nil {
		t.Fatalf("generateRateLimits: %v", err)
	}

	if !strings.Contains(out, "10r/m") {
		t.Errorf("rate-limits.conf does not contain %q\ngot:\n%s", "10r/m", out)
	}

	// Also confirm the default 30r/m is NOT used for the auth zone line.
	// The auth zone line is: zone=auth:10m rate=<value>
	authLine := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "zone=auth:") {
			authLine = line
			break
		}
	}
	if authLine == "" {
		t.Fatal("auth zone line not found in rate-limits.conf")
	}
	if !strings.Contains(authLine, "rate=10r/m") {
		t.Errorf("auth zone line = %q, expected rate=10r/m", authLine)
	}

	t.Logf("auth zone line: %s", authLine)
	_ = fmt.Sprintf("AUTH_RATE_LIMIT override confirmed: %s", cfg.Nginx.AuthRateLimit)
}
