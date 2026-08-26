package database

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestIsTransientPostgresError_Transient verifies the classifier recognizes
// each of the three postgres startup/shutdown FATAL messages observed in
// run 32948887818 (the temporary init-server race), including when wrapped
// with additional context the way runSQLOnDB/createDatabase format them.
func TestIsTransientPostgresError_Transient(t *testing.T) {
	cases := []string{
		"FATAL:  the database system is shutting down",
		"FATAL:  the database system is starting up",
		"FATAL:  the database system is not yet accepting connections",
	}
	for _, msg := range cases {
		wrapped := fmt.Errorf("%s: %w", msg, errors.New("exit status 2"))
		if !isTransientPostgresError(wrapped) {
			t.Errorf("isTransientPostgresError(%q) = false, want true", wrapped)
		}
	}
}

// TestIsTransientPostgresError_NotTransient verifies genuine failures are
// never classified as retryable — retrying these would hide the real error
// behind a 60s timeout instead of surfacing it immediately.
func TestIsTransientPostgresError_NotTransient(t *testing.T) {
	cases := []error{
		errors.New(`psql: error: FATAL:  password authentication failed for user "nself"`),
		errors.New(`psql: error: FATAL:  database "unknown_db" does not exist`),
		errors.New(`psql: error: FATAL:  role "missing_role" does not exist`),
		nil,
	}
	for _, err := range cases {
		if isTransientPostgresError(err) {
			t.Errorf("isTransientPostgresError(%v) = true, want false", err)
		}
	}
}

// TestRetryTransientPG_SucceedsAfterTransientFailures verifies fn is retried
// while it returns a transient error, and the overall call succeeds once fn
// starts succeeding.
func TestRetryTransientPG_SucceedsAfterTransientFailures(t *testing.T) {
	calls := 0
	fn := func() error {
		calls++
		if calls < 3 {
			return errors.New("FATAL:  the database system is shutting down")
		}
		return nil
	}

	if err := retryTransientPGWithBudget(context.Background(), fn, 5*time.Second, time.Millisecond); err != nil {
		t.Fatalf("retryTransientPGWithBudget: unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

// TestRetryTransientPG_ReturnsNonTransientImmediately verifies a genuine
// error short-circuits the retry loop instead of waiting out the budget.
func TestRetryTransientPG_ReturnsNonTransientImmediately(t *testing.T) {
	calls := 0
	wantErr := errors.New(`FATAL:  password authentication failed for user "nself"`)
	fn := func() error {
		calls++
		return wantErr
	}

	err := retryTransientPGWithBudget(context.Background(), fn, 5*time.Second, time.Millisecond)
	if !errors.Is(err, wantErr) {
		t.Fatalf("retryTransientPGWithBudget error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (should not retry a non-transient error)", calls)
	}
}

// TestRetryTransientPG_ExhaustsBudgetAndReturnsLastError verifies that once
// the time budget runs out on a persistently transient error, the original
// error is returned unchanged rather than swallowed.
func TestRetryTransientPG_ExhaustsBudgetAndReturnsLastError(t *testing.T) {
	wantErr := errors.New("FATAL:  the database system is starting up")
	fn := func() error {
		return wantErr
	}

	err := retryTransientPGWithBudget(context.Background(), fn, 20*time.Millisecond, 5*time.Millisecond)
	if !errors.Is(err, wantErr) {
		t.Fatalf("retryTransientPGWithBudget error = %v, want %v", err, wantErr)
	}
}
