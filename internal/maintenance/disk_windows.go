//go:build windows

package maintenance

import (
	"fmt"
	"syscall"
	"unsafe"
)

// GetDiskUsage returns current disk utilisation for the C:\ drive on Windows.
func GetDiskUsage() (DiskUsage, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytes, totalBytes, totalFreeBytes uint64
	root, _ := syscall.UTF16PtrFromString(`C:\`)
	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(root)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if ret == 0 {
		return DiskUsage{}, fmt.Errorf("GetDiskFreeSpaceExW: %w", err)
	}

	usedBytes := totalBytes - freeBytes
	const gb = 1024 * 1024 * 1024
	var usedPct int
	if totalBytes > 0 {
		usedPct = int((usedBytes * 100) / totalBytes)
	}
	return DiskUsage{
		UsedPercent: usedPct,
		TotalGB:     float64(totalBytes) / gb,
		UsedGB:      float64(usedBytes) / gb,
		FreeGB:      float64(freeBytes) / gb,
	}, nil
}

// DiskCleanup is a stub on Windows — only Docker prune is attempted.
func DiskCleanup() CleanupResult {
	result := CleanupResult{}
	before, _ := GetDiskUsage()
	result.Before = before

	dockerOut, dockerErr := runCommand("docker", "system", "prune", "-af", "--volumes=false")
	result.DockerPruneOut = dockerOut
	if dockerErr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("docker prune: %w", dockerErr))
	}

	after, _ := GetDiskUsage()
	result.After = after
	return result
}
