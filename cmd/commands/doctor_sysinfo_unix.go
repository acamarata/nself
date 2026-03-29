//go:build darwin || linux

package commands

import "syscall"

// getFreeDiskGB returns the free disk space in gigabytes for the current
// working directory's filesystem.
func getFreeDiskGB() (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(".", &stat); err != nil {
		return 0, err
	}
	// Available blocks * block size → bytes → GB
	freeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
	return float64(freeBytes) / (1024 * 1024 * 1024), nil
}
