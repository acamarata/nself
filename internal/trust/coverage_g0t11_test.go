// Package trust — coverage_g0t11_test.go: G0-T11 follow-on coverage.
//
// Drives the cross-platform Setup() branches and a few previously-uncovered
// helpers by overriding the currentOS seam. Targets:
//
//   - Setup() default / unsupported-OS branch (the error path)
//   - Setup() linux branch on a Darwin host
//   - Setup() darwin branch on a Linux host
//   - findDnsmasqConf fallback when no candidate exists
//   - Additional structural checks on TrustStatus
package trust

import (
	"os"
	"runtime"
	"testing"
)

// withFakeOS overrides currentOS for the duration of fn, then restores.
func withFakeOS(t *testing.T, os string, fn func()) {
	t.Helper()
	orig := currentOS
	currentOS = func() string { return os }
	defer func() { currentOS = orig }()
	fn()
}

// TestSetup_UnsupportedOS verifies the default switch branch.
func TestSetup_UnsupportedOS(t *testing.T) {
	withFakeOS(t, "windows", func() {
		cfg := TrustConfig{WorkDir: t.TempDir(), BaseDomain: "win.local"}
		_, err := Setup(cfg)
		if err == nil {
			t.Fatal("Setup with unsupported OS should return error")
		}
		if !containsStr(err.Error(), "unsupported OS") {
			t.Errorf("error should say 'unsupported OS', got %v", err)
		}
		if !containsStr(err.Error(), "windows") {
			t.Errorf("error should include the GOOS value, got %v", err)
		}
	})
}

// TestSetup_UnsupportedOS_Other verifies error formatting with another OS.
func TestSetup_UnsupportedOS_Plan9(t *testing.T) {
	withFakeOS(t, "plan9", func() {
		cfg := TrustConfig{WorkDir: t.TempDir()}
		_, err := Setup(cfg)
		if err == nil {
			t.Fatal("Setup(plan9) should error")
		}
	})
}

// TestSetup_LinuxBranchFromDarwin verifies that overriding currentOS to "linux"
// drives Setup into setupLinux even when running on Darwin.
func TestSetup_LinuxBranchOverride(t *testing.T) {
	withFakeOS(t, "linux", func() {
		cfg := TrustConfig{
			WorkDir:    t.TempDir(),
			BaseDomain: "force-linux.local",
			SkipDNS:    true,
			SkipSSL:    true,
			SkipPorts:  true,
		}
		result, err := Setup(cfg)
		if err != nil {
			t.Fatalf("forced-linux Setup all-skip: %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
	})
}

// TestSetup_DarwinBranchOverride verifies the inverse override.
func TestSetup_DarwinBranchOverride(t *testing.T) {
	withFakeOS(t, "darwin", func() {
		cfg := TrustConfig{
			WorkDir:    t.TempDir(),
			BaseDomain: "force-darwin.local",
			SkipDNS:    true,
			SkipSSL:    true,
			SkipPorts:  true,
		}
		result, err := Setup(cfg)
		if err != nil {
			t.Fatalf("forced-darwin Setup all-skip: %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
	})
}

// TestSetup_RealOSStillWorks verifies the production behaviour is unchanged
// when currentOS is the real runtime.GOOS.
func TestSetup_RealOSPath(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("real-OS path test only runs on darwin or linux")
	}
	cfg := TrustConfig{
		WorkDir:    t.TempDir(),
		BaseDomain: "real.local",
		SkipDNS:    true,
		SkipSSL:    true,
		SkipPorts:  true,
	}
	_, err := Setup(cfg)
	if err != nil {
		t.Errorf("real-OS Setup all-skip should succeed: %v", err)
	}
}

// TestCurrentOS_DefaultMatchesRuntime verifies the default seam behaviour.
func TestCurrentOS_DefaultMatchesRuntime(t *testing.T) {
	if got := currentOS(); got != runtime.GOOS {
		t.Errorf("currentOS() default = %q, want %q", got, runtime.GOOS)
	}
}

// TestSetupLinux_AllPaths exercises setupLinux without any skip flags. Each
// step calls into a stub on Darwin (SetupDNSLinux / SetupPortsLinux), so error
// branches accumulate to result.Errors without panic. SetupMkcert may or may
// not succeed depending on whether mkcert is on PATH.
func TestSetupLinux_NoSkipsOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("test exercises setupLinux stubs from Darwin")
	}
	cfg := TrustConfig{
		WorkDir:       t.TempDir(),
		BaseDomain:    "linux-allsteps.local",
		SkipDNS:       false,
		SkipSSL:       false,
		SkipPorts:     false,
		NginxHTTPPort: 8080,
		NginxSSLPort:  8443,
	}
	result, err := setupLinux(cfg, &TrustResult{})
	if err != nil {
		t.Fatalf("setupLinux should not return top-level error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
	// At least DNS + Ports stubs failed — Errors should have entries.
	if len(result.Errors) < 1 {
		t.Errorf("expected at least 1 error from setupLinux stubs on Darwin, got %d", len(result.Errors))
	}
}

// TestSetupDarwin_AllSteps exercises setupDarwin with no skip flags on Darwin.
// The DNS step may succeed or fail depending on dnsmasq; SSL and Ports may
// fail without admin / mkcert. The test just verifies no panic and result
// integrity.
func TestSetupDarwin_NoSkipsOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only")
	}
	// Use unique temp dir to avoid colliding with real ssl/.
	cfg := TrustConfig{
		WorkDir:       t.TempDir(),
		BaseDomain:    "darwin-allsteps.local",
		SkipDNS:       false,
		SkipSSL:       false,
		SkipPorts:     false,
		NginxHTTPPort: 8080,
		NginxSSLPort:  8443,
	}
	result, err := setupDarwin(cfg, &TrustResult{})
	if err != nil {
		t.Fatalf("setupDarwin should not return top-level error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
}

// TestSetupLinux_NoSkipsViaSetupForceLinux drives the no-skip path via the
// public Setup() entry point with the OS forced to linux. This exercises the
// linux switch branch AND setupLinux's no-skip code path.
func TestSetupLinux_NoSkipsViaSetupForceLinux(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("test forces linux from darwin only")
	}
	withFakeOS(t, "linux", func() {
		cfg := TrustConfig{
			WorkDir:       t.TempDir(),
			BaseDomain:    "force-linux-noskip.local",
			NginxHTTPPort: 8080,
			NginxSSLPort:  8443,
		}
		result, err := Setup(cfg)
		if err != nil {
			t.Fatalf("forced-linux no-skip Setup: %v", err)
		}
		if result == nil {
			t.Fatal("result must not be nil")
		}
	})
}

// readSmallFile reads a file fully. Used by darwin-gated tests in
// coverage_g0t11_darwin_test.go.
func readSmallFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

// countOccurrences returns the number of times needle appears in haystack.
func countOccurrences(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	count := 0
	pos := 0
	for {
		idx := indexOf(haystack[pos:], needle)
		if idx < 0 {
			break
		}
		count++
		pos += idx + len(needle)
	}
	return count
}

// indexOf returns the first index of needle in haystack, or -1.
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// --- setupDarwin error-branch coverage via step seams ----------------------

// stubError is a sentinel error type for tests.
type stubError string

func (e stubError) Error() string { return string(e) }

// withStubSteps overrides all setup-step seams to return success / failure as
// configured. Restores originals on cleanup.
func withStubSteps(t *testing.T,
	dns func(TrustConfig) (bool, bool, error),
	mk func(TrustConfig) (bool, error),
	pt func(TrustConfig) (bool, error),
	dnsLin func(TrustConfig) (bool, error),
	ptLin func(TrustConfig) (bool, error),
	fn func(),
) {
	t.Helper()
	o1, o2, o3, o4, o5 := setupDNSDarwinFunc, setupMkcertFunc, setupPortsDarwinFunc, setupDNSLinuxFunc, setupPortsLinuxFunc
	if dns != nil {
		setupDNSDarwinFunc = dns
	}
	if mk != nil {
		setupMkcertFunc = mk
	}
	if pt != nil {
		setupPortsDarwinFunc = pt
	}
	if dnsLin != nil {
		setupDNSLinuxFunc = dnsLin
	}
	if ptLin != nil {
		setupPortsLinuxFunc = ptLin
	}
	defer func() {
		setupDNSDarwinFunc = o1
		setupMkcertFunc = o2
		setupPortsDarwinFunc = o3
		setupDNSLinuxFunc = o4
		setupPortsLinuxFunc = o5
	}()
	fn()
}

// TestSetupDarwin_AllStepsSuccessSeam exercises the success branch of every
// step, capturing alreadyDone propagation into TrustResult.
func TestSetupDarwin_AllStepsSuccessSeam(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only")
	}
	dnsStub := func(_ TrustConfig) (bool, bool, error) { return true, true, nil }
	mkStub := func(_ TrustConfig) (bool, error) { return true, nil }
	ptStub := func(_ TrustConfig) (bool, error) { return false, nil }

	withStubSteps(t, dnsStub, mkStub, ptStub, nil, nil, func() {
		cfg := TrustConfig{WorkDir: t.TempDir(), BaseDomain: "ok.local",
			NginxHTTPPort: 8080, NginxSSLPort: 8443}
		result, err := setupDarwin(cfg, &TrustResult{})
		if err != nil {
			t.Fatalf("setupDarwin: %v", err)
		}
		if !result.DNSConfigured || !result.DNSAlreadyDone {
			t.Errorf("DNS flags wrong: %+v", result)
		}
		if !result.ResolverConfigured || !result.ResolverAlreadyDone {
			t.Errorf("Resolver flags wrong: %+v", result)
		}
		if !result.CertsGenerated || !result.CertsAlreadyDone {
			t.Errorf("Certs flags wrong: %+v", result)
		}
		if !result.PortsConfigured || result.PortsAlreadyDone {
			t.Errorf("Ports flags wrong: %+v", result)
		}
		if len(result.Errors) != 0 {
			t.Errorf("expected no errors, got %v", result.Errors)
		}
	})
}

// TestSetupDarwin_AllStepsErrorSeam exercises every error branch.
func TestSetupDarwin_AllStepsErrorSeam(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only")
	}
	dnsStub := func(_ TrustConfig) (bool, bool, error) {
		return false, false, stubError("dns failed")
	}
	mkStub := func(_ TrustConfig) (bool, error) { return false, stubError("mk failed") }
	ptStub := func(_ TrustConfig) (bool, error) { return false, stubError("pt failed") }

	withStubSteps(t, dnsStub, mkStub, ptStub, nil, nil, func() {
		cfg := TrustConfig{WorkDir: t.TempDir(), BaseDomain: "fail.local",
			NginxHTTPPort: 8080, NginxSSLPort: 8443}
		result, err := setupDarwin(cfg, &TrustResult{})
		if err != nil {
			t.Fatalf("setupDarwin should not return top-level error: %v", err)
		}
		// All three steps failed → 3 errors accumulated.
		if len(result.Errors) != 3 {
			t.Errorf("expected 3 errors, got %d (%v)", len(result.Errors), result.Errors)
		}
		// No flag should be set.
		if result.DNSConfigured || result.CertsGenerated || result.PortsConfigured {
			t.Errorf("no step should report success: %+v", result)
		}
	})
}

// TestSetupLinux_AllStepsSuccessSeam exercises Linux success branches via seams.
func TestSetupLinux_AllStepsSuccessSeam(t *testing.T) {
	dnsLinStub := func(_ TrustConfig) (bool, error) { return true, nil }
	mkStub := func(_ TrustConfig) (bool, error) { return false, nil }
	ptLinStub := func(_ TrustConfig) (bool, error) { return true, nil }

	withStubSteps(t, nil, mkStub, nil, dnsLinStub, ptLinStub, func() {
		cfg := TrustConfig{WorkDir: t.TempDir(), BaseDomain: "linux-ok.local"}
		result, err := setupLinux(cfg, &TrustResult{})
		if err != nil {
			t.Fatalf("setupLinux: %v", err)
		}
		if !result.DNSConfigured || !result.DNSAlreadyDone {
			t.Errorf("DNS flags wrong: %+v", result)
		}
		if !result.ResolverConfigured {
			t.Errorf("Resolver should be set on Linux DNS success: %+v", result)
		}
		if !result.CertsGenerated || result.CertsAlreadyDone {
			t.Errorf("Certs flags wrong: %+v", result)
		}
		if !result.PortsConfigured || !result.PortsAlreadyDone {
			t.Errorf("Ports flags wrong: %+v", result)
		}
		if len(result.Errors) != 0 {
			t.Errorf("expected no errors, got %v", result.Errors)
		}
	})
}

// TestSetupLinux_AllStepsErrorSeam exercises Linux error accumulation.
func TestSetupLinux_AllStepsErrorSeam(t *testing.T) {
	dnsLinStub := func(_ TrustConfig) (bool, error) { return false, stubError("dns") }
	mkStub := func(_ TrustConfig) (bool, error) { return false, stubError("mk") }
	ptLinStub := func(_ TrustConfig) (bool, error) { return false, stubError("pt") }

	withStubSteps(t, nil, mkStub, nil, dnsLinStub, ptLinStub, func() {
		cfg := TrustConfig{WorkDir: t.TempDir()}
		result, err := setupLinux(cfg, &TrustResult{})
		if err != nil {
			t.Fatalf("setupLinux should not return top-level error: %v", err)
		}
		if len(result.Errors) != 3 {
			t.Errorf("expected 3 errors, got %d (%v)", len(result.Errors), result.Errors)
		}
	})
}
