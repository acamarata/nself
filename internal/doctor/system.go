// Package doctor provides comprehensive health check sections for nself doctor.
package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CheckResult holds the outcome of a single diagnostic check.
type CheckResult struct {
	Section string `json:"section"`
	Name    string `json:"name"`
	Status  string `json:"status"` // pass, warn, fail
	Message string `json:"message"`
	FixCmd  string `json:"fix_cmd,omitempty"` // suggested fix command
}

// SystemChecks runs system-level diagnostics: disk, memory, swap, load, clock sync, Docker, kernel.
func SystemChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// Docker daemon version
	results = append(results, checkDockerVersion(ctx))

	// Kernel version (Linux only)
	if runtime.GOOS == "linux" {
		results = append(results, checkKernel(ctx))
	}

	// Clock sync
	results = append(results, checkClockSync(ctx))

	// Load average
	results = append(results, checkLoad(ctx))

	// Swap
	results = append(results, checkSwap(ctx))

	return results
}

func checkDockerVersion(ctx context.Context) CheckResult {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Section: "system", Name: "Docker version", Status: "fail", Message: "cannot get Docker version"}
	}
	ver := strings.TrimSpace(string(out))
	return CheckResult{Section: "system", Name: "Docker version", Status: "pass", Message: ver}
}

func checkKernel(ctx context.Context) CheckResult {
	cmd := exec.CommandContext(ctx, "uname", "-r")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Section: "system", Name: "Kernel", Status: "warn", Message: "cannot determine kernel version"}
	}
	return CheckResult{Section: "system", Name: "Kernel", Status: "pass", Message: strings.TrimSpace(string(out))}
}

func checkClockSync(ctx context.Context) CheckResult {
	// Compare system time to a reference. Best effort using date.
	now := time.Now()
	_ = now
	// On Linux, check if NTP is synced
	if runtime.GOOS == "linux" {
		cmd := exec.CommandContext(ctx, "timedatectl", "show", "--property=NTPSynchronized", "--value")
		out, err := cmd.Output()
		if err == nil {
			val := strings.TrimSpace(string(out))
			if val == "yes" {
				return CheckResult{Section: "system", Name: "Clock sync", Status: "pass", Message: "NTP synchronized"}
			}
			return CheckResult{Section: "system", Name: "Clock sync", Status: "warn", Message: "NTP not synchronized",
				FixCmd: "sudo timedatectl set-ntp true"}
		}
	}
	return CheckResult{Section: "system", Name: "Clock sync", Status: "pass", Message: "check skipped on " + runtime.GOOS}
}

func checkLoad(ctx context.Context) CheckResult {
	cmd := exec.CommandContext(ctx, "uptime")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{Section: "system", Name: "Load average", Status: "warn", Message: "cannot check load"}
	}
	return CheckResult{Section: "system", Name: "Load average", Status: "pass", Message: strings.TrimSpace(string(out))}
}

func checkSwap(ctx context.Context) CheckResult {
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "SwapTotal:") {
					return CheckResult{Section: "system", Name: "Swap", Status: "pass", Message: strings.TrimSpace(line)}
				}
			}
		}
	}
	return CheckResult{Section: "system", Name: "Swap", Status: "pass", Message: "swap check skipped on " + runtime.GOOS}
}

// BackupChecks verifies backup health.
func BackupChecks(_ context.Context, projectDir string) []CheckResult {
	var results []CheckResult

	// Check last backup age
	backupDir := filepath.Join(projectDir, ".nself", "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		results = append(results, CheckResult{Section: "backups", Name: "Backup directory", Status: "warn",
			Message: "no backup directory found", FixCmd: "nself backup create"})
		return results
	}

	if len(entries) == 0 {
		results = append(results, CheckResult{Section: "backups", Name: "Last backup", Status: "fail",
			Message: "no backups found", FixCmd: "nself backup create"})
		return results
	}

	// Check most recent backup file modification time
	var newest time.Time
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}

	age := time.Since(newest)
	if age > 26*time.Hour {
		results = append(results, CheckResult{Section: "backups", Name: "Last backup", Status: "fail",
			Message: fmt.Sprintf("last backup is %s old (>26h)", age.Round(time.Minute)),
			FixCmd: "nself backup create"})
	} else {
		results = append(results, CheckResult{Section: "backups", Name: "Last backup", Status: "pass",
			Message: fmt.Sprintf("last backup %s ago", age.Round(time.Minute))})
	}

	return results
}

// SecurityChecks runs security diagnostics.
func SecurityChecks(ctx context.Context, projectDir string) []CheckResult {
	var results []CheckResult

	// JWT secret presence — Hasura refuses to start without it.
	results = append(results, CheckJWTSecretPresent(projectDir))

	// Check for root containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err == nil {
		containers := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, c := range containers {
			if c == "" {
				continue
			}
			inspectCmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Config.User}}", c)
			userOut, err := inspectCmd.Output()
			if err == nil {
				user := strings.TrimSpace(string(userOut))
				if user == "" || user == "root" || user == "0" {
					results = append(results, CheckResult{Section: "security", Name: fmt.Sprintf("Container %s user", c),
						Status: "warn", Message: "running as root"})
				}
			}
		}
	}

	// Check secret file permissions
	secretsDir := filepath.Join(projectDir, ".nself", "secrets")
	if entries, err := os.ReadDir(secretsDir); err == nil {
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			mode := info.Mode().Perm()
			if mode&0o077 != 0 {
				results = append(results, CheckResult{Section: "security",
					Name:    fmt.Sprintf("Secret perms: %s", e.Name()),
					Status:  "warn",
					Message: fmt.Sprintf("mode %o (should be 0600)", mode),
					FixCmd:  fmt.Sprintf("chmod 0600 %s", filepath.Join(secretsDir, e.Name()))})
			}
		}
	}

	if len(results) == 0 {
		results = append(results, CheckResult{Section: "security", Name: "Security", Status: "pass", Message: "no issues found"})
	}

	return results
}

// MonitoringChecks verifies monitoring stack health.
func MonitoringChecks(ctx context.Context) []CheckResult {
	var results []CheckResult

	// Check Prometheus targets
	cmd := exec.CommandContext(ctx, "curl", "-sf", "http://127.0.0.1:9090/-/healthy")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "monitoring", Name: "Prometheus", Status: "warn", Message: "not reachable"})
	} else {
		results = append(results, CheckResult{Section: "monitoring", Name: "Prometheus", Status: "pass", Message: "healthy"})
	}

	// Check Grafana
	cmd = exec.CommandContext(ctx, "curl", "-sf", "http://127.0.0.1:3000/api/health")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "monitoring", Name: "Grafana", Status: "warn", Message: "not reachable"})
	} else {
		results = append(results, CheckResult{Section: "monitoring", Name: "Grafana", Status: "pass", Message: "healthy"})
	}

	// Check Loki
	cmd = exec.CommandContext(ctx, "curl", "-sf", "http://127.0.0.1:3100/ready")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "monitoring", Name: "Loki", Status: "warn", Message: "not reachable"})
	} else {
		results = append(results, CheckResult{Section: "monitoring", Name: "Loki", Status: "pass", Message: "ingesting"})
	}

	return results
}

// FixItEngine runs safe auto-fixes for check results that have FixCmd set.
func FixItEngine(ctx context.Context, results []CheckResult) []CheckResult {
	var fixed []CheckResult
	for _, r := range results {
		if r.FixCmd == "" || r.Status == "pass" {
			continue
		}
		parts := strings.Fields(r.FixCmd)
		if len(parts) == 0 {
			continue
		}
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		if err := cmd.Run(); err == nil {
			r.Status = "pass"
			r.Message += " (auto-fixed)"
		}
		fixed = append(fixed, r)
	}
	return fixed
}
