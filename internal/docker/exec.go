package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// ExecOptions configures behavior for container exec operations.
type ExecOptions struct {
	Interactive bool
	TTY         bool
	User        string
	WorkDir     string
	Env         []string
}

// Exec runs a command inside a running container, attaching stdin/stdout/stderr
// for interactive sessions.
func Exec(ctx context.Context, container string, cmd []string, opts ExecOptions) error {
	args := buildExecArgs(container, cmd, opts)

	c := exec.CommandContext(ctx, "docker", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return fmt.Errorf("docker exec %s: %w", container, err)
	}
	return nil
}

// buildExecArgs constructs the argument list for `docker exec`.
func buildExecArgs(container string, cmd []string, opts ExecOptions) []string {
	args := []string{"exec"}

	if opts.Interactive {
		args = append(args, "-i")
	}
	if opts.TTY {
		args = append(args, "-t")
	}
	if opts.User != "" {
		args = append(args, "-u", opts.User)
	}
	if opts.WorkDir != "" {
		args = append(args, "-w", opts.WorkDir)
	}
	for _, env := range opts.Env {
		args = append(args, "-e", env)
	}

	args = append(args, container)
	args = append(args, cmd...)
	return args
}

// DefaultCommand returns the default shell or client command for a given service.
// Recognized services get their native CLI; everything else falls back to /bin/sh.
func DefaultCommand(service string) []string {
	switch service {
	case "postgres":
		return []string{"psql", "-U", "postgres"}
	case "redis":
		return []string{"redis-cli"}
	default:
		return []string{"/bin/sh"}
	}
}
