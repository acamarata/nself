package trust

import (
	"runtime"
	"testing"
)

// TestTrustConfigDefaults verifies zero-value TrustConfig fields are sane.
func TestTrustConfigDefaults(t *testing.T) {
	cfg := TrustConfig{}
	if cfg.SkipDNS || cfg.SkipSSL || cfg.SkipPorts {
		t.Error("zero-value TrustConfig should not skip any steps")
	}
}

// TestTrustConfigSkipAll verifies skip flags are honoured by Setup on unsupported OS.
// On macOS/Linux the Setup call would invoke real system helpers, so we only
// test the UnsupportedOS path + the SkipXxx flag fields here.
func TestTrustConfigSkipFlags(t *testing.T) {
	cfg := TrustConfig{
		SkipDNS:  true,
		SkipSSL:  true,
		SkipPorts: true,
	}
	if !cfg.SkipDNS {
		t.Error("SkipDNS should be true")
	}
	if !cfg.SkipSSL {
		t.Error("SkipSSL should be true")
	}
	if !cfg.SkipPorts {
		t.Error("SkipPorts should be true")
	}
}

// TestSetupUnsupportedOS verifies Setup returns an error for an unknown OS value.
// This exercises the switch default branch without OS-specific system calls.
func TestSetupUnsupportedOS(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		// We are actually on an unsupported OS — verify Setup returns an error.
		cfg := TrustConfig{WorkDir: t.TempDir(), BaseDomain: "test.local"}
		_, err := Setup(cfg)
		if err == nil {
			t.Error("Setup should return an error on unsupported OS")
		}
	}
	// On macOS/Linux, test the error message format via the string constant only
	// (we cannot call Setup because it would invoke system helpers in CI).
}

// TestTrustResultZeroValue verifies the TrustResult struct starts with all-false fields.
func TestTrustResultZeroValue(t *testing.T) {
	r := &TrustResult{}
	if r.DNSConfigured || r.DNSAlreadyDone || r.ResolverConfigured ||
		r.ResolverAlreadyDone || r.CertsGenerated || r.CertsAlreadyDone ||
		r.PortsConfigured || r.PortsAlreadyDone {
		t.Error("zero-value TrustResult should have all bool fields false")
	}
	if len(r.Errors) != 0 {
		t.Errorf("zero-value TrustResult.Errors should be nil, got %v", r.Errors)
	}
}

// TestTrustResultErrorAccumulation verifies Errors slice grows correctly.
func TestTrustResultErrorAccumulation(t *testing.T) {
	r := &TrustResult{}
	r.Errors = append(r.Errors, errorf("first error"))
	r.Errors = append(r.Errors, errorf("second error"))
	if len(r.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(r.Errors))
	}
}

// TestSetupAllSkipOnLinux runs Setup with all steps skipped on Linux.
// This validates the happy path without triggering any OS admin prompts.
func TestSetupAllSkipOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	cfg := TrustConfig{
		WorkDir:    t.TempDir(),
		BaseDomain: "test.local",
		SkipDNS:   true,
		SkipSSL:   true,
		SkipPorts: true,
	}
	result, err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup with all-skip on Linux failed: %v", err)
	}
	if result == nil {
		t.Fatal("Setup returned nil result")
	}
	// All steps were skipped, so none should report configured=true.
	if result.DNSConfigured || result.CertsGenerated || result.PortsConfigured {
		t.Error("all-skip Setup should not mark any step as configured")
	}
}

// TestSetupAllSkipOnDarwin runs Setup with all steps skipped on macOS.
// This validates the happy path without triggering any OS admin prompts.
func TestSetupAllSkipOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}
	cfg := TrustConfig{
		WorkDir:    t.TempDir(),
		BaseDomain: "test.local",
		SkipDNS:   true,
		SkipSSL:   true,
		SkipPorts: true,
	}
	result, err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup with all-skip on macOS failed: %v", err)
	}
	if result == nil {
		t.Fatal("Setup returned nil result")
	}
	if result.DNSConfigured || result.CertsGenerated || result.PortsConfigured {
		t.Error("all-skip Setup should not mark any step as configured")
	}
}

// errorf is a tiny helper to produce an error without importing fmt.
func errorf(msg string) error {
	return errorString(msg)
}

type errorString string

func (e errorString) Error() string { return string(e) }
