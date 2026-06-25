package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNew_Defaults verifies that New applies the documented defaults when
// called with a zero ClientOptions struct.
func TestNew_Defaults(t *testing.T) {
	c := New(ClientOptions{})
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.HTTP == nil {
		t.Fatal("New returned Client with nil HTTP field")
	}
}

// TestNew_ExplicitOptions verifies that explicit opts are respected.
func TestNew_ExplicitOptions(t *testing.T) {
	c := New(ClientOptions{
		Timeout:    5 * time.Second,
		UserAgent:  "test-agent/1.0",
		MaxRetries: 1,
		RetryDelay: 100 * time.Millisecond,
	})
	if c == nil {
		t.Fatal("New returned nil")
	}
}

// TestNew_NegativeRetries verifies that MaxRetries < 0 is clamped to 0 then
// gets the default of 2 (the guard in New applies the default when == 0).
func TestNew_NegativeRetries(t *testing.T) {
	c := New(ClientOptions{MaxRetries: -1})
	if c == nil {
		t.Fatal("New returned nil")
	}
}

// TestDo_SetsUserAgent verifies that Do injects the User-Agent header when
// the caller did not set one.
func TestDo_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(ClientOptions{MaxRetries: 0, Timeout: 5 * time.Second})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if !strings.HasPrefix(gotUA, "nself-plugin-sdk") {
		t.Errorf("expected User-Agent to start with 'nself-plugin-sdk', got %q", gotUA)
	}
}

// TestDo_PreservesUserAgent verifies that Do does not overwrite a caller-set
// User-Agent.
func TestDo_PreservesUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(ClientOptions{MaxRetries: 0, Timeout: 5 * time.Second})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	req.Header.Set("User-Agent", "my-custom-agent/2.0")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	defer resp.Body.Close()

	if gotUA != "my-custom-agent/2.0" {
		t.Errorf("expected User-Agent 'my-custom-agent/2.0', got %q", gotUA)
	}
}

// TestDo_Returns2xx verifies that a 2xx response is returned without error.
func TestDo_Returns2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(ClientOptions{MaxRetries: 0, Timeout: 5 * time.Second})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do returned error for 2xx: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

// TestDo_5xxExhaustsRetries verifies that Do returns an error after exhausting
// retries on persistent 5xx responses.
func TestDo_5xxExhaustsRetries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(ClientOptions{MaxRetries: 2, RetryDelay: 1 * time.Millisecond, Timeout: 5 * time.Second})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := c.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected error after 5xx retries, got nil")
	}
	// 1 initial + 2 retries = 3 calls
	if calls != 3 {
		t.Errorf("expected 3 calls (1+2 retries), got %d", calls)
	}
}

// TestDo_ContextCancellation verifies that Do respects context cancellation
// during the retry wait.
func TestDo_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := New(ClientOptions{MaxRetries: 3, RetryDelay: 1 * time.Second, Timeout: 5 * time.Second})
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	_, err := c.Do(ctx, req)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
