package database

import (
	"context"
	"strings"
	"time"
)

// transientPostgresMessages lists substrings (matched case-insensitively)
// that the official postgres image returns while the server is coming up or
// going down, never as a genuine failure. They show up in two situations we
// care about: the ordinary boot sequence, and the narrower first-run race
// where the entrypoint starts a temporary local server to run initdb and
// initdb.d scripts, shuts it down, then starts the real server. A caller
// that lands a query in that shutdown window sees "the database system is
// shutting down" even though nothing is actually wrong.
var transientPostgresMessages = []string{
	"the database system is shutting down",
	"the database system is starting up",
	"the database system is not yet accepting connections",
}

// isTransientPostgresError reports whether err represents one of the
// transient startup/shutdown conditions above, as opposed to a genuine
// failure such as bad credentials, a missing role, or an invalid connection
// string. Those must never be retried or their real message would be
// hidden behind a timeout.
func isTransientPostgresError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range transientPostgresMessages {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// retryTransientPGBudget and retryTransientPGInterval bound retryTransientPG.
// Same 60s budget as waitForPostgresReady's timeout, since both are covering
// the same startup window from different angles.
const (
	retryTransientPGBudget   = 60 * time.Second
	retryTransientPGInterval = 1 * time.Second
)

// retryTransientPG runs fn, retrying only while it returns a transient
// postgres startup/shutdown error (isTransientPostgresError), up to a fixed
// time budget. A non-transient error is returned immediately without
// retrying — the caller sees the real failure right away instead of waiting
// out the budget. If the budget is exhausted while still seeing transient
// errors, the last error is returned unchanged so a genuine outage still
// surfaces with its real message.
//
// This exists because readiness alone (waitForPostgresReady) is not enough:
// a slow enough shutdown of the postgres image's temporary init server can
// still land a later statement — the "check database existence" query that
// runs right after init — in the same window. Both sides of the race need
// covering.
func retryTransientPG(ctx context.Context, fn func() error) error {
	return retryTransientPGWithBudget(ctx, fn, retryTransientPGBudget, retryTransientPGInterval)
}

// retryTransientPGWithBudget is retryTransientPG with an injectable budget
// and poll interval, so tests can exercise the retry/give-up logic without
// waiting out the real 60s production budget.
func retryTransientPGWithBudget(ctx context.Context, fn func() error, budget, interval time.Duration) error {
	deadline := time.Now().Add(budget)
	var lastErr error

	for {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isTransientPostgresError(lastErr) {
			return lastErr
		}
		if !time.Now().Before(deadline) {
			return lastErr
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
