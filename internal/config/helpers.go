package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

// userDefinedPrefixes lists env var prefixes that represent user-defined
// namespaces. Variables whose names start with one of these prefixes are
// intentionally unknown to the schema and must never trigger a warning.
var userDefinedPrefixes = []string{
	// User-defined namespaces (custom services, apps, schemas)
	"CS_",
	"FRONTEND_APP_",
	"REMOTE_SCHEMA_",
	"AUTH_PROVIDER_",
	"INTERNAL_ROUTE_",
	"HASURA_EXTRA_",

	// Plugin-managed vars — injected directly by plugin compose templates,
	// never read by the CLI loader. Listed by plugin:
	"STRIPE_",     // stripe plugin
	"GITHUB_",     // github plugin
	"SHOPIFY_",    // shopify plugin
	"OPENSEARCH_", // search plugin (OpenSearch provider)
	"ZINC_",       // search plugin (Zinc provider)
	"SONIC_",      // search plugin (Sonic provider)

	// Legacy microservice vars (pre-CS_N system, plugin-managed)
	"NESTJS_",
	"BULLMQ_",
	"GOLANG_",
	"PYTHON_",
	"DASHBOARD_",
	"SERVICES_",
}

// WarnUnknownEnvVars emits a slog.Warn for every key in loaded that is:
//   - not present in known, AND
//   - not matched by a user-defined prefix (CS_*, FRONTEND_APP_*, etc.)
//
// It does NOT warn about known vars that are absent from loaded — doing so
// would flood output for every optional service the user hasn't enabled
// (TRAP-11).
//
// When a loaded key is within edit-distance 2 of a known var, the warning
// includes a "did_you_mean" field with the closest match.
func warnUnknownEnvVars(loaded map[string]string, known []string) {
	// Build O(1) lookup for known vars.
	knownSet := make(map[string]bool, len(known))
	for _, k := range known {
		knownSet[k] = true
	}

	for k := range loaded {
		// Skip vars that are in the known set.
		if knownSet[k] {
			continue
		}

		// Skip user-defined namespace prefixes.
		skip := false
		for _, pfx := range userDefinedPrefixes {
			if strings.HasPrefix(k, pfx) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Emit warning. Include a "did you mean" hint when a close match exists.
		suggestion := closestKnownVar(k, known)
		if suggestion != "" {
			slog.Warn("unknown env var — check spelling (ignored)",
				"var", k,
				"did_you_mean", suggestion,
			)
		} else {
			slog.Warn("unknown env var — check spelling (ignored)",
				"var", k,
			)
		}
	}
}

// closestKnownVar returns the known var with the smallest Levenshtein distance
// to candidate, provided that distance is <= 2. Returns "" when no match
// is close enough.
func closestKnownVar(candidate string, known []string) string {
	bestVar := ""
	bestDist := 3 // distance > 2 means "no suggestion"

	for _, k := range known {
		d := editDistance(candidate, k)
		if d < bestDist {
			bestDist = d
			bestVar = k
		}
	}

	if bestDist <= 2 {
		return bestVar
	}
	return ""
}

// editDistance computes the Levenshtein edit distance between a and b using
// the standard two-row dynamic programming algorithm. Cost is O(len(a)*len(b)).
func editDistance(a, b string) int {
	la, lb := len(a), len(b)

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	// Base case: distance from "" to b[0..j] is j insertions.
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = minInt(del, minInt(ins, sub))
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// minInt returns the smaller of x and y.
func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// ── T38 ─────────────────────────────────────────────────────────────────────

// FindNSelfRoot walks up from startDir looking for a nself project root.
// It checks, at each directory level:
//  1. startDir/.backend/.env  → returns startDir/.backend (monorepo case)
//  2. startDir/.env           → returns startDir (already in backend dir)
//
// Walking stops at $HOME, at /, or after 10 levels — whichever comes first.
// Returns an error if no project root is found.
func FindNSelfRoot(startDir string) (string, error) {
	home, _ := os.UserHomeDir()

	dir := startDir
	for i := 0; i < 10; i++ {
		// Stop at home or filesystem root.
		// filepath.Dir(dir) == dir covers both Unix "/" and Windows drive roots
		// like "C:\" where filepath.Dir("C:\\") == "C:\\" (unlike "/" which
		// does not match Windows drive roots).
		if dir == home || filepath.Dir(dir) == dir {
			return "", fmt.Errorf("no nself project found")
		}

		// Monorepo case: .backend/.env exists one level down.
		if _, err := os.Stat(filepath.Join(dir, ".backend", ".env")); err == nil {
			return filepath.Join(dir, ".backend"), nil
		}

		// Already-in-backend case: .env exists directly here.
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return dir, nil
		}

		// Walk up one level.
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without hitting home guard above.
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no nself project found")
}
