package doctor

// hardening_check_helpers.go — shared env-file helpers for hardening checks.
//
// Purpose: Read KEY=VALUE pairs from standard nSelf env files (.env.prod,
//          .env.staging, .env.dev, .env.secrets, .env.local, .env) under
//          projectDir. Used by hardening_check_auth_net.go and
//          hardening_check_infra.go to check whether required env vars are set.
// Inputs:  projectDir string (nSelf working directory) and key string(s).
// Outputs: Missing key list (envKeysMissing), bool presence (envKeySet), or
//          string value (envKeyValue). readEnvFileKey is the leaf reader.
// Constraints: Searches env files in precedence order (prod first, bare .env
//              last). Comments (#) and blank lines are skipped. Values are
//              stripped of surrounding single or double quotes.
// SPORT:   cli/internal/doctor — decomposed from hardening_check.go (T-E2-06).

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// envKeysMissing returns the subset of keys not found in any standard env file
// under projectDir.
func envKeysMissing(projectDir string, keys []string) []string {
	var missing []string
	for _, key := range keys {
		if !envKeySet(projectDir, key) {
			missing = append(missing, key)
		}
	}
	return missing
}

// envKeySet reports whether key is present (non-empty) in any standard env
// file under projectDir.
func envKeySet(projectDir string, key string) bool {
	return envKeyValue(projectDir, key) != ""
}

// envKeyValue returns the value of key from the first standard env file under
// projectDir where it appears. Searches in precedence order:
// .env.prod → .env.staging → .env.dev → .env.secrets → .env.local → .env.
// Returns "" when the key is not found in any file.
func envKeyValue(projectDir string, key string) string {
	candidates := []string{
		".env.prod", ".env.staging", ".env.dev",
		".env.secrets", ".env.local", ".env",
	}
	for _, fname := range candidates {
		val, found := readEnvFileKey(filepath.Join(projectDir, fname), key)
		if found {
			return val
		}
	}
	return ""
}

// readEnvFileKey reads a single KEY=VALUE line from the env file at path.
// Returns the value and true on success; "", false when the file does not
// exist, cannot be opened, or does not contain the key.
func readEnvFileKey(path, key string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	prefix := key + "="
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimPrefix(line, prefix)
			val = strings.Trim(val, `"'`)
			return val, true
		}
	}
	return "", false
}
