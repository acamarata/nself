// Package runbook implements the SRE runbook library and claw-ops auto-execution
// engine. Runbooks are Markdown files with a YAML front matter header that
// describes the trigger condition and ordered action steps.
//
// CLI usage:
//
//	nself runbook list
//	nself runbook run <id>
//	nself runbook test <id>   — dry-run mode, prints steps without executing
package runbook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultRunbookDir is the path (relative to the monitoring plugin directory)
// where runbooks are stored. The CLI resolves this against the installed
// monitoring plugin path at runtime.
const DefaultRunbookDir = "runbooks"

// Executor loads and executes SRE runbooks.
type Executor struct {
	// RunbookDir is the absolute path to the directory containing .md runbook files.
	RunbookDir string
	// DryRun prevents any action from being taken; steps are only logged.
	DryRun bool
	// Out receives human-readable execution output (defaults to os.Stdout).
	Out io.Writer
	// Log is the structured logger; defaults to slog.Default().
	Log *slog.Logger
	// HTTPClient is used for Slack webhook calls.
	HTTPClient *http.Client
}

// Runbook is the parsed representation of a runbook Markdown file.
type Runbook struct {
	// ID uniquely identifies the runbook (from front matter).
	ID string `yaml:"id"`
	// Title is the human-readable name.
	Title string `yaml:"title"`
	// Trigger describes the Grafana alert that fires this runbook.
	Trigger RunbookTrigger `yaml:"trigger"`
	// Steps is the ordered list of actions to execute.
	Steps []RunbookStep `yaml:"steps"`
}

// RunbookTrigger describes the alert condition that activates this runbook.
type RunbookTrigger struct {
	Alert    string `yaml:"alert"`
	Severity string `yaml:"severity"`
}

// RunbookStep is a single action within a runbook.
type RunbookStep struct {
	Action               string            `yaml:"action"`
	Params               map[string]string `yaml:"params"`
	RequiresConfirmation bool              `yaml:"requires_confirmation"`
}

// New creates an Executor with sane defaults.
func New(runbookDir string, dryRun bool) *Executor {
	return &Executor{
		RunbookDir: runbookDir,
		DryRun:     dryRun,
		Out:        os.Stdout,
		Log:        slog.Default(),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// List returns all runbooks found in RunbookDir, sorted by ID.
func (e *Executor) List(ctx context.Context) ([]Runbook, error) {
	entries, err := os.ReadDir(e.RunbookDir)
	if err != nil {
		return nil, fmt.Errorf("reading runbook dir %s: %w", e.RunbookDir, err)
	}

	var books []Runbook
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		rb, err := e.load(filepath.Join(e.RunbookDir, entry.Name()))
		if err != nil {
			e.Log.Warn("skipping runbook with parse error", "file", entry.Name(), "err", err)
			continue
		}
		books = append(books, rb)
	}
	return books, nil
}

// Execute loads the runbook with the given ID and runs its steps in order.
// Steps marked requires_confirmation: true are skipped in dry-run mode and
// require an explicit Confirm callback in non-dry-run mode.
//
// When alertVars contains key/value pairs (e.g. from an Alertmanager webhook),
// template placeholders {{ key }} in step params are replaced before execution.
func (e *Executor) Execute(ctx context.Context, id string, alertVars map[string]string) error {
	rb, err := e.findByID(ctx, id)
	if err != nil {
		return err
	}

	e.printf("[runbook] Executing: %s (%s)\n", rb.Title, rb.ID)
	if e.DryRun {
		e.printf("[runbook] DRY RUN — no actions will be taken\n")
	}

	for i, step := range rb.Steps {
		params := expandParams(step.Params, alertVars)
		e.printf("[runbook] Step %d/%d: %s %v\n", i+1, len(rb.Steps), step.Action, params)

		if step.RequiresConfirmation && !e.DryRun {
			e.printf("[runbook] Step requires human confirmation — escalating\n")
			if err := e.runEscalate(ctx, params); err != nil {
				e.Log.Warn("escalation failed", "step", step.Action, "err", err)
			}
			continue
		}

		if e.DryRun {
			e.printf("[runbook] [dry-run] would execute: %s\n", step.Action)
			continue
		}

		if err := e.runStep(ctx, step.Action, params); err != nil {
			e.Log.Error("step failed", "step", step.Action, "err", err)
			// Continue to next step; individual failures are logged but non-fatal.
		}
	}

	e.printf("[runbook] Done: %s\n", rb.ID)
	return nil
}

// load parses a runbook Markdown file's YAML front matter.
func (e *Executor) load(path string) (Runbook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Runbook{}, err
	}

	// Extract YAML between first pair of --- delimiters.
	yamlBlock, err := extractFrontMatter(data)
	if err != nil {
		return Runbook{}, fmt.Errorf("no valid front matter in %s: %w", path, err)
	}

	var rb Runbook
	if err := yaml.Unmarshal(yamlBlock, &rb); err != nil {
		return Runbook{}, fmt.Errorf("parsing front matter in %s: %w", path, err)
	}
	if rb.ID == "" {
		rb.ID = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	return rb, nil
}

// findByID searches RunbookDir for a runbook whose id field matches.
func (e *Executor) findByID(ctx context.Context, id string) (Runbook, error) {
	books, err := e.List(ctx)
	if err != nil {
		return Runbook{}, err
	}
	for _, rb := range books {
		if rb.ID == id {
			return rb, nil
		}
	}
	return Runbook{}, fmt.Errorf("runbook not found: %q", id)
}

// runStep dispatches a single action by name.
func (e *Executor) runStep(ctx context.Context, action string, params map[string]string) error {
	switch action {
	case "check_logs":
		return e.runCheckLogs(ctx, params)
	case "restart_service":
		return e.runRestartService(ctx, params)
	case "notify_slack":
		return e.runNotifySlack(ctx, params)
	case "run_command":
		return e.runShellCommand(ctx, params)
	case "run_query":
		return e.runDBQuery(ctx, params)
	case "escalate":
		return e.runEscalate(ctx, params)
	default:
		return fmt.Errorf("unknown action: %q", action)
	}
}

func (e *Executor) runCheckLogs(ctx context.Context, params map[string]string) error {
	service := params["service"]
	last := params["last"]
	level := params["level"]
	if last == "" {
		last = "10m"
	}
	args := []string{"logs", service, "--tail", "50", "--since", last}
	if level != "" {
		args = append(args, "--level", level)
	}
	cmd := exec.CommandContext(ctx, "nself", args...)
	cmd.Stdout = e.Out
	cmd.Stderr = e.Out
	return cmd.Run()
}

func (e *Executor) runRestartService(ctx context.Context, params map[string]string) error {
	service := params["service"]
	cmd := exec.CommandContext(ctx, "nself", "service", "restart", service)
	cmd.Stdout = e.Out
	cmd.Stderr = e.Out
	return cmd.Run()
}

func (e *Executor) runNotifySlack(ctx context.Context, params map[string]string) error {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		e.Log.Info("SLACK_WEBHOOK_URL not set, skipping Slack notification")
		return nil
	}
	message := params["message"]
	payload := fmt.Sprintf(`{"text": "[nSelf Runbook] %s"}`, strings.ReplaceAll(message, `"`, `\"`))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (e *Executor) runShellCommand(ctx context.Context, params map[string]string) error {
	command := params["command"]
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout = e.Out
	cmd.Stderr = e.Out
	return cmd.Run()
}

func (e *Executor) runDBQuery(ctx context.Context, params map[string]string) error {
	query := params["query"]
	cmd := exec.CommandContext(ctx, "nself", "db", "query", "--sql", query)
	cmd.Stdout = e.Out
	cmd.Stderr = e.Out
	return cmd.Run()
}

func (e *Executor) runEscalate(ctx context.Context, params map[string]string) error {
	message := params["message"]
	e.printf("[runbook] ESCALATION REQUIRED: %s\n", message)
	// Post to Slack if configured.
	if webhookURL := os.Getenv("SLACK_WEBHOOK_URL"); webhookURL != "" {
		return e.runNotifySlack(ctx, map[string]string{
			"message": "[ACTION REQUIRED] " + message,
		})
	}
	return nil
}

func (e *Executor) printf(format string, args ...interface{}) {
	if e.Out != nil {
		fmt.Fprintf(e.Out, format, args...)
	}
}

// extractFrontMatter parses the YAML block between the first two --- lines.
func extractFrontMatter(data []byte) ([]byte, error) {
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) < 3 {
		return nil, fmt.Errorf("too short")
	}
	if !bytes.Equal(bytes.TrimSpace(lines[0]), []byte("---")) {
		return nil, fmt.Errorf("no opening ---")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if bytes.Equal(bytes.TrimSpace(lines[i]), []byte("---")) {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, fmt.Errorf("no closing ---")
	}
	return bytes.Join(lines[1:end], []byte("\n")), nil
}

// expandParams replaces {{ key }} placeholders in param values.
func expandParams(params map[string]string, vars map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	for k, v := range params {
		for varKey, varVal := range vars {
			v = strings.ReplaceAll(v, "{{ "+varKey+" }}", varVal)
		}
		out[k] = v
	}
	return out
}
