package backup

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/security"
)

// InitKey generates an age keypair for backup encryption.
// Private key: ~/.config/nself/{project}-age.key (mode 0600)
// Public key: written to BACKUP_AGE_RECIPIENTS in the project .env.
func InitKey(cfg *config.Config) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	keyDir := filepath.Join(home, ".config", "nself")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return "", fmt.Errorf("create key directory: %w", err)
	}

	keyFile := filepath.Join(keyDir, cfg.ProjectName+"-age.key")

	// Check if key already exists.
	if _, err := os.Stat(keyFile); err == nil {
		return "", fmt.Errorf("age key already exists at %s — remove it first to regenerate", keyFile)
	}

	// Generate key using age-keygen.
	cmd := exec.Command("age-keygen", "-o", keyFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("age-keygen failed: %s: %w", string(output), err)
	}

	// Enforce 0600 permissions.
	if err := security.EnforceFilePermissions(keyFile, 0600); err != nil {
		return "", fmt.Errorf("enforce key permissions: %w", err)
	}

	// Extract public key from the output.
	// age-keygen output format: "Public key: age1..."
	var pubKey string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "Public key:") {
			pubKey = strings.TrimSpace(strings.TrimPrefix(line, "Public key:"))
			break
		}
	}
	// Also try reading from the key file header.
	if pubKey == "" {
		data, err := os.ReadFile(keyFile)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "# public key:") {
					pubKey = strings.TrimSpace(strings.TrimPrefix(line, "# public key:"))
					break
				}
			}
		}
	}

	if pubKey == "" {
		slog.Warn("could not extract public key from age-keygen output — check key file manually", "path", keyFile)
	}

	slog.Info("age keypair generated",
		"private_key", keyFile,
		"public_key", pubKey,
	)

	return pubKey, nil
}
