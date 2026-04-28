// Package trust — ports_linux_test.go: Linux-targeted tests for ports_linux.go.
//
// Targets:
//   - CheckPortsLinux: returns false when iptables rules are absent / iptables
//     is unavailable / running as non-root (iptables -C requires CAP_NET_ADMIN)
//   - SetupPortsLinux: returns the documented non-root error message including
//     fallback manual instructions
//   - iptablesTimeout sanity
//
// Pure-Go branches only — no real iptables modifications, no sudo, no shell.
package trust

import (
	"os"
	"strings"
	"testing"
)

// TestCheckPortsLinux_NonRoot exercises CheckPortsLinux. Without root,
// `iptables -C` exits non-zero, so the function must return false.
func TestCheckPortsLinux_NonRoot(t *testing.T) {
	requireLinux(t)

	if os.Geteuid() == 0 {
		t.Skip("test runs as root — non-root branch unreachable")
	}

	cfg := TrustConfig{
		NginxSSLPort:  8443,
		NginxHTTPPort: 8080,
	}

	// Either iptables is missing (LookPath fails inside exec.Command), or
	// permission is denied. Both cases produce a non-zero exit and result false.
	got := CheckPortsLinux(cfg)
	if got {
		t.Errorf("CheckPortsLinux should return false without root iptables access; got true")
	}
}

// TestSetupPortsLinux_NonRoot verifies the non-root error path returns the
// documented manual-rule fallback message.
func TestSetupPortsLinux_NonRoot(t *testing.T) {
	requireLinux(t)

	if os.Geteuid() == 0 {
		t.Skip("test runs as root — non-root branch unreachable")
	}

	cfg := TrustConfig{
		NginxSSLPort:  8443,
		NginxHTTPPort: 8080,
	}

	// CheckPortsLinux must return false for SetupPortsLinux to fall through
	// to the non-root error branch. As a non-root user, iptables -C will
	// fail, so this is reliable.
	if CheckPortsLinux(cfg) {
		t.Skip("ports already configured — non-root branch unreachable")
	}

	alreadyDone, err := SetupPortsLinux(cfg)
	if alreadyDone {
		t.Errorf("expected alreadyDone=false on non-root path, got true")
	}
	if err == nil {
		t.Fatal("expected non-root error, got nil")
	}

	msg := err.Error()
	for _, want := range []string{
		"root privileges required",
		"iptables",
		"--dport 443",
		"--dport 80",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message should contain %q; got: %s", want, msg)
		}
	}
}

// TestSetupPortsLinux_AlreadyDone exercises the early-return path. Skips
// when ports aren't already configured (the typical CI case).
func TestSetupPortsLinux_AlreadyDone(t *testing.T) {
	requireLinux(t)

	cfg := TrustConfig{
		NginxSSLPort:  8443,
		NginxHTTPPort: 8080,
	}

	if !CheckPortsLinux(cfg) {
		t.Skip("ports not pre-configured — alreadyDone branch unreachable")
	}

	alreadyDone, err := SetupPortsLinux(cfg)
	if !alreadyDone {
		t.Errorf("expected alreadyDone=true when CheckPortsLinux returns true, got false")
	}
	if err != nil {
		t.Errorf("expected nil err on alreadyDone path, got: %v", err)
	}
}

// TestIPTablesTimeoutSanity confirms the timeout constant is positive.
// Guards against accidental clearing of the constant, which would cause
// every iptables call to time out instantly.
func TestIPTablesTimeoutSanity(t *testing.T) {
	requireLinux(t)

	if iptablesTimeout <= 0 {
		t.Errorf("iptablesTimeout must be positive, got %v", iptablesTimeout)
	}
}
