package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/spf13/cobra"
)

// functionNamePattern validates function names: lowercase alphanumeric + hyphens.
var functionNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*$`)

// functionsCmd is the root of the `nself functions` command group.
var functionsCmd = &cobra.Command{
	Use:   "functions",
	Short: "Manage serverless functions",
	Long: `Deploy, list, invoke, stream logs for, and delete serverless functions
running inside the nself functions service.

The functions service must be enabled and running:
  nself service enable functions
  nself build && nself start`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ---------------------------------------------------------------------------
// deploy
// ---------------------------------------------------------------------------

var functionsDeployFlags struct {
	name     string
	runtime  string
	envPairs []string
}

var functionsDeployCmd = &cobra.Command{
	Use:   "deploy <file|dir>",
	Short: "Deploy a function from a file or directory",
	Long: `Copy the given file or directory into ./functions/<name>/ and signal the
functions container to reload. The name defaults to the base filename (without
extension) unless --name is provided.

Examples:
  nself functions deploy hello-world.ts
  nself functions deploy ./my-fn/ --name my-fn
  nself functions deploy handler.py --runtime python`,
	Args: cobra.ExactArgs(1),
	RunE: runFunctionsDeploy,
}

func init() {
	functionsDeployCmd.Flags().StringVar(&functionsDeployFlags.name, "name", "", "Function name (defaults to filename without extension)")
	functionsDeployCmd.Flags().StringVar(&functionsDeployFlags.runtime, "runtime", "", "Override runtime: node, deno, python")
	functionsDeployCmd.Flags().StringArrayVar(&functionsDeployFlags.envPairs, "env", nil, "Environment variable KEY=VALUE (repeatable)")
}

func runFunctionsDeploy(cmd *cobra.Command, args []string) error {
	src := args[0]

	// Determine function name.
	name := functionsDeployFlags.name
	if name == "" {
		base := filepath.Base(src)
		// Strip extension for files.
		if ext := filepath.Ext(base); ext != "" {
			base = strings.TrimSuffix(base, ext)
		}
		name = base
	}

	// Validate name.
	if !functionNamePattern.MatchString(name) {
		return fmt.Errorf("invalid function name %q: use lowercase alphanumeric and hyphens only", name)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	destDir := filepath.Join(cwd, "functions", name)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("creating function directory: %w", err)
	}

	// Copy source into dest.
	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	if fi.IsDir() {
		if err := copyDir(src, destDir); err != nil {
			return fmt.Errorf("copying directory: %w", err)
		}
	} else {
		destFile := filepath.Join(destDir, filepath.Base(src))
		if err := copyFile(src, destFile); err != nil {
			return fmt.Errorf("copying file: %w", err)
		}
	}

	// Write .env vars into function dir if provided.
	if len(functionsDeployFlags.envPairs) > 0 {
		envFile := filepath.Join(destDir, ".env")
		var buf bytes.Buffer
		for _, pair := range functionsDeployFlags.envPairs {
			buf.WriteString(pair)
			buf.WriteByte('\n')
		}
		if err := os.WriteFile(envFile, buf.Bytes(), 0600); err != nil {
			return fmt.Errorf("writing function .env: %w", err)
		}
	}

	// Signal container reload.
	cfg, _, loadErr := loadHealthConfig()
	if loadErr == nil {
		containerID := fmt.Sprintf("%s_functions", cfg.ProjectName)
		reloadCtx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		// nhost/functions watches for a .reload file touch.
		reloadCmd := exec.CommandContext(reloadCtx, "docker", "exec", containerID,
			"touch", "/opt/project/.reload")
		if out, err := reloadCmd.CombinedOutput(); err != nil {
			// Non-fatal — container may not be running yet.
			fmt.Fprintf(os.Stderr, "Warning: could not signal reload: %v (%s)\n", err, strings.TrimSpace(string(out)))
		}
	}

	fmt.Printf("Function %q deployed to ./functions/%s/\n", name, name)
	fmt.Printf("URL: http://localhost:3008/v1/%s\n", name)
	return nil
}

// copyFile copies a single file src → dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies src directory into dst directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0750)
		}
		return copyFile(path, target)
	})
}

// ---------------------------------------------------------------------------
// list
// ---------------------------------------------------------------------------

var functionsListFlags struct {
	jsonOut bool
	runtime string
}

// FunctionInfo describes a deployed function.
type FunctionInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url"`
	Dir    string `json:"dir"`
}

var functionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployed functions",
	RunE:  runFunctionsList,
}

func init() {
	functionsListCmd.Flags().BoolVar(&functionsListFlags.jsonOut, "json", false, "Output as JSON")
	functionsListCmd.Flags().StringVar(&functionsListFlags.runtime, "runtime", "", "Filter by runtime")
}

func runFunctionsList(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	fnDir := filepath.Join(cwd, "functions")
	entries, err := os.ReadDir(fnDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No functions deployed. Use `nself functions deploy` to deploy one.")
			return nil
		}
		return fmt.Errorf("reading functions directory: %w", err)
	}

	// Determine functions service base URL and project name for status probing.
	port := 3008
	cfg, _, cfgErr := loadHealthConfig()
	if cfgErr == nil && cfg.Functions.Port != 0 {
		port = cfg.Functions.Port
	}

	var fns []FunctionInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info := FunctionInfo{
			Name: name,
			Dir:  filepath.Join(fnDir, name),
			URL:  fmt.Sprintf("http://localhost:%d/v1/%s", port, name),
		}

		// Probe liveness (best-effort, non-fatal).
		probeCtx, cancel := context.WithTimeout(cmd.Context(), 3*time.Second)
		status := probeFunctionHealth(probeCtx, port, name)
		cancel()
		info.Status = status

		fns = append(fns, info)
	}

	if len(fns) == 0 {
		fmt.Println("No functions found in ./functions/")
		return nil
	}

	if functionsListFlags.jsonOut {
		data, err := json.MarshalIndent(fns, "", "  ")
		if err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("%-24s  %-10s  %s\n", "NAME", "STATUS", "URL")
	for _, fn := range fns {
		fmt.Printf("%-24s  %-10s  %s\n", fn.Name, fn.Status, fn.URL)
	}
	return nil
}

// probeFunctionHealth probes GET /v1/<name>/healthz and returns "healthy" or "unknown".
func probeFunctionHealth(ctx context.Context, port int, name string) string {
	url := fmt.Sprintf("http://localhost:%d/v1/%s/healthz", port, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "unknown"
	}
	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return "unreachable"
	}
	resp.Body.Close()
	if resp.StatusCode < 500 {
		return "healthy"
	}
	return "unhealthy"
}

// ---------------------------------------------------------------------------
// invoke
// ---------------------------------------------------------------------------

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
	defer resp.Body.Close()

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

// ---------------------------------------------------------------------------
// logs
// ---------------------------------------------------------------------------

var functionsLogsFlags struct {
	follow bool
	since  string
	tail   int
}

var functionsLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Stream logs for a function",
	Long: `Tail or stream logs for a deployed function, filtered by function name.

Examples:
  nself functions logs hello-world
  nself functions logs hello-world --follow
  nself functions logs hello-world --tail 50
  nself functions logs hello-world --since 1h`,
	Args: cobra.ExactArgs(1),
	RunE: runFunctionsLogs,
}

func init() {
	functionsLogsCmd.Flags().BoolVar(&functionsLogsFlags.follow, "follow", false, "Stream logs continuously")
	functionsLogsCmd.Flags().StringVar(&functionsLogsFlags.since, "since", "", "Show logs since duration (e.g. 1h, 30m)")
	functionsLogsCmd.Flags().IntVar(&functionsLogsFlags.tail, "tail", 100, "Number of recent log lines")
}

func runFunctionsLogs(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !functionNamePattern.MatchString(name) {
		return fmt.Errorf("invalid function name %q", name)
	}

	cfg, _, cfgErr := loadHealthConfig()
	containerID := "functions"
	if cfgErr == nil {
		containerID = fmt.Sprintf("%s_functions", cfg.ProjectName)
	}

	dockerArgs := []string{"logs"}
	if functionsLogsFlags.follow {
		dockerArgs = append(dockerArgs, "--follow")
	}
	if functionsLogsFlags.since != "" {
		dockerArgs = append(dockerArgs, "--since", functionsLogsFlags.since)
	}
	dockerArgs = append(dockerArgs, fmt.Sprintf("--tail=%d", functionsLogsFlags.tail))
	dockerArgs = append(dockerArgs, containerID)

	logsCmd := exec.CommandContext(cmd.Context(), "docker", dockerArgs...)
	stdout, err := logsCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe: %w", err)
	}
	stderr, err := logsCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := logsCmd.Start(); err != nil {
		return fmt.Errorf("starting docker logs: %w", err)
	}

	// Filter lines containing the function name and print.
	filterAndPrint := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, name) || strings.Contains(line, "/v1/"+name) {
				fmt.Println(line)
			}
		}
	}

	go filterAndPrint(stderr)
	filterAndPrint(stdout)

	return logsCmd.Wait()
}

// ---------------------------------------------------------------------------
// delete
// ---------------------------------------------------------------------------

var functionsDeleteFlags struct {
	confirm bool
}

var functionsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a deployed function",
	Long: `Remove a deployed function and its directory from ./functions/<name>/.

Requires --confirm to prevent accidental deletion.

Example:
  nself functions delete hello-world --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runFunctionsDelete,
}

func init() {
	functionsDeleteCmd.Flags().BoolVar(&functionsDeleteFlags.confirm, "confirm", false, "Confirm deletion (required)")
}

func runFunctionsDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !functionNamePattern.MatchString(name) {
		return fmt.Errorf("invalid function name %q", name)
	}

	if !functionsDeleteFlags.confirm {
		return fmt.Errorf("--confirm is required to delete a function")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	fnDir := filepath.Join(cwd, "functions", name)
	if _, err := os.Stat(fnDir); os.IsNotExist(err) {
		return fmt.Errorf("function %q not found at %s", name, fnDir)
	}

	if err := os.RemoveAll(fnDir); err != nil {
		return fmt.Errorf("deleting function directory: %w", err)
	}

	fmt.Printf("Function %q deleted.\n", name)
	return nil
}

// ---------------------------------------------------------------------------
// registration
// ---------------------------------------------------------------------------

func init() {
	functionsCmd.AddCommand(functionsDeployCmd)
	functionsCmd.AddCommand(functionsListCmd)
	functionsCmd.AddCommand(functionsInvokeCmd)
	functionsCmd.AddCommand(functionsLogsCmd)
	functionsCmd.AddCommand(functionsDeleteCmd)

	RootCmd.AddCommand(functionsCmd)
}
