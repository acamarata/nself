package license

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Key-file housekeeping for the license directory.
//
// Split out of keys.go, which crossed the repo's 300-line file cap. The cap is
// a ratchet: raising the budget is the wrong move when the file genuinely has
// two responsibilities, and key storage housekeeping is separable from key
// parsing and validation.

// clearAllKeyFiles removes all key files from the license directory.
//
// A removal target that sits under a path which is not a directory cannot
// exist, so ENOTDIR means the same thing here as ENOENT: nothing to clear.
// The IsNotExist guard alone did not cover that, and callers surfaced the raw
// errno instead ("remove .../license/key: not a directory").
func clearAllKeyFiles(dir string) error {
	// Primary.
	path := dir + "/" + keyFile
	if err := os.Remove(path); err != nil && !isAbsentPath(err) {
		return err
	}
	// Numbered.
	for i := 2; i <= 10; i++ {
		path := fmt.Sprintf("%s/%s.%d", dir, keyFile, i)
		if err := os.Remove(path); err != nil && !isAbsentPath(err) {
			return err
		}
	}
	return nil
}

// isAbsentPath reports whether err means the target is not there to act on,
// covering both a missing entry and a parent that is not a directory.
func isAbsentPath(err error) bool {
	return os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR)
}
