package build

// orchestrator_nginx.go — collects the nginx routes a build will
// generate, for preflight domain-conflict detection. Split from
// orchestrator.go (T-P6-E2-W1-S1-T3).
// Inputs:  resolved *config.Config.
// Outputs: []nginx.NginxRoute — one per core/optional service, custom
//          service, frontend app, and internal route with a set domain.
// Constraints: pure move, same route list/order, no behavior change.

import (
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/nginx"
)

// buildNginxRoutes collects all nginx routes that will be generated for the
// given config. The list is used for preflight conflict detection via
// nginx.HasDomainConflict before any files are written.
//
// The server_name for each route is route + "." + baseDomain, matching the
// logic in the nginx Generator. Location is always "/" in current templates.
func buildNginxRoutes(cfg *config.Config) []nginx.NginxRoute {
	bd := cfg.BaseDomain
	loc := "/"
	var routes []nginx.NginxRoute

	// Core: hasura
	hasuraRoute := cfg.Hasura.Route
	if hasuraRoute == "" {
		hasuraRoute = "api"
	}
	routes = append(routes, nginx.NginxRoute{ServerName: hasuraRoute + "." + bd, Location: loc})

	// Core: auth
	authRoute := cfg.Auth.Route
	if authRoute == "" {
		authRoute = "auth"
	}
	routes = append(routes, nginx.NginxRoute{ServerName: authRoute + "." + bd, Location: loc})

	// Core: storage + storage-console (when MinIO enabled)
	if cfg.Minio.Enabled {
		storageRoute := cfg.Minio.StorageRoute
		if storageRoute == "" {
			storageRoute = "storage"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: storageRoute + "." + bd, Location: loc})

		consoleRoute := cfg.Minio.ConsoleRoute
		if consoleRoute == "" {
			consoleRoute = "storage-console"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: consoleRoute + "." + bd, Location: loc})
	}

	// Optional: admin
	if cfg.Admin.Enabled {
		adminRoute := cfg.Admin.Route
		if adminRoute == "" {
			adminRoute = "admin"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: adminRoute + "." + bd, Location: loc})
	}

	// Optional: search
	if cfg.Search.Enabled {
		searchRoute := cfg.Search.Route
		if searchRoute == "" {
			searchRoute = "search"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: searchRoute + "." + bd, Location: loc})
	}

	// Optional: mail
	if cfg.Mailpit.Enabled {
		mailRoute := cfg.Mailpit.Route
		if mailRoute == "" {
			mailRoute = "mail"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: mailRoute + "." + bd, Location: loc})
	}

	// Optional: functions
	if cfg.Functions.Enabled {
		fnRoute := cfg.Functions.Route
		if fnRoute == "" {
			fnRoute = "functions"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: fnRoute + "." + bd, Location: loc})
	}

	// Custom services (CS_1..CS_10)
	for _, cs := range cfg.CustomServices {
		if cs.Route == "" || cs.Port == 0 {
			continue
		}
		routes = append(routes, nginx.NginxRoute{ServerName: cs.Route + "." + bd, Location: loc})
	}

	// Frontend apps
	for _, fe := range cfg.FrontendApps {
		if fe.Route == "" || fe.Port == 0 {
			continue
		}
		routes = append(routes, nginx.NginxRoute{ServerName: fe.Route + "." + bd, Location: loc})
	}

	// Internal routes
	for _, ir := range cfg.InternalRoutes {
		if ir.Subdomain == "" || ir.Target == "" {
			continue
		}
		routes = append(routes, nginx.NginxRoute{ServerName: ir.Subdomain + "." + bd, Location: loc})
	}

	return routes
}
