package nginx

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// NginxRoute represents a single nginx server block route for conflict detection.
type NginxRoute struct {
	ServerName string // e.g. "auth.example.com"
	Location   string // e.g. "/" (always "/" in current templates)
	PluginName string // owning plugin name — used in conflict error messages
}

// HasDomainConflict detects when two or more routes share the same
// server_name + location combination. Returns true and a list of
// conflict description strings if conflicts are found.
//
// When NginxRoute.PluginName is set on both the current and previously-seen
// route, the conflict message uses the format:
//
//	route conflict: /api claimed by <pluginA> and <pluginB>
func HasDomainConflict(routes []NginxRoute) (bool, []string) {
	// seen maps (serverName+location) -> the NginxRoute that first claimed it,
	// so we can surface both plugin names in conflict messages.
	seen := make(map[string]NginxRoute)
	var conflicts []string
	for _, r := range routes {
		key := r.ServerName + r.Location
		if prev, exists := seen[key]; exists {
			// Produce a named-plugin conflict message when plugin names are available.
			if r.PluginName != "" && prev.PluginName != "" {
				conflicts = append(conflicts, fmt.Sprintf("route conflict: %s%s claimed by %s and %s", r.ServerName, r.Location, prev.PluginName, r.PluginName))
			} else {
				conflicts = append(conflicts, fmt.Sprintf("%s conflicts with %s (same server_name+location: %s%s)", r.ServerName, prev.ServerName, r.ServerName, r.Location))
			}
		} else {
			seen[key] = r
		}
	}
	return len(conflicts) > 0, conflicts
}

// generateAllRoutes generates per-service nginx site configs.
// Returns map of filepath (e.g. "nginx/sites/hasura.conf") to nginx config content.
func (g *Generator) generateAllRoutes() (map[string]string, error) {
	files := make(map[string]string)

	baseDomain := g.cfg.BaseDomain
	sslDir := sslDirName(baseDomain)

	// Core routes (always generated)
	coreRoutes := g.coreRoutes(baseDomain, sslDir)

	// Optional service routes
	optionalRoutes := g.optionalRoutes(baseDomain, sslDir)

	// Custom service routes (CS_1..CS_10)
	csRoutes := g.customServiceRoutes(baseDomain, sslDir)

	// Frontend app routes (skipped on Linux servers)
	var feRoutes []routeEntry
	if !isLinuxServer() {
		feRoutes = g.frontendRoutes(baseDomain, sslDir)
	}

	// Internal routes (INTERNAL_ROUTE_1..INTERNAL_ROUTE_20)
	irRoutes, err := g.internalRoutes(baseDomain, sslDir)
	if err != nil {
		return nil, err
	}

	// Combine all route entries
	allEntries := make([]routeEntry, 0, len(coreRoutes)+len(optionalRoutes)+len(csRoutes)+len(feRoutes)+len(irRoutes))
	allEntries = append(allEntries, coreRoutes...)
	allEntries = append(allEntries, optionalRoutes...)
	allEntries = append(allEntries, csRoutes...)
	allEntries = append(allEntries, feRoutes...)
	allEntries = append(allEntries, irRoutes...)

	// Initialize seen map for this generation batch.
	g.seenRoutes = make(map[string]bool)

	// Propagate HasSSL to every route entry so templates can conditionally
	// omit ssl_certificate directives when certs are not locally managed.
	for i := range allEntries {
		allEntries[i].data.HasSSL = g.hasSSL
	}

	for _, entry := range allEntries {
		// Domain conflict prevention: skip if domain already exists in conf.d/
		if g.hasDomainConflict(entry.data.Route, entry.data.BaseDomain) {
			continue
		}

		content, err := g.render("service.conf.tmpl", entry.data)
		if err != nil {
			return nil, fmt.Errorf("rendering route %s: %w", entry.filename, err)
		}
		files["nginx/sites/"+entry.filename] = content
	}

	return files, nil
}

// routeEntry pairs a filename with its template data.
type routeEntry struct {
	filename string
	data     ServiceRouteData
}

// coreRoutes returns routes for always-present services: hasura, auth, storage, storage-console.
func (g *Generator) coreRoutes(baseDomain, sslDir string) []routeEntry {
	var entries []routeEntry

	// Hasura GraphQL
	hasuraRoute := g.cfg.Hasura.Route
	if hasuraRoute == "" {
		hasuraRoute = "api"
	}
	hasuraPort := g.cfg.Hasura.Port
	if hasuraPort == 0 {
		hasuraPort = 8080
	}
	entries = append(entries, routeEntry{
		filename: "hasura.conf",
		data: ServiceRouteData{
			Route:       hasuraRoute,
			BaseDomain:  baseDomain,
			Upstream:    fmt.Sprintf("hasura:%d", hasuraPort),
			SSLDir:      sslDir,
			RateZone:    "graphql_api",
			Burst:       20,
			ConnLimit:   10,
			WebSocket:   true,
			LazyResolve: true,
		},
	})

	// Auth
	authRoute := g.cfg.Auth.Route
	if authRoute == "" {
		authRoute = "auth"
	}
	authPort := g.cfg.Auth.Port
	if authPort == 0 {
		authPort = 4000
	}
	entries = append(entries, routeEntry{
		filename: "auth.conf",
		data: ServiceRouteData{
			Route:       authRoute,
			BaseDomain:  baseDomain,
			Upstream:    fmt.Sprintf("auth:%d", authPort),
			SSLDir:      sslDir,
			RateZone:    "auth",
			Burst:       5,
			ConnLimit:   5,
			LazyResolve: true,
		},
	})

	// Storage
	if g.cfg.Minio.Enabled {
		storageRoute := g.cfg.Minio.StorageRoute
		if storageRoute == "" {
			storageRoute = "storage"
		}
		entries = append(entries, routeEntry{
			filename: "storage.conf",
			data: ServiceRouteData{
				Route:      storageRoute,
				BaseDomain: baseDomain,
				Upstream:    fmt.Sprintf("minio:%d", g.cfg.Minio.Port),
			LazyResolve: true,
				SSLDir:     sslDir,
				RateZone:   "uploads",
				Burst:      2,
				ConnLimit:  5,
			},
		})

		// Storage Console
		consoleRoute := g.cfg.Minio.ConsoleRoute
		if consoleRoute == "" {
			consoleRoute = "storage-console"
		}
		entries = append(entries, routeEntry{
			filename: "storage-console.conf",
			data: ServiceRouteData{
				Route:       consoleRoute,
				BaseDomain:  baseDomain,
				Upstream:    fmt.Sprintf("minio:%d", minioConsolePort(g.cfg)),
				SSLDir:      sslDir,
				LazyResolve: true,
			},
		})
	}

	return entries
}

// optionalRoutes returns routes for optional services that are conditionally enabled.
func (g *Generator) optionalRoutes(baseDomain, sslDir string) []routeEntry {
	var entries []routeEntry

	// Admin (lazy resolver — may not be running)
	if g.cfg.Admin.Enabled {
		entries = append(entries, g.adminRoute(baseDomain, sslDir))
	}

	// Search
	if g.cfg.Search.Enabled {
		searchRoute := g.cfg.Search.Route
		if searchRoute == "" {
			searchRoute = "search"
		}
		searchPort := g.cfg.Search.Port
		if searchPort == 0 {
			searchPort = 7700
		}
		// Use actual Docker service name as upstream (meilisearch, typesense, etc.)
		searchUpstream := g.cfg.Search.Engine
		if searchUpstream == "" {
			searchUpstream = "meilisearch"
		}
		entries = append(entries, routeEntry{
			filename: "search.conf",
			data: ServiceRouteData{
				Route:       searchRoute,
				BaseDomain:  baseDomain,
				Upstream:    fmt.Sprintf("%s:%d", searchUpstream, searchPort),
				SSLDir:      sslDir,
				RateZone:    "general",
				Burst:       10,
				LazyResolve: true, // Search service may not be running
			},
		})
	}

	// Mail (Mailpit)
	if g.cfg.Mailpit.Enabled {
		mailRoute := g.cfg.Mailpit.Route
		if mailRoute == "" {
			mailRoute = "mail"
		}
		mailPort := g.cfg.Mailpit.UIPort
		if mailPort == 0 {
			mailPort = 8025
		}
		entries = append(entries, routeEntry{
			filename: "mail.conf",
			data: ServiceRouteData{
				Route:      mailRoute,
				BaseDomain: baseDomain,
				Upstream:   fmt.Sprintf("mailpit:%d", mailPort),
				SSLDir:     sslDir,
				RateZone:   "general",
				Burst:      10,
			},
		})
	}

	// MLflow: routes handled by nself-mlflow free plugin

	// Functions
	if g.cfg.Functions.Enabled {
		fnRoute := g.cfg.Functions.Route
		if fnRoute == "" {
			fnRoute = "functions"
		}
		fnPort := g.cfg.Functions.Port
		if fnPort == 0 {
			fnPort = 3008
		}
		entries = append(entries, routeEntry{
			filename: "functions.conf",
			data: ServiceRouteData{
				Route:      fnRoute,
				BaseDomain: baseDomain,
				Upstream:   fmt.Sprintf("functions:%d", fnPort),
				SSLDir:     sslDir,
				RateZone:   "functions",
				Burst:      15,
				ConnLimit:  10,
			},
		})
	}

	// Monitoring: routes handled by nself-monitoring free plugin

	return entries
}

// adminRoute builds the admin route entry with lazy resolver and dev mode support.
func (g *Generator) adminRoute(baseDomain, sslDir string) routeEntry {
	adminRoute := g.cfg.Admin.Route
	if adminRoute == "" {
		adminRoute = "admin"
	}

	var upstream string
	if g.cfg.Admin.DevMode {
		// Dev mode: proxy to host machine for hot-reload
		devPort := g.cfg.Admin.DevPort
		if devPort == 0 {
			devPort = 3000
		}
		if runtime.GOOS == "linux" {
			upstream = fmt.Sprintf("172.17.0.1:%d", devPort)
		} else {
			upstream = fmt.Sprintf("host.docker.internal:%d", devPort)
		}
	} else {
		adminPort := g.cfg.Admin.Port
		if adminPort == 0 {
			adminPort = 3021
		}
		upstream = fmt.Sprintf("nself-admin:%d", adminPort)
	}

	return routeEntry{
		filename: "admin.conf",
		data: ServiceRouteData{
			Route:       adminRoute,
			BaseDomain:  baseDomain,
			Upstream:    upstream,
			SSLDir:      sslDir,
			RateZone:    "general",
			Burst:       10,
			LazyResolve: true,
		},
	}
}

// customServiceRoutes returns routes for CS_1..CS_10 custom services.
// All custom services use lazy resolver since they are optional.
func (g *Generator) customServiceRoutes(baseDomain, sslDir string) []routeEntry {
	var entries []routeEntry

	for _, cs := range g.cfg.CustomServices {
		if cs.Route == "" {
			continue // internal only, no nginx route
		}
		if cs.Port == 0 {
			continue
		}

		entries = append(entries, routeEntry{
			filename: fmt.Sprintf("cs-%s.conf", cs.Name),
			data: ServiceRouteData{
				Route:       cs.Route,
				BaseDomain:  baseDomain,
				Upstream:    fmt.Sprintf("%s:%d", cs.Name, cs.Port),
				SSLDir:      sslDir,
				RateZone:    "general",
				Burst:       10,
				LazyResolve: true,
			},
		})
	}

	return entries
}

// frontendRoutes returns routes for frontend apps.
// These are SKIPPED on Linux servers (caller checks isLinuxServer).
// Uses host.docker.internal since frontend dev servers run on the host.
func (g *Generator) frontendRoutes(baseDomain, sslDir string) []routeEntry {
	var entries []routeEntry

	for _, fe := range g.cfg.FrontendApps {
		if fe.Route == "" || fe.Port == 0 || fe.SystemName == "" {
			continue
		}

		entries = append(entries, routeEntry{
			filename: fmt.Sprintf("frontend-%s.conf", fe.SystemName),
			data: ServiceRouteData{
				Route:       fe.Route,
				BaseDomain:  baseDomain,
				Upstream:    fmt.Sprintf("host.docker.internal:%d", fe.Port),
				SSLDir:      sslDir,
				RateZone:    "static",
				Burst:       10,
				WebSocket:   true, // HMR needs WebSocket
				LazyResolve: true,
			},
		})
	}

	return entries
}

// validateInternalRouteTarget checks that an InternalRoute target is a valid
// http://host:port or https://host:port URL with no nginx-unsafe characters.
func validateInternalRouteTarget(target string) error {
	// Reject injection characters before URL parsing.
	for _, ch := range []rune{'\n', '\r', ';', '{', '}' } {
		if strings.ContainsRune(target, ch) {
			return fmt.Errorf("target contains forbidden character %q", ch)
		}
	}
	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("target is not a valid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("target scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("target is missing host")
	}
	return nil
}

// internalRoutes returns routes for INTERNAL_ROUTE_1..INTERNAL_ROUTE_20.
// These use lazy resolver since they are optional.
func (g *Generator) internalRoutes(baseDomain, sslDir string) ([]routeEntry, error) {
	var entries []routeEntry

	for _, ir := range g.cfg.InternalRoutes {
		if ir.Subdomain == "" || ir.Target == "" {
			continue
		}
		if err := validateInternalRouteTarget(ir.Target); err != nil {
			return nil, fmt.Errorf("internal route %q: invalid target: %w", ir.Name, err)
		}

		rateZone := ir.RateZone
		if rateZone == "" {
			rateZone = "general"
		}

		entries = append(entries, routeEntry{
			filename: fmt.Sprintf("ir-%s.conf", ir.Name),
			data: ServiceRouteData{
				Route:       ir.Subdomain,
				BaseDomain:  baseDomain,
				Upstream:    ir.Target,
				SSLDir:      sslDir,
				RateZone:    rateZone,
				Burst:       10,
				ConnLimit:   10,
				WebSocket:   ir.WebSocket,
				LazyResolve: true,
			},
		})
	}

	return entries, nil
}

// hasDomainConflict checks whether a server_name conflicts with an existing
// route. It checks two sources:
//  1. The current generation batch (duplicate routes within this build).
//  2. Hand-managed nginx/conf.d/ files that already define the server_name.
//
// Returns true if the route should be skipped.
func (g *Generator) hasDomainConflict(route, baseDomain string) bool {
	serverName, err := config.RouteToFQDN(route, baseDomain)
	if err != nil {
		return false
	}

	// Check within the current generation batch.
	if g.seenRoutes[serverName] {
		return true
	}
	g.seenRoutes[serverName] = true

	// Check if this server_name already exists in hand-managed nginx/conf.d/ files.
	confDirs := []string{
		filepath.Join(g.workdir, "nginx", "conf.d"),
		filepath.Join(g.workdir, "nginx", fmt.Sprintf("conf.d-%s", g.cfg.Env)),
	}

	for _, dir := range confDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "default.conf" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			if strings.Contains(string(data), "server_name") && strings.Contains(string(data), serverName) {
				return true
			}
		}
	}
	return false
}

// isLinuxServer returns true when running on a Linux VPS (not Docker Desktop).
// Frontend app routes are skipped on Linux servers because host.docker.internal
// is not available.
func isLinuxServer() bool {
	return runtime.GOOS == "linux"
}

// minioConsolePort returns the MinIO console port from config with default fallback.
func minioConsolePort(cfg *config.Config) int {
	if cfg.Minio.ConsolePort != 0 {
		return cfg.Minio.ConsolePort
	}
	return 9001
}

