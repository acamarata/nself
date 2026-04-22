package commands

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// shouldOpenBrowser reports whether the CLI should launch the user's default
// browser. It returns false in contexts where opening a browser is either
// impossible or actively harmful:
//
//   - Under go test (the binary name ends in .test) — tests must not spawn
//     Safari / xdg-open / cmd start; orphan GUI processes hang the runner
//     and caused a 10-minute timeout in macos-14 CI.
//   - CI=true / GITHUB_ACTIONS=true — headless runners, no display.
//   - NSELF_NO_BROWSER=1 — user opt-out for shell scripts / scripted use.
//
// The CLI caller is expected to print the URL as a fallback when this
// returns false so the user can paste it into a browser themselves.
func shouldOpenBrowser() bool {
	if isTestBinary() {
		return false
	}
	if os.Getenv("NSELF_NO_BROWSER") == "1" {
		return false
	}
	if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
		return false
	}
	return true
}

// isTestBinary reports whether this process is running inside `go test`.
// go test compiles the package into a binary whose basename ends in `.test`
// (e.g. `commands.test`). Detecting this is standard practice for libraries
// that need to skip side-effecting init in tests without pulling in the
// `testing` package from non-test code.
func isTestBinary() bool {
	if len(os.Args) == 0 {
		return false
	}
	base := os.Args[0]
	return strings.HasSuffix(base, ".test") ||
		strings.HasSuffix(base, ".test.exe") ||
		strings.Contains(base, "/_test/")
}

// openBrowserCmd returns the exec.Cmd (not yet started) that, when run, opens
// the given URL in the user's default browser on the current OS. Returns nil
// on platforms with no known open command.
func openBrowserCmd(ctx context.Context, url string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "open", url)
	case "windows":
		return exec.CommandContext(ctx, "cmd", "/c", "start", url)
	default:
		return exec.CommandContext(ctx, "xdg-open", url)
	}
}
