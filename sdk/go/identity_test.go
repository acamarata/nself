package sdk

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestNewIdentityGeneratesUniqueKeypairs(t *testing.T) {
	a, err := NewIdentity("plugin-a")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	b, err := NewIdentity("plugin-b")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	if bytes.Equal(a.PublicKey, b.PublicKey) {
		t.Error("two NewIdentity calls produced the same public key")
	}
	if a.PluginName != "plugin-a" {
		t.Errorf("PluginName: got %q want %q", a.PluginName, "plugin-a")
	}
}

func TestNewIdentityRejectsEmptyName(t *testing.T) {
	_, err := NewIdentity("")
	if err == nil {
		t.Error("expected error for empty pluginName, got nil")
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	id, err := NewIdentity("test-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	msg := []byte("hello inter-plugin world")
	sigStr := id.Sign(msg)

	// Decode the base64 signature and verify it.
	sig, err := base64.StdEncoding.DecodeString(sigStr)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if !id.Verify(msg, sig) {
		t.Error("Verify returned false for a valid signature")
	}
}

func TestVerifyRejectsTamperedMessage(t *testing.T) {
	id, err := NewIdentity("test-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	sigStr := id.Sign([]byte("original"))
	sig, _ := base64.StdEncoding.DecodeString(sigStr)

	if id.Verify([]byte("tampered"), sig) {
		t.Error("Verify accepted a signature for a different message")
	}
}

func TestSignRequestVerifyRequestRoundTrip(t *testing.T) {
	id, err := NewIdentity("caller-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	body := []byte(`{"key":"value"}`)
	req := newTestRequest(t, http.MethodPost, "/api/internal/data", body)

	if err := id.SignRequest(req, body); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	if err := VerifyRequest(req, id.PublicKey, body); err != nil {
		t.Errorf("VerifyRequest: %v", err)
	}
}

func TestSignRequestVerifyRequestNilBody(t *testing.T) {
	id, err := NewIdentity("caller-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	req := newTestRequest(t, http.MethodGet, "/health", nil)
	if err := id.SignRequest(req, nil); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}
	if err := VerifyRequest(req, id.PublicKey, nil); err != nil {
		t.Errorf("VerifyRequest with nil body: %v", err)
	}
}

func TestVerifyRequestRejectsExpiredTimestamp(t *testing.T) {
	id, err := NewIdentity("stale-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	body := []byte("payload")
	req := newTestRequest(t, http.MethodGet, "/check", body)

	if err := id.SignRequest(req, body); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	// Overwrite the timestamp header with one that is 6 minutes in the past.
	stale := time.Now().Add(-6 * time.Minute).Unix()
	req.Header.Set(TimestampHeader, itoa64(stale))

	// Re-sign with the stale timestamp so the signature matches but is expired.
	payload := buildRequestPayload(req.Method, req.URL.RequestURI(), stale, body)
	sig := id.Sign([]byte(payload))
	req.Header.Set(SignatureHeader, sig)

	err = VerifyRequest(req, id.PublicKey, body)
	if err == nil {
		t.Error("expected error for expired timestamp, got nil")
	}
}

func TestVerifyRequestRejectsTamperedBody(t *testing.T) {
	id, err := NewIdentity("integrity-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	originalBody := []byte("original body")
	req := newTestRequest(t, http.MethodPost, "/submit", originalBody)

	if err := id.SignRequest(req, originalBody); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	tamperedBody := []byte("tampered body")
	err = VerifyRequest(req, id.PublicKey, tamperedBody)
	if err == nil {
		t.Error("expected error when verifying with tampered body, got nil")
	}
}

func TestVerifyRequestRejectsMissingTimestamp(t *testing.T) {
	id, err := NewIdentity("missing-ts-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	req := newTestRequest(t, http.MethodGet, "/ping", nil)
	if err := id.SignRequest(req, nil); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}
	req.Header.Del(TimestampHeader)

	if err := VerifyRequest(req, id.PublicKey, nil); err == nil {
		t.Error("expected error for missing timestamp header, got nil")
	}
}

func TestVerifyRequestRejectsMissingSignature(t *testing.T) {
	id, err := NewIdentity("missing-sig-plugin")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	req := newTestRequest(t, http.MethodGet, "/ping", nil)
	if err := id.SignRequest(req, nil); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}
	req.Header.Del(SignatureHeader)

	if err := VerifyRequest(req, id.PublicKey, nil); err == nil {
		t.Error("expected error for missing signature header, got nil")
	}
}

func TestVerifyRequestRejectsWrongPublicKey(t *testing.T) {
	signer, err := NewIdentity("signer")
	if err != nil {
		t.Fatalf("NewIdentity signer: %v", err)
	}
	other, err := NewIdentity("other")
	if err != nil {
		t.Fatalf("NewIdentity other: %v", err)
	}

	body := []byte("data")
	req := newTestRequest(t, http.MethodPost, "/data", body)
	if err := signer.SignRequest(req, body); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	// Verify with the wrong key — must fail.
	if err := VerifyRequest(req, other.PublicKey, body); err == nil {
		t.Error("expected error when verifying with wrong public key, got nil")
	}
}

func TestLoadOrCreatePersistsAndLoads(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "keys", "my-plugin.pem")
	ctx := context.Background()

	// First call: file does not exist — generates and saves.
	id1, err := LoadOrCreate(ctx, "my-plugin", keyPath)
	if err != nil {
		t.Fatalf("LoadOrCreate (create): %v", err)
	}
	if id1 == nil {
		t.Fatal("LoadOrCreate returned nil identity")
	}

	// Verify file was written with restrictive permissions.
	fi, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key file not created: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("key file mode: got %o want 0600", fi.Mode().Perm())
	}

	// Second call: file exists — loads the same identity.
	id2, err := LoadOrCreate(ctx, "my-plugin", keyPath)
	if err != nil {
		t.Fatalf("LoadOrCreate (load): %v", err)
	}
	if !bytes.Equal(id1.PublicKey, id2.PublicKey) {
		t.Error("loaded identity has different public key than original")
	}

	// Verify sign/verify still works after round-trip through disk.
	msg := []byte("persistence check")
	sig := id2.Sign(msg)
	decoded, _ := base64.StdEncoding.DecodeString(sig)
	if !id1.Verify(msg, decoded) {
		t.Error("id1 could not verify signature produced by loaded id2")
	}
}

func TestLoadOrCreateRejectsEmptyName(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadOrCreate(context.Background(), "", filepath.Join(dir, "key.pem"))
	if err == nil {
		t.Error("expected error for empty pluginName")
	}
}

func TestLoadOrCreateRejectsEmptyPath(t *testing.T) {
	_, err := LoadOrCreate(context.Background(), "plugin", "")
	if err == nil {
		t.Error("expected error for empty keyPath")
	}
}

func TestPublicKeyPEMRoundTrip(t *testing.T) {
	id, err := NewIdentity("pem-test")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}

	pemBytes, err := id.PublicKeyPEM()
	if err != nil {
		t.Fatalf("PublicKeyPEM: %v", err)
	}
	if len(pemBytes) == 0 {
		t.Fatal("PublicKeyPEM returned empty bytes")
	}
	if string(pemBytes[:10]) != "-----BEGIN" {
		t.Errorf("unexpected PEM prefix: %q", string(pemBytes[:10]))
	}
}

// newTestRequest builds an *http.Request suitable for signing/verification tests.
func newTestRequest(t *testing.T, method, path string, _ []byte) *http.Request {
	t.Helper()
	u, err := url.Parse("http://plugin-host:8080" + path)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	req := &http.Request{
		Method: method,
		URL:    u,
		Header: make(http.Header),
	}
	return req
}

// itoa64 converts an int64 to its decimal string representation without
// importing strconv in the test (strconv is imported by the package under
// test; use a local helper to keep the test file self-contained).
func itoa64(v int64) string {
	return strconv.FormatInt(v, 10)
}

// ---- PeerRegistry tests ----

func TestPeerRegistryRegisterAndLookup(t *testing.T) {
	reg := NewPeerRegistry()

	id, err := NewIdentity("ai")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	if err := reg.Register("ai", id.PublicKey); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok := reg.Lookup("ai")
	if !ok {
		t.Fatal("Lookup returned false for registered peer")
	}
	if !bytes.Equal(got, id.PublicKey) {
		t.Error("Lookup returned a different public key than registered")
	}
}

func TestPeerRegistryLookupUnknown(t *testing.T) {
	reg := NewPeerRegistry()
	_, ok := reg.Lookup("nonexistent")
	if ok {
		t.Error("Lookup returned true for unregistered peer")
	}
}

func TestPeerRegistryRegisterRejectsEmptyName(t *testing.T) {
	reg := NewPeerRegistry()
	id, err := NewIdentity("p")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	if err := reg.Register("", id.PublicKey); err == nil {
		t.Error("expected error for empty pluginName, got nil")
	}
}

func TestPeerRegistryRegisterRejectsInvalidKey(t *testing.T) {
	reg := NewPeerRegistry()
	if err := reg.Register("plugin", []byte("short")); err == nil {
		t.Error("expected error for invalid key length, got nil")
	}
}

func TestPeerRegistryRemove(t *testing.T) {
	reg := NewPeerRegistry()
	id, err := NewIdentity("mux")
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	if err := reg.Register("mux", id.PublicKey); err != nil {
		t.Fatalf("Register: %v", err)
	}
	reg.Remove("mux")
	if _, ok := reg.Lookup("mux"); ok {
		t.Error("Lookup returned true after Remove")
	}
	// Remove on non-existent name must not panic.
	reg.Remove("nonexistent")
}

func TestPeerRegistryKeyRotation(t *testing.T) {
	reg := NewPeerRegistry()

	id1, err := NewIdentity("claw")
	if err != nil {
		t.Fatalf("NewIdentity id1: %v", err)
	}
	id2, err := NewIdentity("claw")
	if err != nil {
		t.Fatalf("NewIdentity id2: %v", err)
	}

	if err := reg.Register("claw", id1.PublicKey); err != nil {
		t.Fatalf("Register id1: %v", err)
	}
	if err := reg.Register("claw", id2.PublicKey); err != nil {
		t.Fatalf("Register id2: %v", err)
	}

	got, ok := reg.Lookup("claw")
	if !ok {
		t.Fatal("Lookup returned false after key rotation")
	}
	if !bytes.Equal(got, id2.PublicKey) {
		t.Error("Lookup returned old key after rotation")
	}
}

func TestPeerRegistryNames(t *testing.T) {
	reg := NewPeerRegistry()
	for _, name := range []string{"ai", "mux", "claw"} {
		id, err := NewIdentity(name)
		if err != nil {
			t.Fatalf("NewIdentity %s: %v", name, err)
		}
		if err := reg.Register(name, id.PublicKey); err != nil {
			t.Fatalf("Register %s: %v", name, err)
		}
	}
	names := reg.Names()
	if len(names) != 3 {
		t.Errorf("Names() returned %d entries, want 3", len(names))
	}
}

func TestPeerRegistryVerifyRequestIntegration(t *testing.T) {
	// Demonstrates the intended end-to-end usage of PeerRegistry.
	signer, err := NewIdentity("ai")
	if err != nil {
		t.Fatalf("NewIdentity signer: %v", err)
	}
	reg := NewPeerRegistry()
	if err := reg.Register("ai", signer.PublicKey); err != nil {
		t.Fatalf("Register: %v", err)
	}

	body := []byte(`{"prompt":"hello"}`)
	req := newTestRequest(t, http.MethodPost, "/api/ai/infer", body)
	if err := signer.SignRequest(req, body); err != nil {
		t.Fatalf("SignRequest: %v", err)
	}

	pubKey, ok := reg.Lookup("ai")
	if !ok {
		t.Fatal("Lookup: peer not found")
	}
	if err := VerifyRequest(req, pubKey, body); err != nil {
		t.Errorf("VerifyRequest via PeerRegistry: %v", err)
	}
}
