// Package trust — dns_linux_test.go: Linux-targeted tests for dns_linux.go.
//
// Targets pure-Go paths that do not require root, network, or live system
// modification:
//   - detectLinuxResolver branch detection via PATH + /run inspection (read-only)
//   - CheckDNSLinux state inspection (read-only file stats)
//   - SetupDNSLinux non-root error formatting (returns before any privileged call)
//
// All real DNS modification paths (writing /etc/systemd/..., restarting
// services) are gated behind os.Geteuid() == 0, so the non-root assertion
// is the safe ceiling for unit tests in CI.
package trust

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// requireLinux skips the test on non-Linux hosts.
func requireLinux(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
}

// TestDetectLinuxResolver_ReturnsKnownEnum verifies detectLinuxResolver
// returns one of the four documented enum values without panicking. The
// function inspects /run/systemd and PATH; either branch outcome is acceptable
// because CI environments vary.
func TestDetectLinuxResolver_ReturnsKnownEnum(t *testing.T) {
	requireLinux(t)

	got := detectLinuxResolver()
	switch got {
	case resolverUnknown, resolverSystemd, resolverNetworkManager, resolverDnsmasqRaw:
		// All valid.
	default:
		t.Errorf("detectLinuxResolver returned unknown enum value: %d", int(got))
	}
}

// TestCheckDNSLinux_NoPanic verifies CheckDNSLinux executes without panic
// and returns a bool. On a CI runner the result is almost always false
// (no nself.conf installed), but we accept either outcome — the goal is
// covering the read-only state inspection path.
func TestCheckDNSLinux_NoPanic(t *testing.T) {
	requireLinux(t)

	// Just exercise the function; either result is acceptable in CI.
	_ = CheckDNSLinux()
}

// TestSetupDNSLinux_NonRoot returns the documented error message when invoked
// without root. CI runs as a non-root user, so this path is reliably reached.
// We skip if the test happens to run as root.
func TestSetupDNSLinux_NonRoot(t *testing.T) {
	requireLinux(t)

	if os.Geteuid() == 0 {
		t.Skip("test runs as root — non-root branch unreachable")
	}

	// CheckDNSLinux must return false for SetupDNSLinux to fall through to
	// the non-root check. On a CI runner this is the normal state.
	if CheckDNSLinux() {
		t.Skip("DNS already configured on this host — non-root branch unreachable")
	}

	alreadyDone, err := SetupDNSLinux(TrustConfig{})
	if alreadyDone {
		t.Errorf("expected alreadyDone=false on non-root path, got true")
	}
	if err == nil {
		t.Fatal("expected non-root error, got nil")
	}
	if !strings.Contains(err.Error(), "root privileges required") {
		t.Errorf("error should mention root privileges; got: %v", err)
	}
}

// TestSetupDNSLinux_AlreadyDone exercises the early-return path when
// CheckDNSLinux returns true. We can't reliably make CheckDNSLinux return
// true without writing system files, so we skip when already-configured
// state isn't naturally present.
func TestSetupDNSLinux_AlreadyDone(t *testing.T) {
	requireLinux(t)

	if !CheckDNSLinux() {
		t.Skip("DNS not pre-configured on this host — alreadyDone branch unreachable")
	}

	alreadyDone, err := SetupDNSLinux(TrustConfig{})
	if !alreadyDone {
		t.Errorf("expected alreadyDone=true when CheckDNSLinux returns true, got false")
	}
	if err != nil {
		t.Errorf("expected nil err on alreadyDone path, got: %v", err)
	}
}

// TestLinuxDNSResolverConstants verifies the resolver enum is stable and
// distinct — guards against accidental reordering.
func TestLinuxDNSResolverConstants(t *testing.T) {
	requireLinux(t)

	if resolverUnknown == resolverSystemd ||
		resolverSystemd == resolverNetworkManager ||
		resolverNetworkManager == resolverDnsmasqRaw {
		t.Error("linuxDNSResolver enum values must be distinct")
	}
	// Ensure resolverUnknown is the zero value, since detectLinuxResolver
	// returns it as the no-match fallback.
	var zero linuxDNSResolver
	if zero != resolverUnknown {
		t.Errorf("resolverUnknown should be the zero value (got %d)", int(zero))
	}
}

// TestLinuxDNSTimeout verifies the documented timeout constant is non-zero.
// Catches regressions where the constant gets accidentally cleared.
func TestLinuxDNSTimeout(t *testing.T) {
	requireLinux(t)

	if linuxDNSTimeout <= 0 {
		t.Errorf("linuxDNSTimeout must be positive, got %v", linuxDNSTimeout)
	}
}
