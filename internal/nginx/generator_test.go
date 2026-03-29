package nginx

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// ── Security headers ──────────────────────────────────────────────────────────

// TestServiceRouteSecurityHeaders verifies that the rendered service.conf.tmpl
// includes all required security headers.
func TestServiceRouteSecurityHeaders(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg = config.ApplyDefaults(cfg)

	gen := NewGenerator(cfg, t.TempDir())

	data := ServiceRouteData{
		Route:      "api",
		BaseDomain: "example.com",
		Upstream:   "hasura:8080",
		SSLDir:     "example-com",
		RateZone:   "graphql_api",
	}

	out, err := gen.RenderServiceRoute(data)
	if err != nil {
		t.Fatalf("RenderServiceRoute() error: %v", err)
	}

	requiredHeaders := []string{
		`add_header X-Frame-Options "SAMEORIGIN" always`,
		`add_header X-Content-Type-Options "nosniff" always`,
		`add_header X-XSS-Protection "0" always`,
		`add_header Referrer-Policy "strict-origin-when-cross-origin" always`,
		`add_header Permissions-Policy "camera=(), microphone=(), geolocation=()" always`,
		`add_header Content-Security-Policy`,
		`add_header Strict-Transport-Security`,
		`server_tokens off`,
	}

	for _, header := range requiredHeaders {
		if !strings.Contains(out, header) {
			t.Errorf("service.conf.tmpl missing security directive: %q\ngot:\n%s", header, out)
		}
	}
}

// TestServiceRouteProxyHeaders verifies that the rendered service.conf.tmpl
// includes required reverse-proxy headers passed to the upstream.
func TestServiceRouteProxyHeaders(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg = config.ApplyDefaults(cfg)

	gen := NewGenerator(cfg, t.TempDir())

	data := ServiceRouteData{
		Route:      "auth",
		BaseDomain: "example.com",
		Upstream:   "auth:4000",
		SSLDir:     "example-com",
	}

	out, err := gen.RenderServiceRoute(data)
	if err != nil {
		t.Fatalf("RenderServiceRoute() error: %v", err)
	}

	proxyHeaders := []string{
		"proxy_set_header X-Real-IP",
		"proxy_set_header X-Forwarded-For",
		"proxy_set_header X-Forwarded-Proto",
		"proxy_set_header Host",
	}

	for _, h := range proxyHeaders {
		if !strings.Contains(out, h) {
			t.Errorf("service.conf.tmpl missing proxy header: %q\ngot:\n%s", h, out)
		}
	}
}

// TestServiceRouteHideHeaders verifies that the rendered service.conf.tmpl
// includes proxy_hide_header directives to strip upstream-leaked headers.
func TestServiceRouteHideHeaders(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg = config.ApplyDefaults(cfg)

	gen := NewGenerator(cfg, t.TempDir())

	data := ServiceRouteData{
		Route:      "api",
		BaseDomain: "example.com",
		Upstream:   "hasura:8080",
		SSLDir:     "example-com",
	}

	out, err := gen.RenderServiceRoute(data)
	if err != nil {
		t.Fatalf("RenderServiceRoute() error: %v", err)
	}

	for _, header := range []string{"X-Powered-By", "Server"} {
		directive := "proxy_hide_header " + header
		if !strings.Contains(out, directive) {
			t.Errorf("service.conf.tmpl missing hide-header directive for %q\ngot:\n%s", header, out)
		}
	}
}

// TestGeneratedAuthConfSecurityHeaders verifies that the full Generate() output
// for auth.conf includes security headers.
func TestGeneratedAuthConfSecurityHeaders(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg = config.ApplyDefaults(cfg)

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	authConf, ok := files["nginx/sites/auth.conf"]
	if !ok {
		t.Fatal("nginx/sites/auth.conf not in output")
	}

	for _, header := range []string{
		`add_header X-Frame-Options`,
		`add_header X-Content-Type-Options`,
		`server_tokens off`,
	} {
		if !strings.Contains(authConf, header) {
			t.Errorf("auth.conf missing security directive %q\ngot:\n%s", header, authConf)
		}
	}
}

// TestGeneratedAuthConfProxyHeaders verifies that auth.conf includes the
// required proxy headers for upstream communication.
func TestGeneratedAuthConfProxyHeaders(t *testing.T) {
	cfg := &config.Config{
		BaseDomain: "example.com",
	}
	cfg = config.ApplyDefaults(cfg)

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	authConf, ok := files["nginx/sites/auth.conf"]
	if !ok {
		t.Fatal("nginx/sites/auth.conf not in output")
	}

	for _, header := range []string{
		"proxy_set_header X-Real-IP",
		"proxy_set_header X-Forwarded-For",
	} {
		if !strings.Contains(authConf, header) {
			t.Errorf("auth.conf missing proxy header %q\ngot:\n%s", header, authConf)
		}
	}
}

// TestSecurityHeadersConstant verifies the securityHeaders constant itself
// contains all expected directives, so any accidental edit is caught.
func TestSecurityHeadersConstant(t *testing.T) {
	expected := []string{
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Permissions-Policy",
		"server_tokens off",
	}
	for _, e := range expected {
		if !strings.Contains(securityHeaders, e) {
			t.Errorf("securityHeaders constant missing %q", e)
		}
	}
}

// TestProxyHideHeaderDirectives verifies proxyHideHeaderDirectives() produces
// directives for all headers in upstreamHideHeaders.
func TestProxyHideHeaderDirectives(t *testing.T) {
	directives := proxyHideHeaderDirectives()

	for _, h := range upstreamHideHeaders {
		want := "proxy_hide_header " + h
		if !strings.Contains(directives, want) {
			t.Errorf("proxyHideHeaderDirectives() missing %q\ngot:\n%s", want, directives)
		}
	}
}
