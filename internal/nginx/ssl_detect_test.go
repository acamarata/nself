package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// ssl_detect_test.go — cli#385: proves generated per-service site confs
// emit a `listen 443 ssl` block whenever a certificate pair is actually on
// disk, regardless of SSL_MODE, while preserving the pre-existing Bug #68
// guarantees (nginx_test.go) that "local" is always true and "none" is
// always false.

// writeCertPair writes a minimal fullchain.pem + privkey.pem pair under
// workdir/ssl/certificates/<sslDirName(domain)>/, matching the layout
// internal/ssl.Generator writes to.
func writeCertPair(t *testing.T, workdir, domain string) {
	t.Helper()
	certDir := filepath.Join(workdir, "ssl", "certificates", sslDirName(domain))
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", certDir, err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "fullchain.pem"), []byte("fake-fullchain"), 0o644); err != nil {
		t.Fatalf("write fullchain.pem: %v", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "privkey.pem"), []byte("fake-privkey"), 0o600); err != nil {
		t.Fatalf("write privkey.pem: %v", err)
	}
}

func newCfgForSSLDetect(t *testing.T, mode string) *config.Config {
	t.Helper()
	cfg := &config.Config{BaseDomain: "example.com"}
	cfg, err := config.ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	cfg.SSLMode = mode
	return cfg
}

// TestSSLShouldEmit_LetsEncryptWithCertsPresent is the core cli#385
// regression: a "letsencrypt" project whose certs were already issued by an
// earlier certbot/`ssl add` run must emit the 443 block, not silently drop
// it the way SSL_MODE=="local"-only logic did.
func TestSSLShouldEmit_LetsEncryptWithCertsPresent(t *testing.T) {
	workdir := t.TempDir()
	cfg := newCfgForSSLDetect(t, "letsencrypt")
	writeCertPair(t, workdir, cfg.BaseDomain)

	gen := NewGenerator(cfg, workdir)
	if !gen.hasSSL {
		t.Error("cli#385: letsencrypt mode with certs already on disk must set hasSSL=true")
	}
}

// TestSSLShouldEmit_CustomWithCertsPresent mirrors the letsencrypt case for
// operator-supplied ("custom") certs placed at the same conventional path.
func TestSSLShouldEmit_CustomWithCertsPresent(t *testing.T) {
	workdir := t.TempDir()
	cfg := newCfgForSSLDetect(t, "custom")
	writeCertPair(t, workdir, cfg.BaseDomain)

	gen := NewGenerator(cfg, workdir)
	if !gen.hasSSL {
		t.Error("cli#385: custom mode with certs already on disk must set hasSSL=true")
	}
}

// TestSSLShouldEmit_LetsEncryptWithoutCerts preserves the pre-existing Bug
// #68 guarantee: before certbot has ever run, no cert files exist yet, and
// nginx must still be able to start HTTP-only.
func TestSSLShouldEmit_LetsEncryptWithoutCerts(t *testing.T) {
	workdir := t.TempDir()
	cfg := newCfgForSSLDetect(t, "letsencrypt")

	gen := NewGenerator(cfg, workdir)
	if gen.hasSSL {
		t.Error("letsencrypt mode with no certs on disk must set hasSSL=false")
	}
}

// TestSSLShouldEmit_CustomWithoutCerts is the "custom" analog of the above.
func TestSSLShouldEmit_CustomWithoutCerts(t *testing.T) {
	workdir := t.TempDir()
	cfg := newCfgForSSLDetect(t, "custom")

	gen := NewGenerator(cfg, workdir)
	if gen.hasSSL {
		t.Error("custom mode with no certs on disk must set hasSSL=false")
	}
}

// TestSSLShouldEmit_NoneAlwaysFalseEvenWithCertsPresent proves "none" is a
// hard opt-out: even if a cert pair happens to be sitting on disk (e.g. left
// over from a prior SSL_MODE), nginx must never reference it.
func TestSSLShouldEmit_NoneAlwaysFalseEvenWithCertsPresent(t *testing.T) {
	workdir := t.TempDir()
	cfg := newCfgForSSLDetect(t, "none")
	writeCertPair(t, workdir, cfg.BaseDomain)

	gen := NewGenerator(cfg, workdir)
	if gen.hasSSL {
		t.Error("SSL_MODE=none must set hasSSL=false even when cert files are present")
	}
}

// TestSSLShouldEmit_LocalAlwaysTrueEvenWithoutCertsYet preserves Bug #68:
// local mode is trusted unconditionally because `nself build` always
// (re)generates local certs earlier in the same run.
func TestSSLShouldEmit_LocalAlwaysTrueEvenWithoutCertsYet(t *testing.T) {
	workdir := t.TempDir()
	cfg := newCfgForSSLDetect(t, "local")

	gen := NewGenerator(cfg, workdir)
	if !gen.hasSSL {
		t.Error("SSL_MODE=local must set hasSSL=true even before this run's certs are written")
	}
}

// TestServiceConf_LetsEncryptWithCertsPresent_Emits443Block renders an
// actual service.conf.tmpl through RenderServiceRoute and asserts the
// output contains a real `listen 443 ssl` block with the matching
// ssl_certificate paths — the end-to-end shape of the cli#385 fix, not just
// the internal hasSSL flag.
func TestServiceConf_LetsEncryptWithCertsPresent_Emits443Block(t *testing.T) {
	workdir := t.TempDir()
	cfg := newCfgForSSLDetect(t, "letsencrypt")
	writeCertPair(t, workdir, cfg.BaseDomain)

	gen := NewGenerator(cfg, workdir)
	out, err := gen.RenderServiceRoute(ServiceRouteData{
		Route:      "api",
		BaseDomain: cfg.BaseDomain,
		Upstream:   "api_upstream:8080",
		SSLDir:     sslDirName(cfg.BaseDomain),
	})
	if err != nil {
		t.Fatalf("RenderServiceRoute: %v", err)
	}

	if !strings.Contains(out, "listen 443 ssl;") {
		t.Errorf("expected a listen 443 ssl block, got:\n%s", out)
	}
	wantCert := "ssl_certificate /etc/nginx/ssl/certificates/" + sslDirName(cfg.BaseDomain) + "/fullchain.pem;"
	if !strings.Contains(out, wantCert) {
		t.Errorf("expected ssl_certificate directive %q, got:\n%s", wantCert, out)
	}
}
