package plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newMockServer returns an httptest.Server that responds with the given status
// code and body string. The caller must call Close() when done.
func newMockServer(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

// TestValidateLicense_HTTP200ValidTrue verifies that a server returning
// HTTP 200 {"valid": true} causes ValidateLicenseRemote to return (true, nil).
func TestValidateLicense_HTTP200ValidTrue(t *testing.T) {
	srv := newMockServer(t, http.StatusOK, `{"valid":true,"tier":"pro"}`)
	defer srv.Close()

	valid, err := ValidateLicenseRemote(context.Background(), "nself_pro_testkey_aaaaaaaaaaaaaaaa", srv.URL)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !valid {
		t.Fatal("expected valid=true, got false")
	}
}

// TestValidateLicense_HTTP200ValidFalse verifies that a server returning
// HTTP 200 {"valid": false, "reason": "expired"} causes ValidateLicenseRemote
// to return (false, error) where the error message contains "expired".
func TestValidateLicense_HTTP200ValidFalse(t *testing.T) {
	srv := newMockServer(t, http.StatusOK, `{"valid":false,"reason":"expired"}`)
	defer srv.Close()

	valid, err := ValidateLicenseRemote(context.Background(), "nself_pro_testkey_aaaaaaaaaaaaaaaa", srv.URL)
	if valid {
		t.Fatal("expected valid=false, got true")
	}
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected error to contain %q, got: %v", "expired", err)
	}
}

// TestValidateLicense_HTTP200ValidFalseNoReason verifies that a server
// returning HTTP 200 {"valid": false} (no reason field) causes
// ValidateLicenseRemote to return (false, error) where the error message
// contains "invalid".
func TestValidateLicense_HTTP200ValidFalseNoReason(t *testing.T) {
	srv := newMockServer(t, http.StatusOK, `{"valid":false}`)
	defer srv.Close()

	valid, err := ValidateLicenseRemote(context.Background(), "nself_pro_testkey_aaaaaaaaaaaaaaaa", srv.URL)
	if valid {
		t.Fatal("expected valid=false, got true")
	}
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected error to contain %q, got: %v", "invalid", err)
	}
}

// TestValidateLicense_HTTP401 verifies that a server returning HTTP 401 causes
// ValidateLicenseRemote to return (false, nil) — unauthorized is not an error,
// it simply means the license is not valid.
func TestValidateLicense_HTTP401(t *testing.T) {
	srv := newMockServer(t, http.StatusUnauthorized, `{"error":"unauthorized"}`)
	defer srv.Close()

	valid, err := ValidateLicenseRemote(context.Background(), "nself_pro_testkey_aaaaaaaaaaaaaaaa", srv.URL)
	if valid {
		t.Fatal("expected valid=false, got true")
	}
	if err != nil {
		t.Fatalf("expected nil error for 401, got: %v", err)
	}
}

// TestValidateLicense_HTTP500 verifies that a server returning HTTP 500 causes
// ValidateLicenseRemote to return (false, error) — an unexpected server error
// must surface as an error to the caller.
func TestValidateLicense_HTTP500(t *testing.T) {
	srv := newMockServer(t, http.StatusInternalServerError, `{"error":"internal server error"}`)
	defer srv.Close()

	valid, err := ValidateLicenseRemote(context.Background(), "nself_pro_testkey_aaaaaaaaaaaaaaaa", srv.URL)
	if valid {
		t.Fatal("expected valid=false, got true")
	}
	if err == nil {
		t.Fatal("expected an error for HTTP 500, got nil")
	}
}

// TestValidateLicense_InvalidJSON verifies that a server returning HTTP 200
// with an unparsable body causes ValidateLicenseRemote to return (false, error)
// where the error message indicates an unreadable response.
func TestValidateLicense_InvalidJSON(t *testing.T) {
	srv := newMockServer(t, http.StatusOK, `not-json`)
	defer srv.Close()

	valid, err := ValidateLicenseRemote(context.Background(), "nself_pro_testkey_aaaaaaaaaaaaaaaa", srv.URL)
	if valid {
		t.Fatal("expected valid=false, got true")
	}
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "unreadable") {
		t.Fatalf("expected error to contain %q, got: %v", "unreadable", err)
	}
}
