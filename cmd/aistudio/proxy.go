package aistudio

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"
)

// proxyConfig holds runtime configuration for the bridge proxy.
type proxyConfig struct {
	ListenPort   int
	HasuraURL    string
	AdminSecret  string
	AuthToken    string    // optional pre-shared token
	InjectSchema bool      // whether to add X-Nself-Schema-Context
	IPAllowlist  []netip.Prefix // empty = no restriction
	IdleTimeout  time.Duration
}

// proxyServer is the HTTP reverse proxy that sits in front of Hasura.
type proxyServer struct {
	cfg          proxyConfig
	schemaCtx    string       // base64-encoded schema context (refreshed periodically)
	lastActivity time.Time
	mux          *http.ServeMux
}

// newProxyServer builds a proxyServer and wires up routes.
func newProxyServer(ctx context.Context, cfg proxyConfig) (*proxyServer, error) {
	ps := &proxyServer{
		cfg:          cfg,
		lastActivity: time.Now(),
	}

	// Pre-fetch schema context if injection enabled.
	if cfg.InjectSchema {
		sc, err := buildSchemaContext(ctx, cfg.HasuraURL, cfg.AdminSecret)
		if err != nil {
			// Non-fatal: log and continue without context.
			fmt.Fprintf(os.Stderr, "warning: schema context unavailable: %v\n", err)
		} else {
			ps.schemaCtx = sc
		}
	}

	target, err := url.Parse(cfg.HasuraURL)
	if err != nil {
		return nil, fmt.Errorf("parse hasura URL: %w", err)
	}

	rp := buildReverseProxy(target, ps)

	ps.mux = http.NewServeMux()
	ps.mux.Handle("/graphql", ps.authenticate(ps.readonly(rp)))
	ps.mux.Handle("/v1/graphql", ps.authenticate(ps.readonly(rp)))
	ps.mux.Handle("/schema", ps.authenticate(http.HandlerFunc(ps.handleSchema)))
	ps.mux.HandleFunc("/health", ps.handleHealth)
	ps.mux.Handle("/context", ps.authenticate(http.HandlerFunc(ps.handleContext)))
	return ps, nil
}

// buildReverseProxy constructs an httputil.ReverseProxy targeting target with
// the bridge's director applied. Exported as a package-level helper so tests
// can construct a proxy without going through newProxyServer.
func buildReverseProxy(target interface{ String() string }, ps *proxyServer) *httputil.ReverseProxy {
	parsedURL, _ := url.Parse(target.String())
	rp := httputil.NewSingleHostReverseProxy(parsedURL)
	rp.Director = ps.buildDirector(parsedURL, rp.Director)
	return rp
}

// parseHasuraURLInternal parses a raw Hasura URL and returns it as a *url.URL.
// Exposed as an internal helper so tests can construct a proxy target without
// importing net/url directly at test scope.
func parseHasuraURLInternal(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

// buildDirector wraps the default director to inject security headers and block DDL/DML.
func (ps *proxyServer) buildDirector(target *url.URL, orig func(*http.Request)) func(*http.Request) {
	return func(req *http.Request) {
		orig(req)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// Force read-only Hasura role; strip any client-supplied role override.
		req.Header.Del("x-hasura-role")
		req.Header.Del("X-Hasura-Role")
		req.Header.Set("X-Hasura-Role", "ai_studio_read")

		// Inject admin secret for schema access (the ai_studio_read role must exist in Hasura).
		if ps.cfg.AdminSecret != "" {
			req.Header.Set("x-hasura-admin-secret", ps.cfg.AdminSecret)
		}

		// Inject schema context.
		if ps.cfg.InjectSchema && ps.schemaCtx != "" {
			req.Header.Set("X-Nself-Schema-Context", ps.schemaCtx)
		}
	}
}

// authenticate enforces the optional pre-shared token from Authorization: Bearer <token>.
func (ps *proxyServer) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// IP allowlist check.
		if len(ps.cfg.IPAllowlist) > 0 {
			remoteHost, _, _ := net.SplitHostPort(r.RemoteAddr)
			addr, err := netip.ParseAddr(remoteHost)
			if err != nil || !ps.ipAllowed(addr) {
				http.Error(w, "forbidden: IP not in allowlist", http.StatusForbidden)
				return
			}
		}

		// Token check (optional — only enforced when configured).
		if ps.cfg.AuthToken != "" {
			token := bearerToken(r)
			if !hmacEqual(token, ps.cfg.AuthToken) {
				http.Error(w, "forbidden: invalid auth token", http.StatusForbidden)
				return
			}
		}
		ps.lastActivity = time.Now()
		next.ServeHTTP(w, r)
	})
}

// readonly blocks GraphQL mutations (DDL/DML) at the HTTP layer before forwarding.
func (ps *proxyServer) readonly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, err := readBodyPreserving(r)
			if err == nil {
				var gql struct {
					Query string `json:"query"`
				}
				if json.Unmarshal(body, &gql) == nil {
					if isWriteOperation(gql.Query) {
						http.Error(w, `{"errors":[{"message":"write operations are not permitted through the AI Studio bridge"}]}`, http.StatusForbidden)
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// handleHealth returns a simple health response without authentication.
func (ps *proxyServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "role": "ai_studio_read"})
}

// handleSchema returns the current schema context JSON (decoded from base64).
func (ps *proxyServer) handleSchema(w http.ResponseWriter, _ *http.Request) {
	if ps.schemaCtx == "" {
		http.Error(w, "schema context not available", http.StatusServiceUnavailable)
		return
	}
	raw, err := base64.StdEncoding.DecodeString(ps.schemaCtx)
	if err != nil {
		http.Error(w, "schema decode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(raw)
}

// handleContext returns schema context + bridge metadata.
func (ps *proxyServer) handleContext(w http.ResponseWriter, _ *http.Request) {
	out := map[string]interface{}{
		"schema_context_header": "X-Nself-Schema-Context",
		"role":                  "ai_studio_read",
		"inject_schema":         ps.cfg.InjectSchema,
		"schema_b64":            ps.schemaCtx,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// isWriteOperation returns true if the GraphQL query text looks like a mutation or DDL.
func isWriteOperation(query string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(q, "mutation") ||
		strings.Contains(q, "insert_") ||
		strings.Contains(q, "update_") ||
		strings.Contains(q, "delete_") ||
		strings.Contains(q, "drop ") ||
		strings.Contains(q, "alter ") ||
		strings.Contains(q, "create ")
}

// bearerToken extracts the token from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// hmacEqual does a constant-time comparison to prevent timing attacks.
func hmacEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return hmac.Equal(ha[:], hb[:])
}

// generateToken creates a cryptographically random hex token (32 bytes = 64 hex chars).
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// parseIPAllowlist parses a comma-separated list of CIDR prefixes.
func parseIPAllowlist(raw string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Bare IP address: treat as /32 or /128.
		if !strings.Contains(s, "/") {
			addr, err := netip.ParseAddr(s)
			if err != nil {
				return nil, fmt.Errorf("invalid IP %q: %w", s, err)
			}
			bits := 32
			if addr.Is6() {
				bits = 128
			}
			s = fmt.Sprintf("%s/%d", addr.String(), bits)
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", s, err)
		}
		prefixes = append(prefixes, p)
	}
	return prefixes, nil
}

// ipAllowed returns true if addr matches any prefix in the allowlist.
func (ps *proxyServer) ipAllowed(addr netip.Addr) bool {
	for _, p := range ps.cfg.IPAllowlist {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
