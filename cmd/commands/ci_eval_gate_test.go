package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCIEvalGateExitCodes(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		cleared      bool
		expectedCode int
		description  string
	}{
		{
			name:         "exit_0_cleared",
			statusCode:   http.StatusOK,
			cleared:      true,
			expectedCode: 0,
			description:  "HTTP 200 with cleared=true should exit 0",
		},
		{
			name:         "exit_1_regression",
			statusCode:   http.StatusOK,
			cleared:      false,
			expectedCode: 1,
			description:  "HTTP 200 with cleared=false should exit 1 (regression)",
		},
		{
			name:         "exit_3_precondition",
			statusCode:   http.StatusServiceUnavailable,
			cleared:      false,
			expectedCode: 3,
			description:  "HTTP 503 should exit 3 (precondition not met)",
		},
		{
			name:         "exit_2_infra_error",
			statusCode:   http.StatusInternalServerError,
			cleared:      false,
			expectedCode: 2,
			description:  "HTTP 500 should exit 2 (infrastructure error)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				resp := GateResponse{
					Cleared: tc.cleared,
					Message: fmt.Sprintf("Test message for %s", tc.name),
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			// Call evaluateGate
			_, exitCode, err := evaluateGate(server.URL, "semi-auto")

			// Verify exit code
			if exitCode != tc.expectedCode {
				t.Errorf("%s: expected exit code %d, got %d. %s",
					tc.name, tc.expectedCode, exitCode, tc.description)
			}

			// For exit code 0, error should be nil. For others, should not be nil.
			if tc.expectedCode == 0 && err != nil {
				t.Errorf("%s: expected no error for exit code 0, got: %v", tc.name, err)
			}
		})
	}
}

func TestCIEvalGateTierFlag(t *testing.T) {
	// Create a mock HTTP server that accepts any tier and returns success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Accept the tier query parameter
		tier := r.URL.Query().Get("tier")
		if tier == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GateResponse{
			Cleared: true,
			Message: "OK",
		})
	}))
	defer server.Close()

	// Test with different tier values
	_, exitCode, _ := evaluateGate(server.URL, "strict")
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 with 'strict' tier, got %d", exitCode)
	}
}

func TestExitErrorInterface(t *testing.T) {
	// Verify ExitError implements error interface
	var _ error = &ExitError{
		Code:    1,
		Message: "test error",
	}

	// Test Error() method
	e := &ExitError{
		Code:    1,
		Message: "test regression",
	}
	if e.Error() != "test regression" {
		t.Errorf("Expected 'test regression', got '%s'", e.Error())
	}
}
