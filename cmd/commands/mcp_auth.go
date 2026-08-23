package commands

// mcp_auth.go — optional bearer-token gate for the SSE/HTTP MCP transports.
//
// Purpose: CLI-R15. The server already binds to 127.0.0.1 only, but that
//   doesn't stop other local users or processes on a shared host from
//   reaching the port. NSELF_MCP_TOKEN adds a shared-secret check for that
//   case, for operators who want it.
// Inputs:  NSELF_MCP_TOKEN env var; the incoming request's Authorization
//   header ("Bearer <token>").
// Outputs: an http.Handler wrapper that 401s non-matching requests.
// Constraints: a no-op pass-through when NSELF_MCP_TOKEN is unset, so
//   existing setups are unaffected. Uses a constant-time comparison to avoid
//   leaking the token length/prefix through response timing.
// SPORT: CLI-CMD-MCP-001

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

// mcpTokenEnv is the env var that, when set, requires every SSE/HTTP request
// to present a matching bearer token.
const mcpTokenEnv = "NSELF_MCP_TOKEN"

// mcpBearerMiddleware wraps h with a bearer-token check driven by
// NSELF_MCP_TOKEN. Returns h unchanged when the env var is empty.
func mcpBearerMiddleware(h http.Handler) http.Handler {
	token := os.Getenv(mcpTokenEnv)
	if token == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="nself-mcp"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}
