package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// ExitError represents a CLI exit code error.
// Exit codes: 0=cleared, 1=regression, 2=infra-error, 3=precondition-not-met
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

// GateResponse is the response from the eval gate endpoint.
type GateResponse struct {
	Cleared bool   `json:"cleared"`
	Message string `json:"message,omitempty"`
}

var ciEvalGateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Evaluate CI quality gate",
	Long: `Evaluate CI quality gate for the current state.
Exit codes:
  0 = gate cleared (all checks passed)
  1 = regression (gate blocked due to failed checks)
  2 = infrastructure error (transient failure, not a regression)
  3 = precondition not met (gate not ready to evaluate, not a regression)`,
	RunE: runCIEvalGate,
}

func init() {
	ciEvalGateCmd.Flags().StringP("tier", "t", "semi-auto", "Evaluation tier (semi-auto, full, strict)")
	ciEvalGateCmd.Flags().StringP("endpoint", "e", "", "HTTP endpoint for gate evaluation (default: nself backend)")
}

func runCIEvalGate(cmd *cobra.Command, args []string) error {
	tier, _ := cmd.Flags().GetString("tier")
	endpoint, _ := cmd.Flags().GetString("endpoint")

	if endpoint == "" {
		endpoint = "http://localhost:3000/api/v1/ci/eval/gate"
	}

	// Evaluate the gate
	response, exitCode, err := evaluateGate(endpoint, tier)
	if err != nil {
		return err
	}

	// Handle the result
	switch exitCode {
	case 0:
		fmt.Println("✓ Eval gate cleared")
		return nil
	case 1:
		return &ExitError{
			Code:    1,
			Message: fmt.Sprintf("Eval gate blocked: %s", response.Message),
		}
	case 2:
		return &ExitError{
			Code:    2,
			Message: fmt.Sprintf("Infrastructure error: %s", response.Message),
		}
	case 3:
		return &ExitError{
			Code:    3,
			Message: fmt.Sprintf("Precondition not met: %s", response.Message),
		}
	default:
		return &ExitError{
			Code:    2,
			Message: fmt.Sprintf("Unknown exit code: %d", exitCode),
		}
	}
}

func evaluateGate(endpoint, tier string) (*GateResponse, int, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request to endpoint
	url := fmt.Sprintf("%s?tier=%s", endpoint, tier)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 2, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 2, fmt.Errorf("gate evaluation failed: %w", err)
	}
	defer resp.Body.Close()

	// Decode response
	var gateResp GateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gateResp); err != nil {
		return nil, 2, fmt.Errorf("failed to decode gate response: %w", err)
	}

	// Map HTTP status codes to exit codes
	switch resp.StatusCode {
	case http.StatusOK:
		if gateResp.Cleared {
			return &gateResp, 0, nil
		}
		return &gateResp, 1, nil
	case http.StatusServiceUnavailable:
		return &gateResp, 3, nil
	default:
		return &gateResp, 2, nil
	}
}
