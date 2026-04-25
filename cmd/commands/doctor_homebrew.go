package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// checkHomebrewPostgres detects whether Homebrew Postgres is installed and running on macOS,
// which can conflict with the dockerized Postgres on the standard 5432 port.
// This is part of T9 S186-T06 install/onboarding hints.
func checkHomebrewPostgres(verbose bool) []doctorCheckResult {
	if runtime.GOOS != "darwin" {
		return nil
	}

	if _, err := exec.LookPath("brew"); err != nil {
		return nil
	}

	out, err := exec.Command("brew", "services", "list").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		status := fields[1]
		if !strings.HasPrefix(name, "postgresql") && name != "postgres" {
			continue
		}
		if status == "started" || status == "running" {
			displayName := "Homebrew Postgres"
			msg := fmt.Sprintf("%s is running and may conflict with nSelf's dockerized Postgres on port 5432. Stop it with: brew services stop %s", name, name)
			printCheck("warn", displayName, msg, verbose)
			return []doctorCheckResult{{Name: displayName, Status: "warn", Message: msg}}
		}
	}

	return nil
}
