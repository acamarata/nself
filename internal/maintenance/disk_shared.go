package maintenance

import (
	"fmt"
	"os/exec"
	"strings"
)

// DiskUsage holds before/after disk utilisation for a cleanup run.
type DiskUsage struct {
	// UsedPercent is the percentage of the disk that is in use (0-100).
	UsedPercent int
	// TotalGB is the total disk capacity in gigabytes.
	TotalGB float64
	// UsedGB is the used disk space in gigabytes.
	UsedGB float64
	// FreeGB is the free disk space in gigabytes.
	FreeGB float64
}

// CleanupResult summarises what disk-cleanup did.
type CleanupResult struct {
	Before           DiskUsage
	After            DiskUsage
	DockerPruneOut   string
	LogRotationOut   string
	JournalVacuumOut string
	Errors           []error
}

// runCommand executes a command and returns combined stdout+stderr output and any error.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(out))
	if err != nil {
		return trimmed, fmt.Errorf("%w\n%s", err, trimmed)
	}
	return trimmed, nil
}
