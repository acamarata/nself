//go:build windows

package installer

// Purpose: Windows stub for the process-group helpers used by
//          downloadAndRunInstaller. Unreachable in practice: Install()
//          refuses to run on any GOOS other than linux before this code
//          path is reached. The stub exists only so internal/installer
//          builds on windows-2022 CI (go build/vet ./... compiles every
//          package regardless of runtime.GOOS gates elsewhere).
// Inputs:  an *exec.Cmd.
// Outputs: none.
// Constraints: mirrors internal/docker/compose_windows.go's no-op pattern.

import "os/exec"

// setProcGroupAttr is a no-op on Windows — this code path is unreachable
// there, since Install() gates on runtime.GOOS == "linux" first.
func setProcGroupAttr(_ *exec.Cmd) {}

// killProcessGroup is a no-op on Windows for the same reason.
func killProcessGroup(_ *exec.Cmd) {}
