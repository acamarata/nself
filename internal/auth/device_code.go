// Package auth — device code polling loop for the CLI login flow.
package auth

import (
	"context"
	"fmt"
	"time"
)

const (
	// defaultPollInterval is the polling interval between token requests.
	defaultPollInterval = 5 * time.Second
	// defaultPollTimeout is the maximum time to wait for user authorization.
	defaultPollTimeout = 5 * time.Minute
)

// PollResult holds the outcome of a completed device code poll.
type PollResult struct {
	Token *TokenResponse
}

// PollDeviceCode polls the auth server until the user authorizes the device code,
// the timeout expires, or the context is cancelled.
//
// onPoll is called before each poll attempt (use for progress indicators).
// Pass nil to skip the callback.
//
// Returns ErrPollTimeout if the poll window expires without user action.
// Returns context.Canceled/context.DeadlineExceeded if ctx is cancelled.
func PollDeviceCode(ctx context.Context, deviceCode string, onPoll func(elapsed time.Duration)) (*PollResult, error) {
	deadline := time.Now().Add(defaultPollTimeout)
	start := time.Now()

	for {
		// Check context cancellation first.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return nil, ErrPollTimeout
		}

		if onPoll != nil {
			onPoll(time.Since(start))
		}

		token, err := PollToken(deviceCode)
		if err != nil {
			return nil, fmt.Errorf("polling for authorization: %w", err)
		}

		// nil token means authorization_pending — user hasn't clicked Authorize yet.
		if token != nil {
			return &PollResult{Token: token}, nil
		}

		// Wait for the next poll interval, respecting context cancellation.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(defaultPollInterval):
		}
	}
}

// ErrPollTimeout is returned when the device code polling window expires.
var ErrPollTimeout = fmt.Errorf("authorization timed out — the login URL has expired. Run 'nself login' to try again")
