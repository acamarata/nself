package plugin

import (
	"os"
	"testing"
	"time"
)

// TestGracePeriod_Default verifies that gracePeriod returns 30s when the
// DOCKER_STOP_GRACE_PERIOD env var is unset or empty.
func TestGracePeriod_Default(t *testing.T) {
	t.Setenv("DOCKER_STOP_GRACE_PERIOD", "")
	got := gracePeriod()
	if got != 30*time.Second {
		t.Errorf("default grace period: got %v, want 30s", got)
	}
}

// TestGracePeriod_EnvVar verifies that gracePeriod respects a valid
// DOCKER_STOP_GRACE_PERIOD env var value.
func TestGracePeriod_EnvVar(t *testing.T) {
	t.Setenv("DOCKER_STOP_GRACE_PERIOD", "10s")
	got := gracePeriod()
	if got != 10*time.Second {
		t.Errorf("env var 10s: got %v, want 10s", got)
	}
}

// TestGracePeriod_EnvVarMinutes verifies that minute-based durations are parsed.
func TestGracePeriod_EnvVarMinutes(t *testing.T) {
	t.Setenv("DOCKER_STOP_GRACE_PERIOD", "2m")
	got := gracePeriod()
	// 2m = 120s which equals maxGrace, so it should be clamped to 120s.
	if got != 120*time.Second {
		t.Errorf("env var 2m: got %v, want 120s (max)", got)
	}
}

// TestGracePeriod_Invalid verifies that an invalid env var value falls back
// to the 30s default rather than panicking or returning zero.
func TestGracePeriod_Invalid(t *testing.T) {
	t.Setenv("DOCKER_STOP_GRACE_PERIOD", "not-a-duration")
	got := gracePeriod()
	if got != 30*time.Second {
		t.Errorf("invalid env var: got %v, want 30s fallback", got)
	}
}

// TestGracePeriod_ClampMax verifies that values above 120s are clamped to 120s.
func TestGracePeriod_ClampMax(t *testing.T) {
	t.Setenv("DOCKER_STOP_GRACE_PERIOD", "300s")
	got := gracePeriod()
	if got != 120*time.Second {
		t.Errorf("clamp max: got %v, want 120s", got)
	}
}

// TestGracePeriod_ClampMin verifies that values below 1s are clamped to 1s.
func TestGracePeriod_ClampMin(t *testing.T) {
	t.Setenv("DOCKER_STOP_GRACE_PERIOD", "0s")
	got := gracePeriod()
	if got != 1*time.Second {
		t.Errorf("clamp min: got %v, want 1s", got)
	}
}

// TestGracePeriod_Unset verifies behavior when env var key is not present at all.
func TestGracePeriod_Unset(t *testing.T) {
	// Ensure the env var is completely absent.
	_ = os.Unsetenv("DOCKER_STOP_GRACE_PERIOD")
	got := gracePeriod()
	if got != 30*time.Second {
		t.Errorf("unset env var: got %v, want 30s", got)
	}
}
