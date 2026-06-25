package db

import (
	"context"
	"testing"
)

// TestOpen_EmptyDSN verifies that Open rejects an empty DSN without making
// any network calls. This is a pure-logic guard that runs offline.
func TestOpen_EmptyDSN(t *testing.T) {
	_, err := Open(context.Background(), PoolConfig{DSN: ""})
	if err == nil {
		t.Fatal("expected error for empty DSN, got nil")
	}
}

// TestOpen_MalformedDSN verifies that Open rejects a malformed DSN string.
func TestOpen_MalformedDSN(t *testing.T) {
	_, err := Open(context.Background(), PoolConfig{DSN: "not-a-valid-dsn"})
	if err == nil {
		t.Fatal("expected error for malformed DSN, got nil")
	}
}

// TestHealthCheck_NilPool verifies that HealthCheck returns an error for a
// nil pool — this branch is reachable without a live Postgres instance.
func TestHealthCheck_NilPool(t *testing.T) {
	err := HealthCheck(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil pool, got nil")
	}
}
