package doctor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/auth"
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

// CheckJWTRotation implements the JWT-ROT-01 deep doctor check. It verifies:
//
//  1. HASURA_GRAPHQL_JWT_SECRET is set (delegates to CheckJWTSecretPresent for
//     the precise fail/warn/pass logic on where the key lives).
//  2. A rotation log exists at the configured path.
//  3. The last recorded rotation is within the configured window (default 90d).
//
// This check is only included in --deep mode.
func CheckJWTRotation(projectDir string) CheckResult {
	const checkName = "JWT-ROT-01"
	section := "security"

	// (a) JWT secret env var must be set (any env file satisfies this).
	secretCheck := CheckJWTSecretPresent(projectDir)
	if secretCheck.Status == "fail" {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "fail",
			Message: "JWT-ROT-01: " + secretCheck.Message,
			FixCmd:  secretCheck.FixCmd,
		}
	}

	// (b) Rotation log must exist.
	logPath := auth.RotationLogPath()
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("JWT-ROT-01: rotation log not found at %s — run 'nself self-heal --jwt' to initialise", logPath),
			FixCmd:  "nself self-heal --jwt",
		}
	}

	// (c) Last rotation must be within the configured window.
	last, err := auth.LastRotationTime(logPath)
	if err != nil {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("JWT-ROT-01: cannot read rotation log (%v)", err),
		}
	}

	windowDays := auth.RotationWindowDays()
	hardDays := auth.RotationHardDays()

	if last.IsZero() {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("JWT-ROT-01: no rotation recorded in %s — JWT key has never been rotated", logPath),
			FixCmd:  "nself self-heal --jwt",
		}
	}

	age := time.Since(last)
	ageDays := int(age.Hours() / 24)
	if ageDays >= hardDays {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "fail",
			Message: fmt.Sprintf("JWT-ROT-01: last rotation was %d days ago — exceeds hard limit of %dd (NSELF_JWT_ROTATION_HARD_DAYS). Rotate immediately with 'nself self-heal --jwt'", ageDays, hardDays),
			FixCmd:  "nself self-heal --jwt",
		}
	}
	if ageDays >= windowDays {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("JWT-ROT-01: last rotation was %d days ago (window=%dd) — rotate with 'nself self-heal --jwt'", ageDays, windowDays),
			FixCmd:  "nself self-heal --jwt",
		}
	}

	return CheckResult{
		Section: section,
		Name:    checkName,
		Status:  "pass",
		Message: fmt.Sprintf("JWT-ROT-01: last rotation %d days ago (window=%dd)", ageDays, windowDays),
	}
}

// envFileHasKey returns true when the given env file exists on disk and
// contains a line defining KEY=... (comments and blank lines are skipped).
func envFileHasKey(path, key string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

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
