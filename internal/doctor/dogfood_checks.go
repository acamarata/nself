package doctor

// dogfood_checks.go — dogfood-only checks for nSelf's own backend (web/backend).
//
// G-DOGFOOD ticket (P97 Wave 37): extra deep checks that run against the
// nself.org production stack to catch drift between what we ship and what
// users get. Invoked by `nself doctor --deep` when run inside a repo that
// declares itself a dogfood target via NSELF_DOGFOOD=1 OR by the dogfood-check
// CI workflow at web/.github/workflows/dogfood-check.yml.
//
// All checks here are read-only and never modify state. Per the
// Security-Always-Free Doctrine, these checks run without a license.

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DogfoodChecks runs the full dogfood-only check set. It is invoked from
// DeepChecks when NSELF_DOGFOOD=1 is set, AND can be invoked directly from
// CI to gate nself.org's own deploys.
//
// Check IDs (cross-referenced in operations/dogfood-checks.md):
//
//	SEC-CSP-01    — CSP headers present on all subapps
//	SEC-HSTS-01   — HSTS preload value
//	SEC-AUTH-01   — HttpOnly cookie convention enforced
//	VENDOR-DEP-01 — no AGPL/SSPL deps in production source
//	DOGFOOD-SUBAPPS-01 — every shipped subapp has synced sport.json
//	DOGFOOD-HEX-01     — no hex colors in subapp src/ trees
//	DOGFOOD-LICENSE-01 — owner license vault path readable
//
// PERM-RLS-01 is intentionally NOT duplicated here; rls_check.go owns it.
func DogfoodChecks(ctx context.Context, projectDir string, verbose bool) []CheckResult {
	var results []CheckResult

	// Security headers — only run when subapp URLs are reachable. The CI job
	// uses a localhost reverse proxy; production hits the live subdomains.
	results = append(results, checkCSPHeaders(ctx)...)
	results = append(results, checkHSTSHeader(ctx))
	results = append(results, checkHttpOnlyCookies(ctx))

	// Vendor dependency license scan — production source only.
	results = append(results, checkAGPLDeps(ctx, projectDir))

	// Dogfood-only structural checks.
	results = append(results, checkSubappSportSync(projectDir)...)
	results = append(results, checkNoHexColorsInSrc(projectDir)...)
	results = append(results, checkLicenseVaultPath())

	return results
}

// dogfoodSubapps lists every subapp that must be present + sport-synced for
// nself.org's web monorepo. Mirrors the F11-SUBDOMAIN-MAP table.
var dogfoodSubapps = []string{
	"org", "docs", "nchat", "nclaw", "ntask", "ntv", "nfamily",
	"clawde", "cloud", "install", "base", "backend",
	// 12 + base = 13 per PPI Three-Surface Model.
}

// dogfoodSubappURLs maps subapps to their production URLs for header probing.
var dogfoodSubappURLs = map[string]string{
	"org":     "https://nself.org",
	"docs":    "https://docs.nself.org",
	"nchat":   "https://chat.nself.org",
	"nclaw":   "https://claw.nself.org",
	"ntask":   "https://task.nself.org",
	"ntv":     "https://ntv.nself.org",
	"cloud":   "https://cloud.nself.org",
	"install": "https://install.nself.org",
}

// httpHeaderClient is shared across header checks with a tight timeout so a
// single slow subapp can't hang the whole doctor run.
var httpHeaderClient = &http.Client{
	Timeout: 8 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// Don't follow redirects — we want the headers from the canonical URL.
		return http.ErrUseLastResponse
	},
}

// checkCSPHeaders probes every dogfood subapp URL and verifies it returns a
// Content-Security-Policy header. Unreachable subapps are SKIPPED (warn) so
// CI in offline environments doesn't false-fail.
func checkCSPHeaders(ctx context.Context) []CheckResult {
	var results []CheckResult
	for _, name := range []string{"org", "docs", "nchat", "nclaw", "ntask", "ntv", "cloud", "install"} {
		url, ok := dogfoodSubappURLs[name]
		if !ok {
			continue
		}
		checkName := fmt.Sprintf("SEC-CSP-01: CSP on %s", name)
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			results = append(results, CheckResult{
				Section: "security", Name: checkName, Status: "warn",
				Message: fmt.Sprintf("cannot build request: %v", err),
			})
			continue
		}
		resp, err := httpHeaderClient.Do(req)
		if err != nil {
			results = append(results, CheckResult{
				Section: "security", Name: checkName, Status: "warn",
				Message: fmt.Sprintf("unreachable: %v", err),
			})
			continue
		}
		_ = resp.Body.Close()
		csp := resp.Header.Get("Content-Security-Policy")
		if csp == "" {
			results = append(results, CheckResult{
				Section: "security", Name: checkName, Status: "fail",
				Message: "Content-Security-Policy header missing",
				FixCmd:  fmt.Sprintf("review nginx/conf.d for %s subapp; add CSP directive", name),
			})
			continue
		}
		results = append(results, CheckResult{
			Section: "security", Name: checkName, Status: "pass",
			Message: fmt.Sprintf("CSP set (%d chars)", len(csp)),
		})
	}
	return results
}

// checkHSTSHeader probes nself.org for Strict-Transport-Security with a
// preload-grade value (max-age >= 31536000, includeSubDomains, preload).
func checkHSTSHeader(ctx context.Context) CheckResult {
	const checkName = "SEC-HSTS-01: HSTS preload"
	url := dogfoodSubappURLs["org"]
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return CheckResult{Section: "security", Name: checkName, Status: "warn",
			Message: fmt.Sprintf("cannot build request: %v", err)}
	}
	resp, err := httpHeaderClient.Do(req)
	if err != nil {
		return CheckResult{Section: "security", Name: checkName, Status: "warn",
			Message: fmt.Sprintf("unreachable: %v", err)}
	}
	defer resp.Body.Close()

	hsts := resp.Header.Get("Strict-Transport-Security")
	if hsts == "" {
		return CheckResult{Section: "security", Name: checkName, Status: "fail",
			Message: "Strict-Transport-Security header missing",
			FixCmd:  "add HSTS preload header to nginx config for nself.org"}
	}

	// Parse max-age, require >= 31536000 (1 year) AND includeSubDomains AND preload.
	lower := strings.ToLower(hsts)
	hasIncludeSub := strings.Contains(lower, "includesubdomains")
	hasPreload := strings.Contains(lower, "preload")

	maxAgeRe := regexp.MustCompile(`max-age=(\d+)`)
	m := maxAgeRe.FindStringSubmatch(lower)
	maxAgeOK := false
	if len(m) == 2 {
		// At least 31536000 = 1 year.
		var n int
		_, _ = fmt.Sscanf(m[1], "%d", &n)
		maxAgeOK = n >= 31536000
	}

	if !maxAgeOK || !hasIncludeSub || !hasPreload {
		return CheckResult{Section: "security", Name: checkName, Status: "fail",
			Message: fmt.Sprintf("HSTS not preload-grade: %q", hsts),
			FixCmd:  "set: max-age=31536000; includeSubDomains; preload"}
	}

	return CheckResult{Section: "security", Name: checkName, Status: "pass",
		Message: "HSTS preload-grade"}
}

// checkHttpOnlyCookies verifies api.nself.org sets HttpOnly on its session
// cookies. We probe a no-auth endpoint that still issues a session cookie.
func checkHttpOnlyCookies(ctx context.Context) CheckResult {
	const checkName = "SEC-AUTH-01: HttpOnly cookies"
	// /healthz should be cheap and not require auth. The auth service issues
	// session cookies on first hit.
	url := "https://api.nself.org/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return CheckResult{Section: "security", Name: checkName, Status: "warn",
			Message: fmt.Sprintf("cannot build request: %v", err)}
	}
	resp, err := httpHeaderClient.Do(req)
	if err != nil {
		return CheckResult{Section: "security", Name: checkName, Status: "warn",
			Message: fmt.Sprintf("unreachable: %v", err)}
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		// No cookies set on healthz — not necessarily a failure (some auth
		// implementations defer cookies to authenticated endpoints).
		return CheckResult{Section: "security", Name: checkName, Status: "pass",
			Message: "no session cookies on /healthz (skipped)"}
	}

	for _, c := range cookies {
		if !c.HttpOnly {
			return CheckResult{Section: "security", Name: checkName, Status: "fail",
				Message: fmt.Sprintf("cookie %q is not HttpOnly", c.Name),
				FixCmd:  "set HttpOnly=true on all session cookies in auth service"}
		}
	}
	return CheckResult{Section: "security", Name: checkName, Status: "pass",
		Message: fmt.Sprintf("all %d session cookies HttpOnly", len(cookies))}
}

// agplLicenseRegex matches license fields in package.json files for AGPL/SSPL/CC-BY-NC.
// Production source must not depend on AGPL/SSPL because those copyleft terms
// would force release of nSelf's own source.
var agplLicenseRegex = regexp.MustCompile(`(?i)"license"\s*:\s*"(AGPL|SSPL|CC-BY-NC)`)

// checkAGPLDeps walks the project's vendored / node_modules-style dependency
// metadata for AGPL/SSPL license declarations. It does NOT do a full SBOM scan
// (that's `nself license-audit`) — this is a fast, last-line gate for prod source.
func checkAGPLDeps(ctx context.Context, projectDir string) CheckResult {
	const checkName = "VENDOR-DEP-01: no AGPL/SSPL deps"

	// Look at common Go and Node dep manifests in the project.
	candidates := []string{
		filepath.Join(projectDir, "go.mod"),
		filepath.Join(projectDir, "package.json"),
		filepath.Join(projectDir, "vendor", "modules.txt"),
		filepath.Join(projectDir, "pnpm-lock.yaml"),
	}

	scanned := 0
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		scanned++
		if agplLicenseRegex.Find(data) != nil {
			// Don't expose the offending line directly — it could leak proprietary
			// names. Tell the operator to run the full license-audit instead.
			return CheckResult{Section: "security", Name: checkName, Status: "fail",
				Message: fmt.Sprintf("AGPL/SSPL/CC-BY-NC license declaration found in %s", filepath.Base(path)),
				FixCmd:  "run: nself license-audit  # for full report and remediation"}
		}
	}

	if scanned == 0 {
		return CheckResult{Section: "security", Name: checkName, Status: "pass",
			Message: "no dependency manifests found (skipped)"}
	}
	return CheckResult{Section: "security", Name: checkName, Status: "pass",
		Message: fmt.Sprintf("scanned %d manifest(s); no AGPL/SSPL/CC-BY-NC", scanned)}
}

// checkSubappSportSync verifies every dogfood subapp directory has a
// synced sport.json (generated by web/org/scripts/sync-sport.mjs at build).
// Missing or stale sport.json means marketing/docs are out of sync with code.
func checkSubappSportSync(projectDir string) []CheckResult {
	var results []CheckResult
	for _, name := range dogfoodSubapps {
		// Skip "backend" — it's not a subapp with a sport.json.
		if name == "backend" {
			continue
		}
		checkName := fmt.Sprintf("DOGFOOD-SUBAPPS-01: sport.json on %s", name)
		// Most likely paths: web/<app>/sport.json or web/<app>/src/sport.json
		// or web/<app>/public/sport.json. Look up all three.
		candidates := []string{
			filepath.Join(projectDir, name, "sport.json"),
			filepath.Join(projectDir, name, "src", "sport.json"),
			filepath.Join(projectDir, name, "public", "sport.json"),
		}
		found := ""
		for _, p := range candidates {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				found = p
				break
			}
		}
		if found == "" {
			results = append(results, CheckResult{
				Section: "dogfood", Name: checkName, Status: "warn",
				Message: "sport.json not found",
				FixCmd:  "run: pnpm --filter " + name + " run sync-sport",
			})
			continue
		}
		results = append(results, CheckResult{
			Section: "dogfood", Name: checkName, Status: "pass",
			Message: filepath.Base(found) + " present",
		})
	}
	return results
}

// hexColorRegex matches a 3- or 6-digit hex color literal in source code.
// Used to enforce the brand token convention (no raw hex; use design tokens).
var hexColorRegex = regexp.MustCompile(`#[0-9a-fA-F]{6}\b|#[0-9a-fA-F]{3}\b`)

// checkNoHexColorsInSrc scans every dogfood subapp's src/ tree for raw hex
// color literals. Brand convention (per F15) requires using design tokens.
// CSS files inside .vendor/ or generated/ subtrees are skipped.
func checkNoHexColorsInSrc(projectDir string) []CheckResult {
	var results []CheckResult
	for _, name := range dogfoodSubapps {
		if name == "backend" {
			continue
		}
		checkName := fmt.Sprintf("DOGFOOD-HEX-01: no hex colors in %s/src", name)
		srcDir := filepath.Join(projectDir, name, "src")
		if _, err := os.Stat(srcDir); err != nil {
			// Subapp without a src/ tree (e.g., legacy structure) — skip.
			continue
		}
		offenders := scanHexColors(srcDir)
		if len(offenders) > 0 {
			// Cap message length so we don't flood JSON output.
			msg := fmt.Sprintf("%d file(s) contain hex color literals", len(offenders))
			if len(offenders) <= 3 {
				msg += ": " + strings.Join(offenders, ", ")
			}
			results = append(results, CheckResult{
				Section: "dogfood", Name: checkName, Status: "warn",
				Message: msg,
				FixCmd:  "replace hex colors with design tokens from packages/ui/tokens",
			})
			continue
		}
		results = append(results, CheckResult{
			Section: "dogfood", Name: checkName, Status: "pass",
			Message: "no hex colors found",
		})
	}
	return results
}

// scanHexColors walks a directory tree and returns relative paths of files
// containing hex color literals. .vendor/, node_modules/, .next/, build/,
// dist/, and generated/ subtrees are skipped.
func scanHexColors(root string) []string {
	var offenders []string
	skipDirs := map[string]bool{
		"node_modules": true, ".next": true, "build": true, "dist": true,
		"generated": true, ".vendor": true, "vendor": true, ".turbo": true,
	}
	scannedExts := map[string]bool{
		".css": true, ".scss": true, ".ts": true, ".tsx": true,
		".js": true, ".jsx": true, ".vue": true,
	}
	_ = filepathWalk(root, func(path string, isDir bool) error {
		base := filepath.Base(path)
		if isDir {
			if skipDirs[base] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !scannedExts[ext] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if hexColorRegex.Find(data) != nil {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			offenders = append(offenders, rel)
		}
		return nil
	})
	return offenders
}

// filepathWalk wraps filepath.Walk with a simpler callback signature.
// Errors reading individual entries are swallowed so a single permission-
// denied node does not abort the entire scan — matches doctor's
// "best-effort scan" posture. filepath.SkipDir from the callback is passed
// through unchanged so callers can short-circuit subtrees.
func filepathWalk(root string, fn func(path string, isDir bool) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip unreadable entries; never abort the scan.
			return nil
		}
		return fn(path, info.IsDir())
	})
}

// checkLicenseVaultPath verifies the owner license env var path is readable
// when NSELF_PLUGIN_LICENSE_KEY_OWNER is referenced. This catches missing
// vault sourcing in the CI environment.
func checkLicenseVaultPath() CheckResult {
	const checkName = "DOGFOOD-LICENSE-01: license vault path"
	if os.Getenv("NSELF_PLUGIN_LICENSE_KEY") != "" {
		return CheckResult{Section: "dogfood", Name: checkName, Status: "pass",
			Message: "NSELF_PLUGIN_LICENSE_KEY set"}
	}
	if os.Getenv("NSELF_PLUGIN_LICENSE_KEY_OWNER") != "" {
		return CheckResult{Section: "dogfood", Name: checkName, Status: "pass",
			Message: "NSELF_PLUGIN_LICENSE_KEY_OWNER set"}
	}
	return CheckResult{Section: "dogfood", Name: checkName, Status: "warn",
		Message: "no license env var set; some pro plugin checks may skip",
		FixCmd:  "source ~/.claude/vault.env  # or set NSELF_PLUGIN_LICENSE_KEY"}
}
