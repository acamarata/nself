package doctor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// jwtSecretEnvKey is the env var Hasura reads to validate signed JWTs.
// When Hasura is enabled (always true in core nSelf) this value must be
// present in either .env or .env.secrets before the stack will start.
const jwtSecretEnvKey = "HASURA_GRAPHQL_JWT_SECRET"

// CheckJWTSecretPresent reports whether HASURA_GRAPHQL_JWT_SECRET is defined
// in any of the project's env files. Semantics:
//
//   - fail when the key is absent from both .env and .env.secrets and
//     no other env file has it either; this is what the spec calls
//     "Hasura enabled but JWT secret absent from both sources".
//   - warn when the key is in .env.secrets but missing from .env. This is
//     the nSelf-preferred arrangement (secrets live in .env.secrets which is
//     gitignored and chmod 0600), so it is informational only.
//   - pass when the key is present in .env (with or without .env.secrets).
//
// projectDir is the working directory passed through from `nself doctor`.
func CheckJWTSecretPresent(projectDir string) CheckResult {
	name := "jwt-secret-present"

	inEnv := envFileHasKey(filepath.Join(projectDir, ".env"), jwtSecretEnvKey)
	inSecrets := envFileHasKey(filepath.Join(projectDir, ".env.secrets"), jwtSecretEnvKey)

	// Also accept the other standard env cascade files: a user who put
	// the secret in .env.local / .env.prod etc still has a configured stack.
	inOther := false
	for _, other := range []string{".env.local", ".env.dev", ".env.staging", ".env.prod"} {
		if envFileHasKey(filepath.Join(projectDir, other), jwtSecretEnvKey) {
			inOther = true
			break
		}
	}

	switch {
	case !inEnv && !inSecrets && !inOther:
		return CheckResult{
			Section: "security",
			Name:    name,
			Status:  "fail",
			Message: fmt.Sprintf("%s is missing from both .env and .env.secrets — Hasura will refuse to start. Run 'nself build' to auto-generate.", jwtSecretEnvKey),
			FixCmd:  "nself build",
		}
	case !inEnv && (inSecrets || inOther):
		return CheckResult{
			Section: "security",
			Name:    name,
			Status:  "warn",
			Message: fmt.Sprintf("%s not in .env but present in .env.secrets (which takes precedence)", jwtSecretEnvKey),
		}
	default:
		return CheckResult{
			Section: "security",
			Name:    name,
			Status:  "pass",
			Message: fmt.Sprintf("%s is present in .env", jwtSecretEnvKey),
		}
	}
}

// envFileHasKey returns true when the given env file exists on disk and
// contains a line defining KEY=... (comments and blank lines are skipped).
func envFileHasKey(path, key string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// Env values can be long (base64 JWT blobs, long JSON); bump the
	// max token size to handle multi-KB values without truncation errors.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	prefix := key + "="
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}
