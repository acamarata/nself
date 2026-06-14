package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/doctor"
	"github.com/nself-org/cli/internal/maintenance"
	"github.com/nself-org/cli/internal/migration"
	"github.com/nself-org/cli/internal/onboarding"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ports"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// doctorCheckResult holds the outcome of a single diagnostic check.
type doctorCheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "warn", "fail"
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// doctorReport collects all check results for JSON output and exit code logic.
type doctorReport struct {
	Timestamp string              `json:"timestamp"`
	Checks    []doctorCheckResult `json:"checks"`
	Summary   doctorSummary       `json:"summary"`
}

type doctorSummary struct {
	Total    int `json:"total"`
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Failed   int `json:"failed"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run comprehensive system diagnostics",
	Long: `Run comprehensive system diagnostics to validate your nSelf environment.

Checks infrastructure requirements, Docker health, disk space, memory,
network connectivity, configuration, running containers, and more.

Exit codes:
  0  All checks passed
  1  One or more checks failed
  2  Warnings only (no failures)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// ── Legacy global scan (S60-T03) ─────────────────────────────
		// --check-legacy scans for v0.9 stale paths on the host (NOT per-project).
		// Returns 0 on clean install, non-zero with structured output when found.
		checkLegacy, _ := cmd.Flags().GetBool("check-legacy")
		if checkLegacy {
			return runDoctorCheckLegacy()
		}

		// ── Onboarding funnel check (Q08) ────────────────────────────
		// --install-check runs 6-stage onboarding funnel check.
		// Invoked automatically by Homebrew post-install hook; safe to run at any time.
		installCheck, _ := cmd.Flags().GetBool("install-check")
		if installCheck {
			jsonOut2, _ := cmd.Flags().GetBool("json")
			format2, _ := cmd.Flags().GetString("format")
			if format2 == "json" {
				jsonOut2 = true
			}
			return runInstallCheck(jsonOut2)
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		full, _ := cmd.Flags().GetBool("full")
		deep, _ := cmd.Flags().GetBool("deep")
		if deep {
			full = true
		}
		fix, _ := cmd.Flags().GetBool("fix")
		jsonOut, _ := cmd.Flags().GetBool("json")
		formatFlag, _ := cmd.Flags().GetString("format")
		onlySection, _ := cmd.Flags().GetString("only")
		if formatFlag == "json" {
			jsonOut = true
		}

		// ── AI wizard mode (T-05-01) ────────────────────────────────
		aiMode, _ := cmd.Flags().GetBool("ai")
		if aiMode {
			yes, _ := cmd.Flags().GetBool("yes")
			skipOllama, _ := cmd.Flags().GetBool("skip-ollama")
			skipPool, _ := cmd.Flags().GetBool("skip-pool")
			headless, _ := cmd.Flags().GetBool("headless")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			return runDoctorAI(ctx, doctorAIFlags{
				yes:        yes,
				skipOllama: skipOllama,
				skipPool:   skipPool,
				headless:   headless,
				jsonOut:    jsonOut,
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		cwd, _ := os.Getwd()

		// Deep mode: run all 12 subsystem checks via doctor package
		if deep {
			deepResults := doctor.DeepChecks(ctx, cwd, verbose)

			// Apply --only filter
			if onlySection != "" {
				deepResults = doctor.FilterBySection(deepResults, onlySection)
			}

			// Apply fix-it engine
			if fix {
				doctor.FixItEngine(ctx, deepResults)
			}

			// Convert to doctorCheckResult
			var checks []doctorCheckResult
			for _, r := range deepResults {
				checks = append(checks, doctorCheckResult{
					Name:    fmt.Sprintf("[%s] %s", r.Section, r.Name),
					Status:  r.Status,
					Message: r.Message,
					Detail:  r.FixCmd,
				})
			}

			report := buildDoctorReport(checks)
			if jsonOut {
				return printDoctorJSON(report)
			}
			if !jsonOut {
				ui.CommandHeader("nSelf Doctor (Deep)", "All 12 subsystem checks")
			}
			printDoctorSummary(report)
			if report.Summary.Failed > 0 {
				return &plugin.ExitCodeError{Code: 1}
			}
			if report.Summary.Warnings > 0 {
				return &plugin.ExitCodeError{Code: 2}
			}
			return nil
		}

		if !jsonOut {
			ui.CommandHeader("nSelf Doctor", "System diagnostics")
		}

		var checks []doctorCheckResult

		// 1. Infrastructure checks
		if !jsonOut {
			ui.Section("Infrastructure")
		}
		checks = append(checks, checkDockerInstalled(verbose))
		checks = append(checks, checkDockerRunning(ctx, verbose))
		checks = append(checks, checkDockerComposeVersion(ctx, verbose))
		checks = append(checks, checkGitInstalled(verbose))

		// 2. Disk space
		if !jsonOut {
			ui.Section("Disk Space")
		}
		checks = append(checks, checkDiskSpace(verbose))

		// 3. Memory (full mode only — slower)
		if full {
			if !jsonOut {
				ui.Section("Memory")
			}
			checks = append(checks, checkMemory(verbose))
		}

		// 4. Network (full mode only — slower)
		if full {
			if !jsonOut {
				ui.Section("Network")
			}
			checks = append(checks, checkNetwork(ctx, verbose))
		}

		// 5. Port availability
		if !jsonOut {
			ui.Section("Port Availability")
		}
		checks = append(checks, checkPorts(verbose)...)
		checks = append(checks, checkServicePortConflicts(cwd, verbose)...)
		checks = append(checks, checkHomebrewPostgres(verbose)...)

		// 6. Configuration
		if !jsonOut {
			ui.Section("Configuration")
		}
		checks = append(checks, checkEnvExists(cwd, verbose))
		checks = append(checks, checkPasswordStrength(cwd, verbose, fix)...)
		checks = append(checks, checkJWTSecretPresent(cwd, verbose))

		// 7. Route consistency + port range sanity + config validators
		if !jsonOut {
			ui.Section("Config Validation")
		}
		checks = append(checks, checkRouteConsistency(cwd, verbose)...)
		checks = append(checks, checkPortRangeSanity(cwd, verbose)...)
		checks = append(checks, checkConfigValidators(cwd, verbose))

		// 8. Plugin compatibility
		if !jsonOut {
			ui.Section("Plugins")
		}
		checks = append(checks, checkPluginCompatibility(cwd, verbose)...)

		// 9. License cache state
		if !jsonOut {
			ui.Section("License")
		}
		checks = append(checks, checkLicenseCache(verbose))
		checks = append(checks, checkLicenseMigrationRate(verbose))

		// 9. Container health
		if !jsonOut {
			ui.Section("Container Health")
		}
		checks = append(checks, checkContainerHealth(ctx, cwd, verbose)...)

		// Build summary
		report := buildDoctorReport(checks)

		if jsonOut {
			return printDoctorJSON(report)
		}

		// Print summary
		printDoctorSummary(report)

		// Exit code: 1=failures, 2=warnings only, 0=all pass
		if report.Summary.Failed > 0 {
			return &plugin.ExitCodeError{Code: 1}
		}
		if report.Summary.Warnings > 0 {
			return &plugin.ExitCodeError{Code: 2}
		}
		return nil
	},
}

// checkDockerInstalled verifies the docker binary is on PATH.
func checkDockerInstalled(verbose bool) doctorCheckResult {
	name := "Docker installed"
	path, err := exec.LookPath("docker")
	if err != nil {
		printCheck("fail", name, "docker not found in PATH", verbose)
		return doctorCheckResult{Name: name, Status: "fail", Message: "docker not found in PATH"}
	}
	detail := ""
	if verbose {
		detail = path
	}
	printCheck("pass", name, "docker found", verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: "docker found", Detail: detail}
}

// checkDockerRunning verifies the Docker daemon is responsive.
func checkDockerRunning(ctx context.Context, verbose bool) doctorCheckResult {
	name := "Docker daemon running"
	cmd := exec.CommandContext(ctx, "docker", "info")
	out, err := cmd.CombinedOutput()
	if err != nil {
		printCheck("fail", name, "Docker daemon is not running", verbose)
		return doctorCheckResult{Name: name, Status: "fail", Message: "Docker daemon is not running"}
	}
	detail := ""
	if verbose {
		// Extract server version from docker info output
		for _, line := range strings.Split(string(out), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Server Version:") {
				detail = strings.TrimSpace(strings.TrimPrefix(trimmed, "Server Version:"))
				break
			}
		}
	}
	printCheck("pass", name, "Docker daemon is responsive", verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: "Docker daemon is responsive", Detail: detail}
}

// checkDockerComposeVersion verifies docker compose v2 is available and reports its version.
func checkDockerComposeVersion(ctx context.Context, verbose bool) doctorCheckResult {
	name := "Docker Compose v2"
	cmd := exec.CommandContext(ctx, "docker", "compose", "version", "--short")
	out, err := cmd.Output()
	if err != nil {
		printCheck("fail", name, "docker compose not available", verbose)
		return doctorCheckResult{Name: name, Status: "fail", Message: "docker compose not available"}
	}
	version := strings.TrimSpace(string(out))
	msg := fmt.Sprintf("docker compose %s", version)
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg, Detail: version}
}

// checkGitInstalled verifies git is on PATH.
func checkGitInstalled(verbose bool) doctorCheckResult {
	name := "Git installed"
	path, err := exec.LookPath("git")
	if err != nil {
		printCheck("warn", name, "git not found in PATH", verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: "git not found in PATH"}
	}
	detail := ""
	if verbose {
		detail = path
	}
	printCheck("pass", name, "git found", verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: "git found", Detail: detail}
}

// checkDiskSpace verifies at least 5 GB of free disk space.
// When --deep is active and disk usage exceeds 70%, it also appends a
// suggestion to enable the daily maintenance timer.
func checkDiskSpace(verbose bool) doctorCheckResult {
	name := "Disk space"
	freeGB, err := getFreeDiskGB()
	if err != nil {
		msg := fmt.Sprintf("unable to check disk space: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}
	msg := fmt.Sprintf("%.1f GB free", freeGB)
	if freeGB < 5.0 {
		// Also check used-percent so we can surface the maintenance suggestion.
		if usage, uerr := maintenance.GetDiskUsage(); uerr == nil && usage.UsedPercent > 70 {
			detail := fmt.Sprintf("disk is %d%% full — run `nself maintenance schedule --daily` to enable automatic daily cleanup", usage.UsedPercent)
			printCheck("warn", name, msg+" (recommended: 5 GB+) — "+detail, verbose)
			return doctorCheckResult{Name: name, Status: "warn", Message: msg + " (recommended: 5 GB+)", Detail: detail}
		}
		printCheck("warn", name, msg+" (recommended: 5 GB+)", verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg + " (recommended: 5 GB+)"}
	}
	// Even when free space is adequate, warn if disk is >70% full.
	if usage, uerr := maintenance.GetDiskUsage(); uerr == nil && usage.UsedPercent > 70 {
		detail := fmt.Sprintf("disk is %d%% full — run `nself maintenance schedule --daily` to enable automatic daily cleanup", usage.UsedPercent)
		printCheck("warn", name, msg+" — "+detail, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg, Detail: detail}
	}
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}

// checkMemory verifies at least 2 GB of total system memory.
func checkMemory(verbose bool) doctorCheckResult {
	name := "System memory"
	totalMB, err := getTotalMemoryMB()
	if err != nil {
		msg := fmt.Sprintf("unable to check memory: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}
	totalGB := float64(totalMB) / 1024.0
	msg := fmt.Sprintf("%.1f GB total", totalGB)
	if totalMB < 2048 {
		printCheck("warn", name, msg+" (recommended: 2 GB+)", verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg + " (recommended: 2 GB+)"}
	}
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}

// checkNetwork verifies internet connectivity by pinging Docker Hub.
func checkNetwork(ctx context.Context, verbose bool) doctorCheckResult {
	name := "Network / Docker Hub"
	cmd := exec.CommandContext(ctx, "docker", "pull", "--quiet", "hello-world")
	_, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: just try to reach the registry
		cmd2 := exec.CommandContext(ctx, "docker", "manifest", "inspect", "hello-world")
		_, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			printCheck("warn", name, "Docker Hub unreachable", verbose)
			return doctorCheckResult{Name: name, Status: "warn", Message: "Docker Hub unreachable"}
		}
	}
	printCheck("pass", name, "Docker Hub reachable", verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: "Docker Hub reachable"}
}

// checkPorts probes all reserved ports and reports conflicts.
func checkPorts(verbose bool) []doctorCheckResult {
	var results []doctorCheckResult
	conflicts, err := docker.CheckAllPorts(docker.ReservedPorts)
	if err != nil {
		name := "Port check"
		msg := fmt.Sprintf("error checking ports: %v", err)
		printCheck("warn", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "warn", Message: msg}}
	}

	if len(conflicts) == 0 {
		name := "Reserved ports"
		msg := fmt.Sprintf("all %d reserved ports available", len(docker.ReservedPorts))
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}

	// Report each conflicting port individually, with holder info when available.
	for _, c := range conflicts {
		name := fmt.Sprintf("Port %d", c.Port)
		holder, _ := ports.WhoHoldsPort(c.Port)
		msg := ports.FormatConflictMessage(c.Port, holder)
		printCheck("warn", name, msg, verbose)
		results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: msg})
	}
	return results
}

// checkEnvExists verifies that a .env file (or .env.dev) exists in the project directory.
func checkEnvExists(projectDir string, verbose bool) doctorCheckResult {
	name := ".env exists"
	envFiles := []string{".env", ".env.dev"}
	for _, f := range envFiles {
		path := filepath.Join(projectDir, f)
		if _, err := os.Stat(path); err == nil {
			msg := fmt.Sprintf("%s found", f)
			if verbose {
				printCheck("pass", name, msg, true)
			} else {
				printCheck("pass", name, msg, false)
			}
			return doctorCheckResult{Name: name, Status: "pass", Message: msg, Detail: path}
		}
	}
	printCheck("fail", name, "no .env or .env.dev found (run 'nself init')", verbose)
	return doctorCheckResult{Name: name, Status: "fail", Message: "no .env or .env.dev found (run 'nself init')"}
}

// checkPasswordStrength loads config and checks password fields for weakness.
func checkPasswordStrength(projectDir string, verbose, fix bool) []doctorCheckResult {
	var results []doctorCheckResult
	cfg, err := config.Load(projectDir)
	if err != nil {
		// Config load failed — cannot check passwords.
		name := "Password strength"
		msg := fmt.Sprintf("cannot load config: %v", err)
		printCheck("warn", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "warn", Message: msg}}
	}

	// Check each critical password field
	type pwField struct {
		Name   string
		Value  string
		MinLen int
	}

	fields := []pwField{
		{"POSTGRES_PASSWORD", cfg.Postgres.Password, 16},
		{"HASURA_GRAPHQL_ADMIN_SECRET", cfg.Hasura.AdminSecret, 32},
	}
	if cfg.Redis.Enabled {
		fields = append(fields, pwField{"REDIS_PASSWORD", cfg.Redis.Password, 16})
	}
	if cfg.Minio.Enabled {
		fields = append(fields, pwField{"MINIO_ROOT_PASSWORD", cfg.Minio.RootPassword, 16})
	}

	for _, f := range fields {
		name := fmt.Sprintf("Password: %s", f.Name)
		if f.Value == "" {
			printCheck("warn", name, "not set", verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: "not set"})
			continue
		}
		if len(f.Value) < f.MinLen {
			msg := fmt.Sprintf("too short (%d chars, need %d+)", len(f.Value), f.MinLen)
			printCheck("warn", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: msg})
			continue
		}
		if isWeakPassword(f.Value) {
			msg := "contains insecure pattern"
			if fix {
				msg += " (use 'nself init' to regenerate)"
			}
			printCheck("warn", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: msg})
			continue
		}
		printCheck("pass", name, "strong", verbose)
		results = append(results, doctorCheckResult{Name: name, Status: "pass", Message: "strong"})
	}

	// Warn when POSTGRES_USER is the default 'postgres' value in prod/staging.
	// The default is correct for dev; in production it is a predictable attack
	// surface. We do NOT change the default — only surface a warning.
	if cfg.Postgres.User == "postgres" {
		env := cfg.Env
		if env == "prod" || env == "staging" {
			name := "Postgres default credentials"
			msg := fmt.Sprintf("POSTGRES_USER is 'postgres' (the default) in %s — set a unique username to reduce predictable-credential risk", env)
			printCheck("warn", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: msg})
		}
	}

	return results
}

// isWeakPassword checks if a password contains common insecure substrings.
func isWeakPassword(value string) bool {
	insecure := []string{
		"password", "changeme", "secret", "admin",
		"12345", "qwerty", "default", "test",
		"postgres", "minioadmin", "hasura",
	}
	lower := strings.ToLower(value)
	for _, p := range insecure {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// checkJWTSecretPresent reports whether HASURA_GRAPHQL_JWT_SECRET is defined
// in the project's env files. Fails the command if absent everywhere.
// Hasura is always a core service in nSelf, so this check runs unconditionally.
func checkJWTSecretPresent(projectDir string, verbose bool) doctorCheckResult {
	r := doctor.CheckJWTSecretPresent(projectDir)
	printCheck(r.Status, r.Name, r.Message, verbose)
	return doctorCheckResult{Name: r.Name, Status: r.Status, Message: r.Message, Detail: r.FixCmd}
}

// checkContainerHealth inspects running containers and reports their health.
func checkContainerHealth(ctx context.Context, projectDir string, verbose bool) []doctorCheckResult {
	var results []doctorCheckResult
	compose := docker.NewCompose()

	containers, err := compose.ComposePs(ctx, projectDir)
	if err != nil {
		name := "Container health"
		// No compose project here — not necessarily an error
		msg := "no running containers found"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}

	if len(containers) == 0 {
		name := "Container health"
		msg := "no running containers"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}

	for _, c := range containers {
		name := fmt.Sprintf("Container: %s", c.Name)
		status := "pass"
		msg := c.State

		if c.Health != "" && c.Health != "none" {
			msg = fmt.Sprintf("%s (%s)", c.State, c.Health)
		}

		switch {
		case c.State != "running":
			status = "fail"
			msg = fmt.Sprintf("not running (state: %s)", c.State)
		case c.Health == "unhealthy":
			status = "fail"
			msg = "unhealthy"
			// Fetch recent logs for unhealthy containers when verbose
			if verbose {
				logs, logErr := docker.GetContainerLogs(ctx, c.Name, 5)
				if logErr == nil && logs != "" {
					msg += "\n    Recent logs:\n    " + strings.ReplaceAll(strings.TrimSpace(logs), "\n", "\n    ")
				}
			}
		case c.Health == "starting":
			status = "warn"
			msg = "still starting"
		}

		printCheck(status, name, msg, verbose)
		results = append(results, doctorCheckResult{Name: name, Status: status, Message: msg})
	}

	return results
}

// buildDoctorReport aggregates check results into a report with summary counts.
func buildDoctorReport(checks []doctorCheckResult) *doctorReport {
	report := &doctorReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Checks:    checks,
	}
	for _, c := range checks {
		report.Summary.Total++
		switch c.Status {
		case "pass":
			report.Summary.Passed++
		case "warn":
			report.Summary.Warnings++
		case "fail":
			report.Summary.Failed++
		}
	}
	return report
}

// printDoctorJSON renders the full report as JSON to stdout.
func printDoctorJSON(report *doctorReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

// printDoctorSummary prints a final summary box.
func printDoctorSummary(report *doctorReport) {
	fmt.Println()
	ui.Separator()

	items := []string{
		fmt.Sprintf("Total checks: %d", report.Summary.Total),
		fmt.Sprintf("Passed: %d", report.Summary.Passed),
	}
	if report.Summary.Warnings > 0 {
		items = append(items, fmt.Sprintf("Warnings: %d", report.Summary.Warnings))
	}
	if report.Summary.Failed > 0 {
		items = append(items, fmt.Sprintf("Failed: %d", report.Summary.Failed))
	}

	if report.Summary.Failed > 0 {
		ui.Error(fmt.Sprintf("Doctor found %d issue(s) that need attention", report.Summary.Failed))
	} else if report.Summary.Warnings > 0 {
		ui.Warn(fmt.Sprintf("Doctor found %d warning(s)", report.Summary.Warnings))
	} else {
		ui.Success("All checks passed")
	}

	for _, item := range items {
		ui.Bullet(item)
	}
	fmt.Println()
}

// printCheck renders a single check result to the terminal using ui.Checked / ui.Unchecked.
func printCheck(status, name, message string, verbose bool) {
	line := fmt.Sprintf("%s: %s", name, message)
	switch status {
	case "pass":
		ui.Checked(line)
	case "warn":
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.C(ui.Yellow, ui.IconWarning), line)
	case "fail":
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.C(ui.Red, ui.IconFailure), line)
	}
}

// ── New checks: T06 ──────────────────────────────────────────────────────────

// checkRouteConsistency verifies that ROUTE values for enabled services
// do not contain uppercase letters, spaces, or leading slashes. A valid route
// is a bare subdomain label like "api" or "auth-service".
func checkRouteConsistency(projectDir string, verbose bool) []doctorCheckResult {
	cfg, err := config.Load(projectDir)
	if err != nil {
		name := "Route consistency"
		msg := fmt.Sprintf("cannot load config: %v", err)
		printCheck("warn", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "warn", Message: msg}}
	}

	type namedRoute struct {
		label string
		value string
	}

	// Collect all configured route values with their label.
	routes := []namedRoute{
		{"HASURA_ROUTE", cfg.Hasura.Route},
		{"AUTH_ROUTE", cfg.Auth.Route},
	}
	if cfg.Minio.Enabled {
		routes = append(routes, namedRoute{"STORAGE_ROUTE", cfg.Minio.StorageRoute})
		routes = append(routes, namedRoute{"STORAGE_CONSOLE_ROUTE", cfg.Minio.ConsoleRoute})
	}
	if cfg.Admin.Enabled {
		routes = append(routes, namedRoute{"NSELF_ADMIN_ROUTE", cfg.Admin.Route})
	}
	if cfg.Search.Enabled {
		routes = append(routes, namedRoute{"SEARCH_ROUTE", cfg.Search.Route})
	}
	if cfg.Mailpit.Enabled {
		routes = append(routes, namedRoute{"MAILPIT_ROUTE", cfg.Mailpit.Route})
	}
	if cfg.Functions.Enabled {
		routes = append(routes, namedRoute{"FUNCTIONS_ROUTE", cfg.Functions.Route})
	}
	if cfg.MLflow.Enabled {
		routes = append(routes, namedRoute{"MLFLOW_ROUTE", cfg.MLflow.Route})
	}
	for _, cs := range cfg.CustomServices {
		if cs.Route != "" {
			routes = append(routes, namedRoute{fmt.Sprintf("CS_%d_ROUTE", cs.Index), cs.Route})
		}
	}
	for _, fa := range cfg.FrontendApps {
		if fa.Route != "" {
			routes = append(routes, namedRoute{fmt.Sprintf("FRONTEND_APP_%d_ROUTE", fa.Index), fa.Route})
		}
	}

	var results []doctorCheckResult
	for _, r := range routes {
		if r.value == "" {
			continue
		}
		name := fmt.Sprintf("Route: %s", r.label)
		corrected := routeCorrect(r.value)
		if corrected != r.value {
			msg := fmt.Sprintf("%q has invalid format — suggested value: %q", r.value, corrected)
			printCheck("warn", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: msg})
		} else {
			msg := fmt.Sprintf("%q is valid", r.value)
			printCheck("pass", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "pass", Message: msg})
		}
	}

	if len(results) == 0 {
		name := "Route consistency"
		msg := "no routes configured"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}
	return results
}

// routeCorrect returns a corrected form of a ROUTE value: lowercase, no
// leading slashes, no spaces. This mirrors what RouteToFQDN would expect.
func routeCorrect(route string) string {
	corrected := strings.TrimLeft(route, "/")
	corrected = strings.TrimSpace(corrected)
	corrected = strings.ToLower(corrected)
	corrected = strings.ReplaceAll(corrected, " ", "-")
	return corrected
}

// checkPortRangeSanity warns when any configured port is below 1024
// (privileged port) when the process is not running as root.
func checkPortRangeSanity(projectDir string, verbose bool) []doctorCheckResult {
	cfg, err := config.Load(projectDir)
	if err != nil {
		name := "Port range sanity"
		msg := fmt.Sprintf("cannot load config: %v", err)
		printCheck("warn", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "warn", Message: msg}}
	}

	// Only warn when not running as root (uid != 0).
	isRoot := os.Getuid() == 0

	type namedPort struct {
		label string
		port  int
	}

	// Well-known ports 80 and 443 are expected privileged ports (Nginx) — skip them.
	ports := []namedPort{
		{"POSTGRES_PORT", cfg.Postgres.Port},
		{"HASURA_PORT", cfg.Hasura.Port},
		{"AUTH_PORT", cfg.Auth.Port},
	}
	if cfg.Redis.Enabled {
		ports = append(ports, namedPort{"REDIS_PORT", cfg.Redis.Port})
	}
	if cfg.Minio.Enabled {
		ports = append(ports, namedPort{"MINIO_PORT", cfg.Minio.Port})
		ports = append(ports, namedPort{"MINIO_CONSOLE_PORT", cfg.Minio.ConsolePort})
	}
	if cfg.Search.Enabled {
		ports = append(ports, namedPort{"SEARCH_PORT", cfg.Search.Port})
	}
	if cfg.Functions.Enabled {
		ports = append(ports, namedPort{"FUNCTIONS_PORT", cfg.Functions.Port})
	}
	if cfg.MLflow.Enabled {
		ports = append(ports, namedPort{"MLFLOW_PORT", cfg.MLflow.Port})
	}
	if cfg.Admin.Enabled {
		ports = append(ports, namedPort{"NSELF_ADMIN_PORT", cfg.Admin.Port})
	}
	for _, cs := range cfg.CustomServices {
		if cs.Port != 0 {
			ports = append(ports, namedPort{fmt.Sprintf("CS_%d_PORT", cs.Index), cs.Port})
		}
	}

	var results []doctorCheckResult
	for _, p := range ports {
		if p.port == 0 || p.port == 80 || p.port == 443 {
			continue
		}
		name := fmt.Sprintf("Port range: %s", p.label)
		if p.port < 1024 && !isRoot {
			msg := fmt.Sprintf("port %d is privileged (<1024) and may require root to bind", p.port)
			printCheck("warn", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: msg})
		} else {
			msg := fmt.Sprintf("port %d OK", p.port)
			printCheck("pass", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "pass", Message: msg})
		}
	}

	if len(results) == 0 {
		name := "Port range sanity"
		msg := "all configured ports are in the unprivileged range"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}
	return results
}

// checkConfigValidators runs config.Validate against the loaded configuration
// and reports the result. T04 will wire Validate() to call RunAll() internally,
// so this will automatically cover all registered validators after T04 lands.
func checkConfigValidators(projectDir string, verbose bool) doctorCheckResult {
	name := "Config validators"
	cfg, err := config.Load(projectDir)
	if err != nil {
		msg := fmt.Sprintf("cannot load config: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	if err := config.Validate(cfg); err != nil {
		msg := fmt.Sprintf("config validation failed: %v", err)
		printCheck("fail", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "fail", Message: msg}
	}

	msg := "config validators passed"
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}

// licensestaleDays is the number of days after which the license cache is
// considered stale.
const licensestaleDays = 7

// checkLicenseCache inspects the license entitlements cache. It reports the
// cache age and tier when present, and warns if the cache is older than 7 days.
func checkLicenseCache(verbose bool) doctorCheckResult {
	name := "License cache"

	// Use the public LicenseCacheDir helper so the path stays consistent
	// with the plugin manager.
	cacheDir := plugin.LicenseCacheDir()
	entitlementsPath := filepath.Join(cacheDir, "entitlements.json")

	data, err := os.ReadFile(entitlementsPath)
	if os.IsNotExist(err) {
		// No cache file — not an error, just informational.
		msg := "no license cache found (run 'nself license validate' to populate)"
		printCheck("pass", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}
	if err != nil {
		msg := fmt.Sprintf("cannot read license cache: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	// Parse the entitlements JSON (tier + cached_at).
	var cache struct {
		Tier     string `json:"tier"`
		CachedAt string `json:"cached_at"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		msg := fmt.Sprintf("cannot parse license cache: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	cachedAt, err := time.Parse(time.RFC3339, cache.CachedAt)
	if err != nil {
		msg := fmt.Sprintf("cannot parse cache timestamp: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	age := time.Since(cachedAt)
	ageDays := int(age.Hours() / 24)
	tier := cache.Tier
	if tier == "" {
		tier = "unknown"
	}

	if ageDays >= licensestaleDays {
		msg := fmt.Sprintf("tier=%s, cache age=%dd — license cache is stale — run 'nself license refresh'", tier, ageDays)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	msg := fmt.Sprintf("tier=%s, cache age=%dd", tier, ageDays)
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}

// checkLicenseMigrationRate implements the LIC-MIGRATION-01 doctor check (SP-04.O11 T11).
//
// After migration has been running for 60+ days, warns if more than 10% of
// daily license validations are still using unmigrated (legacy) keys.
// Data is read from the ping_api telemetry endpoint if NSELF_PING_API_URL is set,
// otherwise the check is skipped (non-fatal — only prod infra exposes telemetry).
func checkLicenseMigrationRate(verbose bool) doctorCheckResult {
	name := "LIC-MIGRATION-01: License migration rate"

	pingURL := os.Getenv("NSELF_PING_API_URL")
	if pingURL == "" {
		pingURL = defaultPingURL
	}

	// Query the migration telemetry summary endpoint (admin-only, only available
	// when DATABASE_URL is configured on the server — not reachable from end-user CLI).
	// We attempt a HEAD to confirm the endpoint exists; if unreachable we skip.
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(pingURL + "/admin/migration/status")
	if err != nil {
		// Unreachable — not an error for end-user CLI, skip silently.
		msg := "migration telemetry endpoint unreachable — skipped"
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		// Endpoint exists but requires admin auth — skip (not an error for end-user).
		msg := "migration telemetry requires admin access — skipped"
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("migration telemetry returned HTTP %d — skipped", resp.StatusCode)
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	var telemetry struct {
		TotalHits          int     `json:"total_hits"`
		PendingHits        int     `json:"pending_hits"`
		MigratedHits       int     `json:"migrated_hits"`
		MigrationStartDate string  `json:"migration_start_date"`
		PendingRatioPct    float64 `json:"pending_ratio_pct"`
		DaysSinceMigration int     `json:"days_since_migration"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&telemetry); err != nil {
		msg := fmt.Sprintf("cannot parse migration telemetry: %v", err)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	// Only alert after 60 days since migration start (spec: acceptance criterion 7).
	if telemetry.DaysSinceMigration < 60 {
		msg := fmt.Sprintf("migration running for %d days — alert threshold not yet reached (60 days)", telemetry.DaysSinceMigration)
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	if telemetry.TotalHits == 0 {
		msg := "no validation hits recorded today"
		if verbose {
			printCheck("pass", name, msg, verbose)
		}
		return doctorCheckResult{Name: name, Status: "pass", Message: msg}
	}

	ratio := telemetry.PendingRatioPct
	if ratio > 10.0 {
		msg := fmt.Sprintf("%.1f%% of daily license validations are still using unmigrated keys (threshold: 10%%) — run: nself license migrate --account-id <uuid>", ratio)
		printCheck("warn", name, msg, verbose)
		return doctorCheckResult{Name: name, Status: "warn", Message: msg}
	}

	msg := fmt.Sprintf("%.1f%% pending ratio — within threshold", ratio)
	printCheck("pass", name, msg, verbose)
	return doctorCheckResult{Name: name, Status: "pass", Message: msg}
}

// checkPluginCompatibility verifies installed plugins are compatible with the current CLI version.
func checkPluginCompatibility(projectDir string, verbose bool) []doctorCheckResult {
	pluginDir := resolvePluginDir()
	plugins, err := plugin.ListInstalled(pluginDir)
	if err != nil {
		name := "Plugin compatibility"
		msg := fmt.Sprintf("cannot list plugins: %v", err)
		printCheck("warn", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "warn", Message: msg}}
	}

	if len(plugins) == 0 {
		name := "Plugin compatibility"
		msg := "no plugins installed"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}

	var results []doctorCheckResult
	for _, p := range plugins {
		name := fmt.Sprintf("Plugin: %s", p.Name)
		msg := fmt.Sprintf("v%s (%s)", p.Version, p.Status)
		if p.Status == "error" || p.Status == "failed" {
			printCheck("fail", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "fail", Message: msg})
		} else {
			printCheck("pass", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "pass", Message: msg})
		}
	}
	return results
}

// checkServicePortConflicts probes configured service ports against enabled services.
// It catches conflicts between nSelf services (Grafana on 3000, Admin on 3021, etc.)
// and local dev servers that may already be listening.
func checkServicePortConflicts(projectDir string, verbose bool) []doctorCheckResult {
	cfg, err := config.Load(projectDir)
	if err != nil {
		name := "Service port conflicts"
		msg := fmt.Sprintf("cannot load config: %v", err)
		printCheck("warn", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "warn", Message: msg}}
	}

	type servicePort struct {
		name string
		port int
	}
	var ports []servicePort

	if cfg.Monitoring.GrafanaEnabled {
		ports = append(ports, servicePort{"Grafana", cfg.Monitoring.GrafanaPort})
	}
	if cfg.Admin.Enabled {
		ports = append(ports, servicePort{"nSelf Admin", cfg.Admin.Port})
	}
	if cfg.Mailpit.Enabled {
		ports = append(ports, servicePort{"Mailpit UI", cfg.Mailpit.UIPort})
	}
	if cfg.Functions.Enabled {
		ports = append(ports, servicePort{"Functions", cfg.Functions.Port})
	}
	if cfg.MLflow.Enabled {
		ports = append(ports, servicePort{"MLflow", cfg.MLflow.Port})
	}

	if len(ports) == 0 {
		name := "Service port conflicts"
		msg := "no services with dev-port conflict risk enabled"
		printCheck("pass", name, msg, verbose)
		return []doctorCheckResult{{Name: name, Status: "pass", Message: msg}}
	}

	var results []doctorCheckResult
	for _, sp := range ports {
		if sp.port == 0 {
			continue
		}
		name := fmt.Sprintf("Port %d (%s)", sp.port, sp.name)
		inUse, err := docker.CheckPort(sp.port)
		if err != nil {
			printCheck("warn", name, fmt.Sprintf("cannot check port: %v", err), verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: fmt.Sprintf("cannot check port: %v", err)})
			continue
		}
		if inUse {
			msg := fmt.Sprintf("Warning: port %d (%s) is already in use by another process", sp.port, sp.name)
			printCheck("warn", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "warn", Message: msg})
		} else {
			msg := fmt.Sprintf("port %d (%s) is available", sp.port, sp.name)
			printCheck("pass", name, msg, verbose)
			results = append(results, doctorCheckResult{Name: name, Status: "pass", Message: msg})
		}
	}
	return results
}

func init() {
	doctorCmd.Flags().Bool("verbose", false, "Detailed diagnostics")
	doctorCmd.Flags().Bool("full", false, "Run all checks including network and memory (slower)")
	doctorCmd.Flags().Bool("deep", false, "Alias for --full (run all checks)")
	doctorCmd.Flags().Bool("fix", false, "Auto-fix safe issues")
	doctorCmd.Flags().Bool("json", false, "JSON output")
	doctorCmd.Flags().String("section", "", "Run only a specific section (system, core, backups, license, plugins, monitoring, security)")
	doctorCmd.Flags().StringSlice("skip", nil, "Skip specific check sections")
	doctorCmd.Flags().Bool("alerts", false, "Check monitoring alert rules are loaded")
	doctorCmd.Flags().String("format", "", "Output format: json, text (default text)")
	doctorCmd.Flags().String("only", "", "Run only a specific subsystem check (host, docker, postgres, hasura, nginx, ssl, ping, plugins, license, monitoring, backups, security)")
	// AI wizard flags (T-05-01, T-05-02)
	doctorCmd.Flags().Bool("ai", false, "Run the AI first-run wizard (install Ollama, setup Gemini pool, verify)")
	doctorCmd.Flags().Bool("yes", false, "Non-interactive mode: accept all defaults (for CI/scripts)")
	doctorCmd.Flags().Bool("skip-ollama", false, "Skip local Ollama installation step")
	doctorCmd.Flags().Bool("skip-pool", false, "Skip Gemini pool setup step")
	doctorCmd.Flags().Bool("headless", false, "Print OAuth URL instead of opening browser (for SSH/headless servers)")
	doctorCmd.Flags().Bool("check-legacy", false, "Scan host for v0.9 stale paths (global scan, not per-project)")
	doctorCmd.Flags().Bool("install-check", false, "Run 6-stage onboarding funnel check (used by Homebrew post-install hook)")
	RootCmd.AddCommand(doctorCmd)
}

// runDoctorCheckLegacy implements `nself doctor --check-legacy`.
// It scans global host paths for v0.9 stale artifacts (NOT per-project).
// Returns exit code 0 on clean install, non-zero with structured output on findings.
func runDoctorCheckLegacy() error {
	artifacts := migration.ScanLegacyPaths()

	if len(artifacts) == 0 {
		ui.Success("No v0.9 global artifacts detected. Clean install.")
		return nil
	}

	ui.Warn(fmt.Sprintf("Found %d v0.9 stale artifact(s) on this host:", len(artifacts)))
	fmt.Println()
	for i, a := range artifacts {
		fmt.Printf("  %d. %s\n", i+1, a.Path)
		fmt.Printf("     Kind: %s\n", a.Kind)
		fmt.Printf("     Fix:  %s\n", a.Hint)
		fmt.Println()
	}
	fmt.Println("Run the cleanup commands above, then re-run `nself doctor --check-legacy` to verify.")
	return fmt.Errorf("v0.9 stale artifacts found (%d path(s))", len(artifacts))
}

// ── Onboarding funnel check (Q08) ────────────────────────────────────────────

// installCheckJSONReport is the machine-readable shape for --install-check --json.
type installCheckJSONReport struct {
	Stages   []installCheckJSONStage `json:"stages"`
	Position int                     `json:"funnel_position"`
	Next     string                  `json:"next_action"`
}

type installCheckJSONStage struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	Remediation string                 `json:"remediation,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// runInstallCheck implements `nself doctor --install-check`.
//
// It runs the 6-stage onboarding funnel check, prints a friendly checklist,
// fires telemetry events for each passing stage (opt-in), and exits:
//
//	0 — all 6 stages pass
//	1 — one or more stages fail
func runInstallCheck(jsonOut bool) error {
	report := onboarding.RunFunnel()

	if jsonOut {
		return printInstallCheckJSON(report)
	}

	// ── Human-readable output ────────────────────────────────────────────────
	ui.CommandHeader("nSelf Doctor — Onboarding Funnel", "6-stage install readiness check")
	fmt.Println()

	for _, s := range report.Stages {
		label := fmt.Sprintf("Stage %d — %-12s", s.ID, s.Name)
		switch s.Status {
		case onboarding.StatusPass:
			fmt.Printf("  %s %s  %s\n", ui.C(ui.Green, ui.IconSuccess), label, s.Message)
		case onboarding.StatusFail:
			fmt.Fprintf(os.Stderr, "  %s %s  %s\n", ui.C(ui.Red, ui.IconFailure), label, s.Message)
			if s.Remediation != "" {
				fmt.Fprintf(os.Stderr, "    %s %s\n", ui.C(ui.Cyan, "→"), s.Remediation)
			}
		case onboarding.StatusUnknown:
			fmt.Fprintf(os.Stderr, "  %s %s  %s\n", ui.C(ui.Yellow, ui.IconWarning), label, s.Message)
		case onboarding.StatusSkipped:
			fmt.Printf("  %s %s  %s\n", ui.C(ui.Dim, "—"), label, s.Message)
		}
	}

	fmt.Println()
	ui.Separator()
	posStr := fmt.Sprintf("Funnel position: Stage %d/6.", report.Position)
	if report.Position == 6 {
		ui.Success(posStr + " Onboarding complete.")
	} else {
		ui.Info(posStr + " Next: " + report.Next)
	}
	fmt.Println()

	// Exit 1 if any stage failed (skipped does not count as failure for exit code).
	for _, s := range report.Stages {
		if s.Status == onboarding.StatusFail {
			return &plugin.ExitCodeError{Code: 1}
		}
	}
	return nil
}

// printInstallCheckJSON renders the funnel report as JSON.
func printInstallCheckJSON(report onboarding.FunnelReport) error {
	out := installCheckJSONReport{
		Position: report.Position,
		Next:     report.Next,
	}
	for _, s := range report.Stages {
		out.Stages = append(out.Stages, installCheckJSONStage{
			ID:          s.ID,
			Name:        s.Name,
			Status:      string(s.Status),
			Message:     s.Message,
			Remediation: s.Remediation,
			Metadata:    s.Metadata,
		})
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}
