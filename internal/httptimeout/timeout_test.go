package httptimeout

import (
	"testing"
	"time"
)

// Note: Default/Installer/Health/License/Plugin/Backup are package-init vars,
// so env overrides on those package-level clients are only honored at fresh
// process init. We test envDuration directly to cover the override logic
// in isolation. The package-level vars are constructed via the same helper,
// so the parsing semantics are identical.

func TestEnvDurationOverride(t *testing.T) {
	t.Setenv("NSELF_HTTP_TIMEOUT_TEST", "5")
	got := envDuration("NSELF_HTTP_TIMEOUT_TEST", 30*time.Second)
	want := 5 * time.Second
	if got != want {
		t.Fatalf("envDuration override: got %v, want %v", got, want)
	}
}

func TestEnvDurationInvalidFallback(t *testing.T) {
	t.Setenv("NSELF_HTTP_TIMEOUT_TEST", "abc")
	got := envDuration("NSELF_HTTP_TIMEOUT_TEST", 30*time.Second)
	want := 30 * time.Second
	if got != want {
		t.Fatalf("envDuration invalid: got %v, want %v", got, want)
	}
}

func TestEnvDurationZeroFallback(t *testing.T) {
	t.Setenv("NSELF_HTTP_TIMEOUT_TEST", "0")
	got := envDuration("NSELF_HTTP_TIMEOUT_TEST", 30*time.Second)
	want := 30 * time.Second
	if got != want {
		t.Fatalf("envDuration zero: got %v, want %v", got, want)
	}
}

func TestEnvDurationEmptyFallback(t *testing.T) {
	// Make sure the env var is not set in this test process.
	t.Setenv("NSELF_HTTP_TIMEOUT_TEST_UNSET", "")
	got := envDuration("NSELF_HTTP_TIMEOUT_TEST_UNSET", 30*time.Second)
	want := 30 * time.Second
	if got != want {
		t.Fatalf("envDuration empty: got %v, want %v", got, want)
	}
}

func TestEnvDurationNegativeFallback(t *testing.T) {
	t.Setenv("NSELF_HTTP_TIMEOUT_TEST", "-5")
	got := envDuration("NSELF_HTTP_TIMEOUT_TEST", 30*time.Second)
	want := 30 * time.Second
	if got != want {
		t.Fatalf("envDuration negative: got %v, want %v", got, want)
	}
}

func TestWithTimeoutFactory(t *testing.T) {
	c := WithTimeout(5 * time.Second)
	if c == nil {
		t.Fatal("WithTimeout returned nil")
	}
	if c.Timeout != 5*time.Second {
		t.Fatalf("WithTimeout: got %v, want %v", c.Timeout, 5*time.Second)
	}
}

func TestPackageVarsHaveTimeouts(t *testing.T) {
	// Sanity: every exported scoped client must have a positive Timeout.
	clients := map[string]*struct{ Timeout time.Duration }{
		"Default":   {Default.Timeout},
		"Installer": {Installer.Timeout},
		"Health":    {Health.Timeout},
		"License":   {License.Timeout},
		"Plugin":    {Plugin.Timeout},
		"Backup":    {Backup.Timeout},
	}
	for name, c := range clients {
		if c.Timeout <= 0 {
			t.Errorf("%s.Timeout = %v, want > 0", name, c.Timeout)
		}
	}
}
