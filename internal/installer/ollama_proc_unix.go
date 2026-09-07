//go:build darwin || linux

package installer

// Purpose: platform-specific process-group isolation for the Ollama install
//          script child process, so a context cancellation kills the whole
//          subtree (sh + curl + zstd) instead of only the immediate child.
// Inputs:  an *exec.Cmd not yet started (setProcGroupAttr) or already
//          started (killProcessGroup).
// Outputs: SysProcAttr mutation; a best-effort SIGKILL to the process group.
// Constraints: unix-only (darwin/linux); mirrors the identical pattern in
//          internal/docker/compose_unix.go, which fixed the same
//          "Test I/O incomplete" failure mode for `docker compose`.

import (
	"os/exec"
	"syscall"
)

// setProcGroupAttr places cmd (and every process it spawns) in its own
// process group, so killProcessGroup can terminate the entire subtree with
// one signal instead of only the direct child.
func setProcGroupAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup sends SIGKILL to cmd's whole process group. On unix, the
// process group id equals the negation of the leader's PID once Setpgid is
// set, so signalling -pid reaches every descendant (install.sh's curl/zstd
// children included), not just install.sh itself.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
