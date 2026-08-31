package ci

// Purpose: Tests for nself ci serve — webhook server, HMAC verification, event parsing.
// Inputs:  HTTP test server + crafted webhook payloads
// Outputs: Status codes, parsed ciJob structs, HMAC pass/fail
// Constraints: No Docker/gh CLI required — tests the HTTP + signature + dispatch path only.
// SPORT: CLI-CMD-CI-SERVE-001

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// signBody computes the X-Hub-Signature-256 header value for a payload and secret.
func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_valid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"test":"payload"}`)
	sig := signBody(secret, body)
	if err := verifySignature(secret, sig, body); err != nil {
		t.Fatalf("expected valid signature to pass: %v", err)
	}
}

func TestVerifySignature_tampered(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"test":"payload"}`)
	sig := signBody(secret, body)
	// Tamper with body.
	tampered := []byte(`{"test":"tampered"}`)
	if err := verifySignature(secret, sig, tampered); err == nil {
		t.Fatal("expected tampered body to fail signature check")
	}
}

func TestVerifySignature_wrongSecret(t *testing.T) {
	body := []byte(`{"test":"payload"}`)
	sig := signBody("correct-secret", body)
	if err := verifySignature("wrong-secret", sig, body); err == nil {
		t.Fatal("expected wrong secret to fail")
	}
}

func TestVerifySignature_missingHeader(t *testing.T) {
	body := []byte(`{}`)
	if err := verifySignature("secret", "", body); err == nil {
		t.Fatal("expected missing header to fail")
	}
}

func TestParseEvent_push(t *testing.T) {
	payload := map[string]interface{}{
		"after":   "abc123def456abc123def456abc123def456abc1",
		"ref":     "refs/heads/main",
		"deleted": false,
		"repository": map[string]interface{}{
			"full_name": "nself-org/cli",
			"clone_url": "https://github.com/nself-org/cli.git",
		},
	}
	body, _ := json.Marshal(payload)
	job, err := parseEvent("push", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.SHA != "abc123def456abc123def456abc123def456abc1" {
		t.Errorf("SHA mismatch: got %q", job.SHA)
	}
	if job.RepoFullName != "nself-org/cli" {
		t.Errorf("RepoFullName mismatch: got %q", job.RepoFullName)
	}
	if job.EventType != "push" {
		t.Errorf("EventType mismatch: got %q", job.EventType)
	}
}

func TestParseEvent_pushDeleted(t *testing.T) {
	payload := map[string]interface{}{
		"after":   "0000000000000000000000000000000000000000",
		"ref":     "refs/heads/feat/old",
		"deleted": true,
		"repository": map[string]interface{}{
			"full_name": "nself-org/cli",
			"clone_url": "https://github.com/nself-org/cli.git",
		},
	}
	body, _ := json.Marshal(payload)
	_, err := parseEvent("push", body)
	if err == nil {
		t.Fatal("expected delete push to be skipped")
	}
}

func TestParseEvent_pullRequest(t *testing.T) {
	payload := map[string]interface{}{
		"action": "synchronize",
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{
				"sha": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
				"ref": "feat/nsentry-ops-profile",
				"repo": map[string]interface{}{
					"full_name": "nself-org/cli",
					"clone_url": "https://github.com/nself-org/cli.git",
				},
			},
		},
	}
	body, _ := json.Marshal(payload)
	job, err := parseEvent("pull_request", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.SHA != "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef" {
		t.Errorf("SHA mismatch: got %q", job.SHA)
	}
	if job.EventType != "pull_request" {
		t.Errorf("EventType: got %q", job.EventType)
	}
}

func TestParseEvent_unknownEvent(t *testing.T) {
	_, err := parseEvent("ping", []byte(`{}`))
	if err == nil {
		t.Fatal("expected unknown event to be skipped")
	}
}

func TestHandleHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handleHealthz(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("healthz: got %d, want 200", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("healthz body not JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("healthz status: got %q, want ok", resp["status"])
	}
	if resp["port"] != "3845" {
		t.Errorf("healthz port: got %q, want 3845", resp["port"])
	}
}

func TestWebhookHandler_signatureRequired(t *testing.T) {
	secret := "webhook-secret"
	body := []byte(`{"after":"abc123","ref":"refs/heads/main","deleted":false,"repository":{"full_name":"o/r","clone_url":"https://github.com/o/r.git"}}`)

	handler := &webhookHandler{
		secret:     secret,
		sem:        make(chan struct{}, 1),
		binaryPath: "/usr/local/bin/nself-ci",
		cfg:        ServeConfig{Concurrency: 1, JobTimeout: 60},
	}

	// Missing signature — must be 403.
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("missing sig: got %d, want 403", w.Code)
	}

	// Correct signature — must be 202.
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req2.Header.Set("X-GitHub-Event", "push")
	req2.Header.Set("X-Hub-Signature-256", signBody(secret, body))
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusAccepted {
		t.Errorf("valid sig: got %d, want 202 — body: %s", w2.Code, w2.Body.String())
	}
}

func TestWebhookHandler_noSecret(t *testing.T) {
	// When secret is empty, all POSTs are accepted without signature check.
	body := []byte(`{"after":"aabbcc","ref":"refs/heads/feat","deleted":false,"repository":{"full_name":"o/r","clone_url":"https://github.com/o/r.git"}}`)
	handler := &webhookHandler{
		secret:     "",
		sem:        make(chan struct{}, 1),
		binaryPath: "/usr/local/bin/nself-ci",
		cfg:        ServeConfig{Concurrency: 1, JobTimeout: 60},
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("no-secret: got %d, want 202", w.Code)
	}
}

func TestSplitFullName(t *testing.T) {
	cases := []struct{ input, wantOwner, wantRepo string }{
		{"nself-org/cli", "nself-org", "cli"},
		{"owner/repo-name", "owner", "repo-name"},
		{"noSlash", "noSlash", "noSlash"},
	}
	for _, c := range cases {
		owner, repo := splitFullName(c.input)
		if owner != c.wantOwner || repo != c.wantRepo {
			t.Errorf("splitFullName(%q) = (%q, %q), want (%q, %q)",
				c.input, owner, repo, c.wantOwner, c.wantRepo)
		}
	}
}

func TestExtractSummary(t *testing.T) {
	output := "running gates...\n✓ go:fmt\n✓ go:vet\nAll gates passed (go) in 12s"
	s := extractSummary(output)
	if !strings.Contains(s, "All gates passed") {
		t.Errorf("extractSummary got %q, want last non-empty line", s)
	}
}

func TestTruncateStr(t *testing.T) {
	long := strings.Repeat("a", 200)
	got := truncateStr(long, 140)
	if len([]rune(got)) != 140 {
		t.Errorf("truncateStr len = %d, want 140", len([]rune(got)))
	}
	short := "hello"
	if truncateStr(short, 10) != short {
		t.Errorf("short string should not be truncated")
	}
}

func TestHandleInfo(t *testing.T) {
	cfg := ServeConfig{Addr: ":3845", Concurrency: 2, JobTimeout: 600, WorkDir: "/tmp"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handleInfo(w, req, cfg)
	if w.Code != http.StatusOK {
		t.Errorf("info: got %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "3845") {
		t.Errorf("info page missing port 3845: %q", body)
	}
}

func TestWebhookHandler_concurrencyLimit(t *testing.T) {
	// sem of size 0 → always 503.
	body := []byte(`{"after":"aabbcc","ref":"refs/heads/main","deleted":false,"repository":{"full_name":"o/r","clone_url":"https://github.com/o/r.git"}}`)
	handler := &webhookHandler{
		secret:     "",
		sem:        make(chan struct{}, 0),
		binaryPath: "/usr/local/bin/nself-ci",
		cfg:        ServeConfig{Concurrency: 0, JobTimeout: 60},
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("full pool: got %d, want 503", w.Code)
	}
}

func TestMinHelper(t *testing.T) {
	if min(3, 7) != 3 || min(7, 3) != 3 || min(5, 5) != 5 {
		t.Error("min helper incorrect")
	}
}

func TestWebhookHandler_getNotAllowed(t *testing.T) {
	handler := &webhookHandler{
		secret: "",
		sem:    make(chan struct{}, 1),
		cfg:    ServeConfig{JobTimeout: 60},
	}
	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET on webhook: got %d, want 405", w.Code)
	}
}

// TestIntegration_HttpServer starts the actual server on a random port and
// exercises /healthz + a signed push webhook to verify the full HTTP + dispatch path.
func TestIntegration_HttpServer(t *testing.T) {
	secret := "integration-test-secret"
	cfg := ServeConfig{
		Addr:        ":0", // let OS assign port
		Secret:      secret,
		Concurrency: 1,
		WorkDir:     t.TempDir(),
		JobTimeout:  5,
	}

	// Start server in background.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	handler := &webhookHandler{
		secret:     secret,
		sem:        make(chan struct{}, cfg.Concurrency),
		binaryPath: "/usr/local/bin/nself-ci", // existence not required for dispatch test
		cfg:        cfg,
	}
	mux.HandleFunc("/webhook", handler.ServeHTTP)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Health check.
	resp, err := http.Get(fmt.Sprintf("%s/healthz", ts.URL))
	if err != nil {
		t.Fatalf("healthz GET: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthz: got %d, want 200", resp.StatusCode)
	}

	// 2. Signed webhook POST — 202 Accepted.
	body := []byte(`{"after":"cafebabecafebabecafebabecafebabecafebabe","ref":"refs/heads/main","deleted":false,"repository":{"full_name":"nself-org/test","clone_url":"https://github.com/nself-org/test.git"}}`)
	sig := signBody(secret, body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("webhook POST: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusAccepted {
		t.Errorf("signed webhook: got %d, want 202", resp2.StatusCode)
	}

	// 3. Wrong signature — 403.
	req3, _ := http.NewRequest(http.MethodPost, ts.URL+"/webhook", bytes.NewReader(body))
	req3.Header.Set("X-GitHub-Event", "push")
	req3.Header.Set("X-Hub-Signature-256", "sha256=badhex000000000000000000000000000000000000000000000000000000000000")
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("bad-sig POST: %v", err)
	}
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusForbidden {
		t.Errorf("bad-sig: got %d, want 403", resp3.StatusCode)
	}
}
