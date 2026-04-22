//go:build !windows

package plugin

import (
	"os/exec"
	"syscall"
)

// setNewProcessGroup configures cmd so that, when started, the child runs in
// its own process group. This lets terminateProcessGroup / killProcessGroup
// signal the child and all of its descendants with a single kill(-pgid, ...).
func setNewProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateProcessGroup sends SIGTERM to the process group leader at pid.
// If the pid has no explicit process group, it falls back to signalling pid
// directly. Errors are non-fatal by design: callers use this for best-effort
// graceful shutdown, then follow up with killProcessGroup on timeout.
func terminateProcessGroup(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid
	}
	return syscall.Kill(-pgid, syscall.SIGTERM)
}

// killProcessGroup sends SIGKILL to the process group leader at pid.
// If the pid has no explicit process group, it falls back to signalling pid
// directly.
func killProcessGroup(pid int) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}
