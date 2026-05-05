//go:build windows

package auth

import "os"

// lockExclusive on Windows is a no-op. The rotation log is appended via
// os.O_APPEND which is atomic on Windows for writes up to PIPE_BUF; the rare
// cross-process concurrent CLI usage is acceptable to leave unlocked here.
// Cross-platform parity is preserved without pulling in golang.org/x/sys.
func lockExclusive(_ *os.File) (func(), error) {
	return func() {}, nil
}
