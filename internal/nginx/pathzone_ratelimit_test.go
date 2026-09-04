package nginx

// pathzone_ratelimit_test.go — SEC-HARDENING-06 regression coverage
// (P6-E2-W2-S3-T20).
//
// Purpose: assert every service.conf.tmpl-rendered site conf carries the
// path-scoped /auth/login (auth_strict zone) and /api/ (api zone) rate-limit
// locations documented at internal/nginx/generator.go's
// defaultSecurityPathZones, ahead of the existing catch-all `location /`.
// Inputs: fixtures built the same way as the rest of this package's route
// tests (config.ApplyDefaults + a Generator).
// Outputs: pass/fail on the exact real Generate()/RenderServiceRoute() path
// used by `nself build`.
// Constraints: no live nginx binary here — that verification is separate
// (see ticket checkpoint for the `nginx -t` container run); this file only
// checks the generated text.

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// TestPathZones_AllRoutesCarryAuthAndAPILocations exercises the real
// Generate() path (not RenderServiceRoute in isolation) so it also covers
// core, optional, custom-service and internal-route entries built by
// routes_core.go/routes_app.go, matching how `nself build` actually calls
// this package.
func TestPathZones_AllRoutesCarryAuthAndAPILocations(t *testing.T) {
	cfg := &config.Config{BaseDomain: "example.com"}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	// A CS_N custom service is the concrete case the field report named
	// (cs-ping-api.conf / ir-ping.conf): a RateZone: "general" server that
	// also needs the stricter path-scoped zones layered on top.
	cfg.CustomServices = []config.CustomService{
		{Index: 1, Name: "ping-api", Port: 8001, Route: "ping"},
	}

	gen := NewGenerator(cfg, t.TempDir())
	files, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	target := "nginx/sites/cs-ping-api.conf"
	conf, ok := files[target]
	if !ok {
		var keys []string
		for k := range files {
			keys = append(keys, k)
		}
		t.Fatalf("expected %q in Generate() output; got keys: %v", target, keys)
	}

	assertPathZoneBlocks(t, target, conf)

	// The general catch-all zone must still be present and unmodified.
	if !strings.Contains(conf, "limit_req zone=general") {
		t.Errorf("%s: server-wide RateZone (general) location must still be present\ngot:\n%s", target, conf)
	}
}

// TestPathZones_RenderServiceRouteDirect covers the public
// RenderServiceRoute entry point (used by plugin/test callers outside
// generateAllRoutes) gets the same default.
func TestPathZones_RenderServiceRouteDirect(t *testing.T) {
	cfg := &config.Config{BaseDomain: "example.com"}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	gen := NewGenerator(cfg, t.TempDir())

	out, err := gen.RenderServiceRoute(ServiceRouteData{
		Route:      "svc",
		BaseDomain: "example.com",
		Upstream:   "svc:8080",
		SSLDir:     "example-com",
		RateZone:   "general",
		Burst:      10,
	})
	if err != nil {
		t.Fatalf("RenderServiceRoute() error: %v", err)
	}
	assertPathZoneBlocks(t, "RenderServiceRoute output", out)
}

// TestPathZones_ExplicitEmptyOverrideSuppressesDefault verifies a caller
// that explicitly passes a non-nil empty PathZones slice opts out of the
// default (documented on ServiceRouteData.PathZones) rather than silently
// always getting the blocks with no way to disable them.
func TestPathZones_ExplicitEmptyOverrideSuppressesDefault(t *testing.T) {
	cfg := &config.Config{BaseDomain: "example.com"}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	gen := NewGenerator(cfg, t.TempDir())

	out, err := gen.RenderServiceRoute(ServiceRouteData{
		Route:      "svc",
		BaseDomain: "example.com",
		Upstream:   "svc:8080",
		SSLDir:     "example-com",
		RateZone:   "general",
		PathZones:  []PathZone{},
	})
	if err != nil {
		t.Fatalf("RenderServiceRoute() error: %v", err)
	}
	if strings.Contains(out, "auth_strict") {
		t.Errorf("explicit empty PathZones must suppress the default auth_strict block\ngot:\n%s", out)
	}
}

// assertPathZoneBlocks asserts conf contains both required SEC-HARDENING-06
// locations, that they appear before the catch-all `location /` (nginx
// resolves prefix locations by longest match regardless of order, but
// ordering the more specific blocks first keeps the generated file
// readable and is asserted here so a future edit can't silently drop that
// convention), and that the doctor check's exact literal-string
// expectations (internal/doctor/hardening_check_nginx_zones.go) are met.
func assertPathZoneBlocks(t *testing.T, label, conf string) {
	t.Helper()

	for _, want := range []string{
		"location = /auth/login {",
		"limit_req zone=auth_strict burst=5 nodelay;",
		"location /api/ {",
		"limit_req zone=api burst=20 nodelay;",
	} {
		if !strings.Contains(conf, want) {
			t.Errorf("%s: missing %q\ngot:\n%s", label, want, conf)
		}
	}

	// Doctor check (SEC-HARDENING-06) greps for "/auth/login" + "limit_req"
	// and "/api/" + "limit_req" both present in the same file — assert that
	// exact contract here so a template refactor can't accidentally satisfy
	// this test while breaking the doctor check's grep.
	if strings.Contains(conf, "/auth/login") != strings.Contains(conf, "limit_req") {
		t.Errorf("%s: /auth/login and limit_req must co-occur (doctor check greps for both in the same file)", label)
	}

	authIdx := strings.Index(conf, "location = /auth/login")
	apiIdx := strings.Index(conf, "location /api/")
	catchAllIdx := strings.Index(conf, "location / {")
	if authIdx == -1 || apiIdx == -1 || catchAllIdx == -1 {
		t.Fatalf("%s: could not locate all three location blocks for ordering check", label)
	}
	if authIdx > catchAllIdx || apiIdx > catchAllIdx {
		t.Errorf("%s: path-scoped locations must be declared before the catch-all `location /` for readability (auth=%d api=%d catchAll=%d)", label, authIdx, apiIdx, catchAllIdx)
	}
}
