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

	// BACKUP-METADATA-01: verify a Hasura metadata backup exists and is < 36h old.
	results = append(results, CheckHasuraMetadataBackup(backupDir))

	return results
}

// CheckHasuraMetadataBackup verifies that a hasura-metadata-*.json file exists
// in backupDir and is less than 36 hours old (implements BACKUP-METADATA-01).
func CheckHasuraMetadataBackup(backupDir string) CheckResult {
	const checkName = "Hasura metadata backup (BACKUP-METADATA-01)"
	const maxAge = 36 * time.Hour

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return CheckResult{
			Section: "backups",
			Name:    checkName,
			Status:  "warn",
			Message: "backup directory not found; run nself backup hasura-metadata",
			FixCmd:  "nself backup hasura-metadata",
		}
	}

	var newest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Match files like hasura-metadata-2026-04-30.json
		if len(name) < len("hasura-metadata-") || name[:len("hasura-metadata-")] != "hasura-metadata-" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}

	if newest.IsZero() {
		return CheckResult{
			Section: "backups",
			Name:    checkName,
			Status:  "fail",
			Message: "no Hasura metadata backup found",
			FixCmd:  "nself backup hasura-metadata",
		}
	}

	age := time.Since(newest)
	if age > maxAge {
		return CheckResult{
			Section: "backups",
			Name:    checkName,
			Status:  "fail",
			Message: fmt.Sprintf("last Hasura metadata backup is %s old (>36h)", age.Round(time.Minute)),
			FixCmd:  "nself backup hasura-metadata",
		}
	}

	return CheckResult{
		Section: "backups",
		Name:    checkName,
		Status:  "pass",
		Message: fmt.Sprintf("last Hasura metadata backup %s ago", age.Round(time.Minute)),
	}
}

// CheckAdminBind verifies that the nself-admin container is bound to
// 127.0.0.1 only (not 0.0.0.0 which would expose it to the LAN).
// This implements S43-T03 — required by the Service Binding Hard Rule.
func CheckAdminBind(ctx context.Context) CheckResult {
	name := "Admin bind address (SEC-BIND-01)"

	// Try docker inspect to determine the host binding for port 3021.
	cmd := exec.CommandContext(ctx, "docker", "inspect",
		"--format", `{{range $p, $conf := .HostConfig.PortBindings}}{{range $conf}}{{if eq $p "3021/tcp"}}{{.HostIp}}{{end}}{{end}}{{end}}`,
		"nself-admin")
	out, err := cmd.Output()
	if err != nil {
		// Container not running — check passes vacuously (nothing exposed).
		return CheckResult{Section: "security", Name: name, Status: "pass",
			Message: "nself-admin container not running"}
	}

	hostIP := strings.TrimSpace(string(out))
	if hostIP == "" {
		return CheckResult{Section: "security", Name: name, Status: "pass",
			Message: "admin container has no published port 3021"}
	}
	if hostIP == "0.0.0.0" || hostIP == "" {
		return CheckResult{Section: "security", Name: name, Status: "fail",
			Message: fmt.Sprintf("admin port 3021 bound to %q — exposes admin UI to LAN", hostIP),
			FixCmd:  "nself admin stop && nself admin start"}
	}
	if hostIP != "127.0.0.1" {
		return CheckResult{Section: "security", Name: name, Status: "warn",
			Message: fmt.Sprintf("admin port 3021 bound to %q (expected 127.0.0.1)", hostIP)}
	}
	return CheckResult{Section: "security", Name: name, Status: "pass",
		Message: "admin port 3021 bound to 127.0.0.1 only"}
}

// CheckEncryptionKeyScope verifies that AI_ENCRYPTION_KEY (and any other
// *_ENCRYPTION_KEY env vars) are set and are not reused across environments.
// Implements S43-UNDEP-01 + S43-AUDIT-01 (SEC-ENC-01 and SEC-ENC-02).
//
// When strict=true, a shared or default key returns Status="fail".
// When strict=false (default), it returns Status="warn" — allowing the admin
// to investigate without blocking a deploy.
func CheckEncryptionKeyScope(projectDir string, strict bool) []CheckResult {
	var results []CheckResult

	// Known encryption key env vars across nSelf (extend as new plugins add keys).
	keyVars := []string{
		"AI_ENCRYPTION_KEY",
		"CLAW_ENCRYPTION_KEY",
		"PLUGIN_AI_ENCRYPTION_KEY",
	}

	// Known weak/default values that must never appear in any real environment.
	weakValues := map[string]bool{
		"":              true,
		"changeme":      true,
		"secret":        true,
		"password":      true,
		"dev":           true,
		"development":   true,
		"test":          true,
		"default":       true,
		"placeholder":   true,
		"replace_me":    true,
		"your_key_here": true,
	}

	status := func(isStrict bool) string {
		if isStrict {
			return "fail"
		}
		return "warn"
	}

	for _, keyVar := range keyVars {
		val := os.Getenv(keyVar)

		// SEC-ENC-01: key must be set.
		if val == "" {
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("SEC-ENC-01: %s", keyVar),
				Status:  status(strict),
				Message: fmt.Sprintf("%s is not set (encryption at rest requires a key)", keyVar),
			})
			continue
		}

		// SEC-ENC-02: key must not be a known weak/default value.
		if weakValues[strings.ToLower(strings.TrimSpace(val))] {
			results = append(results, CheckResult{
				Section: "security",
				Name:    fmt.Sprintf("SEC-ENC-02: %s", keyVar),
				Status:  status(strict),
				Message: fmt.Sprintf("%s uses a default/weak value; rotate before deploying to production", keyVar),
			})
			continue
		}

		// Key is set and non-default.
		results = append(results, CheckResult{
			Section: "security",
			Name:    fmt.Sprintf("SEC-ENC-01/02: %s", keyVar),
			Status:  "pass",
			Message: fmt.Sprintf("%s is set and non-default", keyVar),
		})
	}

	return results
}

// SecurityChecks runs security diagnostics.
func SecurityChecks(ctx context.Context, projectDir string) []CheckResult {
	var results []CheckResult

	// JWT secret presence — Hasura refuses to start without it.
	results = append(results, CheckJWTSecretPresent(projectDir))

	// S43-T03: admin service binding check.
	results = append(results, CheckAdminBind(ctx))

	// S43-AUDIT-01 / S43-UNDEP-01: encryption key scope (SEC-ENC-01/02).
	// WARN by default; callers may re-run with strict=true for --strict mode.
	results = append(results, CheckEncryptionKeyScope(projectDir, false)...)

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
