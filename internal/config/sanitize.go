package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Sanitized entry points:
// - PROJECT_NAME (loader.go) via SanitizeName
// - BASE_DOMAIN (loader.go) via SanitizeDomain
// - --name flag (cmd/commands/init.go) via SanitizeName
// - --domain flag (cmd/commands/init.go) via SanitizeDomain
// - REMOTE_SCHEMA_N_NAME (remote_schemas.go) via SanitizeName
// - FRONTEND_APP_N_DISPLAY_NAME (frontend_apps.go) via SanitizeName
// - FRONTEND_APP_N_SYSTEM_NAME (frontend_apps.go) via SanitizeName
// - FRONTEND_APP_N_ROUTE (frontend_apps.go) via SanitizeDomain
// - FRONTEND_APP_N_IMAGE (frontend_apps.go) via validateFrontendAppImage
// - CS_N name (custom_services.go) via validateCustomServiceName + SanitizeName
// - REMOTE_SCHEMA_N_URL (remote_schemas.go) via validateRemoteSchemaURL
// - POSTGRES_DB derived from PROJECT_NAME (internal/setup) via SanitizeDBName
// Add new entries here when adding new user-facing inputs.
//
// Audited and not sanitized (different validation concerns):
// - cmd/commands/deploy.go --name/--domain: server infrastructure names (not config values)
// - cmd/commands/plugin.go --key/--version: license keys and semver (no normalization needed)
// - cmd/commands/exec.go --user/--workdir: container exec params (passed to Docker directly)

var (
	reSanitizeInvalid = regexp.MustCompile(`[^a-z0-9-]`)
	reSanitizeHyphens = regexp.MustCompile(`-{2,}`)
	reDomainValid     = regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)
	reDBNameInvalid   = regexp.MustCompile(`[^a-z0-9_]`)
)

// maxDBNameBytes is the PostgreSQL identifier byte limit (NAMEDATALEN-1 = 63).
// Duplicated from internal/database's identical constant: internal/database
// imports internal/config (for *config.Config), so config cannot import
// database without a cycle. Both values encode the same Postgres fact and
// must be kept in sync.
const maxDBNameBytes = 63

// SanitizeName lowercases, trims whitespace, replaces spaces/underscores with
// hyphens, removes all non-alphanumeric-hyphen characters, collapses consecutive
// hyphens, and trims leading/trailing hyphens.
// Returns ("", err) if the result is empty after normalization.
func SanitizeName(input string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(input))
	// Replace spaces and underscores with hyphens before stripping.
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove all characters that are not a-z, 0-9, or hyphen.
	name = reSanitizeInvalid.ReplaceAllString(name, "")
	// Collapse consecutive hyphens.
	name = reSanitizeHyphens.ReplaceAllString(name, "-")
	// Trim leading/trailing hyphens.
	name = strings.Trim(name, "-")
	if name == "" {
		return "", fmt.Errorf("contains no valid characters after sanitization")
	}
	return name, nil
}

// SanitizeDBName converts a Docker-compatible project name (as produced by
// SanitizeName: lowercase alphanumeric + hyphens) into a value that satisfies
// the stricter PostgreSQL unquoted-identifier syntax enforced by
// database.SanitizeIdentifier (start with a letter or underscore; only
// letters, digits, and underscores after that; max 63 bytes).
//
// PROJECT_NAME and POSTGRES_DB have different validity domains — Docker
// container/network names allow hyphens and a leading digit, SQL identifiers
// allow neither — so the raw project name (e.g. from a hyphenated directory
// like "rls-pentest-project") cannot be reused verbatim as the database name.
// This normalizes it instead of rejecting it, so init always produces a
// working POSTGRES_DB regardless of the source directory name.
//
// Returns ("", err) only if nothing usable remains after normalization (e.g.
// an input made entirely of characters outside [a-z0-9_-. ]).
func SanitizeDBName(input string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(input))
	// Fold the characters most likely to appear in a project/directory name
	// but disallowed in a SQL identifier into underscores, rather than
	// dropping them, so "my-project" and "my.project" don't collide into the
	// same "myproject" name.
	name = strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(name)
	// Remove anything else that isn't a lowercase letter, digit, or underscore.
	name = reDBNameInvalid.ReplaceAllString(name, "")
	// identRegex (internal/database) requires the identifier to start with a
	// letter or underscore. PROJECT_NAME permits a leading digit (e.g.
	// "9lives"), which is not a valid identifier start, so prefix it.
	if name != "" && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}
	// Reject a result made entirely of underscores (e.g. input "---"): it is
	// technically a valid SQL identifier, but it carries no trace of the
	// original name and isn't a useful database name.
	if strings.Trim(name, "_") == "" {
		return "", fmt.Errorf("contains no valid characters after sanitization")
	}
	if len(name) > maxDBNameBytes {
		name = strings.TrimRight(name[:maxDBNameBytes], "_")
		if strings.Trim(name, "_") == "" {
			return "", fmt.Errorf("contains no valid characters after truncation to %d bytes", maxDBNameBytes)
		}
	}
	return name, nil
}

// SanitizeRoute lowercases, trims whitespace, strips leading/trailing slashes,
// and replaces spaces with hyphens.
// Returns ("", err) if the result is empty after normalization.
func sanitizeRoute(input string) (string, error) {
	route := strings.ToLower(strings.TrimSpace(input))
	route = strings.Trim(route, "/")
	route = strings.ReplaceAll(route, " ", "-")
	if route == "" {
		return "", fmt.Errorf("contains no valid characters after sanitization")
	}
	return route, nil
}

// SanitizePath cleans the path using filepath.Clean to remove redundant separators
// and resolve dot elements. Returns the cleaned path; returns an error if input
// is empty.
func sanitizePath(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("path is empty")
	}
	return filepath.Clean(input), nil
}

// SanitizePort trims whitespace and validates that the value is numeric.
// Returns ("", err) if the result is empty or non-numeric.
func sanitizePort(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("port is empty")
	}
	if _, err := strconv.Atoi(trimmed); err != nil {
		return "", fmt.Errorf("port %q is not numeric", trimmed)
	}
	return trimmed, nil
}

// SanitizeDomain lowercases, trims whitespace, trims trailing dots, and
// validates that only hostname-safe characters remain: letters, digits, dots,
// and hyphens ([a-zA-Z0-9.-]). Any other character (semicolons, spaces,
// asterisks, dollar signs, newlines, etc.) causes an error to prevent nginx
// config injection and similar attacks.
// Returns ("", err) if the result is empty or contains invalid characters.
func SanitizeDomain(input string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(input))
	domain = strings.TrimRight(domain, ".")
	if domain == "" {
		return "", fmt.Errorf("contains no valid characters after sanitization")
	}
	if !reDomainValid.MatchString(domain) {
		return "", fmt.Errorf("domain %q contains invalid characters (only letters, digits, dots, and hyphens are allowed)", domain)
	}
	return domain, nil
}

// customServiceNameRe: lowercase, alphanumeric + hyphens/underscores, 2-63 chars.
var customServiceNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,61}[a-z0-9]$`)

// validateCustomServiceName validates CS_N service names.
// Must be lowercase, alphanumeric + hyphens/underscores, 2–63 chars.
func validateCustomServiceName(name string) error {
	if !customServiceNameRe.MatchString(name) {
		return fmt.Errorf("invalid custom service name %q (must be lowercase alphanumeric with hyphens/underscores, 2–63 chars)", name)
	}
	return nil
}

// frontendAppImageRe: validates a Docker image reference.
// Rejects shell-injection characters (semicolons, ampersands, spaces, backticks, etc.).
var frontendAppImageRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:\-@]*$`)

// validateFrontendAppImage validates a Docker image reference.
// Returns nil if image is empty (image is optional).
func validateFrontendAppImage(image string) error {
	if image == "" {
		return nil // image is optional
	}
	if !frontendAppImageRe.MatchString(image) {
		return fmt.Errorf("invalid frontend app image %q: contains forbidden characters", image)
	}
	return nil
}

// validateRemoteSchemaURL validates a Hasura remote schema URL.
// Must be http:// or https:// scheme. Rejects empty, javascript:, and other unsafe schemes.
func validateRemoteSchemaURL(raw string) error {
	if raw == "" {
		return nil // URL is optional
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid remote schema URL %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid remote schema URL %q: scheme must be http or https", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid remote schema URL %q: missing host", raw)
	}
	return nil
}
