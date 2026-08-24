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
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
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
