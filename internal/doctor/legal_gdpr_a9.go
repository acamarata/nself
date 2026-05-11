// legal_gdpr_a9.go implements the LEGAL-GDPR-A9-01 deep doctor check.
//
// LEGAL-GDPR-A9-01 verifies that the GDPR Article 9 special-category
// consent flow shipped in the family plugin (P100 Wave 2D) is wired
// correctly before the nFamily product can be marked production-ready.
//
// The check is layered:
//
//	pass — np_family_consents migration on disk AND the privacy page
//	       contains the GDPR Article 9 disclosure section
//	warn — migration on disk but the privacy page is missing the Article 9
//	       disclosure, or NSELF_DPO_EMAIL is unset (recommended for any
//	       deployment processing special-category data)
//	fail — family plugin loaded AND np_family_consents migration absent
//	skip — family plugin not loaded; the check is not applicable
//
// The runtime accessibility of np_family_consents is covered by the existing
// PERM-RLS-01 check (np_* RLS sweep). LEGAL-GDPR-A9-01 stays focused on
// Article 9 preconditions: schema present, privacy disclosure present,
// DPO contact recorded.
//
// S12.T07 follow-up.
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// gdprA9PrivacyMarkers is the set of substrings/regexes that the privacy
// page MUST satisfy for the Article 9 disclosure to be considered present.
//
//  1. A heading or paragraph mentioning "Health and other sensitive" data
//     (the canonical section header shipped in Wave 2D).
//  2. An explicit citation of "Article 9" with a "GDPR" reference somewhere
//     in the same file (Art 9 is the GDPR clause governing special
//     category processing; both tokens must co-occur).
var gdprA9HealthHeading = regexp.MustCompile(`(?i)health and other sensitive`)
var gdprA9ArticleRef = regexp.MustCompile(`(?i)article\s*9`)
var gdprA9GDPRRef = regexp.MustCompile(`(?i)\bgdpr\b`)

// CheckLegalGDPRA9 implements LEGAL-GDPR-A9-01.
func CheckLegalGDPRA9(projectDir string) CheckResult {
	const (
		checkID   = "LEGAL-GDPR-A9-01"
		section   = "legal"
		checkName = checkID + ": GDPR Article 9 special-category wiring"
	)

	if !isFamilyPluginLoaded() {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "skip",
			Message: "family plugin not loaded — GDPR Article 9 check not applicable",
		}
	}

	paidDir := findPluginsProPaidDir(projectDir)
	if paidDir == "" {
		// Bare CLI install — migration audit not possible. The runtime table
		// check belongs to PERM-RLS-01. Treat as pass.
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "pass",
			Message: checkID + ": plugins-pro/paid not found — skipping migration audit (bare CLI install)",
		}
	}

	migrationPath, migrationOK := findFamilyConsentsMigration(paidDir)
	if !migrationOK {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "fail",
			Message: fmt.Sprintf("%s: np_family_consents migration not found under %s/family/migrations — Article 9 consent records cannot be stored", checkID, paidDir),
			FixCmd:  "nself plugin reinstall family",
		}
	}

	// Migration present. Check the privacy disclosure page.
	privacyPath := findFamilyPrivacyPage(projectDir)
	if privacyPath == "" {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: checkID + ": migration present at " + migrationPath + " but web/nfamily privacy page not found — Article 9 disclosure cannot be verified",
			FixCmd:  "Ensure web/nfamily/src/app/privacy/page.tsx exists with the Health and other sensitive section",
		}
	}

	data, err := os.ReadFile(privacyPath)
	if err != nil {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("%s: cannot read privacy page %s: %v", checkID, privacyPath, err),
			FixCmd:  "Verify file permissions on web/nfamily/src/app/privacy/page.tsx",
		}
	}
	contents := string(data)

	if !gdprA9HealthHeading.MatchString(contents) || !gdprA9ArticleRef.MatchString(contents) || !gdprA9GDPRRef.MatchString(contents) {
		var missing []string
		if !gdprA9HealthHeading.MatchString(contents) {
			missing = append(missing, "'Health and other sensitive' section")
		}
		if !gdprA9ArticleRef.MatchString(contents) {
			missing = append(missing, "'Article 9' citation")
		}
		if !gdprA9GDPRRef.MatchString(contents) {
			missing = append(missing, "'GDPR' reference")
		}
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("%s: privacy page %s missing %s", checkID, privacyPath, strings.Join(missing, ", ")),
			FixCmd:  "Add the GDPR Article 9 disclosure section to web/nfamily/src/app/privacy/page.tsx",
		}
	}

	// Disclosure present. Warn if DPO contact is not configured.
	if strings.TrimSpace(os.Getenv("NSELF_DPO_EMAIL")) == "" {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: checkID + ": migration + privacy disclosure present, but NSELF_DPO_EMAIL is unset — Article 9 deployments should expose a DPO contact",
			FixCmd:  "nself env set NSELF_DPO_EMAIL=dpo@example.com",
		}
	}

	return CheckResult{
		Section: section,
		Name:    checkName,
		Status:  "pass",
		Message: checkID + ": migration present, privacy disclosure present, DPO contact set",
	}
}

// findFamilyConsentsMigration looks for the np_family_consents migration in
// the family plugin's migrations directory. The exact filename is not pinned
// (different waves may number it differently), so we scan for a .sql file
// whose contents reference np_family_consents.
func findFamilyConsentsMigration(paidDir string) (string, bool) {
	migrationsDir := filepath.Join(paidDir, "family", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return "", false
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		// Skip down-migrations.
		if strings.HasSuffix(name, ".down.sql") {
			continue
		}
		full := filepath.Join(migrationsDir, name)
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "np_family_consents") {
			return full, true
		}
	}
	return "", false
}

// findFamilyPrivacyPage walks up from projectDir looking for
// web/nfamily/src/app/privacy/page.tsx.
func findFamilyPrivacyPage(projectDir string) string {
	dir := projectDir
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "web", "nfamily", "src", "app", "privacy", "page.tsx")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
