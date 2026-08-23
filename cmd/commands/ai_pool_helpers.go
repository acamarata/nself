package commands

// Purpose: small shared helpers used by the ai pool subcommands: opening the
// system browser for OAuth, and a tolerant string-to-int parse. Inputs are a
// URL or a numeric string; outputs are none (best-effort) or an int.
// Constraints: split out of ai_pool.go (CLI-R12) as a pure move, no behavior change.

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Fprintf(os.Stderr, "Open this URL manually: %s\n", url)
		return
	}
	cmd.Start()
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// -----------------------------------------------------------------------------
// init — wire pool commands + flags
// -----------------------------------------------------------------------------
