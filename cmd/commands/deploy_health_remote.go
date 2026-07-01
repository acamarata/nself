package commands

// Purpose: Remote-probe helpers for `nself deploy health`, extracted out of
//          deploy.go to keep that file from growing further. Implements the
//          gap #12 fix: "nself deploy health <target>" (or --server <name>)
//          actually SSHes into the resolved target and runs 'nself doctor'
//          there, instead of always running local doctor checks regardless
//          of which target name was passed.
// Inputs:  A resolved control-plane server (Host/SSHKeyRef/RemotePath) or a
//          legacy "user@host:/remote/path" NSELF_DEPLOY_HOST_<TARGET> value.
// Outputs: Streams the remote 'nself doctor' output directly to this
//          process's stdout/stderr; returns an error (wrapped for gap #16
//          version-drift clarity) on failure.
// Constraints: A server entry with no Host is local-only by convention
//              (controlplane.Server doc comment) — that case must fall back
//              to local doctor, not error out.
// SPORT: cli/cmd/commands — see gap #12 and gap #16.

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/controlplane"

	"github.com/spf13/cobra"
)

// runDeployHealthOnServer runs 'nself doctor' over SSH on the named
// control-plane server, or falls back to local doctor when that server has
// no Host configured (local-only entry).
func runDeployHealthOnServer(cmd *cobra.Command, workdir, serverFilter string, jsonOut bool) error {
	inv, loadErr := controlplane.Load(workdir)
	if loadErr != nil {
		return fmt.Errorf("deploy health: load inventory: %w", loadErr)
	}
	for _, env := range inv.Environments {
		for _, srv := range env.Servers {
			if srv.Name != serverFilter {
				continue
			}
			if srv.Host == "" {
				// Local-only server entry: run local doctor instead of SSH.
				return runCLISelf(cmd.Context(), workdir, "doctor")
			}
			keyPath := os.Getenv(srv.SSHKeyRef)
			remotePath := srv.RemotePath
			if remotePath == "" {
				remotePath = "/opt/nself"
			}
			return runDeployHealthSSH(cmd, srv.Host, keyPath, remotePath, jsonOut)
		}
	}
	return fmt.Errorf("deploy health: --server %q not found in inventory", serverFilter)
}

// runDeployHealthOverSSH probes a legacy "user@host:/remote/path" target
// (the NSELF_DEPLOY_HOST_<TARGET> convention, no control-plane.yaml entry).
func runDeployHealthOverSSH(cmd *cobra.Command, host string, jsonOut bool) error {
	sshTarget, remotePath := splitDeployHost(host)
	if remotePath == "" {
		remotePath = "/opt/nself"
	}
	return runDeployHealthSSH(cmd, sshTarget, "", remotePath, jsonOut)
}

// runDeployHealthSSH invokes 'nself doctor [--json]' on the remote host via
// SSH, streaming output directly to this process's stdout/stderr. If the
// remote nself binary doesn't support a flag/subcommand used here (gap #16:
// version drift between the local and remote CLI), the wrapped error names
// the likely cause instead of surfacing a bare non-zero exit or raw SSH error.
func runDeployHealthSSH(cmd *cobra.Command, sshTarget, keyPath, remotePath string, jsonOut bool) error {
	if keyPath == "" {
		keyPath = sshKeyPath()
	}
	doctorCmd := fmt.Sprintf("cd %s && nself doctor", remotePath)
	if jsonOut {
		doctorCmd = fmt.Sprintf("cd %s && nself doctor --json", remotePath)
	}
	sc := exec.CommandContext(cmd.Context(), "ssh",
		"-i", keyPath,
		"-o", "BatchMode=yes",
		"-o", "ForwardAgent=no",
		"-o", "StrictHostKeyChecking=accept-new",
		sshTarget, doctorCmd)
	var stderr strings.Builder
	sc.Stdout = os.Stdout
	sc.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := sc.Run(); err != nil {
		return wrapRemoteVersionDriftError(dbRemoteTarget{SSHTarget: sshTarget, EnvName: remotePath}, []string{"doctor"}, err, stderr.String())
	}
	return nil
}

// ResolveLegacyDeployHost returns the NSELF_DEPLOY_HOST_<TARGET> (or legacy
// <TARGET>_DEPLOY_HOST) value for target, mirroring the lookup already used
// by runDeploy's remote push path.
func ResolveLegacyDeployHost(target string) (string, bool) {
	upper := strings.ToUpper(target)
	if host := os.Getenv("NSELF_DEPLOY_HOST_" + upper); host != "" {
		return host, true
	}
	if host := os.Getenv(upper + "_DEPLOY_HOST"); host != "" {
		return host, true
	}
	return "", false
}

// splitDeployHost splits "user@host:/remote/path" into its two components,
// mirroring the parsing already used by remoteDeployPush.
func splitDeployHost(host string) (sshTarget, remotePath string) {
	idx := strings.LastIndex(host, ":")
	if idx < 0 {
		return host, ""
	}
	return host[:idx], host[idx+1:]
}
