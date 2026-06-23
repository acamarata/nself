package plugin

// Purpose: Platform architecture detection for Rust/binary plugin installs.
//          Maps runtime.GOOS/GOARCH to the canonical arch strings used in
//          GitHub Release asset names (darwin-arm64, darwin-amd64,
//          linux-amd64, linux-arm64, windows-amd64).
// Inputs:  GOOS string, GOARCH string.
// Outputs: arch string or error when the combination is unsupported.
// Constraints: Only the 5 platform combinations declared in validArchPairs are
//              accepted; all others return ErrUnsupportedArch.
// SPORT: arch detection path; callers: binary_install.go

import (
	"fmt"
	"runtime"
)

// ErrUnsupportedArch is returned when the running platform has no binary
// plugin asset available for it.
var ErrUnsupportedArch = fmt.Errorf("unsupported platform for binary plugin install")

// archKey is the internal key used to look up the canonical arch string.
type archKey struct {
	goos   string
	goarch string
}

// archTable maps GOOS/GOARCH pairs to the canonical asset suffix used in
// GitHub Release binary names. Matches the logic in the legacy runtime.sh.
var archTable = map[archKey]string{
	{"darwin", "arm64"}:  "darwin-arm64",
	{"darwin", "amd64"}:  "darwin-amd64",
	{"linux", "amd64"}:   "linux-amd64",
	{"linux", "arm64"}:   "linux-arm64",
	{"windows", "amd64"}: "windows-amd64",
}

// currentArch returns the canonical arch string for the running platform.
// Returns ErrUnsupportedArch when the GOOS/GOARCH combination has no entry.
func currentArch() (string, error) {
	return archForPlatform(runtime.GOOS, runtime.GOARCH)
}

// archForPlatform returns the canonical arch string for the given GOOS/GOARCH
// pair. Exported for testing only — call currentArch() in production.
func archForPlatform(goos, goarch string) (string, error) {
	key := archKey{goos: goos, goarch: goarch}
	arch, ok := archTable[key]
	if !ok {
		return "", fmt.Errorf("%w: %s/%s", ErrUnsupportedArch, goos, goarch)
	}
	return arch, nil
}
