package doctor

import (
	"path/filepath"
	"strings"
	"testing"
)

const validA9PrivacyPage = `
# Privacy Policy

## Health and other sensitive information

Under the GDPR (Article 9), we treat health data with special care...
`

const partialA9PrivacyPage = `
# Privacy Policy

We do not process sensitive data, ever.
`

// stagePrivacyPage writes web/nfamily/src/app/privacy/page.tsx beneath
// projectDir. Empty contents means do not write the file.
func stagePrivacyPage(t *testing.T, projectDir, contents string) {
	t.Helper()
	if contents == "" {
		return
	}
	path := filepath.Join(projectDir, "web", "nfamily", "src", "app", "privacy", "page.tsx")
	writeFile(t, path, contents)
}

// stageFamilyConsentsMigration writes a migration file referencing
// np_family_consents under the staged plugins-pro tree.
func stageFamilyConsentsMigration(t *testing.T, projectDir string) {
	t.Helper()
	paid := stagePluginsPro(t, projectDir, "")
	writeFile(t,
		filepath.Join(paid, "family", "migrations", "002_family_consents.sql"),
		"-- migration\nCREATE TABLE np_family_consents();\n",
	)
}

// TestCheckLegalGDPRA9_Skip_NoPlugin: family plugin not loaded → skip.
func TestCheckLegalGDPRA9_Skip_NoPlugin(t *testing.T) {
	t.Setenv("NSELF_FAMILY_LOADED", "")
	t.Setenv("PLUGIN_FAMILY_INTERNAL_URL", "")

	r := CheckLegalGDPRA9(t.TempDir())
	if r.Status != "skip" {
		t.Fatalf("want skip, got %q (msg=%q)", r.Status, r.Message)
	}
}

// TestCheckLegalGDPRA9_Pass: migration + complete privacy disclosure + DPO email set → pass.
func TestCheckLegalGDPRA9_Pass(t *testing.T) {
	dir := t.TempDir()
	stageFamilyConsentsMigration(t, dir)
	stagePrivacyPage(t, dir, validA9PrivacyPage)

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("NSELF_DPO_EMAIL", "dpo@example.com")

	r := CheckLegalGDPRA9(dir)
	if r.Status != "pass" {
		t.Fatalf("want pass, got %q (msg=%q)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "LEGAL-GDPR-A9-01") {
		t.Errorf("expected check ID in message, got %q", r.Message)
	}
}

// TestCheckLegalGDPRA9_Warn_NoDPO: migration + disclosure but no DPO email → warn.
func TestCheckLegalGDPRA9_Warn_NoDPO(t *testing.T) {
	dir := t.TempDir()
	stageFamilyConsentsMigration(t, dir)
	stagePrivacyPage(t, dir, validA9PrivacyPage)

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("NSELF_DPO_EMAIL", "")

	r := CheckLegalGDPRA9(dir)
	if r.Status != "warn" {
		t.Fatalf("want warn, got %q (msg=%q)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "NSELF_DPO_EMAIL") {
		t.Errorf("warn should call out missing DPO email, got %q", r.Message)
	}
}

// TestCheckLegalGDPRA9_Warn_PartialDisclosure: migration present but
// privacy page is missing the Article 9 disclosure → warn.
func TestCheckLegalGDPRA9_Warn_PartialDisclosure(t *testing.T) {
	dir := t.TempDir()
	stageFamilyConsentsMigration(t, dir)
	stagePrivacyPage(t, dir, partialA9PrivacyPage)

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("NSELF_DPO_EMAIL", "dpo@example.com")

	r := CheckLegalGDPRA9(dir)
	if r.Status != "warn" {
		t.Fatalf("want warn, got %q (msg=%q)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "Article 9") && !strings.Contains(r.Message, "Health") && !strings.Contains(r.Message, "GDPR") {
		t.Errorf("warn message should name the missing tokens, got %q", r.Message)
	}
}

// TestCheckLegalGDPRA9_Warn_PrivacyPageMissing: migration present, privacy page absent → warn.
func TestCheckLegalGDPRA9_Warn_PrivacyPageMissing(t *testing.T) {
	dir := t.TempDir()
	stageFamilyConsentsMigration(t, dir)
	// no privacy page

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("NSELF_DPO_EMAIL", "dpo@example.com")

	r := CheckLegalGDPRA9(dir)
	if r.Status != "warn" {
		t.Fatalf("want warn, got %q (msg=%q)", r.Status, r.Message)
	}
	if !strings.Contains(r.Message, "privacy page not found") {
		t.Errorf("warn message should mention missing privacy page, got %q", r.Message)
	}
}

// TestCheckLegalGDPRA9_Fail_NoMigration: plugin loaded, plugins-pro
// directory present but np_family_consents migration absent → fail.
func TestCheckLegalGDPRA9_Fail_NoMigration(t *testing.T) {
	dir := t.TempDir()
	stagePluginsPro(t, dir, "") // family/migrations dir exists, no consents migration
	stagePrivacyPage(t, dir, validA9PrivacyPage)

	t.Setenv("NSELF_FAMILY_LOADED", "1")
	t.Setenv("NSELF_DPO_EMAIL", "dpo@example.com")

	r := CheckLegalGDPRA9(dir)
	if r.Status != "fail" {
		t.Fatalf("want fail, got %q (msg=%q)", r.Status, r.Message)
	}
	if r.FixCmd == "" {
		t.Errorf("fail result should include FixCmd")
	}
}

// TestCheckLegalGDPRA9_BareInstall: plugins-pro absent → pass (defers to runtime checks).
func TestCheckLegalGDPRA9_BareInstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_FAMILY_LOADED", "1")

	r := CheckLegalGDPRA9(dir)
	if r.Status != "pass" {
		t.Fatalf("want pass (bare install), got %q (msg=%q)", r.Status, r.Message)
	}
}
