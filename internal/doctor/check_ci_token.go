package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ciTokenEnvKey is the GitHub token name used by the version-lockstep CI
// workflow to check out the admin/ and homebrew-nself/ repos cross-repo.
// It must have at minimum: contents:read on nself-org/admin and
// nself-org/homebrew-nself.
const ciTokenEnvKey = "NSELF_CI_TOKEN"

// ciTokenRotationDays is the recommended rotation window.
// 90 days matches the GitHub fine-grained token maximum for the free tier.
const ciTokenRotationDays = 90

// ciTokenRotationLogName is stored relative to the project's .nself dir so
// it travels with the project rather than the user's home directory.
const ciTokenRotationLogName = ".nself/ci-token-rotation.log"

// ciTokenSoftWarnDays is the expiry-proximity soft-warn threshold (14 days before expiry).
const ciTokenSoftWarnDays = 14

// ciTokenHardWarnDays is the expiry-proximity hard-fail threshold (1 day before expiry).
const ciTokenHardWarnDays = 1

// CheckCIToken is the CI-TOKEN-01 deep doctor check.
//
// It verifies:
//  1. NSELF_CI_TOKEN is set (in any env cascade file or the environment itself).
//  2. A rotation log exists documenting the last time the token was rotated.
//  3. Expiry-proximity check (primary): if an "expires <RFC3339>" entry exists in the
//     rotation log, warn at ≤14d remaining and fail at ≤1d remaining.
//  4. Rotation-age check (fallback): when no expiry date is present in the log
//     (legacy entries), fall back to the old 90d soft / 120d hard rotation-age logic.
//
// This check is --deep only. It does NOT attempt to call the GitHub API
// (no network) — it only inspects the local rotation log.
//
// To update the rotation log after a token rotation:
//
//	nself doctor --record-ci-token-rotation [--expires <RFC3339>]
//
// (CLI command delegates to internal/doctor.RecordCITokenRotation.)
func CheckCIToken(projectDir string) CheckResult {
	const name = "CI-TOKEN-01"
	section := "security"

	// 1. Token must be set somewhere in the env cascade.
	if !ciTokenIsSet(projectDir) {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "fail",
			Message: fmt.Sprintf("CI-TOKEN-01: %s not found in environment or any env cascade file. "+
				"The version-lockstep CI workflow (version-lockstep.yml) requires a GitHub token "+
				"with contents:read on nself-org/admin and nself-org/homebrew-nself. "+
				"Set NSELF_CI_TOKEN via GitHub Actions secret or .env.secrets.", ciTokenEnvKey),
			FixCmd: fmt.Sprintf("gh secret set NSELF_CI_TOKEN --repo nself-org/cli --body '<token>'"),
		}
	}

	// 2. Rotation log must exist.
	logPath := filepath.Join(projectDir, ciTokenRotationLogName)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "warn",
			Message: fmt.Sprintf("CI-TOKEN-01: rotation log not found at %s. "+
				"Run 'nself doctor --record-ci-token-rotation' after each token rotation "+
				"so this check can track the rotation age.", logPath),
			FixCmd: "nself doctor --record-ci-token-rotation",
		}
	}

	// 3. Read last rotation timestamp and optional expiry date from log.
	last, expiry, err := readRotationLogEntries(logPath)
	if err != nil {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "warn",
			Message: fmt.Sprintf("CI-TOKEN-01: cannot parse rotation log %s: %v", logPath, err),
		}
	}

	if last.IsZero() {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "warn",
			Message: "CI-TOKEN-01: rotation log exists but contains no timestamps — token has never been rotated",
			FixCmd:  "nself doctor --record-ci-token-rotation",
		}
	}

	// 3a. Expiry-proximity check (primary path — when expiry date is recorded).
	if !expiry.IsZero() {
		daysUntilExpiry := int(time.Until(expiry).Hours() / 24)
		if daysUntilExpiry <= ciTokenHardWarnDays {
			return CheckResult{
				Section: section,
				Name:    name,
				Status:  "fail",
				Message: fmt.Sprintf("CI-TOKEN-01: NSELF_CI_TOKEN expires in %d day(s) (at %s) — "+
					"hard deadline reached. Rotate the GitHub secret immediately and run "+
					"'nself doctor --record-ci-token-rotation --expires <new-expiry>'.", daysUntilExpiry, expiry.Format("2006-01-02")),
				FixCmd: "gh secret set NSELF_CI_TOKEN --repo nself-org/cli --body '<new-token>'",
			}
		}
		if daysUntilExpiry <= ciTokenSoftWarnDays {
			return CheckResult{
				Section: section,
				Name:    name,
				Status:  "warn",
				Message: fmt.Sprintf("CI-TOKEN-01: NSELF_CI_TOKEN expires in %d day(s) (at %s). "+
					"Rotate the GitHub secret and run 'nself doctor --record-ci-token-rotation --expires <new-expiry>'.",
					daysUntilExpiry, expiry.Format("2006-01-02")),
				FixCmd: "gh secret set NSELF_CI_TOKEN --repo nself-org/cli --body '<new-token>'",
			}
		}
		ageDays := int(time.Since(last).Hours() / 24)
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "pass",
			Message: fmt.Sprintf("CI-TOKEN-01: NSELF_CI_TOKEN present; expires %s (%d days away); last rotation %d days ago",
				expiry.Format("2006-01-02"), daysUntilExpiry, ageDays),
		}
	}

	// 3b. Rotation-age fallback (legacy entries without expiry date).
	ageDays := int(time.Since(last).Hours() / 24)
	hardDays := ciTokenRotationDays + 30 // 120d hard limit

	if ageDays >= hardDays {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "fail",
			Message: fmt.Sprintf("CI-TOKEN-01: NSELF_CI_TOKEN last rotated %d days ago — "+
				"exceeds hard limit (%dd). Rotate the GitHub secret immediately and run "+
				"'nself doctor --record-ci-token-rotation --expires <expiry-date>'.", ageDays, hardDays),
			FixCmd: "gh secret set NSELF_CI_TOKEN --repo nself-org/cli --body '<new-token>'",
		}
	}
	if ageDays >= ciTokenRotationDays {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "warn",
			Message: fmt.Sprintf("CI-TOKEN-01: NSELF_CI_TOKEN last rotated %d days ago (recommended window: %dd). "+
				"Rotate the GitHub secret and run 'nself doctor --record-ci-token-rotation --expires <expiry-date>'.",
				ageDays, ciTokenRotationDays),
			FixCmd: "gh secret set NSELF_CI_TOKEN --repo nself-org/cli --body '<new-token>'",
		}
	}

	return CheckResult{
		Section: section,
		Name:    name,
		Status:  "pass",
		Message: fmt.Sprintf("CI-TOKEN-01: NSELF_CI_TOKEN present; last rotation %d days ago (window=%dd); no expiry date recorded — add --expires on next rotation",
			ageDays, ciTokenRotationDays),
	}
}

// RecordCITokenRotation writes a new timestamp to the rotation log. Called by
// `nself doctor --record-ci-token-rotation`. Creates parent dirs as needed.
func RecordCITokenRotation(projectDir string) error {
	logPath := filepath.Join(projectDir, ciTokenRotationLogName)
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return fmt.Errorf("creating rotation log dir: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening rotation log: %w", err)
	}
	defer func() { _ = f.Close() }()

	line := fmt.Sprintf("rotated %s\n", time.Now().UTC().Format(time.RFC3339))
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("writing rotation log: %w", err)
	}
	return nil
}

// ciTokenIsSet checks the live environment and env cascade files for NSELF_CI_TOKEN.
func ciTokenIsSet(projectDir string) bool {
	if os.Getenv(ciTokenEnvKey) != "" {
		return true
	}
	// Check env cascade files. The token may live in .env.secrets (never .env.dev).
	for _, name := range []string{".env", ".env.secrets", ".env.local"} {
		if envFileHasKey(filepath.Join(projectDir, name), ciTokenEnvKey) {
			return true
		}
	}
	return false
}

// readRotationLogEntries reads the most recent "rotated <RFC3339>" and
// "expires <RFC3339>" entries from a log file.
// Returns (lastRotation, expiry, err). Either time may be zero if absent.
func readRotationLogEntries(logPath string) (time.Time, time.Time, error) {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("reading %s: %w", logPath, err)
	}

	var last, expiry time.Time
	lines := splitLines(string(data))
	for _, line := range lines {
		const rotatedPrefix = "rotated "
		const expiresPrefix = "expires "
		switch {
		case len(line) > len(rotatedPrefix) && line[:len(rotatedPrefix)] == rotatedPrefix:
			ts, err := time.Parse(time.RFC3339, line[len(rotatedPrefix):])
			if err == nil && ts.After(last) {
				last = ts
			}
		case len(line) > len(expiresPrefix) && line[:len(expiresPrefix)] == expiresPrefix:
			ts, err := time.Parse(time.RFC3339, line[len(expiresPrefix):])
			if err == nil && ts.After(expiry) {
				expiry = ts
			}
		}
	}
	return last, expiry, nil
}

// splitLines splits s on newlines, trimming whitespace, skipping blank lines.
func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := trimSpace(s[start:i])
			if line != "" {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	// Trim leading/trailing spaces and carriage returns.
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
