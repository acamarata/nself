package nginx

// generateRateLimits renders the rate-limits.conf from the embedded template.
//
// This produces all 10 rate limiting zones defined in BUILD_SPEC Part 12:
//
//	Request rate zones (limit_req_zone):
//	  general       — 10r/s   keyed on $binary_remote_addr
//	  graphql_api   — 100r/m  keyed on $binary_remote_addr
//	  auth          — 30r/m   keyed on $binary_remote_addr (configurable via AUTH_RATE_LIMIT)
//	  uploads       — 5r/m    keyed on $binary_remote_addr
//	  user_api      — 1000r/m keyed on $http_authorization
//	  static        — 1000r/m keyed on $binary_remote_addr
//	  webhooks      — 30r/m   keyed on $binary_remote_addr
//	  functions     — 50r/m   keyed on $binary_remote_addr
//
//	Connection zones (limit_conn_zone):
//	  conn_limit_per_ip  — keyed on $binary_remote_addr
//	  conn_limit_server  — keyed on $server_name
//
//	Status codes: limit_req_status 429, limit_conn_status 429
func (g *Generator) generateRateLimits() (string, error) {
	apiRPS := g.cfg.Nginx.RateLimitAPI
	if apiRPS == "" {
		apiRPS = "30"
	}
	authRPS := g.cfg.Nginx.RateLimitAuth
	if authRPS == "" {
		authRPS = "5"
	}
	aiRPS := g.cfg.Nginx.RateLimitAI
	if aiRPS == "" {
		aiRPS = "10"
	}
	data := map[string]string{
		"AuthRateLimit":  g.cfg.Nginx.AuthRateLimit,
		"RateLimitAPI":   apiRPS,
		"RateLimitAuth":  authRPS,
		"RateLimitAI":    aiRPS,
	}
	return g.render("rate-limits.conf.tmpl", data)
}
