// Package secrets implements encrypted secrets management for nSelf projects
// using age encryption (https://age-encryption.org).
//
// Secrets are stored as age-encrypted JSON files per environment:
//
//	.secrets/dev.age, .secrets/staging.age, .secrets/prod.age
//
// Each file encrypts to one or more age recipients (public keys), allowing
// team-based access control.
package secrets

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SecretsDir is the directory name under the project root.
const SecretsDir = ".secrets"

// SecretEntry represents a single secret with metadata.
type SecretEntry struct {
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	RotatedAt string `json:"rotated_at,omitempty"`
}

// SecretStore is the full set of secrets for one environment.
type SecretStore struct {
	Secrets    map[string]SecretEntry `json:"secrets"`
	Recipients []string               `json:"recipients"`
	UpdatedAt  string                 `json:"updated_at"`
}

// ageKeyPath returns the default path for the age private key.
func ageKeyPath() (string, error) {
	if p := os.Getenv("SECRETS_AGE_KEY_PATH"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "nself", "age-key.txt"), nil
}

// secretsDir returns the .secrets directory for the current project.
func secretsDir() string {
	return SecretsDir
}

// envFileName returns the encrypted file name for an environment.
func envFileName(env string) string {
	if env == "" {
		env = "dev"
	}
	return env + ".age"
}

// EnsureAgeInstalled checks that the age CLI is available.
func EnsureAgeInstalled() error {
	if _, err := exec.LookPath("age"); err != nil {
		return fmt.Errorf("age not found in PATH — install with: brew install age")
	}
	if _, err := exec.LookPath("age-keygen"); err != nil {
		return fmt.Errorf("age-keygen not found in PATH — install with: brew install age")
	}
	return nil
}

// Init generates an age keypair if one does not exist and sets up the
// .secrets directory with a .gitignore.
func Init(projectRoot string) error {
	if err := EnsureAgeInstalled(); err != nil {
		return err
	}

	keyPath, err := ageKeyPath()
	if err != nil {
		return err
	}

	// Generate keypair if missing.
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		dir := filepath.Dir(keyPath)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("creating key directory: %w", err)
		}
		cmd := exec.Command("age-keygen", "-o", keyPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("generating age key: %w\n%s", err, output)
		}
		if err := os.Chmod(keyPath, 0600); err != nil {
			return fmt.Errorf("setting key permissions: %w", err)
		}
		slog.Info("generated age key", "path", keyPath)
	} else {
		slog.Info("age key already exists", "path", keyPath)
	}

	// Create .secrets directory.
	sDir := filepath.Join(projectRoot, secretsDir())
	if err := os.MkdirAll(sDir, 0700); err != nil {
		return fmt.Errorf("creating secrets directory: %w", err)
	}

	// Write .gitignore.
	gitignorePath := filepath.Join(sDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, []byte("# Never commit decrypted secrets\n*.json\n*.tmp\n"), 0644); err != nil {
			return fmt.Errorf("writing .gitignore: %w", err)
		}
	}

	// Print the public key for sharing.
	pubKey, err := GetPublicKey(keyPath)
	if err != nil {
		return err
	}
	slog.Info("age public key", "public_key", pubKey)

	return nil
}

// GetPublicKey extracts the public key from an age key file.
func GetPublicKey(keyPath string) (string, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("reading key file: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# public key: ") {
			return strings.TrimPrefix(line, "# public key: "), nil
		}
	}
	return "", fmt.Errorf("no public key found in %s", keyPath)
}
