package config

// loader_helpers.go — small parsing helpers used by the config loader.
//
// Purpose: Three focused helpers that parse dynamic or structured env var sets
//          that cannot be reduced to a single getEnvOr call. Each lives here
//          to keep loader.go and loader_parse_env.go focused on orchestration
//          and field mapping respectively.
// Inputs:  os.Environ (collectPassthrough) and os.Getenv (parseInternalRoutes,
//          parseExtensionList) — no filesystem access.
// Outputs: collectPassthrough → map[string]string; parseInternalRoutes →
//          []InternalRoute; parseExtensionList → []string.
// Constraints: No side effects beyond returning values. Must not call
//              ApplyDefaults or touch the filesystem.
// SPORT:   cli/internal/config — decomposed from loader.go (T-E2-06).

import (
	"fmt"
	"os"
	"strings"
)

// collectPassthrough scans the full environment for dynamic env vars matching
// known prefixes (AUTH_PROVIDER_*, REMOTE_SCHEMA_*, HASURA_EXTRA_*) and returns
// them as a key-value map. These variables cannot be predefined in structs
// because users add them dynamically for OAuth providers, remote schemas, etc.
func collectPassthrough(environ []string) map[string]string {
	prefixes := []string{
		"AUTH_PROVIDER_",
		"REMOTE_SCHEMA_",
		"HASURA_EXTRA_",
	}
	result := make(map[string]string)
	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(parts[0], prefix) {
				result[parts[0]] = parts[1]
			}
		}
	}
	return result
}

// parseInternalRoutes parses INTERNAL_ROUTE_1 through INTERNAL_ROUTE_20
// environment variables into InternalRoute structs. Each route is defined by:
//
//	INTERNAL_ROUTE_N_NAME       — required (skip if empty)
//	INTERNAL_ROUTE_N_SUBDOMAIN
//	INTERNAL_ROUTE_N_TARGET     — e.g., hasura:8080
//	INTERNAL_ROUTE_N_RATE_ZONE  — default: general
//	INTERNAL_ROUTE_N_WEBSOCKET  — bool
func parseInternalRoutes() []InternalRoute {
	var routes []InternalRoute
	for i := 1; i <= 20; i++ {
		prefix := fmt.Sprintf("INTERNAL_ROUTE_%d_", i)
		name := os.Getenv(prefix + "NAME")
		if name == "" {
			continue
		}

		route := InternalRoute{
			Index:     i,
			Name:      name,
			Subdomain: os.Getenv(prefix + "SUBDOMAIN"),
			Target:    os.Getenv(prefix + "TARGET"),
			RateZone:  getEnvOr(prefix+"RATE_ZONE", "general"),
			WebSocket: getEnvBool(prefix+"WEBSOCKET", false),
		}
		routes = append(routes, route)
	}
	return routes
}

// parseExtensionList parses a comma-separated extension list string into a slice.
// Trims whitespace from each element and removes empty entries.
func parseExtensionList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
