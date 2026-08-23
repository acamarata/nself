package commands

// Purpose: shared config helpers: secret-key detection/masking, resolving the
// active env file name, and resolving the project directory. Inputs are a
// config key/value or project/env flags; outputs are bools, masked strings, or
// a resolved path.
// Constraints: split out of config.go (CLI-R12) as a pure move, no behavior change.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// secretKeyParts contains substrings whose presence in a key name indicates
// a secret value that should be masked by default.
var secretKeyParts = []string{"SECRET", "PASSWORD", "KEY", "TOKEN"}

// isSecretKey returns true when the key name contains any of the secret
// indicator substrings (case-insensitive comparison against uppercased key).
func isSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, part := range secretKeyParts {
		if strings.Contains(upper, part) {
			return true
		}
	}
	return false
}

// maskValue returns "****" when reveal is false and the key is a secret key.
func maskValue(key, value string, reveal bool) string {
	if !reveal && isSecretKey(key) && value != "" {
		return "***"
	}
	return value
}

// envFileName returns the .env filename to use for the given env flag value.
// An empty envFlag means use the plain ".env" file.
func envFileName(projectDir, envFlag string) string {
	if envFlag == "" {
		return filepath.Join(projectDir, ".env")
	}
	return filepath.Join(projectDir, ".env."+envFlag)
}

// resolveProjectDir returns the nself project root, used by all config
// subcommands that need to locate .env files.
func resolveProjectDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	dir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("no nself project found in current directory or parents — run 'nself init' first")
	}
	return dir, nil
}

// --- S4-T01: config show ---
