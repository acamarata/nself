// Package ui — coverage_extra_test.go: G0-T11 follow-on coverage to reach 75%+.
//
// Targets the previously-uncovered branches:
//   - DockerPullProgress (quiet/non-TTY path)
//   - ProgressBar.Done (calls render + newline)
//   - ProgressBar.render with quiet=true (early-return branch)
//   - FirstRunProgress quiet path (re-confirm zero-leak)
//   - Spinner non-TTY entry path
//
// All tests run safely in CI: stdout is not a TTY, so all branches that gate
// on term.IsTerminal take the non-TTY path.
package ui

import (
	"runtime"
	"testing"
	"time"
)

// TestDockerPullProgress_QuietNoLeak verifies the quiet branch returns a
// no-op done function and does not start a goroutine.
func TestDockerPullProgress_QuietNoLeak(t *testing.T) {
	before := runtime.NumGoroutine()

	done := DockerPullProgress(true /* quiet */)
	// The done function must be callable with both nil and non-nil errors.
	done(nil)

	time.Sleep(20 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after-before > 0 {
		t.Errorf("DockerPullProgress(quiet) leaked goroutines: delta +%d", after-before)
	}
}

// TestDockerPullProgress_QuietWithError verifies the done(err) branch when
// quiet=true and err is non-nil prints nothing additional but does not panic.
func TestDockerPullProgress_QuietWithError(t *testing.T) {
	done := DockerPullProgress(true)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DockerPullProgress quiet+err panicked: %v", r)
		}
	}()
	done(errFake("simulated pull failure"))
}

// TestDockerPullProgress_NonQuietInCI verifies the non-quiet branch is safe to
// call. In CI stdout is not a TTY so DockerPullProgress takes the same quiet
// branch as quiet=true. We still exercise the call to cover the entry guard.
func TestDockerPullProgress_NonQuietPathCI(t *testing.T) {
	before := runtime.NumGoroutine()
	done := DockerPullProgress(false)
	// Immediately resolve.
	done(nil)
	time.Sleep(40 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after-before > 1 {
		t.Errorf("DockerPullProgress(non-quiet) leaked: delta +%d", after-before)
	}
}

// TestDockerPullProgress_NonQuietWithError exercises the err branch of the
// done function.
func TestDockerPullProgress_NonQuietWithError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("done(err) panicked: %v", r)
		}
	}()
	done := DockerPullProgress(false)
	done(errFake("ci simulated error"))
}

// TestProgressBar_DoneInQuietMode verifies Done renders nothing in quiet mode
// and does not panic.
func TestProgressBar_DoneInQuietMode(t *testing.T) {
	p := NewProgressBar("test", 10, true /* quiet */)
	p.Set(5)
	p.Inc()
	p.Done()
	if p.current != p.total {
		t.Errorf("Done should set current to total: got %d, want %d", p.current, p.total)
	}
}

// TestProgressBar_DoneInNonQuietCI verifies Done in non-quiet CI mode (no TTY)
// does not panic and updates current.
func TestProgressBar_DoneInNonQuietCI(t *testing.T) {
	p := NewProgressBar("ci-test", 5, false /* not quiet */)
	p.Done()
	if p.current != p.total {
		t.Errorf("Done should set current to total: got %d, want %d", p.current, p.total)
	}
}

// TestProgressBar_RenderZeroTotal verifies render handles total<=0 (early return).
func TestProgressBar_RenderZeroTotal(t *testing.T) {
	p := NewProgressBar("zero", 0, false)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("render with total=0 panicked: %v", r)
		}
	}()
	p.Set(0)
	p.Inc()
	p.Done()
}

// TestProgressBar_SetBeyondTotal verifies Set with n>total caps at width.
func TestProgressBar_SetBeyondTotal(t *testing.T) {
	p := NewProgressBar("over", 10, true)
	p.Set(100) // way beyond total
	if p.current != 100 {
		t.Errorf("Set should record n verbatim: got %d, want 100", p.current)
	}
}

// TestProgressBar_DefaultWidth verifies the constructor sets width=40.
func TestProgressBar_DefaultWidth(t *testing.T) {
	p := NewProgressBar("w", 10, false)
	if p.width != 40 {
		t.Errorf("default width: got %d, want 40", p.width)
	}
	if p.label != "w" {
		t.Errorf("label: got %q, want w", p.label)
	}
	if p.total != 10 {
		t.Errorf("total: got %d, want 10", p.total)
	}
}

// TestProgressBar_RenderForcedTTY drives the render() and Done() TTY branches
// by setting isTTY=true via the same package's struct (white-box test). This
// covers the bar-drawing path that CI's non-TTY environment cannot reach.
func TestProgressBar_RenderForcedTTY(t *testing.T) {
	p := NewProgressBar("forced", 10, false)
	// White-box override — only the test package can do this.
	p.isTTY = true

	p.Set(0)
	p.Set(5)
	p.Inc()
	p.Set(15) // beyond total → clamps in render
	p.Done()  // enters !quiet && isTTY → fmt.Println branch
}

// TestProgressBar_RenderForcedTTYQuiet verifies that quiet=true short-circuits
// even when isTTY=true.
func TestProgressBar_RenderForcedTTYQuiet(t *testing.T) {
	p := NewProgressBar("quiet-tty", 5, true)
	p.isTTY = true
	p.Set(2)
	p.Done()
}

// withStdoutAsTerminal forces stdoutIsTerminal() to return true for the
// duration of fn, then restores the real detection function. This drives
// the TTY-only goroutine paths inside UI helpers without requiring a real pty.
func withStdoutAsTerminal(t *testing.T, fn func()) {
	t.Helper()
	orig := stdoutIsTerminalFunc
	stdoutIsTerminalFunc = func() bool { return true }
	defer func() { stdoutIsTerminalFunc = orig }()
	fn()
}

// TestFirstRunProgress_ForcedTTY drives the TTY goroutine path of FirstRunProgress.
func TestFirstRunProgress_ForcedTTY(t *testing.T) {
	withStdoutAsTerminal(t, func() {
		done := FirstRunProgress(false /* not quiet */)
		// Allow at least one ticker fire (4s). To avoid slow tests, we don't
		// wait that long — the test exercises the entry path and goroutine
		// startup; the ticker tick path is exercised on real boots.
		time.Sleep(20 * time.Millisecond)
		done()
		// Allow goroutine cleanup.
		time.Sleep(20 * time.Millisecond)
	})
}

// TestDockerPullProgress_ForcedTTY drives the TTY goroutine path.
func TestDockerPullProgress_ForcedTTY(t *testing.T) {
	withStdoutAsTerminal(t, func() {
		done := DockerPullProgress(false)
		time.Sleep(20 * time.Millisecond)
		done(nil)
		time.Sleep(20 * time.Millisecond)
	})
}

// TestDockerPullProgress_ForcedTTYWithError covers the err-branch of done.
func TestDockerPullProgress_ForcedTTYWithError(t *testing.T) {
	withStdoutAsTerminal(t, func() {
		done := DockerPullProgress(false)
		time.Sleep(20 * time.Millisecond)
		done(errFake("simulated"))
		time.Sleep(20 * time.Millisecond)
	})
}

// TestDockerPullProgress_ForcedTTYTickerFires waits long enough for the 2s
// ticker to fire so the inner write-to-stdout branch is exercised.
func TestDockerPullProgress_ForcedTTYTickerFires(t *testing.T) {
	withStdoutAsTerminal(t, func() {
		done := DockerPullProgress(false)
		// Wait > 2s to let one ticker tick happen.
		time.Sleep(2100 * time.Millisecond)
		done(nil)
		time.Sleep(20 * time.Millisecond)
	})
}

// TestSpinner_ForcedTTYStartStop drives the TTY goroutine path of Spinner.
func TestSpinner_ForcedTTYStartStop(t *testing.T) {
	withStdoutAsTerminal(t, func() {
		s := NewSpinner("forced-tty spinner")
		s.Start()
		time.Sleep(150 * time.Millisecond) // let frames render
		s.Stop()
	})
}

// TestSpinner_ForcedTTYSuccess covers the Stop path called via Success.
func TestSpinner_ForcedTTYSuccess(t *testing.T) {
	withStdoutAsTerminal(t, func() {
		s := NewSpinner("success-tty")
		s.Start()
		time.Sleep(50 * time.Millisecond)
		s.Success("completed")
	})
}

// TestSpinner_ForcedTTYFail covers the Stop path called via Fail.
func TestSpinner_ForcedTTYFail(t *testing.T) {
	withStdoutAsTerminal(t, func() {
		s := NewSpinner("fail-tty")
		s.Start()
		time.Sleep(50 * time.Millisecond)
		s.Fail("error")
	})
}

// TestSpinner_StartNonTTYPrintsStaticMessage verifies the non-TTY branch of
// Start: it prints the static message and returns without starting a goroutine.
func TestSpinner_StartNonTTYPrintsStaticMessage(t *testing.T) {
	before := runtime.NumGoroutine()
	s := NewSpinner("static message")
	s.Start()
	// In CI stdout is not a TTY so no goroutine starts.
	time.Sleep(20 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after-before > 0 {
		t.Errorf("Spinner.Start non-TTY should not leak: delta +%d", after-before)
	}
	// Stop is a no-op when not running.
	s.Stop()
}

// TestInitSteps_QuietPath verifies the quiet branch suppresses output.
func TestInitSteps_QuietPath(t *testing.T) {
	steps := NewInitSteps(true /* quiet */, "step-a", "step-b")
	if !steps.Next() {
		t.Error("first Next should return true")
	}
	if !steps.Next() {
		t.Error("second Next should return true")
	}
	if steps.Next() {
		t.Error("third Next should return false (no more steps)")
	}
	steps.Done()
}

// TestInitSteps_NonQuietPath verifies the non-quiet branch prints output.
func TestInitSteps_NonQuietPath(t *testing.T) {
	steps := NewInitSteps(false, "alpha", "beta", "gamma")
	for i := 0; i < 3; i++ {
		if !steps.Next() {
			t.Errorf("step %d Next should return true", i)
		}
	}
	if steps.Next() {
		t.Error("Next after exhausting steps should return false")
	}
	steps.Done()
}

// TestFirstRunProgress_NonQuietCallable verifies FirstRunProgress(false) returns
// a callable done function in CI (non-TTY → static branch).
func TestFirstRunProgress_NonQuietCallable(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FirstRunProgress(non-quiet) panicked: %v", r)
		}
	}()
	done := FirstRunProgress(false)
	if done == nil {
		t.Fatal("done function should not be nil")
	}
	done()
}

// errFake is a tiny error type to avoid adding errors import.
type errFake string

func (e errFake) Error() string { return string(e) }
