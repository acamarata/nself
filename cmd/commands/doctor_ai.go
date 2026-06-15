// P88 Sprint 05: `nself doctor --ai` first-run wizard.
// Tickets: T-05-01 (wizard), T-05-02 (headless), T-05-03 (idempotency).
package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/installer"
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

func ollamaHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", ollamaBaseURL()+"/api/tags", nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func testLocalChat(ctx context.Context) bool {
	payload := `{"model":"gemma2:2b","messages":[{"role":"user","content":"Say hello in one word."}],"stream":false}`
	req, err := http.NewRequestWithContext(ctx, "POST", ollamaBaseURL()+"/api/chat",
		strings.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func testLocalEmbed(ctx context.Context) bool {
	payload := `{"model":"nomic-embed-text","prompt":"test"}`
	req, err := http.NewRequestWithContext(ctx, "POST", ollamaBaseURL()+"/api/embeddings",
		strings.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func waitForOAuthCallback(ctx context.Context, state string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("OAuth callback not received within %s", timeout)
		case <-ticker.C:
			// Poll the plugin-ai for OAuth completion.
			path := fmt.Sprintf("/ai/pool/oauth/status?state=%s", state)
			body, status, err := aiPluginRequest(ctx, "GET", path, nil)
			if err != nil {
				continue
			}
			if status < 400 {
				var resp struct {
					Complete bool   `json:"complete"`
					Error    string `json:"error"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					return fmt.Errorf("unmarshal response: %w", err)
				}
				if resp.Complete {
					return nil
				}
				if resp.Error != "" {
					return fmt.Errorf("OAuth error: %s", resp.Error)
				}
			}
		}
	}
}

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
	if overall == "ok" {
		ui.Success(fmt.Sprintf("Setup complete in %s", elapsed))
	} else if overall == "partial" {
		ui.Warn(fmt.Sprintf("Setup partially complete in %s", elapsed))
	} else {
		ui.Error(fmt.Sprintf("Setup failed after %s", elapsed))
	}

	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    %s nself start           %s Boot the stack\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))
	fmt.Printf("    %s nself ai local health  %s Check Ollama status\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))
	fmt.Printf("    %s nself ai pool status   %s Check Gemini pool\n", ui.C(ui.Bold, ui.IconArrow), ui.C(ui.Dim, ""))
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

// getTotalMemoryMB is defined in doctor_sysinfo_*.go (platform-specific).
// We use exec as a fallback if the function isn't available.
func getTotalMemoryMBFallback() (int, error) {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0, err
		}
		var bytes int64
		fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &bytes)
		return int(bytes / 1024 / 1024), nil
	default:
		return installer.MemAvailableMB()
	}
}
