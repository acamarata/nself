// P88 Sprint 05: `nself doctor --ai` first-run wizard.
// Tickets: T-05-01 (wizard), T-05-02 (headless), T-05-03 (idempotency).
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nself-org/cli/internal/ui"
)

// doctorAIFlags holds the flags for `nself doctor --ai`.
type doctorAIFlags struct {
	yes        bool // --yes: non-interactive, accept defaults
	skipOllama bool // --skip-ollama: skip local AI setup
	skipPool   bool // --skip-pool: skip Gemini pool setup
	headless   bool // --headless: print OAuth URL, don't open browser
	jsonOut    bool // --json: machine output
}

// wizardStepResult tracks one wizard step outcome.
type wizardStepResult struct {
	Step    string `json:"step"`
	Status  string `json:"status"` // "ok", "skipped", "warn", "fail"
	Message string `json:"message"`
	Elapsed string `json:"elapsed,omitempty"`
}

// runDoctorAI is the entry point for `nself doctor --ai`.
func runDoctorAI(ctx context.Context, flags doctorAIFlags) error {
	started := time.Now()
	var results []wizardStepResult

	if !flags.jsonOut {
		ui.CommandHeader("nSelf AI Setup", "Zero-config unlimited AI in 30 seconds")
		fmt.Println()
	}

	// ── Step 1/4: Local AI (Ollama) ─────────────────────────────────
	stepStart := time.Now()
	r := runWizardStep1Local(ctx, flags)
	r.Elapsed = time.Since(stepStart).Round(time.Millisecond).String()
	results = append(results, r)

	// ── Step 2/4: Gemini Pool ───────────────────────────────────────
	stepStart = time.Now()
	r = runWizardStep2Pool(ctx, flags)
	r.Elapsed = time.Since(stepStart).Round(time.Millisecond).String()
	results = append(results, r)

	// ── Step 3/4: Routing defaults ──────────────────────────────────
	stepStart = time.Now()
	r = runWizardStep3Routing(ctx, flags)
	r.Elapsed = time.Since(stepStart).Round(time.Millisecond).String()
	results = append(results, r)

	// ── Step 4/4: Verification ──────────────────────────────────────
	stepStart = time.Now()
	r = runWizardStep4Verify(ctx, flags)
	r.Elapsed = time.Since(stepStart).Round(time.Millisecond).String()
	results = append(results, r)

	totalElapsed := time.Since(started).Round(time.Millisecond)

	if flags.jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"status":  summaryStatus(results),
			"steps":   results,
			"elapsed": totalElapsed.String(),
		})
	}

	// ── Banner ──────────────────────────────────────────────────────
	printWizardBanner(results, totalElapsed)
	return nil
}

// ── Step 1: Local AI (Ollama) ───────────────────────────────────────
