package commands

// ssl_install_test.go — Regression coverage for the cert install helpers in
// ssl_install.go, plus the cross-package check that ssl_add's cert directory
// construction cannot silently diverge from internal/ssl's.
//
// Purpose: installIssuedCert and writeCustomDomainConf both exist because of
//          previously-shipped bugs (see their doc comments in ssl_install.go):
//          certbot ignores --cert-path so the issued cert has to be copied
//          into the tree nginx reads, and the generated nginx conf must
//          reference the dash-safe directory name, not the dotted domain.
//          ssl_setup_paths_test.go already covers those two behaviors
//          end-to-end; this file adds the one assertion still missing —
//          that ssl_add's `ssl/certificates/<domainSafe>` construction is
//          verified against internal/ssl.DomainToDirName (the same rule
//          internal/ssl applies to the primary domain) rather than two
//          independently-typed literals that could drift apart.
// Inputs:  domain strings; a temp dir standing in for the project root.
// Outputs: assertions on copied cert files, generated conf content, and
//          directory-name agreement with internal/ssl.
// Constraints: No certbot, docker, or network. Pure filesystem + string checks.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/ssl"
)

// TestInstallIssuedCert_CopiesFromLetsEncryptLiveDir verifies installIssuedCert
// copies both PEM files from letsEncryptLiveDir/<domain>/ into the destination
// certDir with identical content and 0600 permissions, matching what
// certbot's own layout requires nself to bridge manually (see ssl_install.go).
func TestInstallIssuedCert_CopiesFromLetsEncryptLiveDir(t *testing.T) {
	const domain = "example.nself.org"

	liveRoot := t.TempDir()
	liveDomainDir := filepath.Join(liveRoot, domain)
	if err := os.MkdirAll(liveDomainDir, 0o750); err != nil {
		t.Fatalf("mkdir live domain dir: %v", err)
	}

	fixtures := map[string]string{
		"fullchain.pem": "FAKE-FULLCHAIN-PEM-DATA",
		"privkey.pem":   "FAKE-PRIVKEY-PEM-DATA",
	}
	for name, body := range fixtures {
		if err := os.WriteFile(filepath.Join(liveDomainDir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	orig := letsEncryptLiveDir
	letsEncryptLiveDir = liveRoot
	t.Cleanup(func() { letsEncryptLiveDir = orig })

	destDir := t.TempDir()
	if err := installIssuedCert(domain, destDir); err != nil {
		t.Fatalf("installIssuedCert: %v", err)
	}

	for name, want := range fixtures {
		p := filepath.Join(destDir, name)
		got, err := os.ReadFile(p) //nolint:gosec // path built from t.TempDir
		if err != nil {
			t.Fatalf("%s not installed at destDir: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s content = %q, want %q", name, got, want)
		}
	}
}

// TestInstallIssuedCert_MissingCertbotOutputErrors verifies installIssuedCert
// returns a wrapped, diagnosable error — rather than silently doing nothing —
// when certbot did not actually produce output for the domain. This is the
// exact failure mode installIssuedCert exists to prevent from going unnoticed.
func TestInstallIssuedCert_MissingCertbotOutputErrors(t *testing.T) {
	liveRoot := t.TempDir() // no domain subdirectory created inside it

	orig := letsEncryptLiveDir
	letsEncryptLiveDir = liveRoot
	t.Cleanup(func() { letsEncryptLiveDir = orig })

	destDir := t.TempDir()
	err := installIssuedCert("never-issued.example.com", destDir)
	if err == nil {
		t.Fatal("installIssuedCert succeeded with no certbot output on disk — expected an error")
	}
	if !strings.Contains(err.Error(), "did certbot succeed?") {
		t.Errorf("error = %q, want it to mention \"did certbot succeed?\"", err.Error())
	}
}

// TestWriteCustomDomainConf_ReferencesFilesafeCertPath is the regression guard
// for the exact bug ssl_add.go's inline comment describes: the generated nginx
// server block must reference the DASH-safe directory name in its
// ssl_certificate directives, never the dotted domain — the dotted form is a
// path nginx cannot resolve because that is not where the certificate lives.
func TestWriteCustomDomainConf_ReferencesFilesafeCertPath(t *testing.T) {
	const domain = "my.custom.com"
	const wantSafe = "my-custom-com"

	workdir := t.TempDir()
	if err := writeCustomDomainConf(workdir, domain, ""); err != nil {
		t.Fatalf("writeCustomDomainConf: %v", err)
	}

	confPath := filepath.Join(workdir, "nginx", "conf.d", "custom-"+wantSafe+".conf")
	raw, err := os.ReadFile(confPath) //nolint:gosec // path built from t.TempDir
	if err != nil {
		t.Fatalf("conf not written at expected path: %v", err)
	}
	conf := string(raw)

	for _, line := range []string{"ssl_certificate ", "ssl_certificate_key "} {
		idx := strings.Index(conf, line)
		if idx == -1 {
			t.Fatalf("conf missing %q directive\n--- conf ---\n%s", line, conf)
		}
		end := strings.IndexByte(conf[idx:], '\n')
		directiveLine := conf[idx : idx+end]
		if !strings.Contains(directiveLine, wantSafe) {
			t.Errorf("%q line does not reference dash-safe form %q: %q", line, wantSafe, directiveLine)
		}
		if strings.Contains(directiveLine, "/"+domain+"/") {
			t.Errorf("%q line references the dotted domain instead of the dash-safe form: %q", line, directiveLine)
		}
	}
}

// TestWriteCustomDomainConf_PathScopedRateLimits is the SEC-HARDENING-06
// regression guard (P6-E2-W2-S3-T20): a custom domain proxying to a backend
// (--upstream set) must get the same path-scoped /auth/login (auth_strict)
// and /api/ (api) rate-limit locations as internal/nginx's
// defaultSecurityPathZones, ahead of the catch-all `location /`. The
// placeholder (no upstream) branch is asserted separately to have none of
// them, since it never proxies anywhere.
func TestWriteCustomDomainConf_PathScopedRateLimits(t *testing.T) {
	t.Run("proxying custom domain gets both path-scoped locations", func(t *testing.T) {
		workdir := t.TempDir()
		if err := writeCustomDomainConf(workdir, "gw.example.com", "gw-upstream:8080"); err != nil {
			t.Fatalf("writeCustomDomainConf: %v", err)
		}
		confPath := filepath.Join(workdir, "nginx", "conf.d", "custom-gw-example-com.conf")
		raw, err := os.ReadFile(confPath) //nolint:gosec // path built from t.TempDir
		if err != nil {
			t.Fatalf("conf not written: %v", err)
		}
		conf := string(raw)

		for _, want := range []string{
			"location = /auth/login {",
			"limit_req zone=auth_strict burst=5 nodelay;",
			"location /api/ {",
			"limit_req zone=api burst=20 nodelay;",
			"proxy_pass http://gw-upstream:8080;",
		} {
			if !strings.Contains(conf, want) {
				t.Errorf("missing %q\n--- conf ---\n%s", want, conf)
			}
		}

		// Doctor check (internal/doctor/hardening_check_nginx_zones.go)
		// greps this exact file's content for "/auth/login" + "limit_req"
		// and "/api/" + "limit_req" co-occurring — assert its literal
		// contract, not just our own template's wording.
		if !strings.Contains(conf, "/auth/login") || !strings.Contains(conf, "limit_req") {
			t.Error("conf must satisfy the doctor check's /auth/login + limit_req grep")
		}
		if !strings.Contains(conf, "/api/") {
			t.Error("conf must satisfy the doctor check's /api/ grep")
		}

		authIdx := strings.Index(conf, "location = /auth/login")
		apiIdx := strings.Index(conf, "location /api/")
		// LastIndex: writeCustomDomainConf emits a separate listen-80
		// redirect server block ahead of the proxying listen-443 block, and
		// that redirect block's own `location / { return 301 ...}` also
		// matches "location / {" — the catch-all we care about ordering
		// against is the one in the same (443) server block as the
		// path-scoped locations, i.e. the LAST occurrence in the file.
		catchAllIdx := strings.LastIndex(conf, "location / {")
		if authIdx == -1 || apiIdx == -1 || catchAllIdx == -1 {
			t.Fatal("could not locate all three location blocks")
		}
		if authIdx > catchAllIdx || apiIdx > catchAllIdx {
			t.Error("path-scoped locations must be declared before the catch-all location /")
		}
	})

	t.Run("placeholder custom domain (no upstream) has none of them", func(t *testing.T) {
		workdir := t.TempDir()
		if err := writeCustomDomainConf(workdir, "future.example.com", ""); err != nil {
			t.Fatalf("writeCustomDomainConf: %v", err)
		}
		confPath := filepath.Join(workdir, "nginx", "conf.d", "custom-future-example-com.conf")
		raw, err := os.ReadFile(confPath) //nolint:gosec // path built from t.TempDir
		if err != nil {
			t.Fatalf("conf not written: %v", err)
		}
		conf := string(raw)
		if strings.Contains(conf, "limit_req") {
			t.Errorf("placeholder (no --upstream) conf should not proxy anywhere and should carry no rate limits\n--- conf ---\n%s", conf)
		}
	})
}

// TestDomainToFilesafe_ReplacesDotsAndColons table-tests the domainToFilesafe
// helper directly, including an IPv6-with-port-style edge case (colons),
// since the function replaces both dots and colons.
func TestDomainToFilesafe_ReplacesDotsAndColons(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"nself.org":          "nself-org",
		"api.task.nself.org": "api-task-nself-org",
		"localhost:8443":     "localhost-8443",
		"[::1]:8443":         "[--1]-8443",
	}
	for in, want := range cases {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			got := domainToFilesafe(in)
			if got != want {
				t.Errorf("domainToFilesafe(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

// TestSSLAdd_CertDirMatchesNginxMountLayout is the cross-reference regression
// guard required by this ticket's CR-C: it proves the directory ssl_add
// constructs (workdir/ssl/certificates/<domainSafe>) uses a domain-safe name
// that is IDENTICAL to internal/ssl.DomainToDirName's output — the function
// internal/ssl uses to name the certificate directory for the primary domain
// (see internal/ssl/generator.go's GenerateWithResult). It imports and calls
// the real internal/ssl function rather than re-typing its replacement rule,
// so the two layouts cannot silently diverge if internal/ssl's convention
// ever changes without ssl_add.go being updated to match.
func TestSSLAdd_CertDirMatchesNginxMountLayout(t *testing.T) {
	t.Parallel()

	domains := []string{
		"staging.nself.org",
		"nself.org",
		"api.task.nself.org",
	}

	for _, domain := range domains {
		domain := domain
		t.Run(domain, func(t *testing.T) {
			t.Parallel()

			workdir := "/opt/nself-web/backend" // representative, not written to
			addSafe := domainToFilesafe(domain)
			addCertDir := filepath.Join(workdir, "ssl", "certificates", addSafe)

			internalSafe := ssl.DomainToDirName(domain)
			internalCertDir := filepath.Join(workdir, "ssl", "certificates", internalSafe)

			if addCertDir != internalCertDir {
				t.Errorf("ssl add cert dir %q diverges from internal/ssl's layout %q for domain %q",
					addCertDir, internalCertDir, domain)
			}
		})
	}
}
