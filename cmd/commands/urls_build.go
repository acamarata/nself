package commands

// Purpose: Builds the full urlsOutput structure for `nself urls` — walks
// the loaded config to assign proxy URLs to every required/optional/
// custom/frontend service, plus the small route/scheme/counting helpers it
// uses. Split out of urls.go (CLI-R12) to separate this large builder from
// the cobra entry point (urls.go), the conflict-detection/display printers
// (urls_display.go), and the diff subcommand (urls_diff.go).
// Inputs: a loaded *config.Config and a showAll flag (include internal-only
// routes).
// Outputs: a populated urlsOutput grouping every service's resolved URL.
// Constraints: pure move — no behavior changes.

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// buildURLOutput computes all URLs from a loaded config.
func buildURLOutput(cfg *config.Config, showAll bool) urlsOutput {
	bd := cfg.BaseDomain
	scheme := urlScheme(cfg)

	out := urlsOutput{
		BaseDomain: bd,
		Env:        cfg.Env,
	}

	// -- Required Services (always present) --
	out.RequiredServices = []serviceURL{
		{
			Name:     "PostgreSQL",
			URL:      fmt.Sprintf("127.0.0.1:%d", cfg.Postgres.Port),
			Internal: true,
			Group:    "Required Services",
		},
		{
			Name:  "Hasura GraphQL",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Hasura.Route, bd),
			Group: "Required Services",
		},
		{
			Name:  "Auth",
			URL:   resolveRoute(scheme, cfg.Auth.Route, bd),
			Group: "Required Services",
		},
		{
			Name:  "Nginx",
			URL:   fmt.Sprintf("%s://%s", scheme, bd),
			Group: "Required Services",
		},
	}

	// -- Optional Services --
	if cfg.Redis.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:     "Redis",
			URL:      fmt.Sprintf("127.0.0.1:%d", cfg.Redis.Port),
			Internal: true,
			Group:    "Optional Services",
		})
	}
	if cfg.Minio.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "MinIO Storage",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Minio.StorageRoute, bd),
			Group: "Optional Services",
		})
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "MinIO Console",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Minio.ConsoleRoute, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Mailpit.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Mailpit UI",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Mailpit.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Functions.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Functions",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Functions.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.MLflow.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "MLflow",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.MLflow.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Admin.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Admin",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Admin.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Search.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Search",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Search.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Monitoring.Enabled && cfg.Monitoring.GrafanaEnabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Grafana",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Monitoring.GrafanaRoute, bd),
			Group: "Optional Services",
		})
	}

	// -- Custom Services --
	for _, cs := range cfg.CustomServices {
		if cs.Route != "" {
			out.CustomServices = append(out.CustomServices, serviceURL{
				Name:  cs.Name,
				URL:   fmt.Sprintf("%s://%s.%s", scheme, cs.Route, bd),
				Group: "Custom Services",
			})
		} else if showAll {
			out.CustomServices = append(out.CustomServices, serviceURL{
				Name:     cs.Name,
				URL:      fmt.Sprintf("127.0.0.1:%d", cs.Port),
				Internal: true,
				Group:    "Custom Services",
			})
		}
	}

	// -- Frontend Apps --
	for _, fa := range cfg.FrontendApps {
		if fa.Route != "" {
			out.FrontendApps = append(out.FrontendApps, serviceURL{
				Name:  fa.DisplayName,
				URL:   fmt.Sprintf("%s://%s.%s", scheme, fa.Route, bd),
				Group: "Frontend Apps",
			})
		}
	}

	// -- Internal admin endpoints (only with --all) --
	if showAll {
		// Hasura admin console (direct port access, not via proxy)
		out.InternalRoutes = append(out.InternalRoutes, serviceURL{
			Name:     "Hasura Admin",
			URL:      fmt.Sprintf("http://localhost:%d/console", cfg.Hasura.Port),
			Internal: true,
			Group:    "Internal Endpoints",
		})
		// Auth service direct port
		out.InternalRoutes = append(out.InternalRoutes, serviceURL{
			Name:     "Auth (direct)",
			URL:      fmt.Sprintf("http://localhost:%d", cfg.Auth.Port),
			Internal: true,
			Group:    "Internal Endpoints",
		})
		if cfg.Minio.Enabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "MinIO Console (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Minio.ConsolePort),
				Internal: true,
				Group:    "Internal Endpoints",
			})
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "MinIO API (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Minio.Port),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
		if cfg.Monitoring.Enabled && cfg.Monitoring.GrafanaEnabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "Grafana (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Monitoring.GrafanaPort),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
		if cfg.Mailpit.Enabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "Mailpit UI (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Mailpit.UIPort),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
		if cfg.Redis.Enabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "Redis (direct)",
				URL:      fmt.Sprintf("127.0.0.1:%d", cfg.Redis.Port),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
	}

	// -- Internal Routes (only with --all) --
	if showAll {
		for _, ir := range cfg.InternalRoutes {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     ir.Name,
				URL:      fmt.Sprintf("%s://%s.%s", scheme, ir.Subdomain, bd),
				Internal: true,
				Group:    "Internal Routes",
			})
		}
	}

	// Count total public routes.
	out.TotalRoutes = countRoutes(out)

	return out
}

// resolveRoute handles routes that may already include the base domain
// (e.g., "auth.local.nself.org") vs bare subdomain prefixes (e.g., "auth").
func resolveRoute(scheme, route, bd string) string {
	if strings.Contains(route, ".") {
		// Fully qualified: auth.local.nself.org
		return fmt.Sprintf("%s://%s", scheme, route)
	}
	return fmt.Sprintf("%s://%s.%s", scheme, route, bd)
}

// urlScheme returns "https" unless SSL is disabled.
func urlScheme(cfg *config.Config) string {
	if cfg.SSLMode == "none" {
		return "http"
	}
	return "https"
}

// countRoutes tallies all non-internal URLs across all groups.
func countRoutes(out urlsOutput) int {
	total := 0
	for _, groups := range [][]serviceURL{
		out.RequiredServices,
		out.OptionalServices,
		out.CustomServices,
		out.FrontendApps,
		out.InternalRoutes,
	} {
		for _, s := range groups {
			if !s.Internal {
				total++
			}
		}
	}
	return total
}
