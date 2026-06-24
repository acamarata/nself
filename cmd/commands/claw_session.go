package commands

// Purpose: Cobra command group for `nself claw session` subcommands.
//   Wires four session lifecycle commands (start, attach, stop, list) to the
//   nself-ai-cc service at AICCPort (3760) via internal/claw/session.go.
// Inputs:  provider name (start), session ID (attach/stop), nSelf JWT (attach)
// Outputs: session_id on start; table of sessions on list; WS URL on attach
// Constraints:
//   - Error UX follows spec §12: "Error: ...\nHint: ...\nExit: N"
//   - attach uses ?token= pattern (CLI-only; see OD-E1-04 for rationale)
//   - No env-var override for target port per spec §11
// SPORT: F02-COMMAND-INVENTORY.md (nself claw session start/attach/stop/list)

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nself-org/cli/internal/claw"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/spf13/cobra"
)

// clawSessionCmd is the parent command group for `nself claw session`.
var clawSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage nself-ai-cc PTY sessions",
	Long: `Manage Claude Code / Codex PTY sessions via nself-ai-cc (port 3760).

Subcommands:
  start <provider>   Start a new session for the given AI provider
  attach <id>        Attach to an active session via WebSocket
  stop <id>          Stop (terminate) an active session
  list               List all sessions (active and recent)

Exit codes:
  0  Success
  1  User input error (missing argument, invalid provider)
  2  Infra error (service unreachable)
  3  Server error (session not found, stop failed)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// clawSessionStartCmd starts a new PTY session via POST /sessions on nself-ai-cc.
var clawSessionStartCmd = &cobra.Command{
	Use:   "start <provider>",
	Short: "Start a new PTY session for an AI provider",
	Long: `Start a new PTY session for the given AI provider via nself-ai-cc.

The provider is the AI binary that nself-ai-cc will spawn (e.g. claude, codex).
On success, the session ID is printed to stdout.

Examples:
  nself claw session start claude
  nself claw session start codex`,
	Args: cobra.ExactArgs(1),
	RunE: runClawSessionStart,
}

// clawSessionAttachCmd attaches to an active session via WebSocket.
var clawSessionAttachCmd = &cobra.Command{
	Use:   "attach <session-id>",
	Short: "Attach to an active PTY session via WebSocket",
	Long: `Attach to an active session using the nself-ai-cc WebSocket endpoint.

The WebSocket URL is printed so you can connect with any WS client.
Authentication uses the ?token= query parameter (CLI-only pattern per OD-E1-04).

Examples:
  nself claw session attach 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	RunE: runClawSessionAttach,
}

// clawSessionStopCmd terminates an active session.
var clawSessionStopCmd = &cobra.Command{
	Use:   "stop <session-id>",
	Short: "Stop (terminate) an active PTY session",
	Long: `Stop an active session by sending DELETE /sessions/:id to nself-ai-cc.

Examples:
  nself claw session stop 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	RunE: runClawSessionStop,
}

// clawSessionListCmd lists all sessions.
var clawSessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all PTY sessions",
	Long: `List active and recent PTY sessions from nself-ai-cc.

Output is a tab-separated table: ID, Provider, Status, Started, Ended.

Examples:
  nself claw session list`,
	RunE: runClawSessionList,
}

func init() {
	clawSessionCmd.AddCommand(clawSessionStartCmd)
	clawSessionCmd.AddCommand(clawSessionAttachCmd)
	clawSessionCmd.AddCommand(clawSessionStopCmd)
	clawSessionCmd.AddCommand(clawSessionListCmd)
}

// sessionHTTPClient returns an http.Client suitable for nself-ai-cc control calls.
// Uses a short timeout; streaming attach uses WebSocket (not this client).
func sessionHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func runClawSessionStart(cmd *cobra.Command, args []string) error {
	provider := args[0]
	if provider == "" {
		fmt.Fprintf(os.Stderr, "Error: provider is required\nHint: run `nself claw session start <provider>` (e.g. claude, codex)\n")
		return &plugin.ExitCodeError{Code: errs.ExitUserError}
	}

	client := sessionHTTPClient()
	sessionID, err := claw.StartSession(cmd.Context(), client, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return &plugin.ExitCodeError{Code: inferSessionExitCode(err)}
	}

	fmt.Println(sessionID)
	return nil
}

func runClawSessionAttach(cmd *cobra.Command, args []string) error {
	sessionID := args[0]
	if sessionID == "" {
		fmt.Fprintf(os.Stderr, "Error: session ID is required\nHint: run `nself claw session list` to find active session IDs\n")
		return &plugin.ExitCodeError{Code: errs.ExitUserError}
	}

	// nSelf JWT for WS auth (?token= pattern per OD-E1-04).
	// The claw API key doubles as the nSelf JWT for session-relay auth.
	nselfJWT := clawAPIKey()

	wsURL, err := claw.AttachSessionURL(sessionID, nselfJWT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return &plugin.ExitCodeError{Code: errs.ExitUserError}
	}

	// Print the WS URL; interactive relay requires a WS client.
	// Full bidirectional relay is deferred to P5 (tracking: P5-E1-WS-RELAY).
	fmt.Printf("WebSocket session URL:\n  %s\n", wsURL)
	fmt.Println()
	fmt.Println("Connect with any WebSocket client, e.g.:")
	fmt.Printf("  websocat %s\n", wsURL)
	return nil
}

func runClawSessionStop(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	client := sessionHTTPClient()
	if err := claw.StopSession(cmd.Context(), client, sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return &plugin.ExitCodeError{Code: inferSessionExitCode(err)}
	}

	fmt.Printf("Session %s stopped.\n", sessionID)
	return nil
}

func runClawSessionList(cmd *cobra.Command, args []string) error {
	client := sessionHTTPClient()
	sessions, err := claw.ListSessions(cmd.Context(), client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return &plugin.ExitCodeError{Code: inferSessionExitCode(err)}
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROVIDER\tSTATUS\tSTARTED\tENDED")
	for _, s := range sessions {
		ended := "-"
		if s.EndedAt != nil {
			ended = s.EndedAt.UTC().Format(time.RFC3339)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			s.ID,
			s.Provider,
			s.Status,
			s.StartedAt.UTC().Format(time.RFC3339),
			ended,
		)
	}
	return w.Flush()
}

// inferSessionExitCode maps nself-ai-cc error messages to canonical exit codes.
// Returns ExitInfraError (2) for unreachable service; ExitUserError (1) for bad input.
func inferSessionExitCode(err error) int {
	if err == nil {
		return errs.ExitOK
	}
	msg := err.Error()
	if strings.Contains(msg, "unreachable") || strings.Contains(msg, "not running") {
		return errs.ExitInfraError
	}
	if strings.Contains(msg, "required") || strings.Contains(msg, "invalid") {
		return errs.ExitUserError
	}
	return errs.ExitInfraError
}

// clawSessionContext returns a background context for session HTTP calls.
// Exported for testing.
func clawSessionContext() context.Context {
	return context.Background()
}
