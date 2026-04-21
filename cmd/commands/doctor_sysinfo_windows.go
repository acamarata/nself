//go:build windows

package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getFreeDiskGB returns the free disk space in gigabytes for the current
// working directory's drive on Windows by shelling out to `wmic`. Returns
// zero and an error if the value cannot be determined.
//
// Windows plugin-runtime and full-stack support is a Phase-2 goal. For v1.0.x
// this helper exists so the CLI vets, builds, and passes unit tests on
// windows-2022 runners.
func getFreeDiskGB() (float64, error) {
	out, err := exec.Command("wmic", "logicaldisk", "where", "DeviceID='C:'", "get", "FreeSpace", "/value").Output()
	if err != nil {
		return 0, fmt.Errorf("wmic failed: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "FreeSpace=") {
			continue
		}
		raw := strings.TrimPrefix(line, "FreeSpace=")
		bytes, perr := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		if perr != nil {
			return 0, fmt.Errorf("parsing FreeSpace: %w", perr)
		}
		return float64(bytes) / (1024 * 1024 * 1024), nil
	}
	return 0, fmt.Errorf("FreeSpace not reported by wmic")
}

// getTotalMemoryMB returns total physical memory in megabytes on Windows by
// shelling out to `wmic`. Returns zero and an error on failure.
func getTotalMemoryMB() (uint64, error) {
	out, err := exec.Command("wmic", "ComputerSystem", "get", "TotalPhysicalMemory", "/value").Output()
	if err != nil {
		return 0, fmt.Errorf("wmic failed: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "TotalPhysicalMemory=") {
			continue
		}
		raw := strings.TrimPrefix(line, "TotalPhysicalMemory=")
		bytes, perr := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
		if perr != nil {
			return 0, fmt.Errorf("parsing TotalPhysicalMemory: %w", perr)
		}
		return bytes / (1024 * 1024), nil
	}
	return 0, fmt.Errorf("TotalPhysicalMemory not reported by wmic")
}
