package commands

// Purpose: Regression tests for PCI ntask-local-url-mismatch: nself start
//          must print reachable localhost:<port> URLs for default local
//          domains instead of the unreachable *.local.nself.org routes.

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

func startURLTestConfig(baseDomain string) *config.Config {
	cfg := &config.Config{
		ProjectName: "urltest",
		BaseDomain:  baseDomain,
		Env:         "dev",
	}
	cfg.Hasura.Port = 8080
	cfg.Auth.Port = 4000
	cfg.Minio.Enabled = true
	cfg.Minio.Port = 9000
	cfg.Mailpit.Enabled = true
	cfg.Mailpit.UIPort = 8025
	return cfg
}

// TestStackURLs_DefaultLocalDomainPrintsLocalhost is the PCI repro: with the
// default local.nself.org base domain, every printed URL must be a curlable
// localhost endpoint, never a bare local.nself.org route.
func TestStackURLs_DefaultLocalDomainPrintsLocalhost(t *testing.T) {
	urls := stackURLs(startURLTestConfig("local.nself.org"))
	joined := strings.Join(urls, "\n")

	for _, want := range []string{
		"http://localhost:8080/v1/graphql",
		"http://localhost:4000",
		"http://localhost:9000",
		"http://localhost:8025",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected reachable URL %q in start output, got:\n%s", want, joined)
		}
	}

	for _, line := range urls {
		if strings.Contains(line, "local.nself.org") && !strings.Contains(line, "dns-setup") {
			t.Errorf("unreachable local.nself.org URL printed without setup hint: %q", line)
		}
	}
}

// TestStackURLs_CustomDomainUnchanged keeps the nginx-routed URLs for real
// domains.
func TestStackURLs_CustomDomainUnchanged(t *testing.T) {
	cfg := startURLTestConfig("api.example.com")
	cfg.Env = "prod"
	urls := stackURLs(cfg)
	joined := strings.Join(urls, "\n")

	if !strings.Contains(joined, "https://api.example.com/v1/graphql") {
		t.Errorf("custom domain URLs changed:\n%s", joined)
	}
	if strings.Contains(joined, "localhost:8080") {
		t.Errorf("custom domain output must not fall back to localhost:\n%s", joined)
	}
}

// TestStackURLs_PortFallbacks covers zero-value ports (defensive defaults).
func TestStackURLs_PortFallbacks(t *testing.T) {
	cfg := &config.Config{BaseDomain: "localhost", Env: "dev"}
	urls := stackURLs(cfg)
	joined := strings.Join(urls, "\n")
	if !strings.Contains(joined, "http://localhost:8080/v1/graphql") {
		t.Errorf("expected 8080 fallback, got:\n%s", joined)
	}
	if !strings.Contains(joined, "http://localhost:4000") {
		t.Errorf("expected 4000 fallback, got:\n%s", joined)
	}
}
