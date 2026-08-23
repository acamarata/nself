package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Purpose: host-level --deep checks — disk, swap, CPU load, kernel taint.
// Inputs: a context and verbose flag; some checks read /proc directly.
// Outputs: []CheckResult per category, or a single CheckResult per helper.
// Constraints: split out of deep.go (CLI-R12) as a pure move; no behavior
// changed.

// HostChecks verifies disk, memory, CPU, time sync, kernel.
func HostChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// Disk: >20% free on / and /var
	for _, mount := range []string{"/", "/var"} {
		results = append(results, checkDiskPercent(ctx, mount))
	}

	// Memory: swap <50% used
	results = append(results, checkSwapUsage(ctx))

	// CPU: 1m load < cores * 0.7
	results = append(results, checkCPULoad(ctx))

	// Time sync: chronyd synced
	results = append(results, checkClockSync(ctx))

	// Kernel: no tainted flag
	if runtime.GOOS == "linux" {
		results = append(results, checkKernelTainted(ctx))
	}

	return results
}

func checkDiskPercent(ctx context.Context, mount string) CheckResult {
	name := fmt.Sprintf("Disk free: %s", mount)
	cmd := exec.CommandContext(ctx, "df", "--output=pcent", mount)
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Section: "host", Name: name, Status: "warn", Message: "cannot check disk"}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return CheckResult{Section: "host", Name: name, Status: "warn", Message: "unexpected df output"}
	}
	pctStr := strings.TrimSpace(strings.TrimSuffix(lines[1], "%"))
	pct, err := strconv.Atoi(pctStr)
	if err != nil {
		return CheckResult{Section: "host", Name: name, Status: "warn", Message: "cannot parse disk usage"}
	}
	free := 100 - pct
	if free < 20 {
		return CheckResult{Section: "host", Name: name, Status: "fail",
			Message: fmt.Sprintf("%d%% free (<20%%)", free),
			FixCmd:  "docker system prune -f"}
	}
	return CheckResult{Section: "host", Name: name, Status: "pass", Message: fmt.Sprintf("%d%% free", free)}
}

func checkSwapUsage(ctx context.Context) CheckResult {
	name := "Swap usage"
	if runtime.GOOS != "linux" {
		return CheckResult{Section: "host", Name: name, Status: "pass", Message: "skipped on " + runtime.GOOS}
	}
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return CheckResult{Section: "host", Name: name, Status: "warn", Message: "cannot read /proc/meminfo"}
	}
	var total, free int64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "SwapTotal:") {
			total = parseKB(line)
		}
		if strings.HasPrefix(line, "SwapFree:") {
			free = parseKB(line)
		}
	}
	if total == 0 {
		return CheckResult{Section: "host", Name: name, Status: "pass", Message: "no swap configured"}
	}
	usedPct := float64(total-free) / float64(total) * 100
	if usedPct > 50 {
		return CheckResult{Section: "host", Name: name, Status: "warn",
			Message: fmt.Sprintf("%.0f%% swap used (>50%%)", usedPct)}
	}
	return CheckResult{Section: "host", Name: name, Status: "pass", Message: fmt.Sprintf("%.0f%% swap used", usedPct)}
}

func parseKB(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		v, _ := strconv.ParseInt(fields[1], 10, 64)
		return v
	}
	return 0
}

func checkCPULoad(ctx context.Context) CheckResult {
	name := "CPU load"
	cmd := exec.CommandContext(ctx, "cat", "/proc/loadavg")
	out, err := cmd.Output()
	if err != nil {
		// Fallback for non-Linux
		cmd2 := exec.CommandContext(ctx, "uptime")
		out2, err2 := cmd2.Output()
		if err2 != nil {
			return CheckResult{Section: "host", Name: name, Status: "warn", Message: "cannot check load"}
		}
		return CheckResult{Section: "host", Name: name, Status: "pass", Message: strings.TrimSpace(string(out2))}
	}
	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		return CheckResult{Section: "host", Name: name, Status: "warn", Message: "unexpected loadavg output"}
	}
	load1m, _ := strconv.ParseFloat(fields[0], 64)
	cores := runtime.NumCPU()
	threshold := float64(cores) * 0.7
	if load1m > threshold {
		return CheckResult{Section: "host", Name: name, Status: "warn",
			Message: fmt.Sprintf("1m load %.2f > %.1f (%d cores * 0.7)", load1m, threshold, cores)}
	}
	return CheckResult{Section: "host", Name: name, Status: "pass",
		Message: fmt.Sprintf("1m load %.2f (threshold %.1f)", load1m, threshold)}
}

func checkKernelTainted(ctx context.Context) CheckResult {
	name := "Kernel tainted"
	data, err := os.ReadFile("/proc/sys/kernel/tainted")
	if err != nil {
		return CheckResult{Section: "host", Name: name, Status: "warn", Message: "cannot read tainted flag"}
	}
	val := strings.TrimSpace(string(data))
	if val != "0" {
		return CheckResult{Section: "host", Name: name, Status: "warn",
			Message: fmt.Sprintf("kernel tainted flag=%s", val)}
	}
	return CheckResult{Section: "host", Name: name, Status: "pass", Message: "kernel not tainted"}
}
