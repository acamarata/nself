package plugin

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/nginx"
)

// CheckPluginRouteConflict checks whether the nginx route declared by plugin
// conflicts with any of the existing service routes. It uses HasDomainConflict
// from the nginx package to detect server_name+location collisions.
//
// plugin: the manifest of the plugin being installed
// existingRoutes: nginx routes already registered by core and optional services
//
// Returns an error if any conflict is detected, nil otherwise.
func checkPluginRouteConflict(plugin *PluginManifest, existingRoutes []nginx.NginxRoute) error {
	// Build plugin's routes from its API endpoints.
	// APIEndpoints are full paths like "/api/v1" or subdomain.example.com/path.
	// We derive server_name from plugin name and location from each endpoint.
	var pluginRoutes []nginx.NginxRoute
	for _, endpoint := range plugin.APIEndpoints {
		// Normalize: strip http(s):// prefix if present
		ep := strings.TrimPrefix(endpoint, "https://")
		ep = strings.TrimPrefix(ep, "http://")
		// Split host/path
		parts := strings.SplitN(ep, "/", 2)
		serverName := parts[0]
		location := "/"
		if len(parts) == 2 && parts[1] != "" {
			location = "/" + parts[1]
		}
		pluginRoutes = append(pluginRoutes, nginx.NginxRoute{
			ServerName: serverName,
			Location:   location,
		})
	}

	// If plugin has no explicit routes, nothing to conflict
	if len(pluginRoutes) == 0 {
		return nil
	}

	// Check plugin routes against existing + plugin routes combined
	combined := append(existingRoutes, pluginRoutes...)
	if hasConflict, conflicts := nginx.HasDomainConflict(combined); hasConflict {
		return fmt.Errorf("plugin %q has conflicting nginx routes: %s",
			plugin.Name, strings.Join(conflicts, "; "))
	}

	return nil
}
