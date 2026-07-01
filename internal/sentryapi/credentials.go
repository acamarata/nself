package sentryapi

// credentials.go — local credential storage + config resolution.
//
// Purpose: Persist the ɳSentry API key at ~/.nself/sentry.json (0600) and
//   resolve the effective api-url/api-key across flag → env → file → default.
// Inputs:  NSELF_SENTRY_API_URL / NSELF_SENTRY_API_KEY env vars, credentials file.
// Outputs: Credentials struct; Resolve() precedence result.
// Constraints: file mode 0600 (P15 env-perms lesson); never log the key.
// SPORT: CLI-PKG-SENTRYAPI-001

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvAPIURL / EnvAPIKey are the environment overrides for the client config.
const (
	EnvAPIURL = "NSELF_SENTRY_API_URL"
	EnvAPIKey = "NSELF_SENTRY_API_KEY"
)

// ErrNotLoggedIn is returned when no credentials file exists.
var ErrNotLoggedIn = errors.New("not logged in to ɳSentry — run 'nself sentry login'")

// Credentials is the on-disk shape of ~/.nself/sentry.json.
type Credentials struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

// credentialsPath returns ~/.nself/sentry.json (honors HOME for tests).
func credentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".nself", "sentry.json"), nil
}

// ReadCredentials loads the stored credentials, or ErrNotLoggedIn.
func ReadCredentials() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotLoggedIn
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &c, nil
}

// WriteCredentials persists credentials at 0600 (dir 0700).
func WriteCredentials(c *Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	// Defence-in-depth: enforce 0600 even if the file pre-existed.
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}

// DeleteCredentials removes the stored credentials (no-op when absent).
func DeleteCredentials() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// Resolve computes the effective (apiURL, apiKey) using precedence:
// explicit args (flags) → env vars → credentials file → defaults.
// A missing key is not an error here — the API returns 401 and the client
// maps it to ErrUnauthorized with the login hint.
func Resolve(flagURL, flagKey string) (apiURL, apiKey string) {
	apiURL = flagURL
	apiKey = flagKey
	if apiURL == "" {
		apiURL = os.Getenv(EnvAPIURL)
	}
	if apiKey == "" {
		apiKey = os.Getenv(EnvAPIKey)
	}
	if apiURL == "" || apiKey == "" {
		if creds, err := ReadCredentials(); err == nil {
			if apiURL == "" {
				apiURL = creds.APIURL
			}
			if apiKey == "" {
				apiKey = creds.APIKey
			}
		}
	}
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	return apiURL, apiKey
}

// ValidateKeyFormat checks the nsk_ prefix without hitting the network.
func ValidateKeyFormat(key string) error {
	if !strings.HasPrefix(key, KeyPrefix) {
		return fmt.Errorf("invalid API key: must start with %q", KeyPrefix)
	}
	if len(key) < len(KeyPrefix)+8 {
		return errors.New("invalid API key: too short")
	}
	return nil
}
