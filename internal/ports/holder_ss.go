package ports

// Purpose: the `ss`-based port-holder lookup (fallback for systems without lsof) plus small helpers (cmdline reading, container detection, OS check, env-var naming) backing WhoHoldsPort in holder.go.
// Inputs: a TCP port number and, for the helpers, a process PID.
// Outputs: a *Holder describing the listening process, or nil/error when not found.
// Constraints: split out of holder.go as a pure move (CLI-R12); no behavior change.

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// whoHoldsPortSS uses `ss -tlnp sport = :<port>` to find the listener.
// ss output for a listening socket looks like:
//
//	State   Recv-Q Send-Q Local Address:Port Peer Address:Port Process
//	LISTEN  0      128    0.0.0.0:5432      0.0.0.0:*         users:(("postgres",pid=1234,fd=4))
func whoHoldsPortSS(port int) (*Holder, error) {
	portStr := strconv.Itoa(port)
	cmd := exec.Command("ss", "-tlnp", "sport", "=", ":"+portStr)
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("ss not found in PATH")
		}
		return nil, fmt.Errorf("running ss: %w", err)
	}

	return parseSSOutput(out, port)
}

// parseSSOutput extracts PID and process name from ss output.
// The Process column has the form: users:(("name",pid=N,fd=M))
func parseSSOutput(out []byte, port int) (*Holder, error) {
	portSuffix := ":" + strconv.Itoa(port)

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		// Only look at LISTEN lines that reference our port.
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		if !strings.Contains(line, portSuffix) {
			continue
		}
		// Extract users:((...)) section.
		usersIdx := strings.Index(line, "users:((")
		if usersIdx < 0 {
			continue
		}
		usersPart := line[usersIdx:]

		name, pid, err := parseSSUsers(usersPart)
		if err != nil {
			continue
		}

		cmdline := readCmdline(pid)
		return &Holder{
			PID:     pid,
			Name:    name,
			Command: cmdline,
			IsOurs:  isOurContainer(pid),
		}, nil
	}

	return nil, fmt.Errorf("ss: no listener found on port %d", port)
}

// parseSSUsers parses users:(("postgres",pid=1234,fd=4)) and returns name, pid.
func parseSSUsers(s string) (string, int, error) {
	// Isolate the inner content between the outer parens.
	start := strings.Index(s, "((")
	end := strings.Index(s, "))")
	if start < 0 || end < 0 || end <= start {
		return "", 0, fmt.Errorf("unexpected users format: %s", s)
	}
	inner := s[start+2 : end] // e.g. "postgres",pid=1234,fd=4

	// Extract name (first quoted field).
	var name string
	if len(inner) > 0 && inner[0] == '"' {
		closeQ := strings.Index(inner[1:], "\"")
		if closeQ >= 0 {
			name = inner[1 : closeQ+1]
		}
	}

	// Extract pid=N.
	var pid int
	for _, field := range strings.Split(inner, ",") {
		field = strings.TrimSpace(field)
		if strings.HasPrefix(field, "pid=") {
			n, err := strconv.Atoi(field[4:])
			if err == nil {
				pid = n
			}
		}
	}

	if pid == 0 {
		return "", 0, fmt.Errorf("pid not found in: %s", inner)
	}

	return name, pid, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// readCmdline reads /proc/<pid>/cmdline on Linux to get the full command line.
// On macOS /proc doesn't exist; returns an empty string gracefully.
func readCmdline(pid int) string {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// /proc/PID/cmdline uses NUL bytes as arg separators.
	return strings.ReplaceAll(string(data), "\x00", " ")
}

// isOurContainer returns true if the process identified by pid belongs to a
// Docker container that nSelf launched. Detection logic:
//
//   - On Linux: read /proc/<pid>/cgroup and check for "docker" substring.
//   - On macOS (/proc absent): skip and return false.
func isOurContainer(pid int) bool {
	path := fmt.Sprintf("/proc/%d/cgroup", pid)
	data, err := os.ReadFile(path)
	if err != nil {
		// /proc not available (macOS) or process gone.
		return false
	}
	return strings.Contains(string(data), "docker")
}

// isLinux returns true when the current OS is Linux (i.e. /proc exists).
func isLinux() bool {
	_, err := os.Stat("/proc")
	return err == nil
}

// portEnvVar maps well-known nSelf ports to their .env variable name so the
// conflict message can include a targeted hint.
func portEnvVar(port int) string {
	switch port {
	case 80:
		return "NGINX_HTTP_PORT"
	case 443:
		return "NGINX_HTTPS_PORT"
	case 5432:
		return "POSTGRES_PORT"
	case 8080:
		return "HASURA_PORT"
	case 4000:
		return "AUTH_PORT"
	case 6379:
		return "REDIS_PORT"
	case 9000:
		return "MINIO_PORT"
	case 9001:
		return "MINIO_CONSOLE_PORT"
	case 7700:
		return "SEARCH_PORT"
	case 3021:
		return "NSELF_ADMIN_PORT"
	case 1025:
		return "MAILPIT_SMTP_PORT"
	case 8025:
		return "MAILPIT_UI_PORT"
	case 3008:
		return "FUNCTIONS_PORT"
	case 5000:
		return "MLFLOW_PORT"
	}
	return ""
}
