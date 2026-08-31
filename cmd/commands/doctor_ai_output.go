package commands

// Purpose: terminal I/O helpers for the AI wizard: yes/no prompts, per-step
// status lines, the final banner, and the getTotalMemoryMBFallback probe.
// Inputs are wizard results/timings; outputs are printed text or a memory
// size in MB.
// Constraints: split out of doctor_ai.go (CLI-R12) as a pure move, no behavior change.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"
)

func promptYesNo(defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}

func printWizardLine(status, step, msg string) {
	var icon string
	switch status {
	case "ok":
		icon = ui.C(ui.Green, ui.IconSuccess)
	case "skip":
		icon = ui.C(ui.Dim, "-")
	case "warn":
		icon = ui.C(ui.Yellow, ui.IconWarning)
	case "fail":
		icon = ui.C(ui.Red, ui.IconFailure)
	}
	fmt.Printf("  %s [%s] %s\n", icon, step, msg)
}

func printWizardBanner(results []wizardStepResult, elapsed time.Duration) {
	fmt.Println()
	ui.Separator()
	fmt.Println()

	for _, r := range results {
		var marker string
		switch r.Status {
		case "ok":
			marker = ui.C(ui.Green, ui.IconSuccess)
		case "skipped":
			marker = ui.C(ui.Dim, "-")
		case "warn":
			marker = ui.C(ui.Yellow, ui.IconWarning)
		case "fail":
			marker = ui.C(ui.Red, ui.IconFailure)
		}
		fmt.Printf("  %s %s: %s", marker, r.Step, r.Message)
		if r.Elapsed != "" {
			fmt.Printf(" (%s)", r.Elapsed)
		}
		fmt.Println()
	}

	fmt.Println()
	overall := summaryStatus(results)
	switch overall {
	case "ok":
		ui.Success(fmt.Sprintf("Setup complete in %s", elapsed))
	case "partial":
		ui.Warn(fmt.Sprintf("Setup partially complete in %s", elapsed))
	default:
		ui.Error(fmt.Sprintf("Setup failed after %s", elapsed))
	}

	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    %s nself start           %s Boot the stack\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))

	// `nself ai` moved to the ai-cli plugin under CLI-R11. Suggesting it to
	// someone who has not installed it sends them into an install hint from a
	// screen that is meant to be a list of things they can do right now, so the
	// suggestion appears only when the command is actually there.
	if plugin.IsCommandInstalled("ai") {
		fmt.Printf("    %s nself ai local health  %s Check Ollama status\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))
		fmt.Printf("    %s nself ai pool status   %s Check Gemini pool\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))
	} else {
		fmt.Printf("    %s nself install ai-cli   %s AI commands (Ollama, Gemini pool)\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))
	}

	fmt.Printf("    %s localhost:3021          %s Admin UI\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))
	fmt.Println()
}

func summaryStatus(results []wizardStepResult) string {
	hasFail := false
	hasOK := false
	for _, r := range results {
		if r.Status == "fail" {
			hasFail = true
		}
		if r.Status == "ok" {
			hasOK = true
		}
	}
	if hasFail && hasOK {
		return "partial"
	}
	if hasFail {
		return "fail"
	}
	return "ok"
}
