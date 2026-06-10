// Package deploy provides deployment management for the nSelf CLI.
// This file implements the SSH deploy helper used by 'nself deploy --env staging|prod' (G-003).
//
// Required env vars:
//   NSELF_DEPLOY_HOST   - Remote host in user@host:/remote/path format
//   NSELF_DEPLOY_USER   - SSH user (overridden by the user@host prefix when set)
//   NSELF_DEPLOY_KEY_PATH - Path to SSH private key (default: ~/.ssh/id_ed25519)
package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SSHConfig holds parameters for an SSH-based remote deploy.
type SSHConfig struct {
	// Host is the remote target in "user@host:/remote/path" format.
	// NSELF_DEPLOY_HOST is the conventional env var name.
	Host string

	// KeyPath is the path to the SSH private key.
	// Falls back to NSELF_DEPLOY_KEY_PATH, then ~/.ssh/id_ed25519.
	KeyPath string

	// Follow streams container logs after a successful deploy until the
	// context is cancelled (e.g. Ctrl-C from the user).
	Follow bool
}

// SSHConfigFromEnv builds an SSHConfig from environment variables.
// target should be "staging" or "prod".
func SSHConfigFromEnv(target string) SSHConfig {
	upper := strings.ToUpper(target)
	host := os.Getenv("NSELF_DEPLOY_HOST_" + upper)
	if host == "" {
		// Fallback: STAGING_DEPLOY_HOST / PROD_DEPLOY_HOST (legacy convention).
		host = os.Getenv(upper + "_DEPLOY_HOST")
	}
	return SSHConfig{
		Host:    host,
		KeyPath: sshKeyPathEnv(),
	}
}

// DeployViaSsh copies the compose file to the remote host via rsync, pulls
// updated images, and runs 'docker compose up -d' on the remote.
// If cfg.Follow is true, it streams 'docker compose logs --follow' until the
// context is cancelled.
//
// Steps:
//  1. rsync composePath to remote:/tmp/nself-compose.yml
//  2. ssh exec "docker compose -f /tmp/nself-compose.yml pull"
//  3. ssh exec "docker compose -f /tmp/nself-compose.yml up -d"
//  4. if follow: stream "docker compose -f /tmp/nself-compose.yml logs --follow"
func DeployViaSsh(ctx context.Context, cfg SSHConfig, composePath string) error {
	if cfg.Host == "" {
		return fmt.Errorf("SSHConfig.Host is empty; set NSELF_DEPLOY_HOST_<TARGET>")
	}

	sshTarget, remotePath, err := splitHost(cfg.Host)
	if err != nil {
		return err
	}

	remoteCompose := filepath.Join(remotePath, "nself-compose.yml")
	if remotePath == "" || remotePath == "/" {
		remoteCompose = "/tmp/nself-compose.yml"
	}

	sshArgs := sshBaseArgs(cfg.KeyPath)

	// 1. rsync compose file to remote.
	rsyncArgs := []string{
		// Agent forwarding is disabled via ForwardAgent=no in sshBaseArgs (the -e
		// command below) — it is an ssh option and must never appear in rsync argv.
		"-az",
		"-e", "ssh " + strings.Join(sshArgs, " "),
		composePath,
		fmt.Sprintf("%s:%s", sshTarget, remoteCompose),
	}
	rc := exec.CommandContext(ctx, "rsync", rsyncArgs...)
	rc.Env = os.Environ()
	if out, rerr := rc.CombinedOutput(); rerr != nil {
		return fmt.Errorf("rsync to %s: %w\n%s", sshTarget, rerr, strings.TrimSpace(string(out)))
	}

	// 2. docker compose pull on remote.
	if err := runSSH(ctx, sshTarget, cfg.KeyPath,
		fmt.Sprintf("docker compose -f %s pull", remoteCompose)); err != nil {
		return fmt.Errorf("remote pull: %w", err)
	}

	// 3. docker compose up -d on remote.
	if err := runSSH(ctx, sshTarget, cfg.KeyPath,
		fmt.Sprintf("docker compose -f %s up -d", remoteCompose)); err != nil {
		return fmt.Errorf("remote up: %w", err)
	}

	// 4. Optional log stream.
	if cfg.Follow {
		sc := exec.CommandContext(ctx, "ssh",
			append(sshBaseArgs(cfg.KeyPath), sshTarget,
				fmt.Sprintf("docker compose -f %s logs --follow", remoteCompose))...)
		sc.Stdout = os.Stdout
		sc.Stderr = os.Stderr
		// Ctrl-C or context cancellation; non-zero exit is not an error from the
		// user's perspective.
		_ = sc.Run()
	}

	return nil
}

// runSSH executes a command on the remote host via SSH and returns any error.
func runSSH(ctx context.Context, sshTarget, keyPath, command string) error {
	args := append(sshBaseArgs(keyPath), sshTarget, command)
	sc := exec.CommandContext(ctx, "ssh", args...)
	sc.Env = os.Environ()
	if out, err := sc.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh %s %q: %w\n%s", sshTarget, command, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// sshBaseArgs returns the common SSH flags used for all remote operations.
func sshBaseArgs(keyPath string) []string {
	return []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ForwardAgent=no",
	}
}

// splitHost splits "user@host:/remote/path" into ("user@host", "/remote/path").
// The path component may be empty when no colon is present, in which case /tmp
// is used by the caller.
func splitHost(host string) (sshTarget, remotePath string, err error) {
	idx := strings.LastIndex(host, ":")
	if idx < 0 {
		return host, "", nil
	}
	return host[:idx], host[idx+1:], nil
}

// sshKeyPathEnv returns the SSH key path from NSELF_DEPLOY_KEY_PATH or
// NSELF_DEPLOY_SSH_KEY (legacy), falling back to ~/.ssh/id_ed25519.
func sshKeyPathEnv() string {
	if k := os.Getenv("NSELF_DEPLOY_KEY_PATH"); k != "" {
		return k
	}
	if k := os.Getenv("NSELF_DEPLOY_SSH_KEY"); k != "" {
		return k
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "id_ed25519")
}
