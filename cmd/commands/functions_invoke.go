package commands

// Purpose: `nself functions invoke` split out of functions.go (CLI-R12
// Batch B mechanical file-size split). Sends an HTTP request to a deployed
// function and prints the response.
// Inputs: cobra command flags (--payload, --auth, --method) and the
// positional function name.
// Outputs: the HTTP status and (pretty-printed, if JSON) response body on
// stdout; a non-nil error on HTTP >= 400 or request failures.
// Constraints: pure move, no behavior change. functionNamePattern and
// functionsCmd (parent) remain in functions.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/spf13/cobra"
)

var functionsInvokeFlags struct {
	payload string
	auth    string
	method  string
}

var functionsInvokeCmd = &cobra.Command{
	Use:   "invoke <name>",
	Short: "Invoke a deployed function",
	Long: `Send an HTTP request to a deployed function and print the response.

Examples:
  nself functions invoke hello-world
  nself functions invoke hello-world --payload '{"name":"Alice"}'
  nself functions invoke hello-world --method GET`,
	Args: cobra.ExactArgs(1),
	RunE: runFunctionsInvoke,
}

func init() {
	functionsInvokeCmd.Flags().StringVar(&functionsInvokeFlags.payload, "payload", "", "JSON payload body")
	functionsInvokeCmd.Flags().StringVar(&functionsInvokeFlags.auth, "auth", "", "Bearer token for Authorization header")
	functionsInvokeCmd.Flags().StringVar(&functionsInvokeFlags.method, "method", "POST", "HTTP method: GET, POST, PUT, PATCH, DELETE")
}

func runFunctionsInvoke(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !functionNamePattern.MatchString(name) {
		return fmt.Errorf("invalid function name %q", name)
	}

	port := 3008
	cfg, _, cfgErr := loadHealthConfig()
	if cfgErr == nil && cfg.Functions.Port != 0 {
		port = cfg.Functions.Port
	}

	url := fmt.Sprintf("http://localhost:%d/v1/%s", port, name)
	method := strings.ToUpper(functionsInvokeFlags.method)

	var body io.Reader
	if functionsInvokeFlags.payload != "" {
		body = strings.NewReader(functionsInvokeFlags.payload)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	if functionsInvokeFlags.payload != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	// Auth token is set but not echoed in logs.
	if functionsInvokeFlags.auth != "" {
		req.Header.Set("Authorization", "Bearer "+functionsInvokeFlags.auth)
	}

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return fmt.Errorf("invoking %s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	fmt.Printf("HTTP %d\n", resp.StatusCode)

	// Pretty-print JSON responses.
	var pretty bytes.Buffer
	if json.Indent(&pretty, respBody, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(respBody))
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("function returned HTTP %d", resp.StatusCode)
	}
	return nil
}
