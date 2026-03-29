package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Compose shells out to `docker compose` for lifecycle operations.
type Compose struct {
	// DockerPath overrides the docker binary location. Empty uses PATH lookup.
	DockerPath string

	// ComposeFiles lists compose file paths passed as -f flags.
	// Order matters: first file is the base, subsequent files extend/override.
	// If empty, no -f flags are added and Docker uses its default discovery.
	ComposeFiles []string
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

// ComposeRestart runs `docker compose restart [services...]` in the given workdir.
func (c *Compose) ComposeRestart(ctx context.Context, workdir string, services ...string) error {
	args := c.buildBaseArgs()
	args = append(args, "restart")
	args = append(args, services...)
	return c.Run(ctx, workdir, args...)
}

// ComposePull runs `docker compose pull` in the given workdir.
func (c *Compose) ComposePull(ctx context.Context, workdir string) error {
	args := c.buildBaseArgs()
	args = append(args, "pull")
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
// into a slice of ContainerInfo. Docker Compose v2 emits one JSON object
// per stdout line.
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
		infos = append(infos, ContainerInfo{
			ID:     entry.ID,
			Name:   entry.Name,
			Image:  entry.Image,
			State:  entry.State,
			Health: entry.Health,
			Ports:  parsePorts(entry.Ports),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading compose ps output: %w", err)
	}
	return infos, nil
}

// ComposeConfig runs `docker compose config` in the given workdir to validate
// the compose configuration. Returns nil on success.
func (c *Compose) ComposeConfig(ctx context.Context, workdir string) error {
	args := c.buildBaseArgs()
	args = append(args, "config", "--quiet")
	return c.Run(ctx, workdir, args...)
}

// Run executes a docker command with the given arguments, setting the working
// directory and inheriting the context for cancellation. stdout and stderr are
// streamed directly to the terminal so that Docker Compose can detect the TTY
// and render progress correctly. This also ensures long-running operations like
// "compose stop" and "compose down" are not silently killed by a hidden pipe.
func (c *Compose) Run(ctx context.Context, workdir string, args ...string) error {
	cmd := exec.CommandContext(ctx, c.dockerBin(), args...)
	cmd.Dir = workdir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
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
