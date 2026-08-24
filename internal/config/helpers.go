package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getEnvOr returns the value of the environment variable named by key,
// or fallback if the variable is empty or unset.
func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt parses the environment variable named by key as an integer.
// Returns fallback if the variable is empty, unset, or not a valid integer.
func getEnvInt(key string, fallback int) int {
	s := os.Getenv(key)
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}

// getEnvBool parses the environment variable named by key as a boolean.
// Accepts true/1/yes/on/enabled (case-insensitive) as true values.
// Returns fallback if the variable is empty or unset.
func getEnvBool(key string, fallback bool) bool {
	s := strings.ToLower(os.Getenv(key))
	if s == "" {
		return fallback
	}
	return s == "true" || s == "1" || s == "yes" || s == "on" || s == "enabled"
}

// normalizeEnv normalizes environment name aliases to canonical short forms.
// development/develop/devel become "dev", production becomes "prod",
// stage becomes "staging". Unknown values are lowercased and returned as-is.
func normalizeEnv(env string) string {
	switch strings.ToLower(env) {
	case "development", "develop", "devel":
		return "dev"
	case "production":
		return "prod"
	case "stage":
		return "staging"
	default:
		return strings.ToLower(env)
	}
}

// ── RouteToFQDN ──────────────────────────────────────────────────────────────

// RouteToFQDN constructs a valid FQDN from a route segment and base domain.
// It trims whitespace, removes leading/trailing dots and slashes from both
// inputs, lowercases both, and returns "route.domain".
// Returns error if either input is empty after normalization.
func RouteToFQDN(route, baseDomain string) (string, error) {
	r := strings.ToLower(strings.TrimSpace(route))
	r = strings.Trim(r, "/.")
	if r == "" {
		return "", fmt.Errorf("route is empty after normalization")
	}

	d := strings.ToLower(strings.TrimSpace(baseDomain))
	d = strings.TrimRight(d, ".")
	if d == "" {
		return "", fmt.Errorf("baseDomain is empty after normalization")
	}

	return r + "." + d, nil
}

// ── T14 ─────────────────────────────────────────────────────────────────────
