// Package ports provides port-holder identification utilities for the nSelf CLI.
// It augments the raw TCP port probing in internal/docker with process-level
// information so that conflict error messages name the offending process.
package ports

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

// Holder describes the process that is listening on a TCP port.
type Holder struct {
	PID     int
	Name    string // process name (short, e.g. "postgres")
	Command string // full command line
	IsOurs  bool   // true if the holder is one of our own Docker containers
}

// WhoHoldsPort returns the process holding the given TCP port.
// Returns nil (not an error) if the port is free or if holder detection
// is unavailable on the current platform (e.g. lsof/ss not installed).
// A non-nil error is returned only for unexpected failures after a holder
// was partially identified.
func WhoHoldsPort(port int) (*Holder, error) {
	// Primary: lsof — available on macOS and most Linux distributions.
	holder, err := whoHoldsPortLsof(port)
	if err == nil {
		return holder, nil
	}

	// Fallback: ss — Linux only. Silently skip on macOS where ss is absent.
	if isLinux() {
		holder, err = whoHoldsPortSS(port)
		if err == nil {
			return holder, nil
		}
	}

	// Could not determine holder; return nil gracefully.
	return nil, nil
}

// FormatConflictMessage formats a human-readable port conflict error message.
// Example: "port 5432 is held by postgres (pid 1234) — stop it or change POSTGRES_PORT"
// If holder is nil (port is free or detection unavailable), the message omits holder info.
func FormatConflictMessage(port int, holder *Holder) string {
	if holder == nil {
		return fmt.Sprintf("port %d is already in use — stop the conflicting process or change the port in your .env", port)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("port %d is held by %s (pid %d)", port, holder.Name, holder.PID))

	if holder.IsOurs {
		sb.WriteString(" [nSelf container]")
	}

	sb.WriteString(" — stop it or change the port in your .env")

	// Suggest the likely env var name based on the port.
	if envVar := portEnvVar(port); envVar != "" {
		sb.WriteString(fmt.Sprintf(" (e.g. %s)", envVar))
	}

	return sb.String()
}

// ── lsof implementation ───────────────────────────────────────────────────────

// whoHoldsPortLsof uses `lsof -F pcn` to identify the holder of port.
// The -F flag requests field-based output:
//
//	p<pid>   — process ID
//	c<name>  — command name (short)
//	n<addr>  — network address (e.g. *:5432)
func whoHoldsPortLsof(port int) (*Holder, error) {
	portStr := strconv.Itoa(port)
	cmd := exec.Command("lsof",
		"-i", "TCP:"+portStr,
		"-sTCP:LISTEN",
		"-n", "-P",
		"-F", "pcn",
	)
	out, err := cmd.Output()
	if err != nil {
		// lsof exits 1 when no processes are found; exec.ExitError is normal.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Non-zero exit with no output == port is free.
			if len(bytes.TrimSpace(out)) == 0 {
				return nil, fmt.Errorf("lsof: port %d appears free", port)
			}
			// Non-zero exit with output — parse what we have.
		} else if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("lsof not found in PATH")
		} else {
			return nil, fmt.Errorf("running lsof: %w", err)
		}
	}

	return parseLsofOutput(out, port)
}

// parseLsofOutput scans lsof -F pcn field output and builds a Holder.
// Field lines look like:
//
//	p1234
//	cpostgres
//	n*:5432
func parseLsofOutput(out []byte, port int) (*Holder, error) {
	var pid int
	var name string
	portSuffix := ":" + strconv.Itoa(port)

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
			if err == nil {
				pid = n
			}
		case 'c':
			name = strings.TrimSpace(line[1:])
		case 'n':
			// We only care about lines that reference the port we asked about.
			// When we see the network line for this port, the preceding p/c
			// lines belong to the holder we want.
			if strings.HasSuffix(line[1:], portSuffix) && pid != 0 {
				// We have enough information.
				cmdline := readCmdline(pid)
				holder := &Holder{
					PID:     pid,
					Name:    name,
					Command: cmdline,
					IsOurs:  isOurContainer(pid),
				}
				return holder, nil
			}
		}
	}

	return nil, fmt.Errorf("lsof: no listener found on port %d", port)
}

// ── ss implementation (Linux fallback) ───────────────────────────────────────

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
