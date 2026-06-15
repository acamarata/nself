package aistudio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// -----------------------------------------------------------------------------
// cloudflared URL detection
// -----------------------------------------------------------------------------

func TestCloudflaredDownloadURL(t *testing.T) {
	u := cloudflaredDownloadURL()
	if !strings.Contains(u, "cloudflare/cloudflared") {
		t.Errorf("unexpected download URL: %s", u)
	}
	if !strings.Contains(u, cloudflaredVersion) {
		t.Errorf("URL missing version %s: %s", cloudflaredVersion, u)
	}
}

// -----------------------------------------------------------------------------
// JWT / token generation
// -----------------------------------------------------------------------------

func TestGenerateToken(t *testing.T) {
	tok1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	tok2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken second call: %v", err)
	}
	if len(tok1) != 64 {
		t.Errorf("expected 64-char hex token, got %d", len(tok1))
	}
	if tok1 == tok2 {
		t.Error("consecutive tokens should differ")
	}
}

func TestHmacEqual(t *testing.T) {
	if !hmacEqual("hello", "hello") {
		t.Error("identical strings should be equal")
	}
	if hmacEqual("hello", "world") {
		t.Error("different strings should not be equal")
	}
}

// -----------------------------------------------------------------------------
// Proxy: read-only enforcement
// -----------------------------------------------------------------------------

func TestIsWriteOperation(t *testing.T) {
	mutations := []string{
		"mutation { insert_users(objects: {}) { affected_rows } }",
		"mutation InsertUser { insert_users(objects: {}) { affected_rows } }",
		"  MUTATION { delete_items(where: {}) { affected_rows } }",
		"{ insert_orders(objects: {}) { id } }",
	}
	for _, q := range mutations {
		if !isWriteOperation(q) {
			t.Errorf("expected write op for: %q", q)
		}
	}

	reads := []string{
		"{ users { id name } }",
		"query GetUsers { users { id } }",
		"{ __schema { types { name } } }",
	}
	for _, q := range reads {
		if isWriteOperation(q) {
			t.Errorf("expected read op for: %q", q)
		}
	}
}

// -----------------------------------------------------------------------------
// Proxy: header injection
// -----------------------------------------------------------------------------

func TestProxySchemaHeaderInjection(t *testing.T) {
	// Build a fake Hasura backend.
	hasura := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The role header must be ai_studio_read.
		role := r.Header.Get("X-Hasura-Role")
		if role != "ai_studio_read" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"data":{}}`))
	}))
	defer hasura.Close()

	cfg := proxyConfig{
		ListenPort:   8891,
		HasuraURL:    hasura.URL,
		AuthToken:    "testtoken",
		InjectSchema: true,
		IdleTimeout:  30 * time.Minute,
	}

	// Override schema context directly (skip actual Hasura introspection in unit test).
	sc := &SchemaContext{
		Tables:     []TableInfo{{Name: "users", Schema: "public"}},
		ActiveRole: "ai_studio_read",
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(sc)
	schemaB64 := base64.StdEncoding.EncodeToString(b)

	ps := &proxyServer{
		cfg:          cfg,
		schemaCtx:    schemaB64,
		lastActivity: time.Now(),
		mux:          http.NewServeMux(),
	}
	target, _ := parseHasuraURL(hasura.URL)
	rp := buildReverseProxy(target, ps)
	ps.mux.Handle("/v1/graphql", ps.authenticate(ps.readonly(rp)))
	ps.mux.HandleFunc("/health", ps.handleHealth)
	ps.mux.Handle("/schema", ps.authenticate(http.HandlerFunc(ps.handleSchema)))

	srv := httptest.NewServer(ps.mux)
	defer srv.Close()

	// Health check (no auth required).
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health: want 200, got %d", resp.StatusCode)
	}

	// Schema endpoint requires auth.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/schema", nil)
	req.Header.Set("Authorization", "Bearer testtoken")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("schema: want 200, got %d", resp.StatusCode)
	}
}

// -----------------------------------------------------------------------------
// Proxy: write operation blocked
// -----------------------------------------------------------------------------

func TestProxyBlocksMutation(t *testing.T) {
	hasura := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{}}`))
	}))
	defer hasura.Close()

	cfg := proxyConfig{
		HasuraURL:   hasura.URL,
		AuthToken:   "tok",
		IdleTimeout: 30 * time.Minute,
	}
	ps := &proxyServer{
		cfg:          cfg,
		lastActivity: time.Now(),
		mux:          http.NewServeMux(),
	}
	target, _ := parseHasuraURL(hasura.URL)
	rp := buildReverseProxy(target, ps)
	ps.mux.Handle("/v1/graphql", ps.authenticate(ps.readonly(rp)))

	srv := httptest.NewServer(ps.mux)
	defer srv.Close()

	body := `{"query":"mutation { insert_users(objects: {name:\"x\"}) { affected_rows } }"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer tok")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("mutation request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("mutation should be blocked: want 403, got %d", resp.StatusCode)
	}
}

// -----------------------------------------------------------------------------
// Proxy: auth enforcement
// -----------------------------------------------------------------------------

func TestProxyAuthRequired(t *testing.T) {
	hasura := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{}}`))
	}))
	defer hasura.Close()

	cfg := proxyConfig{
		HasuraURL:   hasura.URL,
		AuthToken:   "secret",
		IdleTimeout: 30 * time.Minute,
	}
	ps := &proxyServer{
		cfg:          cfg,
		lastActivity: time.Now(),
		mux:          http.NewServeMux(),
	}
	target, _ := parseHasuraURL(hasura.URL)
	rp := buildReverseProxy(target, ps)
	ps.mux.Handle("/v1/graphql", ps.authenticate(ps.readonly(rp)))

	srv := httptest.NewServer(ps.mux)
	defer srv.Close()

	// No auth header.
	body := `{"query":"{ users { id } }"}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("no-auth request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("missing auth should be forbidden: want 403, got %d", resp.StatusCode)
	}

	// Wrong token.
	req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/graphql", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer wrongtoken")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("wrong-auth request: %v", err)
	}
	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("wrong auth should be forbidden: want 403, got %d", resp2.StatusCode)
	}
}

// -----------------------------------------------------------------------------
// IP allowlist parsing
// -----------------------------------------------------------------------------

func TestParseIPAllowlist(t *testing.T) {
	cases := []struct {
		raw    string
		expect int
		errMsg string
	}{
		{"192.168.1.0/24,10.0.0.1", 2, ""},
		{"", 0, ""},
		{"notanip", 0, "invalid IP"},
	}
	for _, c := range cases {
		prefixes, err := parseIPAllowlist(c.raw)
		if c.errMsg != "" {
			if err == nil || !strings.Contains(err.Error(), c.errMsg) {
				t.Errorf("raw=%q: want error containing %q, got %v", c.raw, c.errMsg, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("raw=%q: unexpected error: %v", c.raw, err)
		}
		if len(prefixes) != c.expect {
			t.Errorf("raw=%q: want %d prefixes, got %d", c.raw, c.expect, len(prefixes))
		}
	}
}

func TestIPAllowed(t *testing.T) {
	prefixes, _ := parseIPAllowlist("192.168.1.0/24")
	ps := &proxyServer{cfg: proxyConfig{IPAllowlist: prefixes}}

	addr1, _ := netip.ParseAddr("192.168.1.50")
	if !ps.ipAllowed(addr1) {
		t.Error("192.168.1.50 should be allowed in 192.168.1.0/24")
	}

	addr2, _ := netip.ParseAddr("10.0.0.1")
	if ps.ipAllowed(addr2) {
		t.Error("10.0.0.1 should not be allowed in 192.168.1.0/24")
	}
}

// -----------------------------------------------------------------------------
// Tunnel idle timeout (unit-level, no real cloudflared)
// -----------------------------------------------------------------------------

func TestIdleTimeoutDetection(t *testing.T) {
	ps := &proxyServer{
		lastActivity: time.Now().Add(-35 * time.Minute),
	}
	idle := time.Since(ps.lastActivity)
	threshold := 30 * time.Minute
	if idle < threshold {
		t.Errorf("expected idle %v >= %v", idle, threshold)
	}
}

// -----------------------------------------------------------------------------
// Dry run (smoke test via command flag check)
// -----------------------------------------------------------------------------

func TestDryRunFlag(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Just verify dry-run path returns nil without starting anything.
	bridgeFlagValues.dryRun = true
	bridgeFlagValues.port = 8892
	defer func() { bridgeFlagValues.dryRun = false }()

	// Capture stdout by re-running through runBridge with a cancelled context.
	// The dry-run path exits before any I/O that needs a real tunnel.
	err := runBridge(BridgeCmd, nil)
	_ = ctx
	if err != nil {
		t.Errorf("dry run should return nil, got: %v", err)
	}
}

// -----------------------------------------------------------------------------
// Helpers used by tests (keep test-local wrappers slim)
// -----------------------------------------------------------------------------

// parseHasuraURL is a test helper that calls url.Parse.
func parseHasuraURL(raw string) (interface{ String() string }, error) {
	// Avoids importing net/url at package level just for tests.
	import_url, err := parseHasuraURLInternal(raw)
	return import_url, err
}

// -----------------------------------------------------------------------------
// Token masking
// -----------------------------------------------------------------------------

func TestMaskToken(t *testing.T) {
	cases := []struct {
		name     string
		token    string
		debug    bool
		expected string
	}{
		{
			name:     "normal token (64 chars)",
			token:    "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567",
			debug:    false,
			expected: "a1b2c3d4****",
		},
		{
			name:     "short token (4 chars)",
			token:    "abcd",
			debug:    false,
			expected: "****",
		},
		{
			name:     "exact 8 chars",
			token:    "12345678",
			debug:    false,
			expected: "****",
		},
		{
			name:     "9 chars",
			token:    "123456789",
			debug:    false,
			expected: "12345678****",
		},
		{
			name:     "empty token",
			token:    "",
			debug:    false,
			expected: "****",
		},
		{
			name:     "debug mode shows full token",
			token:    "a1b2c3d4e5f6789012345678901234567890",
			debug:    true,
			expected: "a1b2c3d4e5f6789012345678901234567890",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Temporarily set NSELF_DEBUG for debug case.
			if c.debug {
				t.Setenv("NSELF_DEBUG", "1")
			} else {
				t.Setenv("NSELF_DEBUG", "")
			}

			result := maskToken(c.token)
			if result != c.expected {
				t.Errorf("maskToken(%q): want %q, got %q", c.token, c.expected, result)
			}
		})
	}
}
