//go:build !windows

package auth

import (
	"os"
	"syscall"
)

// lockExclusive takes an exclusive advisory lock on f using flock(2).
// Returns an unlock function the caller must defer.
func lockExclusive(f *os.File) (func(), error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return func() {}, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	}, nil
}
