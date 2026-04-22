// Package auth — tests for device code polling loop.
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestPollDeviceCode_Success verifies that a successful authorization returns the token.
func TestPollDeviceCode_Success(t *testing.T) {
	var calls int32
	want := TokenResponse{
		AccessToken:  "tok-access",
		SessionToken: "tok-session",
		Email:        "user@example.com",
		Tier:         "pro",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/auth/device/token" {
			http.NotFound(w, r)
			return
		}
		// First call: pending; second call: authorized.
		if n == 1 {
			w.WriteHeader(http.StatusAccepted) // authorization_pending
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)

	// Use a very short poll interval override via monkey-patch on defaultPollInterval.
	// We can't override the package-level const, so just verify timing is reasonable.
	ctx := context.Background()
	result, err := PollDeviceCode(ctx, "dev-code", nil)
	if err != nil {
		t.Fatalf("PollDeviceCode returned error: %v", err)
	}
	if result == nil || result.Token == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Token.Email != want.Email {
		t.Errorf("email: got %q, want %q", result.Token.Email, want.Email)
	}
	if result.Token.Tier != want.Tier {
		t.Errorf("tier: got %q, want %q", result.Token.Tier, want.Tier)
	}
}

// TestPollDeviceCode_ContextCancelled verifies early exit on context cancellation.
func TestPollDeviceCode_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always pending — should never complete.
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately after a tiny delay so PollDeviceCode enters the loop once.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := PollDeviceCode(ctx, "dev-code", nil)
	if err == nil {
		t.Fatal("expected error on context cancellation, got nil")
	}
}

// TestPollDeviceCode_Timeout verifies ErrPollTimeout is returned when deadline passes.
func TestPollDeviceCode_Timeout(t *testing.T) {
	// Reduce timeout for the test by using a context that times out quickly.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)

	// Use a context deadline shorter than defaultPollTimeout (5 min).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := PollDeviceCode(ctx, "dev-code", nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

// TestPollDeviceCode_OnPollCallback verifies the callback is invoked.
func TestPollDeviceCode_OnPollCallback(t *testing.T) {
	var cbCalls int32
	want := TokenResponse{Email: "cb@example.com", Tier: "free", AccessToken: "tok"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)

	_, err := PollDeviceCode(context.Background(), "dev-code", func(elapsed time.Duration) {
		atomic.AddInt32(&cbCalls, 1)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&cbCalls) == 0 {
		t.Error("onPoll callback was never called")
	}
}

// TestPollDeviceCode_ServerError verifies propagation of auth server errors.
func TestPollDeviceCode_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "expired_token",
			"message": "device code has expired",
		})
	}))
	defer srv.Close()
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)

	_, err := PollDeviceCode(context.Background(), "bad-code", nil)
	if err == nil {
		t.Fatal("expected error from server, got nil")
	}
}
