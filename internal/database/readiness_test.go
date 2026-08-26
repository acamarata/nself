package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fastReadinessParams keeps the streak tests fast (milliseconds, not the
// production 60s/1s) while still exercising the real loop in
// waitForPostgresReady.
const (
	fastMaxWait  = 200 * time.Millisecond
	fastInterval = time.Millisecond
)

// TestWaitForPostgresReady_SingleSuccessNotEnough verifies that a probe
// which only succeeds once is NOT reported ready — this is the exact defect
// this fix closes: the postgres image's temporary init-time server answers
// one probe successfully before shutting down, and a first-success readiness
// check returns right into that shutdown (run 32948887818).
func TestWaitForPostgresReady_SingleSuccessNotEnough(t *testing.T) {
	calls := 0
	probe := func(_ context.Context) error {
		calls++
		if calls == 1 {
			// One success: the temporary server, then it goes away.
			return nil
		}
		return errors.New("FATAL:  the database system is shutting down")
	}

	err := waitForPostgresReady(context.Background(), probe, fastMaxWait, fastInterval)
	if !errors.Is(err, errPostgresNotReady) {
		t.Fatalf("waitForPostgresReady = %v, want errPostgresNotReady (single success must not count as ready)", err)
	}
	if calls < 2 {
		t.Errorf("calls = %d, want probe to keep being called after the lone success", calls)
	}
}

// TestWaitForPostgresReady_StreakResetsOnFailure verifies a failure
// occurring mid-streak resets the counter to zero — i.e. success, success,
// FAILURE, success, success, success is required to reach the streak (5
// successes total), not just 3.
func TestWaitForPostgresReady_StreakResetsOnFailure(t *testing.T) {
	var calls int
	results := []error{
		nil, // streak 1
		nil, // streak 2
		errors.New("FATAL:  the database system is shutting down"), // reset to 0
		nil, // streak 1
		nil, // streak 2
		nil, // streak 3 -> ready
	}
	probe := func(_ context.Context) error {
		idx := calls
		calls++
		if idx >= len(results) {
			return errors.New("unexpected extra call")
		}
		return results[idx]
	}

	if err := waitForPostgresReady(context.Background(), probe, fastMaxWait, fastInterval); err != nil {
		t.Fatalf("waitForPostgresReady: unexpected error: %v", err)
	}
	if calls != len(results) {
		t.Errorf("calls = %d, want %d (streak must restart after the failure)", calls, len(results))
	}
}

// TestWaitForPostgresReady_AlwaysReadySucceedsQuickly verifies the happy
// path: a probe that always succeeds reaches the streak in exactly
// postgresReadinessStreak calls.
func TestWaitForPostgresReady_AlwaysReadySucceedsQuickly(t *testing.T) {
	calls := 0
	probe := func(_ context.Context) error {
		calls++
		return nil
	}

	if err := waitForPostgresReady(context.Background(), probe, fastMaxWait, fastInterval); err != nil {
		t.Fatalf("waitForPostgresReady: unexpected error: %v", err)
	}
	if calls != postgresReadinessStreak {
		t.Errorf("calls = %d, want %d", calls, postgresReadinessStreak)
	}
}

// TestWaitForPostgresReady_NeverReadyTimesOut verifies a probe that never
// succeeds returns errPostgresNotReady once maxWait elapses, rather than
// blocking forever.
func TestWaitForPostgresReady_NeverReadyTimesOut(t *testing.T) {
	probe := func(_ context.Context) error {
		return errors.New("FATAL:  the database system is starting up")
	}

	err := waitForPostgresReady(context.Background(), probe, fastMaxWait, fastInterval)
	if !errors.Is(err, errPostgresNotReady) {
		t.Fatalf("waitForPostgresReady = %v, want errPostgresNotReady", err)
	}
}

// TestWaitForPostgresReady_ContextCancellation verifies an already-canceled
// context short-circuits the loop and returns ctx.Err(), not
// errPostgresNotReady.
func TestWaitForPostgresReady_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	probe := func(_ context.Context) error {
		t.Fatal("probe should not be called with an already-canceled context")
		return nil
	}

	err := waitForPostgresReady(ctx, probe, fastMaxWait, fastInterval)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForPostgresReady = %v, want context.Canceled", err)
	}
}
