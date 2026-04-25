// Package config provides env-first configuration loading for nSelf plugins.
// Every plugin follows the same pattern: read env vars, fall back to defaults,
// fail fast on missing required values.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env returns the value of key or def when key is unset or empty.
func Env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// EnvRequired returns the value of key, or an error when key is unset or empty.
// Plugins call this in their Load() path to halt startup when a required var
// is missing (e.g. DATABASE_URL, HASURA_ADMIN_SECRET).
func EnvRequired(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("config: required env var %s is not set", key)
	}
	return v, nil
}

// EnvInt parses key as int, falling back to def on miss or parse error.
func EnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// EnvBool parses key as bool ("true", "1", "yes" → true; "false", "0", "no" → false).
// Returns def on miss or unknown value.
func EnvBool(key string, def bool) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "true", "1", "yes", "y", "on":
		return true
	case "false", "0", "no", "n", "off":
		return false
	default:
		return def
	}
}

// EnvDuration parses key as time.Duration (e.g. "30s", "5m"), falling back to def.
func EnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

// EnvList splits key on commas and trims whitespace. Empty input → empty slice.
func EnvList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
