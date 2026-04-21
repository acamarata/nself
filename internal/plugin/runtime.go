package plugin

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// PluginStatus describes the current state of a plugin process.
type PluginStatus struct {
	Name  string
	State string // starting, running, stopping, stopped, failed
	PID   int
}

// runtimeBase returns the base directory for plugin runtime files.
func runtimeBase() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "runtime")
	}
	return filepath.Join(home, ".nself", "runtime")
}

// pidPath returns the file path for a plugin's PID file.
func pidPath(name string) string {
	return filepath.Join(runtimeBase(), "pids", name+".pid")
}

// statePath returns the file path for a plugin's state file.
func statePath(name string) string {
	return filepath.Join(runtimeBase(), "states", name+".state")
}

// logPath returns the file path for a plugin's log file.
func logPath(name string) string {
	return filepath.Join(runtimeBase(), "logs", name+".log")
}

// writeState writes the given state string to the plugin's state file.
func writeState(name string, state string) error {
	dir := filepath.Dir(statePath(name))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	return os.WriteFile(statePath(name), []byte(state), 0644)
}

// writePID writes the given PID to the plugin's PID file.
func writePID(name string, pid int) error {
	dir := filepath.Dir(pidPath(name))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating pid directory: %w", err)
	}
	return os.WriteFile(pidPath(name), []byte(strconv.Itoa(pid)), 0644)
}

// readPID reads the PID from the plugin's PID file.
func readPID(name string) (int, error) {
	data, err := os.ReadFile(pidPath(name))
	if err != nil {
		return 0, fmt.Errorf("reading pid file for %s: %w", name, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parsing pid for %s: %w", name, err)
	}
	return pid, nil
}

// readState reads the state from the plugin's state file.
func readState(name string) string {
	data, err := os.ReadFile(statePath(name))
	if err != nil {
		return "stopped"
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return "stopped"
	}
	return s
}

// isProcessRunning checks whether a process with the given PID is alive.
func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 tests for process existence without actually sending a signal.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// Start launches a plugin process in the background. It locates the plugin's
// entry point inside pluginDir, starts it, writes the PID to
// ~/.nself/runtime/pids/{name}.pid, and records the state in
// ~/.nself/runtime/states/{name}.state.
func Start(ctx context.Context, pluginDir string, name string) error {
	// Check if already running.
	if pid, err := readPID(name); err == nil && isProcessRunning(pid) {
		return fmt.Errorf("plugin %s is already running (pid %d)", name, pid)
	}

	// Set state to starting.
	if err := writeState(name, "starting"); err != nil {
		return fmt.Errorf("setting starting state for %s: %w", name, err)
	}

	// Determine entry point from manifest if available.
	entryPoint := ""
	manifestPath := filepath.Join(pluginDir, "plugin.json")
	var manifest *PluginManifest
	if m, err := parseManifest(manifestPath); err == nil {
		manifest = m
		if m.EntryPoint != "" {
			entryPoint = filepath.Join(pluginDir, m.EntryPoint)
		}
	}

	// Build the command to execute.
	var cmd *exec.Cmd
	if entryPoint != "" {
		// Detect runtime from file extension.
		switch {
		case strings.HasSuffix(entryPoint, ".js"):
			cmd = exec.CommandContext(ctx, "node", entryPoint)
		case strings.HasSuffix(entryPoint, ".ts"):
			cmd = exec.CommandContext(ctx, "npx", "tsx", entryPoint)
		default:
			// Assume a binary.
			cmd = exec.CommandContext(ctx, entryPoint)
		}
	} else {
		// Fallback: look for an executable named after the plugin.
		binPath := filepath.Join(pluginDir, name)
		if _, err := os.Stat(binPath); err != nil {
			_ = writeState(name, "failed")
			return fmt.Errorf("no entry point found for plugin %s", name)
		}
		cmd = exec.CommandContext(ctx, binPath)
	}

	cmd.Dir = pluginDir

	// Load plugin .env if present.
	envFile := filepath.Join(pluginDir, ".env")
	cmd.Env = os.Environ()
	if data, err := os.ReadFile(envFile); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			cmd.Env = append(cmd.Env, line)
		}
	}

	// Redirect stdout and stderr to the log file.
	logDir := filepath.Dir(logPath(name))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		_ = writeState(name, "failed")
		return fmt.Errorf("creating log directory: %w", err)
	}

	logFile, err := os.OpenFile(logPath(name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		_ = writeState(name, "failed")
		return fmt.Errorf("opening log file for %s: %w", name, err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the process in a new process group so we can kill children later.
	// setNewProcessGroup is OS-specific (see runtime_unix.go / runtime_windows.go).
	setNewProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		logFile.Close()
		_ = writeState(name, "failed")
		return fmt.Errorf("starting plugin %s: %w", name, err)
	}

	pid := cmd.Process.Pid
	logFile.Close()

	// Write PID file.
	if err := writePID(name, pid); err != nil {
		_ = writeState(name, "failed")
		return fmt.Errorf("writing pid for %s: %w", name, err)
	}

	// Poll for the plugin to become healthy (or confirm the process is alive
	// when no health endpoint is available). Timeout: 10 seconds at 500ms
	// intervals.
	const (
		healthTimeout  = 10 * time.Second
		healthInterval = 500 * time.Millisecond
	)

	deadline := time.Now().Add(healthTimeout)
	var lastErr error
	healthy := false

	for time.Now().Before(deadline) {
		// Check context cancellation first.
		select {
		case <-ctx.Done():
			_ = writeState(name, "failed")
			_ = os.Remove(pidPath(name))
			return fmt.Errorf("plugin %s start cancelled: %w", name, ctx.Err())
		default:
		}

		// If the process has already exited, fail fast.
		if !isProcessRunning(pid) {
			_ = writeState(name, "failed")
			_ = os.Remove(pidPath(name))
			return fmt.Errorf("plugin %s exited immediately after start", name)
		}

		// If the manifest declared a port, verify via HTTP health check.
		if manifest != nil && manifest.Port > 0 {
			ok, err := health(ctx, name, manifest.Port)
			if ok {
				healthy = true
				break
			}
			if err != nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("plugin %s health endpoint returned non-200", name)
			}
		} else {
			// No port declared — process alive is sufficient.
			healthy = true
			break
		}

		time.Sleep(healthInterval)
	}

	// If we exhausted the deadline without a healthy response, report failure.
	if !healthy {
		if lastErr == nil {
			lastErr = fmt.Errorf("timed out waiting for plugin to become healthy")
		}
		_ = writeState(name, "failed")
		_ = os.Remove(pidPath(name))
		return fmt.Errorf("plugin %s failed health check after start: %w", name, lastErr)
	}

	if err := writeState(name, "running"); err != nil {
		return fmt.Errorf("setting running state for %s: %w", name, err)
	}

	// Check plugin interface contract compliance (non-blocking).
	// Logs warnings for non-compliant plugins but does not prevent startup.
	if manifest != nil && manifest.Port > 0 {
		CheckCompliance(ctx, name, manifest.Port)
	}

	return nil
}

// gracePeriod returns the SIGTERM-to-SIGKILL grace window for plugin shutdown.
// It reads DOCKER_STOP_GRACE_PERIOD env var (e.g. "30s", "10s") to match the
// compose layer's stop_grace_period setting. Defaults to 30s if unset or invalid.
// Bounded to [1s, 120s] to prevent accidental runaway or instant-kill values.
// DEP-03: fixes hardcoded 5s that conflicted with compose 30s stop_grace_period.
func gracePeriod() time.Duration {
	const defaultGrace = 30 * time.Second
	const maxGrace = 120 * time.Second
	const minGrace = 1 * time.Second

	raw := strings.TrimSpace(os.Getenv("DOCKER_STOP_GRACE_PERIOD"))
	if raw == "" {
		return defaultGrace
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		// Invalid value: fall back to default with a silent warning.
		// The caller will apply the default; misconfiguration is non-fatal.
		return defaultGrace
	}
	if d < minGrace {
		return minGrace
	}
	if d > maxGrace {
		return maxGrace
	}
	return d
}

// Stop gracefully shuts down a running plugin. It sends SIGTERM first, waits
// up to gracePeriod() for the process to exit, then sends SIGKILL if it is
// still alive. The grace window respects DOCKER_STOP_GRACE_PERIOD env var
// to match the compose layer's stop_grace_period (default 30s). PID and
// state files are cleaned up.
func Stop(ctx context.Context, name string) error {
	pid, err := readPID(name)
	if err != nil {
		return fmt.Errorf("plugin %s is not running: %w", name, err)
	}

	if !isProcessRunning(pid) {
		// Process already gone. Clean up stale files.
		_ = os.Remove(pidPath(name))
		_ = writeState(name, "stopped")
		return nil
	}

	_ = writeState(name, "stopping")

	// Send graceful stop signal to the process group (SIGTERM on Unix,
	// taskkill /T on Windows). Implementation is OS-specific.
	_ = terminateProcessGroup(pid)

	// Wait up to gracePeriod() for graceful shutdown.
	// DEP-03: reads DOCKER_STOP_GRACE_PERIOD env var; defaults to 30s.
	deadline := time.Now().Add(gracePeriod())
	for time.Now().Before(deadline) {
		if !isProcessRunning(pid) {
			break
		}
		// Check context cancellation.
		select {
		case <-ctx.Done():
			// Force kill on context cancellation.
			_ = killProcessGroup(pid)
			_ = os.Remove(pidPath(name))
			_ = writeState(name, "stopped")
			return ctx.Err()
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}

	// If still running after graceful timeout, hard kill.
	if isProcessRunning(pid) {
		_ = killProcessGroup(pid)
		// Brief wait for the kill signal to take effect.
		time.Sleep(200 * time.Millisecond)
	}

	// Clean up.
	_ = os.Remove(pidPath(name))
	_ = writeState(name, "stopped")

	return nil
}

// Status returns the current status of a plugin by reading its PID and state
// files. If the PID file indicates a process that is no longer running, the
// state is corrected to "stopped".
func Status(name string) (*PluginStatus, error) {
	state := readState(name)
	pid := 0

	if p, err := readPID(name); err == nil {
		pid = p
		// Reconcile: if state claims running but the process is dead, correct it.
		if (state == "running" || state == "starting") && !isProcessRunning(pid) {
			state = "stopped"
			_ = writeState(name, "stopped")
			_ = os.Remove(pidPath(name))
			pid = 0
		}
	} else {
		// No PID file. If state claims running, correct it.
		if state == "running" || state == "starting" {
			state = "stopped"
			_ = writeState(name, "stopped")
		}
	}

	return &PluginStatus{
		Name:  name,
		State: state,
		PID:   pid,
	}, nil
}

// health performs an HTTP health check against a plugin by sending a GET
// request to http://localhost:{port}/health. It returns true if the response
// status is 200 OK.
func health(ctx context.Context, name string, port int) (bool, error) {
	url := fmt.Sprintf("http://localhost:%d/health", port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("creating health request for %s: %w", name, err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
