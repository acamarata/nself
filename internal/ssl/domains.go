package ssl

import (
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// CollectDomains gathers all Subject Alternative Names (SANs) needed for the
// project's SSL certificate. The list is deduplicated. It is exported for use
// by the dns-setup command.
func (g *Generator) CollectDomains() []string {
	return g.collectDomains()
}

// collectDomains gathers all Subject Alternative Names (SANs) needed for the
// project's SSL certificate. The list is deduplicated.
func (g *Generator) collectDomains() []string {
	seen := make(map[string]bool)
	var domains []string

	add := func(d string) {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		domains = append(domains, d)
	}

	// Always included: localhost, loopback addresses.
	add("localhost")
	add("127.0.0.1")
	add("::1")

	bd := g.cfg.BaseDomain

	// BASE_DOMAIN and wildcard.
	if bd != "" {
		add(bd)
		add("*." + bd)
	}

	// local.nself.org and wildcard (always, for local dev tooling).
	add("local.nself.org")
	add("*.local.nself.org")

	// Core services (always enabled).
	if bd != "" {
		// Hasura
		if route := g.cfg.Hasura.Route; route != "" {
			add(route)
		} else {
			add(strings.TrimPrefix(config.BuildServiceURL("api", bd), "https://"))
		}

		// Auth
		if route := g.cfg.Auth.Route; route != "" {
			add(route)
		} else {
			add(strings.TrimPrefix(config.BuildServiceURL("auth", bd), "https://"))
		}

		// Storage (MinIO + nhost-storage)
		if g.cfg.Minio.Enabled {
			if route := g.cfg.Minio.StorageRoute; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("storage", bd), "https://"))
			}
			if route := g.cfg.Minio.ConsoleRoute; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("storage-console", bd), "https://"))
			}
		}

		// Admin
		if g.cfg.Admin.Enabled {
			if route := g.cfg.Admin.Route; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("admin", bd), "https://"))
			}
		}

		// Search
		if g.cfg.Search.Enabled {
			if route := g.cfg.Search.Route; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("search", bd), "https://"))
			}
		}

		// Mailpit
		if g.cfg.Mailpit.Enabled {
			if route := g.cfg.Mailpit.Route; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("mail", bd), "https://"))
			}
		}

		// Functions
		if g.cfg.Functions.Enabled {
			if route := g.cfg.Functions.Route; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("functions", bd), "https://"))
			}
		}

		// MLflow
		if g.cfg.MLflow.Enabled {
			if route := g.cfg.MLflow.Route; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("mlflow", bd), "https://"))
			}
		}

		// Monitoring (Grafana)
		if g.cfg.Monitoring.Enabled && g.cfg.Monitoring.GrafanaEnabled {
			if route := g.cfg.Monitoring.GrafanaRoute; route != "" {
				add(route)
			} else {
				add(strings.TrimPrefix(config.BuildServiceURL("grafana", bd), "https://"))
			}
		}

		// Custom services (only if they have a public route).
		for _, cs := range g.cfg.CustomServices {
			if cs.Route != "" && cs.Public {
				// Route may be a full domain or a subdomain prefix.
				if strings.Contains(cs.Route, ".") {
					add(cs.Route)
				} else {
					add(cs.Route + "." + bd)
				}
			}
		}

		// Frontend apps.
		for _, fa := range g.cfg.FrontendApps {
			if fa.Route != "" {
				if strings.Contains(fa.Route, ".") {
					add(fa.Route)
				} else {
					add(fa.Route + "." + bd)
				}
			}
		}
	}

	// EXTRA_SSL_DOMAINS (comma-separated).
	if extra := g.cfg.ExtraSSLDomains; extra != "" {
		for _, d := range strings.Split(extra, ",") {
			add(d)
		}
	}

	return domains
}

