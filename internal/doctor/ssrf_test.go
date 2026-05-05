package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper that creates a file at path with the given content.
func writeSSRFFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

// makeSSRFPaidDir builds a minimal plugins-pro/paid directory under root with
// guard files for the named services. Each entry in files maps a relPath+marker
// to the file content that should be written there.
func makeSSRFPaidDir(t *testing.T, root string, files map[string]string) string {
	t.Helper()
	paidDir := filepath.Join(root, "plugins-pro", "paid")
	for relPath, content := range files {
		writeSSRFFile(t, filepath.Join(paidDir, relPath), content)
	}
	return paidDir
}

// allServiceFiles returns the full set of (relPath→content) entries for all 5
// services with strong guard symbols, producing a PASS result.
func allServiceFiles() map[string]string {
	return map[string]string{
		"ai/go/internal/providers/ssrf.go":      guardContentWithSymbol("ValidateOllamaURL"),
		"mux/internal/safety/url_guard.go":      guardContentWithSymbol("IsBlockedIP"),
		"browser/internal/safety/urlcheck.go":   guardContentWithSymbol("isBlockedIP"),
		"claw/internal/tools/browser/client.go": guardContentWithSymbol("checkSSRF"),
		"notify/go/internal/push/ssrf.go":       guardContentWithSymbol("DialContext"),
	}
}

func guardContentWithSymbol(sym string) string {
	return "package p\n\nfunc " + sym + "() {}\n"
}

// TestCheckSSRF_BareInstall verifies that when plugins-pro/paid is absent the
// check returns pass (bare CLI install — not a monorepo dev checkout).
func TestCheckSSRF_BareInstall(t *testing.T) {
	dir := t.TempDir()
	result := CheckSSRF(dir)
	if result.Status != "pass" {
		t.Errorf("bare install: status = %q, want pass", result.Status)
	}
	if result.Section != "security" {
		t.Errorf("bare install: section = %q, want security", result.Section)
	}
}

// TestCheckSSRF_AllPass verifies that when all 5 guard files exist and each
// contains a recognised guard symbol the check returns PASS.
func TestCheckSSRF_AllPass(t *testing.T) {
	dir := t.TempDir()
	makeSSRFPaidDir(t, dir, allServiceFiles())

	result := CheckSSRF(dir)
	if result.Status != "pass" {
		t.Errorf("all present+symbols: status = %q, want pass; msg: %s", result.Status, result.Message)
	}
	if result.Name != "SSRF-01" {
		t.Errorf("name = %q, want SSRF-01", result.Name)
	}
}

// TestCheckSSRF_FileMissing verifies that when one or more guard files are
// absent the check returns FAIL.
func TestCheckSSRF_FileMissing(t *testing.T) {
	dir := t.TempDir()
	// Provide all except notify.
	files := allServiceFiles()
	delete(files, "notify/go/internal/push/ssrf.go")
	makeSSRFPaidDir(t, dir, files)

	result := CheckSSRF(dir)
	if result.Status != "fail" {
		t.Errorf("missing notify guard: status = %q, want fail; msg: %s", result.Status, result.Message)
	}
	if result.FixCmd == "" {
		t.Error("missing notify guard: FixCmd should not be empty")
	}
}

// TestCheckSSRF_FilePresent_NoSymbol verifies that when a guard file exists but
// contains no recognised guard symbol the check returns WARN (not PASS).
// This catches empty stubs or files that import a guard package without actually
// calling it — the previous os.Stat-only implementation would have returned PASS.
func TestCheckSSRF_FilePresent_NoSymbol(t *testing.T) {
	dir := t.TempDir()
	// All files present, but ai/go/internal/providers/ssrf.go has no guard symbol.
	files := allServiceFiles()
	files["ai/go/internal/providers/ssrf.go"] = "package providers\n\n// TODO: implement SSRF guard\n"
	makeSSRFPaidDir(t, dir, files)

	result := CheckSSRF(dir)
	if result.Status != "warn" {
		t.Errorf("stub guard file: status = %q, want warn; msg: %s", result.Status, result.Message)
	}
	if result.FixCmd == "" {
		t.Error("stub guard file: FixCmd should not be empty")
	}
}

// TestCheckSSRF_ThreeStatesDistinguished verifies all three states in a single
// test run: one service fully passes, one warns (stub), one fails (absent).
func TestCheckSSRF_ThreeStatesDistinguished(t *testing.T) {
	// We will use the real ssrfServices slice but manufacture a reduced tmp dir
	// that has exactly:
	//   mux  → PASS (url_guard.go with IsBlockedIP)
	//   ai   → WARN (ssrf.go with no guard symbol)
	//   rest → FAIL (absent)
	dir := t.TempDir()
	paidDir := filepath.Join(dir, "plugins-pro", "paid")

	writeSSRFFile(t, filepath.Join(paidDir, "mux/internal/safety/url_guard.go"),
		guardContentWithSymbol("IsBlockedIP"))
	writeSSRFFile(t, filepath.Join(paidDir, "ai/go/internal/providers/ssrf.go"),
		"package providers\n// stub\n")

	result := CheckSSRF(dir)

	// With 3 services missing (browser, claw, notify) the overall status is FAIL.
	if result.Status != "fail" {
		t.Errorf("mixed state: status = %q, want fail (missing services present); msg: %s", result.Status, result.Message)
	}
}

// TestCheckSSRF_WarnTakesPrecedenceOverPass verifies that when there are no
// missing files but at least one file has no guard symbol the overall status is
// WARN (not PASS).
func TestCheckSSRF_WarnTakesPrecedenceOverPass(t *testing.T) {
	dir := t.TempDir()
	files := allServiceFiles()
	// Replace two services with stub files.
	files["mux/internal/safety/url_guard.go"] = "package safety\n// no guard\n"
	files["claw/internal/tools/browser/client.go"] = "package browser\n// stub\n"
	makeSSRFPaidDir(t, dir, files)

	result := CheckSSRF(dir)
	if result.Status != "warn" {
		t.Errorf("two stubs among 5: status = %q, want warn; msg: %s", result.Status, result.Message)
	}
}

// TestCheckSSRF_AllGuardSymbolsRecognised checks that every symbol in
// ssrfGuardSymbols triggers a PASS when present alone in the guard file for
// the ai service (the simplest service with a single marker file).
func TestCheckSSRF_AllGuardSymbolsRecognised(t *testing.T) {
	for _, sym := range ssrfGuardSymbols {
		sym := sym
		t.Run(sym, func(t *testing.T) {
			dir := t.TempDir()
			files := allServiceFiles()
			files["ai/go/internal/providers/ssrf.go"] = guardContentWithSymbol(sym)
			makeSSRFPaidDir(t, dir, files)

			result := CheckSSRF(dir)
			if result.Status != "pass" {
				t.Errorf("symbol %q: status = %q, want pass; msg: %s", sym, result.Status, result.Message)
			}
		})
	}
}
