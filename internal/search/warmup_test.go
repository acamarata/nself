package search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ── QueriesFromEnv ────────────────────────────────────────────────────────────

func TestQueriesFromEnv_Unset(t *testing.T) {
	t.Setenv(EnvWarmupQueries, "")
	if got := QueriesFromEnv(); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestQueriesFromEnv_Single(t *testing.T) {
	t.Setenv(EnvWarmupQueries, "hello")
	got := QueriesFromEnv()
	if len(got) != 1 || got[0] != "hello" {
		t.Fatalf("unexpected %v", got)
	}
}

func TestQueriesFromEnv_Multiple(t *testing.T) {
	t.Setenv(EnvWarmupQueries, "alpha, beta , gamma")
	got := QueriesFromEnv()
	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("len %d != %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] got %q want %q", i, got[i], w)
		}
	}
}

func TestQueriesFromEnv_EmptySegmentsSkipped(t *testing.T) {
	t.Setenv(EnvWarmupQueries, "a,,b,  ,c")
	got := QueriesFromEnv()
	if len(got) != 3 {
		t.Fatalf("expected 3 queries, got %v", got)
	}
}

// ── Warmup no-op ──────────────────────────────────────────────────────────────

func TestWarmup_NoQueriesIsNoop(t *testing.T) {
	cfg := WarmupConfig{Host: "localhost", Port: 0, Queries: nil}
	result, err := Warmup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.QueriesAttempted != 0 || result.QueriesSucceeded != 0 {
		t.Fatalf("expected zero result, got %+v", result)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mockMeiliServer(t *testing.T, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/multi-search") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(statusCode)
	}))
}

func serverPort(srv *httptest.Server) int {
	addr := srv.Listener.Addr().String()
	var port int
	_, _ = fmt.Sscanf(strings.SplitN(addr, ":", 2)[1], "%d", &port)
	return port
}

// ── Warmup happy path ─────────────────────────────────────────────────────────

func TestWarmup_Success(t *testing.T) {
	srv := mockMeiliServer(t, http.StatusOK)
	defer srv.Close()

	cfg := WarmupConfig{
		Host:    "127.0.0.1",
		Port:    serverPort(srv),
		Queries: []string{"test", "hello"},
	}
	result, err := Warmup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.QueriesAttempted != 2 {
		t.Errorf("attempted: want 2 got %d", result.QueriesAttempted)
	}
	if result.QueriesSucceeded != 2 {
		t.Errorf("succeeded: want 2 got %d", result.QueriesSucceeded)
	}
	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestWarmup_404TreatedAsSuccess(t *testing.T) {
	srv := mockMeiliServer(t, http.StatusNotFound)
	defer srv.Close()

	cfg := WarmupConfig{
		Host:    "127.0.0.1",
		Port:    serverPort(srv),
		Queries: []string{"anything"},
	}
	result, err := Warmup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.QueriesSucceeded != 1 {
		t.Errorf("expected 1 success, got %d", result.QueriesSucceeded)
	}
}

func TestWarmup_ServerError_CollectedNotFatal(t *testing.T) {
	srv := mockMeiliServer(t, http.StatusInternalServerError)
	defer srv.Close()

	cfg := WarmupConfig{
		Host:    "127.0.0.1",
		Port:    serverPort(srv),
		Queries: []string{"q1", "q2"},
	}
	result, err := Warmup(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if result.QueriesSucceeded != 0 {
		t.Errorf("expected 0 successes, got %d", result.QueriesSucceeded)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %v", result.Errors)
	}
}

func TestWarmup_ContextCancelledMidway(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := WarmupConfig{
		Host:    "127.0.0.1",
		Port:    serverPort(srv),
		Queries: []string{"slow1", "slow2", "slow3"},
	}
	result, err := Warmup(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if result.QueriesAttempted == 0 {
		t.Error("expected at least 1 attempted query")
	}
	// With a 50ms context and 500ms server delay, no queries should succeed.
	if result.QueriesSucceeded > 0 {
		t.Errorf("expected 0 successes against slow server, got %d", result.QueriesSucceeded)
	}
}

func TestWarmup_MasterKeyAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := WarmupConfig{
		Host:      "127.0.0.1",
		Port:      serverPort(srv),
		MasterKey: "secret-key",
		Queries:   []string{"check"},
	}
	_, _ = Warmup(context.Background(), cfg)
	if gotAuth != "Bearer secret-key" {
		t.Errorf("expected Bearer secret-key, got %q", gotAuth)
	}
}
