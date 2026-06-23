package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// Purpose: CLI command to check autonomy tier clearance via evaluation gates.
// Invoked as: nself ci eval gate --tier semi-auto
// Inputs: --tier flag (autonomy tier name, e.g. "supervised", "semi-auto", "full-auto")
// Outputs: Exit 0=cleared, 1=blocked, 2=infra-error, 3=precondition-not-met
// Constraints: Must distinguish between regression (exit 1) and precondition failure (exit 3).
// Must call nself-eval-gate plugin HTTP endpoint /eval/gate/{tier}.
// SPORT: F02-COMMAND-INVENTORY.md (command registry)

var ciEvalGateFlags struct {
	tier   string
	json   bool
	verbose bool
}

var ciEvalGateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Check autonomy tier clearance via evaluation gates",
	Long: `Check whether an autonomy tier has passed its required evaluation gates.

Exit codes:
  0 = tier is cleared (all required eval suites passed)
  1 = tier is blocked (one or more eval suites below threshold) — regression detected
  2 = infrastructure error (plugin unavailable, network issue, etc.)
  3 = precondition not met (e.g., BGE-M3 embedding plugin not wired) — not a regression

Example:
  nself ci eval gate --tier semi-auto
  echo $? # prints 0, 1, 2, or 3`,
	RunE: runCIEvalGate,
}

func init() {
	ciEvalGateCmd.Flags().StringVar(&ciEvalGateFlags.tier, "tier", "", "Autonomy tier to check (required)")
	ciEvalGateCmd.Flags().BoolVar(&ciEvalGateFlags.json, "json", false, "Emit JSON output")
	ciEvalGateCmd.Flags().BoolVar(&ciEvalGateFlags.verbose, "verbose", false, "Show detailed gate status")
	_ = ciEvalGateCmd.MarkFlagRequired("tier")
}

// EvalGateResponse represents the JSON response from /eval/gate/{tier} plugin endpoint.
type EvalGateResponse struct {
	Tier            string   `json:"tier"`
	Cleared         bool     `json:"cleared"`
	BlockingSuites  []string `json:"blocking_suites,omitempty"`
	Error           string   `json:"error,omitempty"`
	Precondition    string   `json:"precondition,omitempty"`
	Message         string   `json:"message,omitempty"`
}

func runCIEvalGate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	tier := ciEvalGateFlags.tier

	if !ciEvalGateFlags.json {
		ui.CommandHeader("nself ci eval gate", fmt.Sprintf("Check tier: %s", tier))
	}

	// ─ Call nself-eval-gate plugin endpoint /eval/gate/{tier} ─
	resp, exitCode := checkEvalGateTier(ctx, tier)

	// ─ Emit output ─
	if ciEvalGateFlags.json {
		outputJSON := map[string]any{
			"tier":      tier,
			"cleared":   resp.Cleared,
			"exit_code": exitCode,
		}
		if resp.BlockingSuites != nil {
			outputJSON["blocking_suites"] = resp.BlockingSuites
		}
		if resp.Precondition != "" {
			outputJSON["precondition"] = resp.Precondition
		}
		if resp.Error != "" {
			outputJSON["error"] = resp.Error
		}
		jsonData, _ := json.MarshalIndent(outputJSON, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		// Human-readable output
		if resp.Cleared {
			ui.Success(fmt.Sprintf("Tier %s is cleared", tier))
		} else {
			if resp.Precondition != "" {
				ui.Error(fmt.Sprintf("Precondition not met: %s", resp.Precondition))
			} else if len(resp.BlockingSuites) > 0 {
				ui.Error(fmt.Sprintf("Tier %s blocked by: %v", tier, resp.BlockingSuites))
			} else if resp.Error != "" {
				ui.Error(fmt.Sprintf("Gate error: %s", resp.Error))
			} else {
				ui.Error(fmt.Sprintf("Tier %s is not cleared", tier))
			}
		}

		if ciEvalGateFlags.verbose && len(resp.BlockingSuites) > 0 {
			fmt.Printf("  Blocking suites: %v\n", resp.BlockingSuites)
		}
	}

	// ─ Exit with contract-defined exit code ─
	// Exit codes per eval-harness-foundation-spec.md §CLI:
	//   0 = cleared
	//   1 = regression (not cleared, not precondition)
	//   2 = infra error
	//   3 = precondition not met
	//
	// Note: we do not use os.Exit() here; instead we return an error for exit code != 0.
	// The CLI framework will translate os.Exit(n) to that exit code.
	if exitCode != 0 {
		return NewExitError(exitCode, resp.Message)
	}

	return nil
}

// checkEvalGateTier calls the nself-eval-gate plugin HTTP endpoint and returns the response + exit code.
// Returns (response, exitCode).
// Exit codes: 0=cleared, 1=blocked, 2=infra-error, 3=precondition-not-met.
func checkEvalGateTier(ctx context.Context, tier string) (*EvalGateResponse, int) {
	// ─ Determine plugin endpoint ─
	// In a running nSelf stack, the plugin runs on localhost at its configured port (3770 per spec).
	// Default: http://localhost:3770
	pluginURL := os.Getenv("NSELF_EVAL_GATE_PLUGIN_URL")
	if pluginURL == "" {
		pluginURL = "http://localhost:3770"
	}

	endpoint := fmt.Sprintf("%s/eval/gate/%s", pluginURL, tier)

	// ─ Make HTTP GET request ─
	client := httptimeout.Default

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return &EvalGateResponse{Error: fmt.Sprintf("failed to construct request: %v", err)}, 2
	}

	// ─ Add internal plugin auth header (X-Nself-Plugin-Token) ─
	// In production, this header is set by nself start when injecting plugin env.
	// For testing/integration, it can be mocked.
	token := os.Getenv("NSELF_PLUGIN_TOKEN")
	if token != "" {
		req.Header.Set("X-Nself-Plugin-Token", token)
	}

	httpResp, err := client.Do(req)
	if err != nil {
		return &EvalGateResponse{Error: fmt.Sprintf("failed to call eval gate: %v", err)}, 2
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return &EvalGateResponse{Error: fmt.Sprintf("failed to read response: %v", err)}, 2
	}

	var gateResp EvalGateResponse
	if err := json.Unmarshal(body, &gateResp); err != nil {
		return &EvalGateResponse{Error: fmt.Sprintf("failed to parse response: %v", err)}, 2
	}

	// ─ Determine exit code based on HTTP status + response content ─
	switch httpResp.StatusCode {
	case http.StatusOK:
		// HTTP 200: check cleared field
		if gateResp.Cleared {
			return &gateResp, 0
		}
		// Not cleared but no precondition error: this is a regression
		return &gateResp, 1

	case http.StatusServiceUnavailable:
		// HTTP 503: precondition not met (e.g., BGE-M3 not wired)
		if gateResp.Precondition == "" {
			gateResp.Precondition = "service unavailable"
		}
		return &gateResp, 3

	case http.StatusBadRequest:
		// HTTP 400: validation error or bad tier name
		if gateResp.Error == "" {
			gateResp.Error = "invalid request"
		}
		return &gateResp, 2

	case http.StatusNotFound:
		// HTTP 404: tier not found
		if gateResp.Error == "" {
			gateResp.Error = "tier not found"
		}
		return &gateResp, 2

	default:
		// Any other error: infra error
		if gateResp.Error == "" {
			gateResp.Error = fmt.Sprintf("unexpected status: %d", httpResp.StatusCode)
		}
		return &gateResp, 2
	}
}

// ExitError implements error and signals a specific exit code.
// Used to return from command RunE with a custom exit code.
type ExitError struct {
	code    int
	message string
}

func NewExitError(code int, message string) *ExitError {
	return &ExitError{code: code, message: message}
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d: %s", e.code, e.message)
}

// ExitCode returns the intended exit code.
func (e *ExitError) ExitCode() int {
	return e.code
}
