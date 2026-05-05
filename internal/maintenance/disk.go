//go:build darwin || linux

package maintenance

import (
	"fmt"
	"syscall"
)

// GetDiskUsage returns current disk utilisation for the root filesystem ("/").
func GetDiskUsage() (DiskUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return DiskUsage{}, fmt.Errorf("statfs /: %w", err)
	}

	blockSize := uint64(stat.Bsize)
	totalBytes := stat.Blocks * blockSize
	freeBytes := uint64(stat.Bavail) * blockSize
	usedBytes := totalBytes - freeBytes

	const gb = 1024 * 1024 * 1024
	totalGB := float64(totalBytes) / gb
	freeGB := float64(freeBytes) / gb
	usedGB := float64(usedBytes) / gb

	var usedPct int
	if totalBytes > 0 {
		usedPct = int((usedBytes * 100) / totalBytes)
	}

	return DiskUsage{
		UsedPercent: usedPct,
		TotalGB:     totalGB,
		UsedGB:      usedGB,
		FreeGB:      freeGB,
	}, nil
}

// DiskCleanup runs all three cleanup steps and returns a summary.
// It never aborts early — it collects all errors and reports at the end.
func DiskCleanup() CleanupResult {
	result := CleanupResult{}

	before, err := GetDiskUsage()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("read disk usage (before): %w", err))
	}
	result.Before = before

	// 1. Docker prune: images + containers, keep volumes.
	dockerOut, dockerErr := runCommand("docker", "system", "prune", "-af", "--volumes=false")
	result.DockerPruneOut = dockerOut
	if dockerErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("docker prune: %w", dockerErr))
	}

	// 2. Log rotation: delete compressed logs older than 14 days.
	logOut, logErr := runCommand("find", "/var/log", "-name", "*.gz", "-mtime", "+14", "-delete")
	result.LogRotationOut = logOut
	if logErr != nil {
		// non-fatal — /var/log may not exist on all platforms
		result.Errors = append(result.Errors, fmt.Errorf("log rotation: %w", logErr))
	}

	// 3. Journalctl vacuum (Linux only; harmless no-op on macOS).
	journalOut, journalErr := runCommand("journalctl", "--vacuum-time=7d")
	result.JournalVacuumOut = journalOut
	if journalErr != nil {
		// non-fatal on macOS
		result.Errors = append(result.Errors, fmt.Errorf("journalctl vacuum: %w", journalErr))
	}

	after, err := GetDiskUsage()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("read disk usage (after): %w", err))
	}
	result.After = after

	return result
}
