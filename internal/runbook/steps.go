package runbook

// Purpose: per-action step execution (log check, restart, Slack notify, shell command, DB query, escalate) plus front-matter parsing and param expansion, backing Execute in executor.go.
// Inputs: an action name and its string-keyed params, and alert variables for expansion.
// Outputs: side effects per action (log output, restarted service, sent notification, run command) or an error.
// Constraints: split out of executor.go as a pure move (CLI-R12); no behavior change.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

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
	defer func() { _ = resp.Body.Close() }()
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
		_, _ = fmt.Fprintf(e.Out, format, args...)
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
