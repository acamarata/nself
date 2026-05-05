package recover_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	safeRecover "github.com/nself-org/cli/internal/recover"
)

// bufHandler is a slog.Handler that writes records into a buffer for inspection.
type bufHandler struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (h *bufHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *bufHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.buf.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		h.buf.WriteString(" " + a.Key + "=" + a.Value.String())
		return true
	})
	h.buf.WriteString("\n")
	return nil
}
func (h *bufHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *bufHandler) WithGroup(_ string) slog.Handler      { return h }

// TestSafeGo_PanicRecovery verifies that a panicking goroutine does not crash
// the process and that a structured "goroutine_panic" error is emitted via slog.
//
// Timing note: fn's own defers run during panic unwinding, before SafeGo's
// outer recover() fires. We cannot close a channel from fn and use it to gate
// the log-assertion (the channel close happens before the log is written).
// A fixed 500 ms window is well within any CI runner's capability.
func TestSafeGo_PanicRecovery(t *testing.T) {
	h := &bufHandler{}
	old := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(old) })

	safeRecover.SafeGo("test-goroutine", func() {
		panic("test panic value")
	})

	// Allow the goroutine's recover + slog.Error to complete.
	time.Sleep(500 * time.Millisecond)

	h.mu.Lock()
	output := h.buf.String()
	h.mu.Unlock()

	if !strings.Contains(output, "goroutine_panic") {
		t.Errorf("expected slog message 'goroutine_panic', got: %q", output)
	}
	if !strings.Contains(output, "test-goroutine") {
		t.Errorf("expected goroutine name 'test-goroutine' in log, got: %q", output)
	}
	if !strings.Contains(output, "test panic value") {
		t.Errorf("expected panic value 'test panic value' in log, got: %q", output)
	}
	// Stack trace from runtime/debug.Stack() always starts with "goroutine ".
	if !strings.Contains(output, "goroutine ") {
		t.Errorf("expected stack trace in log output, got: %q", output)
	}
}

// TestSafeGo_NoPanicRunsNormally verifies that non-panicking goroutines
// complete without emitting any recovery log event.
func TestSafeGo_NoPanicRunsNormally(t *testing.T) {
	h := &bufHandler{}
	old := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(old) })

	done := make(chan struct{})
	safeRecover.SafeGo("clean-goroutine", func() {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not complete within timeout")
	}

	h.mu.Lock()
	output := h.buf.String()
	h.mu.Unlock()

	if strings.Contains(output, "goroutine_panic") {
		t.Errorf("unexpected goroutine_panic log for clean goroutine: %q", output)
	}
}
