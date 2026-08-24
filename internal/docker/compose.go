package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"
)

// Compose shells out to `docker compose` for lifecycle operations.
type Compose struct {
	// DockerPath overrides the docker binary location. Empty uses PATH lookup.
	DockerPath string

	// ComposeFiles lists compose file paths passed as -f flags.
	// Order matters: first file is the base, subsequent files extend/override.
	// If empty, no -f flags are added and Docker uses its default discovery.
	ComposeFiles []string

	// EnvFiles lists env file paths passed as --env-file flags for ${VAR}
	// interpolation (secrets are env-templated out of the generated compose
	// YAML — see internal/build secret templating). Order matters: later
	// files win on conflict. If empty, Docker Compose falls back to its
	// default .env discovery.
	EnvFiles []string
}

// NewCompose returns a Compose instance configured with the given compose files.
// If no files are provided, Docker Compose uses its default file discovery.
func NewCompose(files ...string) *Compose {
	return &Compose{ComposeFiles: files}
}

// dockerBin returns the path to the docker binary.
func (c *Compose) dockerBin() string {
	if c.DockerPath != "" {
		return c.DockerPath
	}
	return "docker"
}

// buildBaseArgs returns the base command arguments including any -f file flags.
// The returned slice always starts with "compose" followed by zero or more
// "-f <path>" pairs.
func (c *Compose) buildBaseArgs() []string {
	args := []string{"compose"}
	for _, f := range c.ComposeFiles {
		args = append(args, "-f", f)
	}
	for _, e := range c.EnvFiles {
		args = append(args, "--env-file", e)
	}
	return args
}

// ComposeUp runs `docker compose up -d [services...]` in the given workdir.
func (c *Compose) ComposeUp(ctx context.Context, workdir string, services ...string) error {
	args := c.buildBaseArgs()
	args = append(args, "up", "-d")
	args = append(args, services...)
	return c.Run(ctx, workdir, args...)
}

// ComposeDown runs `docker compose down` with the given options in the given workdir.
func (c *Compose) ComposeDown(ctx context.Context, workdir string, opts DownOptions) error {
	args := c.buildBaseArgs()
	args = append(args, "down")
	if opts.RemoveVolumes {
		args = append(args, "-v")
	}
	if opts.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if opts.Timeout > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", opts.Timeout))
	}
	return c.Run(ctx, workdir, args...)
}

// ComposeUpNoDeps runs `docker compose up -d --no-deps [services...]`.
//
// This is the primitive that means "make these containers match the compose
// file": Compose recreates a container whose definition changed and leaves an
// unchanged one alone. `restart` does neither — it bounces the existing
// container and never re-reads the file. --no-deps keeps the operation scoped
// to the named services instead of pulling their dependencies up with them.
func (c *Compose) ComposeUpNoDeps(ctx context.Context, workdir string, services ...string) error {
	args := c.buildBaseArgs()
	args = append(args, "up", "-d", "--no-deps")
	args = append(args, services...)
	return c.Run(ctx, workdir, args...)
}

// ComposeRestart runs `docker compose restart [services...]` in the given workdir.
func (c *Compose) ComposeRestart(ctx context.Context, workdir string, services ...string) error {
	args := c.buildBaseArgs()
	args = append(args, "restart")
	args = append(args, services...)
	return c.Run(ctx, workdir, args...)
}

// ComposePull runs `docker compose pull` in the given workdir, pulling images
// for all services defined in the compose file.
func (c *Compose) ComposePull(ctx context.Context, workdir string) error {
	args := c.buildBaseArgs()
	args = append(args, "pull")
	return c.Run(ctx, workdir, args...)
}

// ComposePullService runs `docker compose pull <service>` in the given workdir,
// pulling a fresh image only for the named service.
func (c *Compose) ComposePullService(ctx context.Context, workdir, service string) error {
	args := c.buildBaseArgs()
	args = append(args, "pull", service)
	return c.Run(ctx, workdir, args...)
}

// ComposeStop runs `docker compose stop [services...]` in the given workdir.
// Unlike ComposeDown, this stops containers without removing them, preserving
// their state for a subsequent ComposeUp.
func (c *Compose) ComposeStop(ctx context.Context, workdir string, services ...string) error {
	args := c.buildBaseArgs()
	args = append(args, "stop")
	args = append(args, services...)
	return c.Run(ctx, workdir, args...)
}

// ComposeScale runs `docker compose up -d --scale <service>=<n>` in the given
// workdir, adjusting the replica count for the named service.
func (c *Compose) ComposeScale(ctx context.Context, workdir, service string, replicas int) error {
	args := c.buildBaseArgs()
	args = append(args, "up", "-d", "--no-deps",
		fmt.Sprintf("--scale=%s=%d", service, replicas), service,
	)
	return c.Run(ctx, workdir, args...)
}

// composePsEntry maps the JSON output of `docker compose ps --format json`.
// Docker Compose v2 emits one JSON object per line (not an array).
type composePsEntry struct {
	ID      string `json:"ID"`
	Name    string `json:"Name"`
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
	Image   string `json:"Image"`
	Status  string `json:"Status"`
	Ports   string `json:"Ports"`
}

// ComposePs runs `docker compose ps --format json` and parses the output
// into a slice of ContainerInfo.
//
// Docker Compose v2 changed its JSON output format across versions:
//   - v2.20 and earlier: one JSON object per stdout line (NDJSON)
//   - v2.21 and later:   a single JSON array containing all objects
//
// Both formats are handled: the raw output is first attempted as a JSON array;
// if that fails (or yields nothing), the output is re-scanned line by line as
// NDJSON. This makes the function version-agnostic.
func (c *Compose) ComposePs(ctx context.Context, workdir string) ([]ContainerInfo, error) {
	args := c.buildBaseArgs()
	args = append(args, "ps", "--format", "json")
	cmd := exec.CommandContext(ctx, c.dockerBin(), args...)
	cmd.Dir = workdir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker compose ps: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}

	// Attempt 1: JSON array format (Docker Compose v2.21+).
	if strings.HasPrefix(raw, "[") {
		var entries []composePsEntry
		if err := json.Unmarshal([]byte(raw), &entries); err != nil {
			return nil, fmt.Errorf("parsing compose ps JSON array: %w", err)
		}
		return composePsEntriesToInfos(entries), nil
	}

	// Attempt 2: NDJSON format (Docker Compose v2.20 and earlier) —
	// one JSON object per line.
	var infos []ContainerInfo
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry composePsEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing compose ps JSON: %w", err)
		}
		infos = append(infos, composePsEntryToInfo(entry))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading compose ps output: %w", err)
	}
	return infos, nil
}

// composePsEntriesToInfos converts a slice of composePsEntry into ContainerInfo.
func composePsEntriesToInfos(entries []composePsEntry) []ContainerInfo {
	infos := make([]ContainerInfo, 0, len(entries))
	for _, e := range entries {
		infos = append(infos, composePsEntryToInfo(e))
	}
	return infos
}

// composePsEntryToInfo converts a single composePsEntry into ContainerInfo.
func composePsEntryToInfo(e composePsEntry) ContainerInfo {
	return ContainerInfo{
		ID:      e.ID,
		Name:    e.Name,
		Service: e.Service,
		Image:   e.Image,
		State:   e.State,
		Health:  e.Health,
		Ports:   parsePorts(e.Ports),
	}
}

// ComposeConfig runs `docker compose config` in the given workdir to validate
// the compose configuration. Returns nil on success.
func (c *Compose) ComposeConfig(ctx context.Context, workdir string) error {
	args := c.buildBaseArgs()
	args = append(args, "config", "--quiet")
	return c.Run(ctx, workdir, args...)
}

// Run executes a docker command with the given arguments, setting the working
// directory and inheriting the context for cancellation. Output is forwarded to
// os.Stdout/os.Stderr via goroutine-owned pipes rather than direct fd inheritance.
//
// Direct fd inheritance (cmd.Stdout = os.Stdout) passes the test binary's stdout
// file descriptor to docker compose and all its children. The docker daemon, which
// is a separate long-lived process, may hold that fd open via IPC even after the
// docker compose process is killed — causing "*** Test I/O incomplete N s after
// exiting" in the Go test harness. Piped copying breaks the inheritance chain: the
// subprocess gets the write end of a pipe; our goroutine owns the read end and
// copies to os.Stdout. When the subprocess exits (or is killed), it closes the
// write end, the goroutine finishes, and the test binary's stdout fd is never
// held by any docker process.
//
// WaitDelay gives the subprocess up to 5 seconds to drain pending I/O after the
// context is cancelled and the process is killed.
//
// Setpgid places docker compose in its own process group so SIGKILL from the
// Cancel hook propagates to all spawned child processes, not just the top-level
// docker process.
func (c *Compose) Run(ctx context.Context, workdir string, args ...string) error {
	cmd := exec.CommandContext(ctx, c.dockerBin(), args...)
	cmd.Dir = workdir
	cmd.WaitDelay = 5 * time.Second

	// Put docker compose and all its spawned children in a new process group so
	// Cancel can kill the entire group with a single signal.
	// setProcGroupAttr and killProcessGroup are implemented per-OS in
	// compose_unix.go (darwin/linux) and compose_windows.go (windows).
	setProcGroupAttr(cmd)
	cmd.Cancel = func() error {
		killProcessGroup(cmd)
		return cmd.Process.Kill()
	}

	// Use pipe-based I/O forwarding instead of direct fd inheritance. This ensures
	// the test binary's stdout/stderr fds are never held open by docker processes.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("docker %s: stdout pipe: %w", strings.Join(args, " "), err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("docker %s: stderr pipe: %w", strings.Join(args, " "), err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
	}

	// In interactive TTY sessions forward output to the real terminal so Docker
	// Compose can render progress correctly. In non-TTY contexts (tests, CI,
	// piped output) discard subprocess output — the test harness captures output
	// via cobra's SetOut/SetErr and does not need raw docker progress lines.
	// Using io.Discard here prevents the goroutines from ever touching os.Stdout
	// after the subprocess exits, which eliminates "Test I/O incomplete" failures
	// caused by goroutines writing to os.Stdout past the test deadline.
	outSink := io.Writer(os.Stdout)
	errSink := io.Writer(os.Stderr)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		outSink = io.Discard
		errSink = io.Discard
	}

	done := make(chan struct{}, 2)
	go func() { io.Copy(outSink, stdoutPipe); done <- struct{}{} }() //nolint:errcheck
	go func() { io.Copy(errSink, stderrPipe); done <- struct{}{} }() //nolint:errcheck

	waitErr := cmd.Wait()
	<-done
	<-done

	if waitErr != nil {
		return fmt.Errorf("docker %s: %w", strings.Join(args, " "), waitErr)
	}
	return nil
}

// parsePorts splits a Docker Compose ports string (e.g. "0.0.0.0:80->80/tcp, 0.0.0.0:443->443/tcp")
// into individual port mappings.
func parsePorts(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ", ")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
