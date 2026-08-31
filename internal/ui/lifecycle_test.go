package ui

import (
	"runtime"
	"testing"
	"time"
)

// TestSpinner_NoGoroutineLeak verifies that creating a Spinner, calling Start
// and Stop, does not permanently increase the goroutine count.
//
// In non-TTY environments (CI), Spinner.Start() takes the non-TTY path and
// prints a static message without starting a goroutine. The test therefore
// validates zero delta in that path.
//
// In a TTY environment the goroutine is expected to start and then be cleaned
// up by Stop(). We allow up to 200ms for cleanup and assert delta ≤ 0 after
// that window.
func TestSpinner_NoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	s := NewSpinner("testing spinner lifecycle")
	s.Start()
	// Brief settle period.
	time.Sleep(20 * time.Millisecond)
	s.Stop()
	// Allow goroutine to exit.
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()
	delta := after - before
	if delta > 0 {
		t.Errorf("Spinner goroutine leak: %d goroutines before, %d after, delta +%d", before, after, delta)
	}
}

// TestSpinner_DoubleStart verifies that calling Start twice does not start a
// second goroutine or panic (idempotent guard).
func TestSpinner_DoubleStart(t *testing.T) {
	before := runtime.NumGoroutine()

	s := NewSpinner("double start test")
	s.Start()
	s.Start() // second call must be a no-op
	time.Sleep(20 * time.Millisecond)
	s.Stop()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()
	delta := after - before
	if delta > 0 {
		t.Errorf("double Start leaked goroutines: delta +%d", delta)
	}
}

// TestSpinner_DoubleStop verifies that calling Stop twice does not panic
// (closing a closed channel is a panic in Go without the running guard).
func TestSpinner_DoubleStop(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("double Stop panicked: %v", r)
		}
	}()

	s := NewSpinner("double stop test")
	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Stop()
	s.Stop() // must not panic
}

// TestSpinner_SuccessDoesNotLeak verifies Spinner.Success stops the goroutine.
func TestSpinner_SuccessDoesNotLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	s := NewSpinner("success path")
	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Success("done")
	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after-before > 0 {
		t.Errorf("Spinner.Success goroutine leak: delta +%d", after-before)
	}
}

// TestSpinner_FailDoesNotLeak verifies Spinner.Fail stops the goroutine.
func TestSpinner_FailDoesNotLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	s := NewSpinner("fail path")
	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Fail("error occurred")
	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after-before > 0 {
		t.Errorf("Spinner.Fail goroutine leak: delta +%d", after-before)
	}
}

// TestFirstRunProgress_NoGoroutineLeak verifies that FirstRunProgress in quiet
// mode (non-TTY, no goroutine) does not leak.
func TestFirstRunProgress_NoGoroutineLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	done := FirstRunProgress(true /* quiet */)
	done()
	time.Sleep(20 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after-before > 0 {
		t.Errorf("FirstRunProgress(quiet) leaked goroutines: delta +%d", after-before)
	}
}

// TestSpinner_StopWithoutStart verifies that calling Stop on a never-started
// Spinner is a no-op and does not panic.
func TestSpinner_StopWithoutStart(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Stop on never-started spinner panicked: %v", r)
		}
	}()
	s := NewSpinner("never started")
	s.Stop() // must be safe
}
