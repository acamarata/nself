package commands

// Purpose: wizard steps 1-2 of the "nself doctor --ai" first-run wizard:
// checking/setting up the local Ollama install and the model pool. Inputs
// are a context and doctorAIFlags; outputs are a wizardStepResult.
// Constraints: split out of doctor_ai.go (CLI-R12) as a pure move, no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/installer"
)

func runWizardStep1Local(ctx context.Context, flags doctorAIFlags) wizardStepResult {
	step := "1/4 Local AI (Ollama)"

	if flags.skipOllama {
		if !flags.jsonOut {
			printWizardLine("skip", step, "skipped (--skip-ollama)")
		}
		return wizardStepResult{Step: step, Status: "skipped", Message: "skipped via --skip-ollama"}
	}

	// Check RAM.
	totalMB, err := getTotalMemoryMB()
	if err != nil {
		totalMB = 0
	}
	if totalMB > 0 && totalMB < 4096 {
		msg := fmt.Sprintf("RAM %d MB < 4 GB. Skipping local install, proceeding to pool.", totalMB)
		if !flags.jsonOut {
			printWizardLine("warn", step, msg)
		}
		return wizardStepResult{Step: step, Status: "warn", Message: msg}
	}

	// Check if Ollama is already running (idempotency T-05-03).
	if ollamaHealthy(ctx) {
		if !flags.jsonOut {
			printWizardLine("ok", step, "Ollama already installed and healthy")
		}
		return wizardStepResult{Step: step, Status: "ok", Message: "already installed and healthy"}
	}

	// Interactive prompt: ask user if they want Ollama.
	if !flags.yes && !flags.jsonOut {
		fmt.Printf("\n  Install Ollama for local AI? [Y/n] ")
		if !promptYesNo(true) {
			printWizardLine("skip", step, "user declined")
			return wizardStepResult{Step: step, Status: "skipped", Message: "user declined"}
		}
	}

	// Only attempt install on Linux (matching installer package behavior).
	if runtime.GOOS != "linux" {
		msg := "non-Linux OS: install Ollama manually from https://ollama.com"
		if !flags.jsonOut {
			printWizardLine("warn", step, msg)
		}
		return wizardStepResult{Step: step, Status: "warn", Message: msg}
	}

	// Run the installer.
	opts := installer.InstallOptions{
		Yes:  true,
		JSON: flags.jsonOut,
		LogFn: func(level, msg string, kv map[string]any) {
			if !flags.jsonOut {
				fmt.Fprintf(os.Stderr, "    [%s] %s\n", level, msg)
			}
		},
	}
	res, err := installer.Install(ctx, opts)
	if err != nil {
		msg := fmt.Sprintf("Ollama install failed: %v", err)
		if !flags.jsonOut {
			printWizardLine("fail", step, msg)
		}
		return wizardStepResult{Step: step, Status: "fail", Message: msg}
	}

	msg := fmt.Sprintf("Ollama %s installed, models: %s", res.OllamaVersion, strings.Join(res.ModelsPulled, ", "))
	if !flags.jsonOut {
		printWizardLine("ok", step, msg)
	}
	return wizardStepResult{Step: step, Status: "ok", Message: msg}
}

// ── Step 2: Gemini Pool ─────────────────────────────────────────────

func runWizardStep2Pool(ctx context.Context, flags doctorAIFlags) wizardStepResult {
	step := "2/4 Gemini Pool"

	if flags.skipPool {
		if !flags.jsonOut {
			printWizardLine("skip", step, "skipped (--skip-pool)")
		}
		return wizardStepResult{Step: step, Status: "skipped", Message: "skipped via --skip-pool"}
	}

	// Check if pool already has keys (idempotency T-05-03).
	body, status, err := aiPluginRequest(ctx, "GET", "/ai/pool/status", nil)
	if err == nil && status < 400 {
		var ps struct {
			TotalKeys int `json:"total_keys"`
		}
		if err := json.Unmarshal(body, &ps); err != nil {
			msg := fmt.Sprintf("unmarshal pool status: %v", err)
			if !flags.jsonOut {
				printWizardLine("fail", step, msg)
			}
			return wizardStepResult{Step: step, Status: "fail", Message: msg}
		}
		if ps.TotalKeys > 0 {
			if !flags.jsonOut {
				printWizardLine("ok", step, fmt.Sprintf("%d key(s) already in pool", ps.TotalKeys))
				fmt.Printf("    Add more accounts? [y/N] ")
				if !flags.yes && !promptYesNo(false) {
					return wizardStepResult{Step: step, Status: "ok", Message: fmt.Sprintf("%d keys present, user declined adding more", ps.TotalKeys)}
				}
			} else if flags.yes {
				// --yes mode: don't add more if already present.
				return wizardStepResult{Step: step, Status: "ok", Message: fmt.Sprintf("%d keys present", ps.TotalKeys)}
			}
		}
	}

	// Interactive prompt.
	if !flags.yes && !flags.jsonOut {
		fmt.Printf("\n  Connect a Google account for free Gemini API access? [Y/n] ")
		if !promptYesNo(true) {
			printWizardLine("skip", step, "user declined")
			return wizardStepResult{Step: step, Status: "skipped", Message: "user declined"}
		}
	} else if flags.yes {
		// --yes mode: skip pool setup (no interactive OAuth possible).
		if !flags.jsonOut {
			printWizardLine("skip", step, "skipped in non-interactive mode")
		}
		return wizardStepResult{Step: step, Status: "skipped", Message: "skipped in non-interactive mode (--yes)"}
	}

	// Start OAuth flow.
	oauthBody, oauthStatus, oauthErr := aiPluginRequest(ctx, "POST", "/ai/pool/oauth/start", []byte(`{}`))
	if oauthErr != nil || oauthStatus >= 400 {
		msg := "could not start OAuth flow (is the AI plugin running?)"
		if oauthErr != nil {
			msg = fmt.Sprintf("OAuth start failed: %v", oauthErr)
		}
		if !flags.jsonOut {
			printWizardLine("fail", step, msg)
		}
		return wizardStepResult{Step: step, Status: "fail", Message: msg}
	}

	var oauthResp struct {
		AuthURL string `json:"auth_url"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(oauthBody, &oauthResp); err != nil {
		msg := fmt.Sprintf("unmarshal oauth response: %v", err)
		if !flags.jsonOut {
			printWizardLine("fail", step, msg)
		}
		return wizardStepResult{Step: step, Status: "fail", Message: msg}
	}

	if flags.headless {
		// T-05-02: headless mode - print URL, wait for callback.
		if !flags.jsonOut {
			fmt.Printf("\n    Open this URL to authorize:\n    %s\n\n", oauthResp.AuthURL)
			fmt.Println("    Waiting for OAuth callback (timeout: 10m)...")
		}
		if err := waitForOAuthCallback(ctx, oauthResp.State, 10*time.Minute); err != nil {
			msg := fmt.Sprintf("OAuth timeout: %v", err)
			if !flags.jsonOut {
				printWizardLine("fail", step, msg)
			}
			return wizardStepResult{Step: step, Status: "fail", Message: msg}
		}
	} else {
		// Interactive: open browser.
		if !flags.jsonOut {
			fmt.Println("    Opening browser for Google authorization...")
		}
		openBrowser(oauthResp.AuthURL)
		if !flags.jsonOut {
			fmt.Println("    After authorizing, the key will be auto-provisioned.")
			fmt.Println("    Waiting for OAuth callback (timeout: 10m)...")
		}
		if err := waitForOAuthCallback(ctx, oauthResp.State, 10*time.Minute); err != nil {
			msg := fmt.Sprintf("OAuth timeout: %v", err)
			if !flags.jsonOut {
				printWizardLine("fail", step, msg)
			}
			return wizardStepResult{Step: step, Status: "fail", Message: msg}
		}
	}

	if !flags.jsonOut {
		printWizardLine("ok", step, "Gemini pool key provisioned")
	}
	return wizardStepResult{Step: step, Status: "ok", Message: "key provisioned"}
}

// ── Step 3: Routing ─────────────────────────────────────────────────
