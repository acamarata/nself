// Package ports provides port-holder identification utilities for the nSelf CLI.
// It augments the raw TCP port probing in internal/docker with process-level
// information so that conflict error messages name the offending process.
package ports

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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
