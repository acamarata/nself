//go:build !darwin

package trust

import "fmt"

// SetupDNSDarwin is not supported on this platform.
func SetupDNSDarwin(_ TrustConfig) (dnsAlreadyDone bool, resolverAlreadyDone bool, err error) {
	return false, false, fmt.Errorf("SetupDNSDarwin is only supported on macOS")
}

// CheckDNSDarwin is not supported on this platform.
func CheckDNSDarwin() (dnsConfigured bool, resolverConfigured bool) {
	return false, false
}

// SetupPortsDarwin is not supported on this platform.
func SetupPortsDarwin(_ TrustConfig) (alreadyDone bool, err error) {
	return false, fmt.Errorf("SetupPortsDarwin is only supported on macOS")
}

// CheckPortsDarwin is not supported on this platform.
func CheckPortsDarwin(_ TrustConfig) bool {
	return false
}

// launchDaemonPlistPath is defined in ports_common.go (no build tag) so it
// is available on all platforms. No stub needed here.

// launchDaemonLoaded is not supported on this platform.
func launchDaemonLoaded() bool { return false }

// pfAnchorContent is not supported on this platform.
func pfAnchorContent(_ TrustConfig) string { return "" }

// escapeForOsascript is not supported on this platform.
func escapeForOsascript(s string) string { return s }

// findDnsmasqConf is a macOS-only function. On non-Darwin platforms this stub
// exists to satisfy test compilation; callers are guarded by runtime.GOOS checks.
func findDnsmasqConf() string { return "" }

// dnsmasqConfPaths is a macOS-only function. On non-Darwin platforms this stub
// exists to satisfy test compilation; callers are guarded by runtime.GOOS checks.
func dnsmasqConfPaths() []string { return nil }

// writePfAnchorDirect is a macOS-only function. On non-Darwin platforms this stub
// exists to satisfy test compilation; callers are guarded by runtime.GOOS checks.
func writePfAnchorDirect(_ TrustConfig) error {
	return fmt.Errorf("writePfAnchorDirect is only supported on macOS")
}

// configureDnsmasqConf is a macOS-only function. On non-Darwin platforms this stub
// exists to satisfy test compilation; callers are guarded by runtime.GOOS checks.
func configureDnsmasqConf() error {
	return fmt.Errorf("configureDnsmasqConf is only supported on macOS")
}

// ensureDnsmasqInstalled is a macOS-only function. On non-Darwin platforms this stub
// exists to satisfy test compilation; callers are guarded by runtime.GOOS checks.
func ensureDnsmasqInstalled() error {
	return fmt.Errorf("ensureDnsmasqInstalled is only supported on macOS")
}

// restartDnsmasq is a macOS-only function. On non-Darwin platforms this stub
// exists to satisfy test compilation; callers are guarded by runtime.GOOS checks.
func restartDnsmasq() error {
	return fmt.Errorf("restartDnsmasq is only supported on macOS")
}

// flushDNSCache is a macOS-only function. On non-Darwin platforms this stub
// exists to satisfy test compilation; callers are guarded by runtime.GOOS checks.
func flushDNSCache() error {
	return fmt.Errorf("flushDNSCache is only supported on macOS")
}
