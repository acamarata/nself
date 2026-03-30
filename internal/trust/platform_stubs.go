//go:build !linux

package trust

import "fmt"

// SetupDNSLinux is not supported on this platform.
func SetupDNSLinux(_ TrustConfig) (alreadyDone bool, err error) {
	return false, fmt.Errorf("SetupDNSLinux is only supported on Linux")
}

// CheckDNSLinux is not supported on this platform.
func CheckDNSLinux() bool {
	return false
}

// SetupPortsLinux is not supported on this platform.
func SetupPortsLinux(_ TrustConfig) (alreadyDone bool, err error) {
	return false, fmt.Errorf("SetupPortsLinux is only supported on Linux")
}

// CheckPortsLinux is not supported on this platform.
func CheckPortsLinux(_ TrustConfig) bool {
	return false
}
