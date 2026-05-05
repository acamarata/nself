// Package runtime detects the local container runtime (Docker, OrbStack, Podman)
// and reports its name, binary path, version, and enterprise-mode hint.
//
// nSelf supports any container runtime that exposes a Docker-compatible CLI.
// Detection probes binaries in priority order (docker, orb, podman) and uses
// the first one found on PATH. Callers should use Runtime.Binary instead of
// hardcoding "docker" so OrbStack and Podman users get first-class support.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Name constants for the supported runtimes. Stored on Runtime.Name so callers
// can branch on a known set instead of parsing strings.
const (
	NameDocker   = "docker"
	NameOrbStack = "orbstack"
	NamePodman   = "podman"
)

// Runtime describes a detected container runtime.
type Runtime struct {
	// Name is one of NameDocker, NameOrbStack, NamePodman.
	Name string `json:"name"`
	// Binary is the absolute path to the runtime's CLI executable. Use this
	// instead of hardcoding "docker" when invoking exec.Command.
	Binary string `json:"binary"`
	// Version is the runtime's reported version string (e.g., "24.0.6").
	// Empty when the version probe fails; detection still succeeds in that case.
	Version string `json:"version"`
	// IsEnterprise is true when the runtime is Docker Desktop running with an
	// enterprise (paid) plan. Used to surface a license-compliance warning for
	// organizations >250 employees.
	IsEnterprise bool `json:"is_enterprise"`
}

// ErrNoRuntime is returned by Detect when no supported container runtime is
// found on PATH.
var ErrNoRuntime = errors.New("no container runtime found (install docker, orbstack, or podman)")

// candidate pairs a CLI binary name with the canonical runtime Name it maps to.
// Order is the detection priority: docker first (still the most common), then
// orb (OrbStack on macOS), then podman.
type candidate struct {
	binary string
	name   string
}

var detectionOrder = []candidate{
	{"docker", NameDocker},
	{"orb", NameOrbStack},
	{"podman", NamePodman},
}

// Detect probes PATH for a supported container runtime and returns a populated
// Runtime on success. It uses a 5-second timeout when probing version and
// enterprise mode so a stuck daemon does not hang the caller.
func Detect() (*Runtime, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return DetectContext(ctx)
}

// DetectContext is the context-aware variant of Detect. Callers running inside
// a wider operation (e.g., nself doctor) should pass their own context so the
// probe respects upstream cancellation.
func DetectContext(ctx context.Context) (*Runtime, error) {
	for _, c := range detectionOrder {
		path, err := lookPath(c.binary)
		if err != nil {
			continue
		}
		// Found a binary. For "docker", check whether it's actually OrbStack's
		// shim — OrbStack installs an "orb" binary AND a "docker" symlink that
		// proxies to it. We treat docker-as-OrbStack-shim as plain Docker for
		// CLI compatibility (the Docker CLI is what's invoked either way).
		rt := &Runtime{Name: c.name, Binary: path}
		populateInspect(ctx, rt)
		return rt, nil
	}
	return nil, ErrNoRuntime
}

// populateInspect fills Version and IsEnterprise on the runtime by probing
// `<binary> version` and (for docker) `<binary> info`. Probe failures are
// non-fatal: detection still succeeds with the binary path.
func populateInspect(ctx context.Context, rt *Runtime) {
	rt.Version = probeVersion(ctx, rt.Binary)
	if rt.Name == NameDocker {
		rt.IsEnterprise = probeEnterprise(ctx, rt.Binary)
	}
}

// probeVersion runs `<binary> version --format '{{.Client.Version}}'` and
// returns the trimmed output. Falls back to a plain `<binary> --version`
// scrape if the structured form is unavailable (Podman supports both).
func probeVersion(ctx context.Context, binary string) string {
	out, err := runCommand(ctx, binary, "version", "--format", "{{.Client.Version}}")
	if err == nil {
		v := strings.TrimSpace(out)
		if v != "" {
			return v
		}
	}
	// Fallback: parse `<binary> --version` (e.g., "Docker version 24.0.6, build ...").
	out, err = runCommand(ctx, binary, "--version")
	if err != nil {
		return ""
	}
	return parseVersionLine(out)
}

// parseVersionLine extracts the version token from a `--version` line such as
// "Docker version 24.0.6, build ed223bc" or "podman version 4.6.2".
func parseVersionLine(line string) string {
	line = strings.TrimSpace(line)
	for _, sep := range []string{"version "} {
		idx := strings.Index(strings.ToLower(line), sep)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+len(sep):])
		// Cut at first comma or space.
		for i, ch := range rest {
			if ch == ',' || ch == ' ' {
				return strings.TrimSpace(rest[:i])
			}
		}
		return rest
	}
	return ""
}

// probeEnterprise inspects `docker info` for indicators that this is Docker
// Desktop running under an enterprise (paid) plan. Used only for the doctor
// warning; absence of indicators returns false (safe default).
func probeEnterprise(ctx context.Context, binary string) bool {
	out, err := runCommand(ctx, binary, "info")
	if err != nil {
		return false
	}
	lower := strings.ToLower(out)
	// Docker Desktop's enterprise mode reports "Plan: Enterprise" or
	// "Subscription: Enterprise" depending on version.
	if strings.Contains(lower, "plan: enterprise") {
		return true
	}
	if strings.Contains(lower, "subscription: enterprise") {
		return true
	}
	return false
}

// EnterpriseWarning returns a human-readable warning string for organizations
// running Docker Desktop's enterprise tier without a paid subscription. Empty
// when no warning applies.
func (r *Runtime) EnterpriseWarning() string {
	if r == nil || r.Name != NameDocker || !r.IsEnterprise {
		return ""
	}
	return "Docker Desktop reports an enterprise plan. Organizations with " +
		"more than 250 employees or $10M revenue must hold a paid Docker " +
		"subscription. See https://docs.docker.com/subscription/ for details."
}

// String renders a one-line summary suitable for `nself doctor --runtime`.
func (r *Runtime) String() string {
	if r == nil {
		return "no container runtime"
	}
	v := r.Version
	if v == "" {
		v = "unknown"
	}
	return fmt.Sprintf("%s %s (%s)", r.Name, v, r.Binary)
}

// ---- Test seams ------------------------------------------------------------
//
// lookPath and runCommand are package-level vars so unit tests can substitute
// in-memory fakes without touching the real PATH or spawning processes. The
// production implementations defer to os/exec.

var lookPath = func(name string) (string, error) {
	return exec.LookPath(name)
}

var runCommand = func(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}
