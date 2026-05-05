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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
		fmt.Printf("Generated age key at %s\n", keyPath)
	} else {
		fmt.Printf("Age key already exists at %s\n", keyPath)
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
	fmt.Printf("Your public key (share with team): %s\n", pubKey)

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

// loadStore decrypts and reads the secret store for an environment.
// Returns an empty store if the file does not exist.
func loadStore(projectRoot, env string) (*SecretStore, error) {
	path := filepath.Join(projectRoot, secretsDir(), envFileName(env))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &SecretStore{
			Secrets: make(map[string]SecretEntry),
		}, nil
	}

	keyPath, err := ageKeyPath()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("age", "--decrypt", "-i", keyPath, path)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("decrypting secrets: %w", err)
	}

	var store SecretStore
	if err := json.Unmarshal(output, &store); err != nil {
		return nil, fmt.Errorf("parsing secrets: %w", err)
	}
	if store.Secrets == nil {
		store.Secrets = make(map[string]SecretEntry)
	}
	return &store, nil
}

// saveStore encrypts and writes the secret store for an environment.
func saveStore(projectRoot, env string, store *SecretStore) error {
	store.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling secrets: %w", err)
	}

	path := filepath.Join(projectRoot, secretsDir(), envFileName(env))
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating secrets directory: %w", err)
	}

	// Build age encrypt command with recipients.
	args := []string{"--encrypt"}
	if len(store.Recipients) > 0 {
		for _, r := range store.Recipients {
			args = append(args, "-r", r)
		}
	} else {
		// Use our own public key as the only recipient.
		keyPath, err := ageKeyPath()
		if err != nil {
			return err
		}
		pubKey, err := GetPublicKey(keyPath)
		if err != nil {
			return err
		}
		store.Recipients = []string{pubKey}
		// Re-marshal with recipients included.
		data, err = json.MarshalIndent(store, "", "  ")
		if err != nil {
			return fmt.Errorf("marshalling secrets: %w", err)
		}
		args = append(args, "-r", pubKey)
	}
	args = append(args, "-o", path)

	cmd := exec.Command("age", args...)
	cmd.Stdin = strings.NewReader(string(data))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("encrypting secrets: %w\n%s", err, output)
	}
	return nil
}

// Set adds or updates a secret.
func Set(projectRoot, env, key, value string) error {
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	entry, exists := store.Secrets[key]
	if !exists {
		entry = SecretEntry{CreatedAt: now}
	}
	entry.Value = value
	entry.UpdatedAt = now
	store.Secrets[key] = entry
	if saveErr := saveStore(projectRoot, env, store); saveErr != nil {
		slog.Error("secrets_set_failed", "key_name", key, "env", env, "error", saveErr)
		return saveErr
	}
	slog.Info("secrets_set", "key_name", key, "env", env)
	return nil
}

// Get retrieves a secret value.
func Get(projectRoot, env, key string) (string, error) {
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return "", err
	}
	entry, ok := store.Secrets[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found in %s environment", key, env)
	}
	return entry.Value, nil
}

// List returns all secret keys with metadata for an environment.
func List(projectRoot, env string) ([]string, map[string]SecretEntry, error) {
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return nil, nil, err
	}
	var keys []string
	for k := range store.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, store.Secrets, nil
}

// Rotate generates a new value for a secret based on its type/name pattern.
func Rotate(projectRoot, env, key string) (string, error) {
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return "", err
	}
	_, ok := store.Secrets[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found in %s environment", key, env)
	}

	newValue, hint := generateRotationValue(key)
	if hint != "" {
		fmt.Printf("Note: %s\n", hint)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	store.Secrets[key] = SecretEntry{
		Value:     newValue,
		CreatedAt: store.Secrets[key].CreatedAt,
		UpdatedAt: now,
		RotatedAt: now,
	}
	if err := saveStore(projectRoot, env, store); err != nil {
		slog.Error("secrets_rotate_failed", "key_name", key, "env", env, "error", err)
		return "", err
	}
	slog.Info("secrets_rotated", "key_name", key, "env", env)
	return newValue, nil
}

// generateRotationValue produces a new secret value based on the key name.
func generateRotationValue(key string) (value string, hint string) {
	keyUpper := strings.ToUpper(key)
	switch {
	case strings.HasSuffix(keyUpper, "_PASSWORD") || strings.HasSuffix(keyUpper, "_PASS"):
		return generateRandomString(32), "Remember to update the corresponding database/service password."
	case strings.Contains(keyUpper, "JWT") || strings.Contains(keyUpper, "SECRET"):
		return generateRandomString(64), "Services using this secret will need to be restarted."
	case strings.Contains(keyUpper, "API_KEY") || strings.Contains(keyUpper, "TOKEN"):
		return "", "API keys and tokens must be rotated through the provider's dashboard. Update the value with 'nself secrets set'."
	default:
		return generateRandomString(32), ""
	}
}

// generateRandomString generates a cryptographically random alphanumeric string.
func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}

// Audit checks for secrets that haven't been rotated in over 90 days.
func Audit(projectRoot, env string) ([]AuditFinding, error) {
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return nil, err
	}

	var findings []AuditFinding
	for key, entry := range store.Secrets {
		lastRotated := entry.RotatedAt
		if lastRotated == "" {
			lastRotated = entry.CreatedAt
		}
		if lastRotated == "" {
			findings = append(findings, AuditFinding{
				Key:      key,
				Issue:    "no creation or rotation timestamp",
				Severity: "high",
			})
			continue
		}
		t, err := time.Parse(time.RFC3339, lastRotated)
		if err != nil {
			continue
		}
		age := time.Since(t)
		if age > 90*24*time.Hour {
			findings = append(findings, AuditFinding{
				Key:      key,
				Issue:    fmt.Sprintf("last rotated %d days ago (threshold: 90 days)", int(age.Hours()/24)),
				Severity: "warning",
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Key < findings[j].Key
	})
	return findings, nil
}

// AuditFinding represents a single audit finding.
type AuditFinding struct {
	Key      string
	Issue    string
	Severity string
}

// DecryptForDeploy decrypts secrets and outputs them as KEY=VALUE lines
// suitable for .env.computed or CI/CD injection.
func DecryptForDeploy(projectRoot, env string) (string, error) {
	// Check for DEPLOY_AGE_KEY env var (used in CI).
	deployKey := os.Getenv("DEPLOY_AGE_KEY")
	if deployKey != "" {
		return decryptWithKey(projectRoot, env, deployKey)
	}
	// Fall back to local key.
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return "", err
	}
	var lines []string
	var keys []string
	for k := range store.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, store.Secrets[k].Value))
	}
	return strings.Join(lines, "\n"), nil
}

// decryptWithKey decrypts using a raw age private key string (for CI/CD).
func decryptWithKey(projectRoot, env, key string) (string, error) {
	path := filepath.Join(projectRoot, secretsDir(), envFileName(env))

	// Write key to a temp file.
	tmpKey, err := os.CreateTemp("", "nself-deploy-key-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpKey.Name())
	if _, err := tmpKey.WriteString(key); err != nil {
		return "", err
	}
	tmpKey.Close()

	cmd := exec.Command("age", "--decrypt", "-i", tmpKey.Name(), path)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("decrypting with deploy key: %w", err)
	}

	var store SecretStore
	if err := json.Unmarshal(output, &store); err != nil {
		return "", fmt.Errorf("parsing secrets: %w", err)
	}
	var lines []string
	var keys []string
	for k := range store.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, store.Secrets[k].Value))
	}
	return strings.Join(lines, "\n"), nil
}

// Rekey re-encrypts all secret files, removing the specified public key
// from the recipients list. Used when a team member leaves.
func Rekey(projectRoot, removePubKey string) error {
	sDir := filepath.Join(projectRoot, secretsDir())
	entries, err := os.ReadDir(sDir)
	if err != nil {
		return fmt.Errorf("reading secrets directory: %w", err)
	}

	rekeyCount := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".age") {
			continue
		}
		env := strings.TrimSuffix(entry.Name(), ".age")
		store, err := loadStore(projectRoot, env)
		if err != nil {
			return fmt.Errorf("loading %s: %w", env, err)
		}

		// Remove the specified recipient.
		var newRecipients []string
		removed := false
		for _, r := range store.Recipients {
			if r == removePubKey {
				removed = true
				continue
			}
			newRecipients = append(newRecipients, r)
		}
		if !removed {
			continue
		}
		if len(newRecipients) == 0 {
			return fmt.Errorf("cannot remove last recipient from %s — add a new recipient first", env)
		}
		store.Recipients = newRecipients
		if err := saveStore(projectRoot, env, store); err != nil {
			return fmt.Errorf("re-encrypting %s: %w", env, err)
		}
		rekeyCount++
		fmt.Printf("Rekeyed %s.age (removed recipient)\n", env)
	}

	if rekeyCount == 0 {
		fmt.Println("No files contained the specified recipient.")
	} else {
		fmt.Printf("Rekeyed %d environment(s).\n", rekeyCount)
	}
	return nil
}

// LintSecrets checks for plaintext secrets in git-tracked files.
func LintSecrets(projectRoot string) ([]LintFinding, error) {
	// Check if gitleaks is installed.
	gitleaksPath, err := exec.LookPath("gitleaks")
	if err != nil {
		return nil, fmt.Errorf("gitleaks not found in PATH — install with: brew install gitleaks")
	}

	cmd := exec.Command(gitleaksPath, "detect", "--source", projectRoot, "--no-git", "--report-format", "json", "--report-path", "/dev/stdout")
	output, err := cmd.Output()

	// gitleaks exits 1 when findings are present.
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Parse findings.
			var findings []LintFinding
			if jsonErr := json.Unmarshal(output, &findings); jsonErr != nil {
				// If JSON parse fails, just report raw output.
				return []LintFinding{{
					File:    "unknown",
					Rule:    "parse-error",
					Message: string(output),
				}}, nil
			}
			return findings, nil
		}
		return nil, fmt.Errorf("running gitleaks: %w", err)
	}

	return nil, nil
}

// LintFinding represents a detected secret in source code.
type LintFinding struct {
	File    string `json:"File"`
	Rule    string `json:"RuleID"`
	Message string `json:"Description"`
	Line    int    `json:"StartLine"`
}
