package commands

// Purpose: wizard steps 3-4 of the "nself doctor --ai" first-run wizard:
// configuring routing and verifying the result end to end. Inputs are a
// context and doctorAIFlags; outputs are a wizardStepResult.
// Constraints: split out of doctor_ai.go (CLI-R12) as a pure move, no behavior change.

import (
	"context"
	"fmt"

	"github.com/nself-org/cli/internal/ui"
)

func runWizardStep3Routing(ctx context.Context, flags doctorAIFlags) wizardStepResult {
	step := "3/4 Routing"

	// Check if routing rows already exist (idempotency).
	body, status, err := aiPluginRequest(ctx, "GET", "/ai/routing/config", nil)
	if err == nil && status < 400 && len(body) > 10 {
		if !flags.jsonOut {
			printWizardLine("ok", step, "routing config already present")
		}
		return wizardStepResult{Step: step, Status: "ok", Message: "routing already configured"}
	}

	// Print default routing info.
	if !flags.jsonOut {
		fmt.Println()
		fmt.Println("    Default routing (AI_PROFILE=auto):")
		fmt.Println("      Chat:       local -> oauth -> pool -> paid")
		fmt.Println("      Background: local only")
		fmt.Println("      Embeddings: local -> oauth -> pool")
		fmt.Println()
		printWizardLine("ok", step, "defaults applied")
	}
	return wizardStepResult{Step: step, Status: "ok", Message: "defaults applied"}
}

// ── Step 4: Verification ────────────────────────────────────────────

func runWizardStep4Verify(ctx context.Context, flags doctorAIFlags) wizardStepResult {
	step := "4/4 Verification"

	var passed, failed int

	// Test local chat.
	if ollamaHealthy(ctx) {
		chatOK := testLocalChat(ctx)
		if chatOK {
			if !flags.jsonOut {
				fmt.Printf("    %s Local chat test\n", ui.C(ui.Green, ui.IconSuccess))
			}
			passed++
		} else {
			if !flags.jsonOut {
				fmt.Printf("    %s Local chat test\n", ui.C(ui.Red, ui.IconFailure))
			}
			failed++
		}

		// Test local embeddings.
		embedOK := testLocalEmbed(ctx)
		if embedOK {
			if !flags.jsonOut {
				fmt.Printf("    %s Local embed test\n", ui.C(ui.Green, ui.IconSuccess))
			}
			passed++
		} else {
			if !flags.jsonOut {
				fmt.Printf("    %s Local embed test\n", ui.C(ui.Red, ui.IconFailure))
			}
			failed++
		}
	} else {
		if !flags.jsonOut {
			fmt.Printf("    %s Local AI (Ollama not available)\n", ui.C(ui.Yellow, ui.IconWarning))
		}
	}

	// Test pool chat if plugin-ai is reachable.
	poolBody, poolStatus, poolErr := aiPluginRequest(ctx, "POST", "/ai/pool/test", []byte(`{"all":true}`))
	if poolErr == nil && poolStatus < 400 {
		if !flags.jsonOut {
			fmt.Printf("    %s Pool test\n", ui.C(ui.Green, ui.IconSuccess))
		}
		_ = poolBody
		passed++
	} else {
		if !flags.jsonOut {
			fmt.Printf("    %s Pool test (plugin-ai unreachable or no keys)\n", ui.C(ui.Yellow, ui.IconWarning))
		}
	}

	if failed > 0 {
		msg := fmt.Sprintf("%d passed, %d failed", passed, failed)
		if !flags.jsonOut {
			printWizardLine("fail", step, msg)
		}
		return wizardStepResult{Step: step, Status: "fail", Message: msg}
	}

	msg := fmt.Sprintf("%d checks passed", passed)
	if !flags.jsonOut {
		printWizardLine("ok", step, msg)
	}
	return wizardStepResult{Step: step, Status: "ok", Message: msg}
}

// ── Helpers ─────────────────────────────────────────────────────────
