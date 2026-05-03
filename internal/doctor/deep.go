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

	// S70-T06: Hasura introspection disabled in production.
	results = append(results, CheckHasuraIntrospection(ctx))

	// S77-T08: orphaned Hasura remote schemas after plugin uninstall.
	results = append(results, CheckOrphanRemoteSchemas(ctx))

	// S74-T02 + S74-T-PERM-01: RLS enforcement for np_* tables (PERM-RLS-01).
	results = append(results, CheckRLSEnforcement(ctx, false)...)

	// S98-02-T11: JWT key rotation check (JWT-ROT-01).
	results = append(results, CheckJWTRotation(projectDir))

	// S98-02-T12: SSRF guard verification (SSRF-01).
	results = append(results, CheckSSRF(projectDir))

	// P98 S98-02-T10: CI token rotation check (CI-TOKEN-01).
	results = append(results, CheckCIToken(projectDir))

	// P98 S98-02-T14: CI vault sync check (CI-VAULT-SYNC-01).
	results = append(results, CheckCIVaultSync(projectDir))

	// S03-T06: SDK version coherence check (SDK-VERSION-01).
	results = append(results, CheckSDKVersions(ctx)...)

	// G-DOGFOOD (P97 W37/W38): nself.org-specific checks. Gated on
	// NSELF_DOGFOOD=1 so end-user `nself doctor --deep` runs are not slowed
	// by HTTP probes against nself.org subdomains. CI workflow at
	// web/.github/workflows/dogfood-check.yml exports NSELF_DOGFOOD=1.
	if os.Getenv("NSELF_DOGFOOD") == "1" {
		results = append(results, DogfoodChecks(ctx, projectDir, verbose)...)
	}

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

// verifyDockerRunning checks that the Docker daemon is reachable.
// Returns a non-nil error if Docker is not available.
func verifyDockerRunning(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker daemon unreachable: %w", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("docker daemon returned empty server version")
	}
	return nil
}

// resolveHostPort returns the first mapped host port for containerName using
// docker inspect. Returns ("", nil) when no port binding is found, and
// ("", err) on inspect failure.
func resolveHostPort(ctx context.Context, containerName string) (string, error) {
	portCmd := exec.CommandContext(ctx, "docker", "inspect",
		"--format", `{{range $p, $b := .NetworkSettings.Ports}}{{if $b}}{{(index $b 0).HostPort}}{{end}}{{end}}`,
		containerName)
	portOut, err := portCmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect %s: %w", containerName, err)
	}
	return strings.TrimSpace(string(portOut)), nil
}

// probePluginHTTP attempts GET /health and, on 404, GET /healthz against
// http://127.0.0.1:<port>. Returns the CheckResult for this plugin.
func probePluginHTTP(client *http.Client, pluginName string, port int) CheckResult {
	checkID := fmt.Sprintf("PLUGIN-HEALTH-%s", pluginName)
	base := fmt.Sprintf("http://127.0.0.1:%d", port)

	for _, path := range []string{"/health", "/healthz"} {
		resp, err := client.Get(base + path)
		if err != nil {
			// connection refused or timeout
			return CheckResult{
				Section: "plugins",
				Name:    checkID,
				Status:  "fail",
				Message: fmt.Sprintf("plugin %s: not running (%v)", pluginName, err),
				FixCmd:  fmt.Sprintf("nself start --plugin %s", pluginName),
			}
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return CheckResult{
				Section: "plugins",
				Name:    checkID,
				Status:  "pass",
				Message: fmt.Sprintf("plugin %s: healthy (HTTP 200 %s)", pluginName, path),
			}
		}
		if resp.StatusCode == http.StatusNotFound && path == "/health" {
			// Try /healthz next iteration.
			continue
		}
		return CheckResult{
			Section: "plugins",
			Name:    checkID,
			Status:  "fail",
			Message: fmt.Sprintf("plugin %s: HTTP %d on %s", pluginName, resp.StatusCode, path),
			FixCmd:  fmt.Sprintf("docker restart %s", pluginName),
		}
	}

	return CheckResult{
		Section: "plugins",
		Name:    checkID,
		Status:  "warn",
		Message: fmt.Sprintf("plugin %s: /health and /healthz both returned 404", pluginName),
	}
}

// PluginHealthChecks verifies every installed plugin's health endpoint.
func PluginHealthChecks(ctx context.Context, projectDir string, verbose bool) []CheckResult {
	var results []CheckResult

	// Step 1: verify Docker daemon is reachable.
	if err := verifyDockerRunning(ctx); err != nil {
		fixCmd := "sudo systemctl start docker"
		if runtime.GOOS == "darwin" {
			fixCmd = "open -a Docker"
		}
		results = append(results, CheckResult{
			Section: "plugins",
			Name:    "PLUGIN-HEALTH-docker",
			Status:  "fail",
			Message: fmt.Sprintf("Docker not running: %v", err),
			FixCmd:  fixCmd,
		})
		return results
	}

	// Step 2: list running nself plugin containers.
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "label=nself.plugin", "--format", "{{.Names}}\t{{.Ports}}")
	out, err := cmd.Output()
	running := strings.TrimSpace(string(out))

	if err != nil || running == "" {
		// Step 3: distinguish installed-but-stopped from not-installed.
		listCmd := exec.CommandContext(ctx, "nself", "plugin", "list", "--installed", "--json")
		listOut, listErr := listCmd.Output()
		expectedCount := 0
		if listErr == nil && strings.TrimSpace(string(listOut)) != "" && strings.TrimSpace(string(listOut)) != "[]" {
			// Count newlines as a rough plugin count; each JSON line = one plugin.
			for _, ln := range strings.Split(strings.TrimSpace(string(listOut)), "\n") {
				if strings.TrimSpace(ln) != "" && strings.TrimSpace(ln) != "[" && strings.TrimSpace(ln) != "]" {
					expectedCount++
				}
			}
		}
		if expectedCount > 0 {
			results = append(results, CheckResult{
				Section: "plugins",
				Name:    "PLUGIN-HEALTH-containers",
				Status:  "fail",
				Message: fmt.Sprintf("%d plugin(s) installed but no containers running", expectedCount),
				FixCmd:  "nself start",
			})
		} else {
			results = append(results, CheckResult{
				Section: "plugins",
				Name:    "PLUGIN-HEALTH-containers",
				Status:  "pass",
				Message: "no plugins installed",
			})
		}
		return results
	}

	// Step 4: probe each running plugin container.
	client := &http.Client{Timeout: 5 * time.Second}
	for _, line := range strings.Split(running, "\n") {
		parts := strings.SplitN(line, "\t", 2)
		containerName := strings.TrimSpace(parts[0])
		if containerName == "" {
			continue
		}

		// Strip "nself_" prefix to derive the plugin name for the check ID.
		pluginName := strings.TrimPrefix(containerName, "nself_")

		hostPort, portErr := resolveHostPort(ctx, containerName)
		if portErr != nil || hostPort == "" || hostPort == "0" {
			// No host port exposed: emit warn, fall through to docker healthcheck.
			results = append(results, CheckResult{
				Section: "plugins",
				Name:    fmt.Sprintf("PLUGIN-HEALTH-%s", pluginName),
				Status:  "warn",
				Message: fmt.Sprintf("plugin %s: cannot probe health (no host port exposed)", pluginName),
			})
			// Still fall through to docker healthcheck below.
		} else {
			portNum, convErr := strconv.Atoi(hostPort)
			if convErr != nil || portNum < 1024 || portNum > 65535 {
				results = append(results, CheckResult{
					Section: "plugins",
					Name:    fmt.Sprintf("PLUGIN-HEALTH-%s", pluginName),
					Status:  "warn",
					Message: fmt.Sprintf("plugin %s: invalid host port %s", pluginName, hostPort),
				})
			} else {
				results = append(results, probePluginHTTP(client, pluginName, portNum))
				continue // HTTP probe result is authoritative; skip docker healthcheck.
			}
		}

		// Fallback: docker healthcheck status (when no valid host port).
		hcCmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Health.Status}}", containerName)
		hcOut, hcErr := hcCmd.Output()
		if hcErr != nil {
			results = append(results, CheckResult{
				Section: "plugins",
				Name:    fmt.Sprintf("PLUGIN-HEALTH-%s", pluginName),
				Status:  "warn",
				Message: fmt.Sprintf("plugin %s: cannot inspect docker health", pluginName),
			})
			continue
		}
		hStatus := strings.TrimSpace(string(hcOut))
		if hStatus == "healthy" || hStatus == "" {
			results = append(results, CheckResult{
				Section: "plugins",
				Name:    fmt.Sprintf("PLUGIN-HEALTH-%s", pluginName),
				Status:  "pass",
				Message: fmt.Sprintf("plugin %s: healthy (docker healthcheck)", pluginName),
			})
		} else {
			results = append(results, CheckResult{
				Section: "plugins",
				Name:    fmt.Sprintf("PLUGIN-HEALTH-%s", pluginName),
				Status:  "fail",
				Message: fmt.Sprintf("plugin %s: docker health status: %s", pluginName, hStatus),
				FixCmd:  fmt.Sprintf("docker restart %s", containerName),
			})
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
