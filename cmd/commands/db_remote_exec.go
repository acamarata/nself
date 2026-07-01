package commands

// Purpose: Low-level SSH process execution + version-drift error wrapping
//          used by runRemoteNselfCommand (db_remote.go). Split out to keep
//          db_remote.go under the 300-line file cap.
// Inputs:  A fully-built ssh argv and the dbRemoteTarget/command context
//          needed to produce an actionable error message.
// Outputs: Combined stdout+stderr text and a wrapped error.
// Constraints: Never panics on a non-zero remote exit; always returns a Go
//              error the caller can propagate as the command's final error.
// SPORT: cli/cmd/commands — see gap #9 (remote exec) and gap #16 (version drift).

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// runSSHCaptured runs `ssh <sshArgs...>` and returns combined stdout+stderr
// plus any error from the process itself (non-zero exit, ssh connection
// failure, etc.).
func runSSHCaptured(ctx context.Context, sshArgs []string) (string, error) {
	sc := exec.CommandContext(ctx, "ssh", sshArgs...)
	out, err := sc.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// remoteCommandNotFoundMarkers lists substrings that indicate the remote
// nself binary doesn't recognize the subcommand we tried to run — the
// telltale sign of CLI version drift between the machine running this
// command and the target box (gap #16). Cobra emits "unknown command" and
// most POSIX shells emit "command not found" / "not found" for a missing
// binary or subcommand.
var remoteCommandNotFoundMarkers = []string{
	"unknown command",
	"command not found",
	"executable file not found",
}

// wrapRemoteVersionDriftError inspects a failed remote command's output and,
// when it looks like the remote nself CLI doesn't support the subcommand we
// tried to run, returns a clear, actionable error instead of a raw SSH/exec
// error or stack trace. Otherwise the original error is wrapped with context
// (target, command) and returned unchanged in kind.
func wrapRemoteVersionDriftError(rt dbRemoteTarget, args []string, execErr error, out string) error {
	cmdStr := "nself " + strings.Join(args, " ")
	lowerOut := strings.ToLower(out)

	for _, marker := range remoteCommandNotFoundMarkers {
		if strings.Contains(lowerOut, marker) {
			return fmt.Errorf(
				"remote command %q failed on %s (env=%s): the nself CLI on that host may be an older version that doesn't support this subcommand yet. Run 'ssh %s nself --version' to check, and upgrade the remote CLI if needed.\nremote output: %s",
				cmdStr, rt.SSHTarget, rt.EnvName, rt.SSHTarget, out,
			)
		}
	}

	return fmt.Errorf("remote command %q failed on %s (env=%s): %w\n%s", cmdStr, rt.SSHTarget, rt.EnvName, execErr, out)
}
