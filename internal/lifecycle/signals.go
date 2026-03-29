package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nself-org/cli/internal/ui"
)

// SignalMessage returns a human-readable message for a signal.
// SIGINT  → "Interrupted. Cleaning up..."
// SIGTERM → "Terminating. Cleaning up..."
// other   → "Signal received. Cleaning up..."
func SignalMessage(sig os.Signal) string {
	switch sig {
	case os.Interrupt:
		return "Interrupted. Cleaning up..."
	case syscall.SIGTERM:
		return "Terminating. Cleaning up..."
	default:
		return "Signal received. Cleaning up..."
	}
}

// TrapSignals registers SIGINT and SIGTERM handlers.
// On signal: prints a message via ui.Warn, calls cancel(), runs cleanupFn in
// a goroutine, waits up to timeout for it to complete, then calls exitFn with
// the conventional signal exit code (130 for SIGINT, 143 for SIGTERM).
// Pass os.Exit as exitFn in production; pass a stub in tests.
func TrapSignals(ctx context.Context, cancel context.CancelFunc, cleanupFn func(), timeout time.Duration, exitFn func(int)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigCh:
			ui.Warn(SignalMessage(sig))
			cancel()

			done := make(chan struct{})
			go func() {
				cleanupFn()
				close(done)
			}()

			select {
			case <-done:
				// cleanup finished in time
			case <-time.After(timeout):
				ui.Warn("Cleanup timed out. Forcing exit.")
			}

			if sig == syscall.SIGTERM {
				exitFn(143)
				return
			}
			exitFn(130)

		case <-ctx.Done():
			// context cancelled externally — stop listening
			signal.Stop(sigCh)
		}
	}()
}
