package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMigrationAdminAudit(t *testing.T) {
	sql := MigrationAdminAudit()
	if sql == "" {
		t.Fatal("MigrationAdminAudit returned empty SQL")
	}
	// Verify all 9 columns mentioned
	for _, col := range []string{
		"id", "event_time", "actor_email", "actor_ip",
		"method", "path", "body_hash", "result_code",
		"duration_ms", "session_id",
	} {
		if !contains(sql, col) {
			t.Errorf("missing column %q in migration SQL", col)
		}
	}
}

func TestMigrationAdminACL(t *testing.T) {
	sql := MigrationAdminACL()
	if sql == "" {
		t.Fatal("MigrationAdminACL returned empty SQL")
	}
	for _, col := range []string{"user_email", "project", "role"} {
		if !contains(sql, col) {
			t.Errorf("missing column %q in ACL migration SQL", col)
		}
	}
	// Role CHECK constraint
	if !contains(sql, "owner") || !contains(sql, "operator") || !contains(sql, "viewer") {
		t.Error("ACL migration missing role CHECK values")
	}
}

func TestShouldAudit(t *testing.T) {
	// Writes always audited
	if !ShouldAudit("POST", 0) {
		t.Error("POST should always be audited")
	}
	if !ShouldAudit("PUT", 0) {
		t.Error("PUT should always be audited")
	}
	if !ShouldAudit("DELETE", 0) {
		t.Error("DELETE should always be audited")
	}
	// GET with 0% sample rate = never
	if ShouldAudit("GET", 0) {
		t.Error("GET with 0 sample rate should not be audited")
	}
	// GET with 100% sample rate = always
	if !ShouldAudit("GET", 1.0) {
		t.Error("GET with 1.0 sample rate should always be audited")
	}
}

func TestHashBody(t *testing.T) {
	if HashBody(nil) != "" {
		t.Error("nil body should return empty hash")
	}
	if HashBody([]byte("")) != "" {
		t.Error("empty body should return empty hash")
	}
	h := HashBody([]byte("test"))
	if len(h) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("expected 64-char hash, got %d", len(h))
	}
}

func TestNewSessionToken(t *testing.T) {
	tok, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64-char token, got %d", len(tok))
	}
}

func TestAdminURL(t *testing.T) {
	// Token must never appear in the URL (moved to BootstrapSession POST).
	u := AdminURL(3021, "")
	if u != "http://localhost:3021" {
		t.Errorf("unexpected bare URL: %s", u)
	}
	u = AdminURL(3021, "nself")
	if u != "http://localhost:3021?project=nself" {
		t.Errorf("unexpected URL with project: %s", u)
	}
	// Verify no token appears regardless of the project value.
	if contains(u, "token=") {
		t.Errorf("AdminURL must not contain token=, got: %s", u)
	}
}

func TestBootstrapSession(t *testing.T) {
	var receivedToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/bootstrap" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		receivedToken = body.Token
		w.Header().Set("Set-Cookie", "nself-session="+body.Token+"; HttpOnly; SameSite=Strict; Path=/")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// srv.URL is http://127.0.0.1:<port> — extract port for BootstrapSession.
	var port int
	if _, err := fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port); err != nil {
		t.Fatalf("could not parse server port from %s: %v", srv.URL, err)
	}

	token := "deadbeef01234567deadbeef01234567deadbeef01234567deadbeef01234567"
	if err := BootstrapSession(port, token); err != nil {
		t.Fatalf("BootstrapSession returned error: %v", err)
	}
	if receivedToken != token {
		t.Errorf("server received token %q, want %q", receivedToken, token)
	}
}

func TestAlertRules(t *testing.T) {
	rules := AlertRules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 alert rules, got %d", len(rules))
	}
	if rules[0].Name != "AdminAuthFailures" {
		t.Errorf("first rule name = %s, want AdminAuthFailures", rules[0].Name)
	}
	if rules[1].Name != "AdminPortExternallyReachable" {
		t.Errorf("second rule name = %s, want AdminPortExternallyReachable", rules[1].Name)
	}
}

func TestCheckACLSQL(t *testing.T) {
	sql := CheckACLSQL()
	if sql == "" {
		t.Fatal("CheckACLSQL returned empty")
	}
}

func TestSeedOwnerSQL(t *testing.T) {
	sql := SeedOwnerSQL("ali@example.com")
	if !contains(sql, "ali@example.com") {
		t.Error("SeedOwnerSQL should include the email")
	}
	if !contains(sql, "'*'") {
		t.Error("SeedOwnerSQL should include wildcard project")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && // avoid false positive on empty
		stringContains(s, substr)
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
