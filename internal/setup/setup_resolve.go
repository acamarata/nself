package setup

// setup_resolve.go — project name, env and secret resolution.
//
// Purpose: resolve the project name, environment and base domain for a new project, and generate secrets/passwords used while scaffolding it, used by Initialize in setup.go, split out for file size.
// Inputs: the user-supplied Options for the init run.
// Outputs: resolved strings (project name, env, base domain) and generated secrets/passwords.
// Constraints: pure move from setup.go (CLI-R12 Batch E); no behaviour change.

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nself-org/cli/internal/errs"
)

// resolveProjectName picks the project name from flag, env, directory, or default.
func resolveProjectName(opts Options) string {
	if opts.Name != "" {
		return opts.Name
	}
	if v := os.Getenv("PROJECT_NAME"); v != "" {
		return v
	}
	// Use directory name, sanitized.
	dir := filepath.Base(opts.WorkDir)
	name := strings.ToLower(dir)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Strip non-alphanumeric/hyphen chars.
	clean := regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "")
	if clean == "" || len(clean) < 2 {
		return "myproject"
	}
	if len(clean) > 30 {
		clean = clean[:30]
	}
	return clean
}

// validateProjectName checks the name against the required pattern.
func validateProjectName(name string) error {
	if len(name) < 2 || len(name) > 30 {
		return fmt.Errorf("project name must be 2-30 characters, got %d: %q: %w", len(name), name, errs.ErrInvalidProjectName)
	}
	if !projectNameRe.MatchString(name) {
		return fmt.Errorf("project name must be lowercase alphanumeric with hyphens (no leading/trailing hyphen): %q: %w", name, errs.ErrInvalidProjectName)
	}
	return nil
}

// resolveEnv determines the environment from env vars or defaults.
func resolveEnv(opts Options) string {
	if v := os.Getenv("ENV"); v != "" {
		switch strings.ToLower(v) {
		case "production", "prod":
			return "prod"
		case "staging", "stage":
			return "staging"
		default:
			return "dev"
		}
	}
	return "dev"
}

// resolveBaseDomain determines the base domain.
// Priority: opts.Domain (flag/wizard) > BASE_DOMAIN env var > default.
func resolveBaseDomain(opts Options, env string) string {
	if opts.Domain != "" {
		return opts.Domain
	}
	if v := os.Getenv("BASE_DOMAIN"); v != "" {
		return v
	}
	if env == "dev" {
		return "local.nself.org"
	}
	return "localhost"
}

// GenerateSecret returns a crypto/rand base64url string of the given length,
// with URL-unsafe characters stripped. Suitable for passwords and API keys.
func GenerateSecret(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	// Strip characters that can cause issues in shell/.env.
	encoded = strings.NewReplacer("+", "", "/", "", "=", "").Replace(encoded)
	if len(encoded) < length {
		return encoded, nil
	}
	return encoded[:length], nil
}

// generateSecurePassword returns a cryptographically random alphanumeric
// password of the given length using crypto/rand.
func generateSecurePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// knownBadMinioPasswords is the set of well-known default MinIO credentials
// that must be replaced with a secure random password on init.
var knownBadMinioPasswords = map[string]bool{
	"minioadmin": true,
	"minio123":   true,
	"admin":      true,
}

// resolveMinioPassword returns a secure MinIO password. If the provided
// value is empty or a known-bad default, a new 32-char random alphanumeric
// password is generated. Any other value (user-set) is returned unchanged.
func resolveMinioPassword(current string) (string, error) {
	if current == "" || knownBadMinioPasswords[current] {
		return generateSecurePassword(32)
	}
	return current, nil
}
