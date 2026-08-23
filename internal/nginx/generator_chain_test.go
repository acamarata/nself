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
