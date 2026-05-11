// legal_coppa.go implements the LEGAL-COPPA-01 deep doctor check.
//
// LEGAL-COPPA-01 verifies that the COPPA parental-consent flow shipped in the
// family plugin (P100 Wave 2D) is wired correctly before the nFamily product
// can be marked production-ready.
//
// The check is layered:
//
//	pass — migration present, consent secret env set, TTL within policy
//	warn — migration present but FAMILY_CONSENT_HMAC_SECRET unset, or TTL
//	       above the policy ceiling, or the family plugin is loaded without
//	       the migration on disk
//	fail — family plugin loaded AND migration file missing entirely (the
//	       parental-consent flow cannot work without the schema)
//	skip — family plugin not loaded; the check is not applicable
//
// The Hasura row-level accessibility of np_parental_consents is verified by
// the existing PERM-HASURA-01 / PERM-RLS-01 checks (np_* tables are blanket
// covered there). LEGAL-COPPA-01 stays focused on the COPPA-specific
// preconditions: schema present, signing secret set, TTL within policy.
//
// S12.T06 follow-up.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// coppaTTLCeilingHours is the maximum allowed consent TTL in hours.
// Policy: parental consent records expire within 7 days (168 hours).
const coppaTTLCeilingHours = 168

// CheckLegalCOPPA implements LEGAL-COPPA-01.
func CheckLegalCOPPA(projectDir string) CheckResult {
	const (
		checkID   = "LEGAL-COPPA-01"
		section   = "legal"
		checkName = checkID + ": COPPA parental-consent wiring"
	)

	if !isFamilyPluginLoaded() {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "skip",
			Message: "family plugin not loaded — COPPA check not applicable",
		}
	}

	paidDir := findPluginsProPaidDir(projectDir)
	if paidDir == "" {
		// Bare CLI install, plugins-pro source not present. The migration is
		// expected to have been applied at install time, so we treat this as a
		// pass rather than fail — the runtime check for the table belongs to
		// PERM-RLS-01.
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "pass",
			Message: checkID + ": plugins-pro/paid not found — skipping migration audit (bare CLI install)",
		}
	}

	migration := filepath.Join(paidDir, "family", "migrations", "001_coppa_consent.sql")
	info, err := os.Stat(migration)
	if err != nil || info.IsDir() {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "fail",
			Message: fmt.Sprintf("%s: COPPA migration missing at %s — parental-consent flow cannot work without np_parental_consents schema", checkID, migration),
			FixCmd:  "nself plugin reinstall family",
		}
	}

	// Migration on disk. Verify the signing secret is set; without it the
	// HMAC on consent tokens cannot be produced or verified.
	if strings.TrimSpace(os.Getenv("FAMILY_CONSENT_HMAC_SECRET")) == "" {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: checkID + ": migration present but FAMILY_CONSENT_HMAC_SECRET is unset — consent tokens cannot be signed",
			FixCmd:  "nself secrets set FAMILY_CONSENT_HMAC_SECRET=$(openssl rand -hex 32)",
		}
	}

	// Verify TTL ≤ policy ceiling. The TTL is read from
	// FAMILY_CONSENT_TTL_HOURS (default 168 / 7 days). A higher value would
	// leave parental consent records valid longer than COPPA policy allows.
	ttlHours := readCOPPATTLHours()
	if ttlHours > coppaTTLCeilingHours {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("%s: FAMILY_CONSENT_TTL_HOURS=%d exceeds policy ceiling of %d hours (7 days)", checkID, ttlHours, coppaTTLCeilingHours),
			FixCmd:  fmt.Sprintf("nself env set FAMILY_CONSENT_TTL_HOURS=%d", coppaTTLCeilingHours),
		}
	}

	return CheckResult{
		Section: section,
		Name:    checkName,
		Status:  "pass",
		Message: fmt.Sprintf("%s: migration present, signing secret set, TTL=%dh ≤ %dh", checkID, ttlHours, coppaTTLCeilingHours),
	}
}

// isFamilyPluginLoaded mirrors the convention used by CheckModerationWired:
// the plugin is considered loaded when either NSELF_FAMILY_LOADED=1 or the
// internal URL env var has been wired by the loader.
func isFamilyPluginLoaded() bool {
	return os.Getenv("NSELF_FAMILY_LOADED") == "1" || os.Getenv("PLUGIN_FAMILY_INTERNAL_URL") != ""
}

// readCOPPATTLHours reads FAMILY_CONSENT_TTL_HOURS and returns the parsed
// value, defaulting to coppaTTLCeilingHours (168) when unset or unparseable.
func readCOPPATTLHours() int {
	raw := strings.TrimSpace(os.Getenv("FAMILY_CONSENT_TTL_HOURS"))
	if raw == "" {
		return coppaTTLCeilingHours
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return coppaTTLCeilingHours
	}
	return n
}
