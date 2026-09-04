package nginx

// pathzones.go — SEC-HARDENING-06: path-scoped nginx rate-limit locations
// (P6-E2-W2-S3-T20). Split out of generator.go (CLI-R12 pattern, file-size
// budget) since the field it backs (ServiceRouteData.PathZones) is declared
// there.
//
// Purpose: describe and default the /auth/login + /api/ path-scoped
// `location` blocks layered on top of every generated service's
// server-wide RateZone.
// Inputs: none — defaultSecurityPathZones is a pure literal constructor.
// Outputs: []PathZone, consumed by service.conf.tmpl (internal/nginx) and
// by cmd/commands/ssl_install.go's writeCustomDomainConf (which redeclares
// the same two blocks as inline nginx text rather than importing this
// type, since that package renders custom-domain confs via fmt.Sprintf,
// not text/template).
// Constraints: Zone names here (auth_strict, api) must already exist in
// rate-limits.conf.tmpl — this package does not validate that at render
// time; a typo here would render nginx config that fails `nginx -t`
// with an unknown-zone error, not a Go-level error.

// PathZone describes one path-scoped rate-limit location block
// (SEC-HARDENING-06 — see defaultSecurityPathZones).
type PathZone struct {
	// Path is the location match target, e.g. "/auth/login" or "/api/".
	Path string
	// Exact renders `location = Path` (exact match) instead of the default
	// prefix match `location Path`.
	Exact bool
	// Zone is the limit_req zone name. Must already be defined in
	// rate-limits.conf.tmpl (auth_strict, api, etc.) — this package does not
	// validate the name against the rendered rate-limits.conf.
	Zone string
	// Burst is the limit_req burst size for this zone.
	Burst int
}

// defaultSecurityPathZones returns the SEC-HARDENING-06 path-scoped
// rate-limit locations applied to every generated service conf and every
// `nself ssl add` custom-domain conf that proxies to a backend.
//
// ServiceRouteData.PathZones (generator.go) documents when this default is
// applied: every route in the real `nself build` path
// (generateAllRoutes in routes.go) and every RenderServiceRoute caller,
// unless the caller passes its own non-nil slice (including an explicit
// empty one, which opts out).
//
// Applied unconditionally (not conditioned on per-service knowledge of what
// paths its upstream actually serves) per the ticket's stated fallback:
// harmless when the upstream never receives these paths (nginx simply
// proxies the more-strictly-limited request through, same as any other
// path under `location /`), and it closes the gap for every service without
// requiring routes_core.go/routes_app.go to track which services front
// auth/API sub-paths — a fact the generator has no reliable way to know for
// CS_N/INTERNAL_ROUTE_N services pointing at arbitrary upstreams.
//
// nginx resolves the longest-matching prefix `location` regardless of
// declaration order in the file, so these path-scoped blocks always win
// over the catch-all `location /` for the paths they name.
//
// Zone rates come from rate-limits.conf.tmpl (auth_strict reads
// RATE_LIMIT_AUTH_RPS, api reads RATE_LIMIT_API_RPS via ratelimit.go) — only
// the burst multipliers here are literal, matching the existing convention
// of per-route literal Burst values in routes_core.go/routes_app.go.
func defaultSecurityPathZones() []PathZone {
	return []PathZone{
		{Path: "/auth/login", Exact: true, Zone: "auth_strict", Burst: 5},
		{Path: "/api/", Zone: "api", Burst: 20},
	}
}
