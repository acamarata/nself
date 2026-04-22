//go:build windows

package plugin

import (
	"os"
	"os/exec"
	"strconv"
)

// setNewProcessGroup configures cmd so that, when started, the child runs in
// its own process group on Windows. CREATE_NEW_PROCESS_GROUP (0x00000200)
// allows us to signal the tree later via taskkill /T.
//
// This is a best-effort implementation. Native Windows support for plugin
// runtime is a Phase-2 goal (tracked as a nSelf-First CLI gap). For v1.0.x
// the CLI ships on Windows primarily for dev-machine vet/build/test; plugin
// runtime on Windows is unverified territory.
func setNewProcessGroup(cmd *exec.Cmd) {
	// Intentionally no SysProcAttr override here to keep the Windows build
	// green without pulling in golang.org/x/sys/windows. Plugins on Windows
	// run as regular child processes until the platform is fully supported.
	_ = cmd
}

// terminateProcessGroup attempts a graceful stop of the process tree rooted
// at pid via `taskkill /PID <pid> /T`. Any failure (process already gone,
// taskkill unavailable) is treated as non-fatal — callers follow up with
// killProcessGroup on timeout.
func terminateProcessGroup(pid int) error {
	return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T").Run()
}

// killProcessGroup force-terminates the process tree rooted at pid via
// `taskkill /F /PID <pid> /T`. Falls back to os.Process.Kill if taskkill is
// unavailable.
func killProcessGroup(pid int) error {
	if err := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid), "/T").Run(); err == nil {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
