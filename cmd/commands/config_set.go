package commands

// Purpose: RunE for "nself config set" plus its key/value validators and the
// setEnvFileLine helper that rewrites the env file in place. Inputs are the
// cobra command/args; outputs are an updated env file or an error.
// Constraints: split out of config.go (CLI-R12) as a pure move, no behavior change.

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/nself-org/cli/internal/security"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]
	envFlag, _ := cmd.Flags().GetString("env")

	// Validate key: only [A-Z0-9_] allowed, max 128 chars.
	if err := validateConfigKey(key); err != nil {
		return err
	}
	// Validate value: no null bytes.
	if err := validateConfigValue(value); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)

	// Read the existing file line by line and update in-place.
	// If the key doesn't exist, append it.
	updated, err := setEnvFileLine(envFile, key, value)
	if err != nil {
		return err
	}

	// FindNSelfRoot's monorepo convenience silently resolves to a
	// `.backend/` project root above cwd when one exists — fine for reads,
	// but a write with no indication of where it actually landed is
	// dangerous: a user working in a nested app dir sees "Added KEY to
	// .env" with no signal the file is two directories up. Surface the
	// redirection explicitly whenever it happens; this does not change
	// where the write lands, only what gets printed.
	if projectDir != cwd {
		ui.Info(fmt.Sprintf("Using monorepo backend at %s (found via .backend/ in a parent directory)", projectDir))
	}

	if updated {
		ui.Success(fmt.Sprintf("Updated %s in %s", key, envFile))
	} else {
		ui.Success(fmt.Sprintf("Added %s to %s", key, envFile))
	}
	return nil
}

// configKeyRe matches valid config keys: uppercase letters, digits, underscores only.
var configKeyRe = regexp.MustCompile(`^[A-Z0-9_]+$`)

// validateConfigKey returns an error if key contains invalid characters,
// is empty, or exceeds 128 characters.
func validateConfigKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("config key cannot be empty")
	}
	if len(key) > 128 {
		return fmt.Errorf("config key too long (max 128 characters)")
	}
	if !configKeyRe.MatchString(key) {
		return fmt.Errorf("config key %q contains invalid characters (only A-Z, 0-9, _ allowed)", key)
	}
	return nil
}

// validateConfigValue returns an error if value contains null bytes.
func validateConfigValue(value string) error {
	for i, b := range []byte(value) {
		if b == 0 {
			return fmt.Errorf("config value contains null byte at position %d", i)
		}
	}
	return nil
}

// setEnvFileLine updates KEY=VALUE in the named file in-place, preserving all
// comments and blank lines. If the key is not found it is appended. Returns
// true when an existing line was replaced, false when the key was appended.
func setEnvFileLine(envFile, key, value string) (updated bool, err error) {
	// quoteIfNeeded wraps value in double-quotes when it contains spaces,
	// special characters, or is empty.
	quoteIfNeeded := func(v string) string {
		if v == "" || strings.ContainsAny(v, " \t#\"'\\$`") {
			return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
		}
		return v
	}
	newLine := key + "=" + quoteIfNeeded(value)

	// If the file does not exist, create it with just this key.
	if _, statErr := os.Stat(envFile); os.IsNotExist(statErr) {
		if writeErr := os.WriteFile(envFile, []byte(newLine+"\n"), 0600); writeErr != nil {
			return false, fmt.Errorf("creating %s: %w", envFile, writeErr)
		}
		if chmodErr := security.EnforceFilePermissions(envFile, 0600); chmodErr != nil {
			return false, fmt.Errorf("enforcing permissions on %s: %w", envFile, chmodErr)
		}
		return false, nil
	}

	// Read existing file.
	f, openErr := os.Open(envFile)
	if openErr != nil {
		return false, fmt.Errorf("opening %s: %w", envFile, openErr)
	}
	defer func() { _ = f.Close() }()

	var lines []string
	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		// Skip comment and blank lines.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
			continue
		}
		// Match KEY= or KEY =
		eqIdx := strings.IndexByte(trimmed, '=')
		if eqIdx > 0 {
			lineKey := strings.TrimSpace(trimmed[:eqIdx])
			if lineKey == key {
				lines = append(lines, newLine)
				found = true
				continue
			}
		}
		lines = append(lines, line)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return false, fmt.Errorf("scanning %s: %w", envFile, scanErr)
	}
	_ = f.Close()

	if !found {
		lines = append(lines, newLine)
	}

	content := strings.Join(lines, "\n") + "\n"
	if writeErr := os.WriteFile(envFile, []byte(content), 0600); writeErr != nil {
		return false, fmt.Errorf("writing %s: %w", envFile, writeErr)
	}
	if chmodErr := security.EnforceFilePermissions(envFile, 0600); chmodErr != nil {
		return false, fmt.Errorf("enforcing permissions on %s: %w", envFile, chmodErr)
	}
	return found, nil
}

// --- S4-T04: config list ---

// knownVarsExported mirrors loader.go's knownEnvVars. We access it through the
// exported function below rather than duplicating the list.
