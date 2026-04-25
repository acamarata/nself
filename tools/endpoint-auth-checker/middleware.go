// Package main — middleware.go
// Classifies a route's middleware chain against the allowlist.
package main

// CheckResult holds the outcome of checking one route.
type CheckResult struct {
	Route      RouteRegistration
	Passed     bool   // true = has an allowed middleware or is exempt
	Exempt     bool   // true = path matched an ExemptRoute
	MatchedMW  string // which allowlisted middleware matched (empty if exempt)
	UnknownMWs []string // middlewares that are not in the allowlist and not exempt
}

// ClassifyRoute checks a single route against the allowlist and exempt list.
func ClassifyRoute(r RouteRegistration, failOnUnknown bool) CheckResult {
	cr := CheckResult{Route: r}

	// 1. Check exemptions first.
	if r.Path != "" && isExempt(r.Path) {
		cr.Passed = true
		cr.Exempt = true
		return cr
	}

	// 2. Check each middleware against the allowlist.
	for _, mw := range r.Middlewares {
		if _, ok := AllowedAuthMiddleware[mw]; ok {
			cr.Passed = true
			cr.MatchedMW = mw
			return cr
		}
	}

	// 3. Collect unknown middlewares for reporting.
	for _, mw := range r.Middlewares {
		if _, ok := AllowedAuthMiddleware[mw]; !ok {
			cr.UnknownMWs = append(cr.UnknownMWs, mw)
		}
	}

	// 4. No auth middleware found.
	cr.Passed = false
	return cr
}

// ClassifyAll runs ClassifyRoute on every route and returns only the failures.
func ClassifyAll(routes []RouteRegistration) []CheckResult {
	var failures []CheckResult
	for _, r := range routes {
		cr := ClassifyRoute(r, true)
		if !cr.Passed {
			failures = append(failures, cr)
		}
	}
	return failures
}
