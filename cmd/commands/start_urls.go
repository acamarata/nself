package commands

// Purpose: Compute the "nSelf Stack Running" URL list printed by nself start.
//          For default local-dev domains (local.nself.org / localhost) the
//          printed URLs must be the actually-curlable localhost:<port>
//          endpoints — the bare local.nself.org URLs require DNS/hosts/mkcert
//          setup and 502 on a fresh machine (ntask dogfood gap #20).
// Inputs:  resolved *config.Config.
// Outputs: display strings for ui.SummaryBox.
// Constraints: Custom domains keep the domain-based URLs unchanged.
// SPORT: cli/cmd/commands — start output (PCI ntask-local-url-mismatch).

import (
	"fmt"

	"github.com/nself-org/cli/internal/config"
)

// defaultLocalDomains are the BASE_DOMAIN values that mean "no custom domain
// configured" — a fresh local-dev setup where only localhost ports are
// guaranteed reachable.
var defaultLocalDomains = map[string]bool{
	"":                true,
	"localhost":       true,
	"local.nself.org": true,
}

// portOr returns port when > 0, else fallback.
func portOr(port, fallback int) int {
	if port > 0 {
		return port
	}
	return fallback
}

// stackURLs returns the URL lines for the start summary box.
//
// Local-dev (default domain): direct localhost:<port> endpoints that work
// with zero DNS/hosts/SSL setup, plus a hint that the *.local.nself.org
// routes need `nself dns-setup`.
//
// Custom domain: the nginx-routed https URLs, unchanged.
func stackURLs(cfg *config.Config) []string {
	domain := cfg.BaseDomain
	if !defaultLocalDomains[domain] {
		scheme := "https"
		if cfg.Env == "dev" {
			scheme = "http"
		}
		urls := []string{
			fmt.Sprintf("API:      %s://%s", scheme, domain),
			fmt.Sprintf("Hasura:   %s://%s/v1/graphql", scheme, domain),
			fmt.Sprintf("Console:  %s://%s/console", scheme, domain),
			fmt.Sprintf("Auth:     %s://%s/v1/auth", scheme, domain),
		}
		if cfg.Minio.Enabled {
			urls = append(urls, fmt.Sprintf("Storage:  %s://%s/v1/storage", scheme, domain))
		}
		if cfg.Mailpit.Enabled {
			urls = append(urls, fmt.Sprintf("Mail UI:  %s://%s/mailpit", scheme, domain))
		}
		if cfg.Monitoring.GrafanaEnabled {
			urls = append(urls, fmt.Sprintf("Grafana:  %s://%s/grafana", scheme, domain))
		}
		if cfg.Admin.Enabled {
			urls = append(urls, fmt.Sprintf("Admin:    http://localhost:%d", portOr(cfg.Admin.Port, 3021)))
		}
		return urls
	}

	// Default local domain — print the reachable localhost endpoints.
	hasuraPort := portOr(cfg.Hasura.Port, 8080)
	urls := []string{
		fmt.Sprintf("GraphQL:  http://localhost:%d/v1/graphql", hasuraPort),
		fmt.Sprintf("Hasura:   http://localhost:%d", hasuraPort),
		fmt.Sprintf("Auth:     http://localhost:%d", portOr(cfg.Auth.Port, 4000)),
	}
	if cfg.Minio.Enabled {
		urls = append(urls, fmt.Sprintf("Storage:  http://localhost:%d", portOr(cfg.Minio.Port, 9000)))
	}
	if cfg.Mailpit.Enabled {
		urls = append(urls, fmt.Sprintf("Mail UI:  http://localhost:%d", portOr(cfg.Mailpit.UIPort, 8025)))
	}
	if cfg.Admin.Enabled {
		urls = append(urls, fmt.Sprintf("Admin:    http://localhost:%d", portOr(cfg.Admin.Port, 3021)))
	}
	if domain != "" && domain != "localhost" {
		urls = append(urls, fmt.Sprintf("Domains:  https://*.%s (requires: nself dns-setup)", domain))
	}
	return urls
}
