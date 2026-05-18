package controlplane

import (
	"fmt"
	"time"
)

// Prober is the interface through which the resolver obtains runtime facts
// about a server. By injecting this interface the resolver itself contains
// zero os/exec calls, making it fully unit-testable without network access.
//
// Implementations must be safe for concurrent use from multiple goroutines.
type Prober interface {
	// SSHReachable attempts an SSH connection to s and returns (true, nil) on
	// success or (false, err) with a descriptive error on failure. The err
	// value is used to populate TargetStatus.Reason; callers must not treat
	// non-nil err as fatal — it may indicate a transient network condition.
	SSHReachable(s Server) (reachable bool, latencyMS int, err error)

	// DockerOK runs "docker info" over SSH and returns true when the Docker
	// daemon responds. It is only called when SSHReachable returns true.
	DockerOK(s Server) (bool, error)

	// KeyState checks whether the key referenced by s.SSHKeyRef exists on the
	// local filesystem. It returns the resolved key path for logging.
	// KeyState must not perform any network I/O.
	KeyState(s Server) (present bool, path string)
}

// Reason strings used by the resolver. They are defined here (§5.3) so that
// callers can pattern-match the TargetStatus.Reason field without parsing
// free text.
const (
	reasonNoKey          = "no SSH key for %s"
	reasonSSHUnreachable = "SSH unreachable (%s)"
	reasonDockerDown     = "SSH ok, docker not reachable"
)

// Resolve evaluates the capability of every server in inv by running probes
// through p and returns one TargetStatus per server.
//
// The decision algorithm follows §5.1 of the architecture spec:
//
//  1. Local kind env → CapManage, no probe.
//  2. Host empty → CapHidden.
//  3. SSHKeyRef unset → KeyPresent=false → CapReadOnly + reason "no SSH key for <ref>".
//  4. KeyState present=false → CapReadOnly + reason "no SSH key for <ref>".
//  5. SSHReachable → if false → CapReadOnly + reason "SSH unreachable (<err class>)".
//  6. DockerOK → if false → CapReadOnly + reason "SSH ok, docker not reachable".
//  7. Both SSH and Docker ok → CapManage.
//
// Resolve never calls os/exec directly; all I/O is delegated to p.
func Resolve(inv *Inventory, p Prober) []TargetStatus {
	var results []TargetStatus

	for envName, env := range inv.Environments {
		for _, srv := range env.Servers {
			ts := resolveOne(envName, env, srv, p)
			results = append(results, ts)
		}
	}

	return results
}

// resolveOne applies the §5.1 decision tree to a single server.
func resolveOne(envName string, env Environment, srv Server, p Prober) TargetStatus {
	now := time.Now()

	ts := TargetStatus{
		Env:      envName,
		Server:   srv.Name,
		ProbedAt: now,
	}

	// Step 1: Local environment — always manage, no probes.
	if env.Kind == "local" {
		ts.Capability = CapManage
		ts.KeyPresent = true
		ts.SSHReachable = true
		ts.DockerOK = true
		return ts
	}

	// Step 2: Host empty → hidden.
	if srv.Host == "" {
		ts.Capability = CapHidden
		return ts
	}

	// Step 3: SSHKeyRef unset → no key → read-only.
	if srv.SSHKeyRef == "" {
		ts.Capability = CapReadOnly
		// Use "(unset)" sentinel so the reason string is not a blank tail.
		ts.Reason = fmt.Sprintf(reasonNoKey, "(unset)")
		return ts
	}

	// Step 4: Key must exist on disk.
	present, _ := p.KeyState(srv)
	ts.KeyPresent = present
	if !present {
		ts.Capability = CapReadOnly
		ts.Reason = fmt.Sprintf(reasonNoKey, srv.SSHKeyRef)
		return ts
	}

	// Step 5: SSH reachability probe.
	reachable, latencyMS, sshErr := p.SSHReachable(srv)
	ts.SSHReachable = reachable
	ts.LatencyMS = latencyMS
	if !reachable {
		ts.Capability = CapReadOnly
		errClass := errClass(sshErr)
		ts.Reason = fmt.Sprintf(reasonSSHUnreachable, errClass)
		return ts
	}

	// Step 6: Docker probe.
	dockerOK, _ := p.DockerOK(srv)
	ts.DockerOK = dockerOK
	if !dockerOK {
		ts.Capability = CapReadOnly
		ts.Reason = reasonDockerDown
		return ts
	}

	// Step 7: Everything ok — full manage access.
	ts.Capability = CapManage
	return ts
}

// errClass returns a concise error-class string for use in reason messages.
// It must not reveal key bytes or host→key mappings.
func errClass(err error) string {
	if err == nil {
		return "unknown"
	}
	msg := err.Error()
	switch {
	case containsFold(msg, "timeout"):
		return "timeout"
	case containsFold(msg, "connection refused"):
		return "connection refused"
	case containsFold(msg, "no route"):
		return "no route to host"
	case containsFold(msg, "network unreachable"):
		return "network unreachable"
	case containsFold(msg, "permission denied"):
		return "permission denied"
	case containsFold(msg, "host key"):
		return "host key mismatch"
	default:
		return "network error"
	}
}

// containsFold reports whether s contains substr, ASCII-case-insensitively.
func containsFold(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	sLow := toLowerASCII(s)
	subLow := toLowerASCII(substr)
	for i := 0; i <= len(sLow)-len(subLow); i++ {
		if sLow[i:i+len(subLow)] == subLow {
			return true
		}
	}
	return false
}

// toLowerASCII returns s with all ASCII uppercase letters converted to
// lowercase. It does not depend on unicode/strings.ToLower.
func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
