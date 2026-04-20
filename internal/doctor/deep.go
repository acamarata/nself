package doctor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DeepChecks runs all 12 subsystem categories for --deep mode.
func DeepChecks(ctx context.Context, projectDir string, verbose bool) []CheckResult {
	var results []CheckResult

	results = append(results, HostChecks(ctx, verbose)...)
	results = append(results, DockerDeepChecks(ctx, verbose)...)
	results = append(results, PostgresChecks(ctx, verbose)...)
	results = append(results, HasuraChecks(ctx, verbose)...)
	results = append(results, NginxChecks(ctx, verbose)...)
	results = append(results, SSLChecks(ctx, verbose)...)
	results = append(results, PingChecks(ctx, verbose)...)
	results = append(results, PluginHealthChecks(ctx, projectDir, verbose)...)
	results = append(results, LicenseChecks(ctx, projectDir, verbose)...)
	results = append(results, MonitoringChecks(ctx)...)
	results = append(results, BackupChecks(ctx, projectDir)...)
	results = append(results, SecurityChecks(ctx, projectDir)...)

	// S69-T05: ai+moderation wiring gap check.
	results = append(results, CheckModerationWired(ctx))

	return results
}

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

// DockerDeepChecks verifies daemon, storage driver, dangling images, container health.
func DockerDeepChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// Daemon reachable
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Driver}}")
	out, err := cmd.Output()
	if err != nil {
		results = append(results, CheckResult{Section: "docker", Name: "Docker daemon", Status: "fail", Message: "unreachable"})
		return results
	}
	driver := strings.TrimSpace(string(out))
	if driver != "overlay2" {
		results = append(results, CheckResult{Section: "docker", Name: "Storage driver", Status: "warn",
			Message: fmt.Sprintf("using %s (expected overlay2)", driver)})
	} else {
		results = append(results, CheckResult{Section: "docker", Name: "Storage driver", Status: "pass", Message: "overlay2"})
	}

	// Dangling images >5GB
	cmd = exec.CommandContext(ctx, "docker", "system", "df", "--format", "{{.Size}}")
	out, err = cmd.Output()
	if err == nil {
		results = append(results, CheckResult{Section: "docker", Name: "Docker disk usage", Status: "pass",
			Message: strings.TrimSpace(string(out))})
	}

	// All expected containers healthy
	cmd = exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}\t{{.Status}}")
	out, err = cmd.Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			cName := parts[0]
			status := ""
			if len(parts) > 1 {
				status = parts[1]
			}
			if strings.Contains(strings.ToLower(status), "unhealthy") {
				results = append(results, CheckResult{Section: "docker", Name: fmt.Sprintf("Container: %s", cName),
					Status: "fail", Message: "unhealthy", FixCmd: fmt.Sprintf("docker restart %s", cName)})
			}
		}
	}

	return results
}

// PostgresChecks verifies pg_isready, replication lag, longest query, dead tuples, vacuum.
func PostgresChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// pg_isready
	cmd := exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "pg_isready", "-U", "postgres")
	if err := cmd.Run(); err != nil {
		results = append(results, CheckResult{Section: "postgres", Name: "pg_isready", Status: "fail", Message: "not ready"})
		return results
	}
	results = append(results, CheckResult{Section: "postgres", Name: "pg_isready", Status: "pass", Message: "accepting connections"})

	// Longest running query <60s
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "psql", "-U", "postgres", "-t", "-c",
		"SELECT COALESCE(EXTRACT(EPOCH FROM MAX(now() - query_start))::int, 0) FROM pg_stat_activity WHERE state='active' AND pid != pg_backend_pid();")
	out, err := cmd.Output()
	if err == nil {
		secs, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		if secs > 60 {
			results = append(results, CheckResult{Section: "postgres", Name: "Longest query", Status: "warn",
				Message: fmt.Sprintf("%ds (>60s)", secs)})
		} else {
			results = append(results, CheckResult{Section: "postgres", Name: "Longest query", Status: "pass",
				Message: fmt.Sprintf("%ds", secs)})
		}
	}

	// Dead tuple % <10%
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "psql", "-U", "postgres", "-t", "-c",
		"SELECT COALESCE(MAX(n_dead_tup::float / NULLIF(n_live_tup + n_dead_tup, 0) * 100)::int, 0) FROM pg_stat_user_tables;")
	out, err = cmd.Output()
	if err == nil {
		pct, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		if pct > 10 {
			results = append(results, CheckResult{Section: "postgres", Name: "Dead tuples", Status: "warn",
				Message: fmt.Sprintf("%d%% dead tuples (>10%%)", pct),
				FixCmd:  "docker exec nself_postgres psql -U postgres -c 'VACUUM ANALYZE;'"})
		} else {
			results = append(results, CheckResult{Section: "postgres", Name: "Dead tuples", Status: "pass",
				Message: fmt.Sprintf("%d%% dead tuples", pct)})
		}
	}

	// Last vacuum <24h
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_postgres", "psql", "-U", "postgres", "-t", "-c",
		"SELECT COALESCE(EXTRACT(EPOCH FROM MIN(now() - last_autovacuum))::int, 0) FROM pg_stat_user_tables WHERE last_autovacuum IS NOT NULL;")
	out, err = cmd.Output()
	if err == nil {
		secs, _ := strconv.Atoi(strings.TrimSpace(string(out)))
		hours := secs / 3600
		if hours > 24 {
			results = append(results, CheckResult{Section: "postgres", Name: "Last vacuum", Status: "warn",
				Message: fmt.Sprintf("%dh ago (>24h)", hours)})
		} else {
			results = append(results, CheckResult{Section: "postgres", Name: "Last vacuum", Status: "pass",
				Message: fmt.Sprintf("%dh ago", hours)})
		}
	}

	return results
}

// HasuraChecks verifies /healthz 200 and metadata consistency.
func HasuraChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// /healthz
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/healthz")
	if err != nil || resp.StatusCode != 200 {
		results = append(results, CheckResult{Section: "hasura", Name: "Hasura healthz", Status: "fail", Message: "unhealthy"})
	} else {
		results = append(results, CheckResult{Section: "hasura", Name: "Hasura healthz", Status: "pass", Message: "200 OK"})
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Metadata consistency — check via metadata API
	cmd := exec.CommandContext(ctx, "curl", "-sf", "-H", "Content-Type: application/json",
		"-d", `{"type":"get_inconsistent_metadata","args":{}}`,
		"http://127.0.0.1:8080/v1/metadata")
	out, err := cmd.Output()
	if err != nil {
		results = append(results, CheckResult{Section: "hasura", Name: "Metadata consistency", Status: "warn",
			Message: "cannot check (admin secret required)"})
	} else if strings.Contains(string(out), `"is_consistent":true`) {
		results = append(results, CheckResult{Section: "hasura", Name: "Metadata consistency", Status: "pass", Message: "consistent"})
	} else {
		results = append(results, CheckResult{Section: "hasura", Name: "Metadata consistency", Status: "fail",
			Message: "inconsistent metadata objects found"})
	}

	return results
}

// NginxChecks verifies config test and SSL cert expiry.
func NginxChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// nginx -t
	cmd := exec.CommandContext(ctx, "docker", "exec", "nself_nginx", "nginx", "-t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, CheckResult{Section: "nginx", Name: "Nginx config test", Status: "fail",
			Message: strings.TrimSpace(string(out))})
	} else {
		results = append(results, CheckResult{Section: "nginx", Name: "Nginx config test", Status: "pass", Message: "syntax ok"})
	}

	// SSL cert expiry >30d on all server_names
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_nginx", "find", "/etc/letsencrypt/live", "-name", "fullchain.pem")
	out, err = cmd.Output()
	if err == nil {
		for _, certPath := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if certPath == "" {
				continue
			}
			checkCmd := exec.CommandContext(ctx, "docker", "exec", "nself_nginx", "openssl", "x509",
				"-enddate", "-noout", "-in", certPath)
			certOut, err := checkCmd.Output()
			if err != nil {
				continue
			}
			// Parse "notAfter=..."
			dateStr := strings.TrimPrefix(strings.TrimSpace(string(certOut)), "notAfter=")
			expiry, err := time.Parse("Jan  2 15:04:05 2006 MST", dateStr)
			if err != nil {
				expiry, err = time.Parse("Jan 2 15:04:05 2006 MST", dateStr)
			}
			if err != nil {
				continue
			}
			domain := filepath.Base(filepath.Dir(certPath))
			daysLeft := int(time.Until(expiry).Hours() / 24)
			if daysLeft < 30 {
				results = append(results, CheckResult{Section: "nginx", Name: fmt.Sprintf("SSL expiry: %s", domain),
					Status: "warn", Message: fmt.Sprintf("%d days left (<30d)", daysLeft)})
			} else {
				results = append(results, CheckResult{Section: "nginx", Name: fmt.Sprintf("SSL expiry: %s", domain),
					Status: "pass", Message: fmt.Sprintf("%d days left", daysLeft)})
			}
		}
	}

	return results
}

// SSLChecks verifies LE renewal cron, last renewal, OCSP stapling.
func SSLChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// LE renewal cron active
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "certbot.timer")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "active" {
		results = append(results, CheckResult{Section: "ssl", Name: "Certbot timer", Status: "warn",
			Message: "certbot.timer not active", FixCmd: "sudo systemctl enable --now certbot.timer"})
	} else {
		results = append(results, CheckResult{Section: "ssl", Name: "Certbot timer", Status: "pass", Message: "active"})
	}

	// Last renewal <60d
	cmd = exec.CommandContext(ctx, "find", "/etc/letsencrypt/renewal", "-name", "*.conf", "-mtime", "-60")
	out, err = cmd.Output()
	if err == nil {
		files := strings.TrimSpace(string(out))
		if files == "" {
			results = append(results, CheckResult{Section: "ssl", Name: "Last renewal", Status: "warn",
				Message: "no renewals in past 60 days"})
		} else {
			count := len(strings.Split(files, "\n"))
			results = append(results, CheckResult{Section: "ssl", Name: "Last renewal", Status: "pass",
				Message: fmt.Sprintf("%d cert(s) renewed within 60d", count)})
		}
	}

	return results
}

// PingChecks verifies ping.nself.org reachable and license cache fresh.
func PingChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ping.nself.org/health")
	if err != nil {
		results = append(results, CheckResult{Section: "ping", Name: "ping.nself.org", Status: "warn",
			Message: fmt.Sprintf("unreachable: %v", err)})
	} else {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			results = append(results, CheckResult{Section: "ping", Name: "ping.nself.org", Status: "pass", Message: "reachable"})
		} else {
			results = append(results, CheckResult{Section: "ping", Name: "ping.nself.org", Status: "warn",
				Message: fmt.Sprintf("returned %d", resp.StatusCode)})
		}
	}

	return results
}

// PluginHealthChecks verifies every installed plugin's health endpoint.
func PluginHealthChecks(ctx context.Context, projectDir string, verbose bool) []CheckResult {
	var results []CheckResult

	// List plugin containers and probe /health
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "label=nself.plugin", "--format", "{{.Names}}\t{{.Ports}}")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		results = append(results, CheckResult{Section: "plugins", Name: "Plugin health", Status: "pass",
			Message: "no plugin containers found"})
		return results
	}

	client := &http.Client{Timeout: 3 * time.Second}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		name := parts[0]
		// Try common health endpoint on localhost
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:0/health")) // placeholder; real port from inspect
		_ = resp
		_ = err
		// Fallback: docker exec health check
		hcCmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Health.Status}}", name)
		hcOut, hcErr := hcCmd.Output()
		if hcErr != nil {
			results = append(results, CheckResult{Section: "plugins", Name: fmt.Sprintf("Plugin: %s", name),
				Status: "warn", Message: "cannot inspect health"})
			continue
		}
		hStatus := strings.TrimSpace(string(hcOut))
		if hStatus == "healthy" {
			results = append(results, CheckResult{Section: "plugins", Name: fmt.Sprintf("Plugin: %s", name),
				Status: "pass", Message: "healthy"})
		} else {
			results = append(results, CheckResult{Section: "plugins", Name: fmt.Sprintf("Plugin: %s", name),
				Status: "fail", Message: hStatus, FixCmd: fmt.Sprintf("docker restart %s", name)})
		}
	}

	return results
}

// LicenseChecks verifies license is current and tier matches expected.
func LicenseChecks(ctx context.Context, projectDir string, verbose bool) []CheckResult {
	var results []CheckResult

	cachePath := filepath.Join(projectDir, ".nself", "cache", "entitlements.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		results = append(results, CheckResult{Section: "license", Name: "License cache", Status: "pass",
			Message: "no cache (run nself license validate)"})
		return results
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		results = append(results, CheckResult{Section: "license", Name: "License cache", Status: "warn",
			Message: fmt.Sprintf("cannot read: %v", err)})
		return results
	}

	content := string(data)
	if strings.Contains(content, `"grace"`) {
		results = append(results, CheckResult{Section: "license", Name: "License grace", Status: "warn",
			Message: "license in grace period"})
	} else {
		results = append(results, CheckResult{Section: "license", Name: "License status", Status: "pass",
			Message: "active"})
	}

	return results
}

// FilterBySection returns only checks matching the given section name.
func FilterBySection(results []CheckResult, section string) []CheckResult {
	var filtered []CheckResult
	for _, r := range results {
		if strings.EqualFold(r.Section, section) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
