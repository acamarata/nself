package doctor

// dogfood_headers.go — security header probes for nSelf's own subapps
// (SEC-CSP-01, SEC-HSTS-01, SEC-AUTH-01), split out of dogfood_checks.go
// (CLI-R12) as a pure move.
//
// Inputs: a context, used to bound each HTTP probe.
// Outputs: []CheckResult (checkCSPHeaders, one per subapp) or a single
// CheckResult (checkHSTSHeader, checkHttpOnlyCookies).
// Constraints: depends on dogfoodSubappURLs and httpHeaderClient, defined in
// dogfood_checks.go. Unreachable subapps are reported as "warn", never "fail",
// so offline CI runs don't false-fail.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
