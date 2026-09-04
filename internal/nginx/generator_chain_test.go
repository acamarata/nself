package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// newLocalSSLGenerator builds a generator in local-SSL mode rooted at dir.
func newLocalSSLGenerator(t *testing.T, dir string) *Generator {
	t.Helper()
	cfg := &config.Config{BaseDomain: "local.nself.org", SSLMode: "local"}
	return NewGenerator(cfg, dir)
}

// TestServiceRoute_OmitsTrustedChainWhenMissing is the regression guard for the
// bug the ntask clean-fork self-host drill found on 2026-08-24: the template
// emitted ssl_trusted_certificate .../chain.pem unconditionally, but `nself
// build` writes only fullchain.pem and privkey.pem. nginx treats a missing file
// there as fatal, so every fresh project failed to start with
// "cannot load certificate .../chain.pem: BIO_new_file() failed".
func TestServiceRoute_OmitsTrustedChainWhenMissing(t *testing.T) {
	dir := t.TempDir()
	g := newLocalSSLGenerator(t, dir)

	out, err := g.RenderServiceRoute(ServiceRouteData{
		Route:      "api",
		BaseDomain: "local.nself.org",
		Upstream:   "hasura:8080",
		SSLDir:     "local-nself-org",
	})
	if err != nil {
		t.Fatalf("RenderServiceRoute: %v", err)
	}

	if strings.Contains(out, "ssl_trusted_certificate") {
		t.Error("emitted ssl_trusted_certificate with no chain.pem on disk — nginx will refuse to start")
	}
	if strings.Contains(out, "ssl_stapling") {
		t.Error("emitted ssl_stapling without a trusted chain; stapling needs one")
	}
	// The certificate that DOES exist must still be wired up.
	if !strings.Contains(out, "ssl_certificate ") || !strings.Contains(out, "fullchain.pem") {
		t.Error("dropped the ssl_certificate directive along with the chain")
	}
}

// TestServiceRoute_EmitsTrustedChainWhenPresent proves the directive is not
// simply deleted: a real chain (a Let's Encrypt lineage, say) still gets
// stapling configured.
func TestServiceRoute_EmitsTrustedChainWhenPresent(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "ssl", "certificates", "local-nself-org")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "chain.pem"), []byte("-----BEGIN CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatalf("write chain.pem: %v", err)
	}

	g := newLocalSSLGenerator(t, dir)
	out, err := g.RenderServiceRoute(ServiceRouteData{
		Route:      "api",
		BaseDomain: "local.nself.org",
		Upstream:   "hasura:8080",
		SSLDir:     "local-nself-org",
	})
	if err != nil {
		t.Fatalf("RenderServiceRoute: %v", err)
	}

	if !strings.Contains(out, "ssl_trusted_certificate") {
		t.Error("chain.pem exists but ssl_trusted_certificate was not emitted")
	}
	if !strings.Contains(out, "ssl_stapling on") {
		t.Error("chain.pem exists but stapling was not enabled")
	}
}

// TestGenerate_RealBuildPathEmitsTrustedChainWhenPresent is the regression
// guard for a live-path gap found 2026-09-03 verifying P6-E11-W2-S1-T2:
// TestServiceRoute_EmitsTrustedChainWhenPresent above proves RenderServiceRoute
// sets HasTrustedChain correctly, but `nself build` never calls
// RenderServiceRoute directly — it calls Generate(), which calls
// generateAllRoutes(), which builds every nginx/sites/*.conf via its own
// propagation loop. That loop set HasSSL and UpstreamName on each route's
// data but never HasTrustedChain, so every route kept the Go zero value
// (false) regardless of whether a real chain.pem was on disk: a genuine CA
// chain (e.g. Let's Encrypt) placed in ssl/certificates/<dir>/chain.pem was
// silently never reflected in any file `nself build` actually writes. This
// test drives the real Generate() entrypoint, not RenderServiceRoute, so it
// would have caught the gap the isolated unit test above could not.
func TestGenerate_RealBuildPathEmitsTrustedChainWhenPresent(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "ssl", "certificates", "local-nself-org")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "chain.pem"), []byte("-----BEGIN CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatalf("write chain.pem: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "fullchain.pem"), []byte("-----BEGIN CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatalf("write fullchain.pem: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "privkey.pem"), []byte("-----BEGIN PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatalf("write privkey.pem: %v", err)
	}

	g := newLocalSSLGenerator(t, dir)
	files, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	hasura, ok := files["nginx/sites/hasura.conf"]
	if !ok {
		t.Fatal("Generate did not produce nginx/sites/hasura.conf (core route)")
	}
	if !strings.Contains(hasura, "ssl_trusted_certificate") {
		t.Error("real Generate() build path: chain.pem exists on disk but " +
			"nginx/sites/hasura.conf has no ssl_trusted_certificate — " +
			"generateAllRoutes() is not propagating HasTrustedChain")
	}
	if !strings.Contains(hasura, "ssl_stapling on") {
		t.Error("real Generate() build path: chain.pem exists but stapling was not enabled in hasura.conf")
	}
}

// TestGenerate_RealBuildPathOmitsTrustedChainWhenMissing is the companion
// no-chain case for the same real Generate() path, guarding against a fix to
// the present-case regressing the already-fixed absent-case (nginx must
// never be pointed at a chain.pem that does not exist).
func TestGenerate_RealBuildPathOmitsTrustedChainWhenMissing(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "ssl", "certificates", "local-nself-org")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "fullchain.pem"), []byte("-----BEGIN CERTIFICATE-----\n"), 0o600); err != nil {
		t.Fatalf("write fullchain.pem: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "privkey.pem"), []byte("-----BEGIN PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatalf("write privkey.pem: %v", err)
	}

	g := newLocalSSLGenerator(t, dir)
	files, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	hasura, ok := files["nginx/sites/hasura.conf"]
	if !ok {
		t.Fatal("Generate did not produce nginx/sites/hasura.conf (core route)")
	}
	if strings.Contains(hasura, "ssl_trusted_certificate") {
		t.Error("real Generate() build path: no chain.pem on disk but " +
			"nginx/sites/hasura.conf emitted ssl_trusted_certificate — nginx will refuse to start")
	}
}
