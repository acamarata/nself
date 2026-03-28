package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getTotalMemoryMB returns total physical memory in megabytes on macOS
// by shelling out to sysctl.
func getTotalMemoryMB() (uint64, error) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, fmt.Errorf("sysctl hw.memsize: %w", err)
	}
	bytes, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing hw.memsize: %w", err)
	}
	return bytes / (1024 * 1024), nil
}
