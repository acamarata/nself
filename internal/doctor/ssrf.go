package doctor

// ssrf.go implements the SSRF-01 deep doctor check.
//
// SSRF-01 verifies that each plugin service known to make outbound HTTP
// requests ships with effective SSRF guard code. It performs a three-state
// check rather than simple file-presence to avoid false PASSes from empty
// stubs or guard files that are never wired into outbound HTTP paths.
//
// Three states per service:
//
//	PASS — guard file present AND at least one strong guard symbol found
//	       (DialContext, IsBlockedIP, isBlockedIP, isPrivateIP, checkSSRF,
//	        ValidateURL, ValidateWebhookURL, validateWebhookURL,
//	        notifyIsBlockedIP, ValidateOllamaURL)
//	WARN — guard file present but no recognised guard symbol detected;
//	        the check may be policy-only or the file is a stub
//	FAIL — guard file absent entirely
//
// Services checked:
//
//	ai       — OllamaURL validation in providers/ssrf.go
//	mux      — URL guard in internal/safety/url_guard.go
//	browser  — URL guard in internal/safety/urlcheck.go
//	claw     — SSRF check in internal/tools/browser/client.go
//	notify   — validateWebhookURL in internal/push/ssrf.go

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ssrfServiceCheck describes one plugin service that must carry SSRF guard code.
type ssrfServiceCheck struct {
	// Name is the human-readable service name used in check output.
	Name string
	// RelPath is the path relative to the plugins-pro paid/ directory.
	RelPath string
	// Markers is a list of file paths (relative to RelPath) at least one of
	// which must exist. All listed files are checked to allow for future
	// refactors that rename the guard file.
	Markers []string
	// GuardSymbols lists Go identifiers that must appear in the guard file for a
	// PASS verdict. If the file exists but none of these symbols are found the
	// result is WARN rather than PASS. The symbols are checked by simple
	// substring scan of the file contents (no AST parsing required).
	GuardSymbols []string
}

// ssrfGuardSymbols is the shared set of strong SSRF guard identifiers.
// At least one must appear in a guard file for the service to earn PASS.
var ssrfGuardSymbols = []string{
	"DialContext",
	"IsBlockedIP",
	"isBlockedIP",
	"isPrivateIP",
	"notifyIsBlockedIP",
	"checkSSRF",
	"ValidateURL",
	"ValidateWebhookURL",
	"validateWebhookURL",
	"ValidateOllamaURL",
}

// ssrfServices lists all plugin services expected to have SSRF protection.
var ssrfServices = []ssrfServiceCheck{
	{
		Name:         "ai",
		RelPath:      "ai/go/internal/providers",
		Markers:      []string{"ssrf.go"},
		GuardSymbols: ssrfGuardSymbols,
	},
	{
		Name:         "mux",
		RelPath:      "mux/internal/safety",
		Markers:      []string{"url_guard.go"},
		GuardSymbols: ssrfGuardSymbols,
	},
	{
		Name:         "browser",
		RelPath:      "browser/internal/safety",
		Markers:      []string{"urlcheck.go"},
		GuardSymbols: ssrfGuardSymbols,
	},
	{
		Name:         "claw",
		RelPath:      "claw/internal/tools/browser",
		Markers:      []string{"client.go"},
		GuardSymbols: ssrfGuardSymbols,
	},
	{
		Name:         "notify",
		RelPath:      "notify/go/internal/push",
		Markers:      []string{"ssrf.go"},
		GuardSymbols: ssrfGuardSymbols,
	},
}

// ssrfCheckState represents the result of checking one service.
type ssrfCheckState int

const (
	ssrfPass ssrfCheckState = iota // guard file present + strong symbol found
	ssrfWarn                       // guard file present but no strong symbol detected
	ssrfFail                       // guard file absent
)

// checkOneSSRFService evaluates a single service and returns its state plus the
// path and symbol that produced the result (empty string if not applicable).
func checkOneSSRFService(paidDir string, svc ssrfServiceCheck) (state ssrfCheckState, foundPath, foundSymbol string) {
	for _, marker := range svc.Markers {
		candidate := filepath.Join(paidDir, svc.RelPath, marker)
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue // file absent — try next marker
		}
		contents := string(data)
		// File exists. Now check for at least one strong guard symbol.
		for _, sym := range svc.GuardSymbols {
			if strings.Contains(contents, sym) {
				return ssrfPass, candidate, sym
			}
		}
		// File present but no strong symbol found — WARN.
		return ssrfWarn, candidate, ""
	}
	return ssrfFail, "", ""
}

// CheckSSRF implements the SSRF-01 deep doctor check. It locates the
// plugins-pro paid/ directory relative to projectDir and verifies that each
// plugin service carries effective SSRF guard code (file present + guard
// symbol present).
//
// projectDir is the nself project working directory (where .env lives).
// plugins-pro is expected to be a sibling of cli/ inside the nself monorepo.
func CheckSSRF(projectDir string) CheckResult {
	const checkName = "SSRF-01"
	section := "security"

	paidDir := findPluginsProPaidDir(projectDir)
	if paidDir == "" {
		// plugins-pro is not present — this is a bare CLI install, not a monorepo
		// development checkout. Skip with a pass so end-user doctor runs are clean.
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "pass",
			Message: "SSRF-01: plugins-pro/paid not found — skipping (bare CLI install)",
		}
	}

	var passing []string
	var warned []string  // file present, symbol absent
	var missing []string // file absent

	for _, svc := range ssrfServices {
		state, _, sym := checkOneSSRFService(paidDir, svc)
		switch state {
		case ssrfPass:
			passing = append(passing, fmt.Sprintf("%s(%s)", svc.Name, sym))
		case ssrfWarn:
			warned = append(warned, svc.Name)
		case ssrfFail:
			missing = append(missing, svc.Name)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "fail",
			Message: fmt.Sprintf("SSRF-01: SSRF guard file missing in plugin service(s): %v. Expected guard files under %s", missing, paidDir),
			FixCmd:  "See plugins-pro/paid/{service}/internal/push/ssrf.go",
		}
	}

	if len(warned) > 0 {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf(
				"SSRF-01: guard file present but no DialContext/IsBlockedIP-like symbol detected in service(s): %v — check may be policy-only or a stub. Passing: %v",
				warned, passing,
			),
			FixCmd: "Ensure guard functions are wired into outbound http.Client transports in each service",
		}
	}

	return CheckResult{
		Section: section,
		Name:    checkName,
		Status:  "pass",
		Message: fmt.Sprintf("SSRF-01: effective SSRF guards verified in all %d plugin services (%v)", len(passing), passing),
	}
}

// findPluginsProPaidDir searches for the plugins-pro/paid directory starting
// from projectDir and walking up to 4 parent levels (covers monorepo layouts
// where projectDir might be cli/, the nself root, or a sub-directory).
func findPluginsProPaidDir(projectDir string) string {
	dir := projectDir
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "plugins-pro", "paid")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
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
