package commands

// Purpose: nself claw session subcommands — start, attach, stop, list.
//   Wires to nself-ai-cc (port 3760) for PTY session lifecycle management.
// Inputs: Session ID (attach/stop), provider name (start).
// Outputs: Session JSON or streaming PTY data.
// Constraints: Uses ports.AICCPort constant; attach uses WS with ?token= auth per OD-E1-04.
// SPORT: F02 — 4 new nself claw session commands.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/nself-org/cli/internal/auth"
	"github.com/nself-org/cli/internal/ports"
	"github.com/spf13/cobra"
)

// aiCCBaseURL returns the base URL for nself-ai-cc.
func aiCCBaseURL() string {
	if u := os.Getenv("NSELF_AICC_URL"); u != "" {
		return u
	}
	return fmt.Sprintf("http://localhost:%d", ports.AICCPort)
}

// aiCCClient returns an HTTP client for nself-ai-cc with JWT auth.
// Returns the client, token, and an error if not logged in.
func aiCCClient() (*http.Client, string, error) {
	af, err := auth.ReadAuthFile()
	if err != nil {
		return nil, "", fmt.Errorf("not logged in\n\nHint: run `nself login` first\nExit: 2")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	return client, af.AccessToken, nil
}

// doAICC performs a JSON request to nself-ai-cc and decodes the response.
func doAICC(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	client, token, err := aiCCClient()
	if err != nil {
		return nil, err
	}
	url := aiCCBaseURL() + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w\n\nHint: ensure nself-ai-cc is running (`nself plugin status nself-ai-cc`)\nExit: 3", err)
	}
	return resp, nil
}

// --- nself claw session ---

var clawSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage Claude Code PTY sessions",
	Long: `Manage Claude Code PTY session lifecycle via nself-ai-cc.

Subcommands:
  start [provider]   Start a new PTY session (default provider: claude)
  list               List all active sessions
  stop <id>          Stop a session
  attach <id>        Attach to a session (WebSocket, interactive)`,
}

// --- nself claw session start ---

var clawSessionStartCmd = &cobra.Command{
	Use:   "start [provider]",
	Short: "Start a new Claude Code PTY session",
	Long: `Start a new Claude Code PTY session via nself-ai-cc.

Provider defaults to "claude". Other providers may be registered in
the gateway. The session_id returned is used with attach/stop.

Exit codes:
  0  Session started successfully
  1  Server error
  2  Authentication error
  3  Connection error (nself-ai-cc not running)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runClawSessionStart,
}

func runClawSessionStart(cmd *cobra.Command, args []string) error {
	provider := "claude"
	if len(args) > 0 {
		provider = args[0]
	}
	if provider == "" {
		return fmt.Errorf("provider must not be empty\n\nHint: use `nself claw session start claude`\nExit: 1")
	}

	body, _ := json.Marshal(map[string]string{"provider": provider})
	import_bytes := &import_reader{data: body}
	resp, err := doAICC(cmd.Context(), http.MethodPost, "/sessions", import_bytes)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s\n\nHint: check nself-ai-cc logs\nExit: 1", resp.StatusCode, string(raw))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode error: %w", err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	return nil
}

// --- nself claw session list ---

var clawSessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active Claude Code PTY sessions",
	Long: `List all active PTY sessions managed by nself-ai-cc.

Exit codes:
  0  Success
  2  Authentication error
  3  Connection error`,
	RunE: runClawSessionList,
}

func runClawSessionList(cmd *cobra.Command, args []string) error {
	resp, err := doAICC(cmd.Context(), http.MethodGet, "/sessions", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s\n\nHint: check nself-ai-cc logs\nExit: 1", resp.StatusCode, string(raw))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode error: %w", err)
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	return nil
}

// --- nself claw session stop ---

var clawSessionStopCmd = &cobra.Command{
	Use:   "stop <id>",
	Short: "Stop a Claude Code PTY session",
	Long: `Stop a running PTY session by ID.

Exit codes:
  0  Session stopped
  1  Server error or session not found
  2  Authentication error
  3  Connection error`,
	Args: cobra.ExactArgs(1),
	RunE: runClawSessionStop,
}

func runClawSessionStop(cmd *cobra.Command, args []string) error {
	id := args[0]
	if id == "" {
		return fmt.Errorf("session ID required\n\nHint: use `nself claw session list` to see active sessions\nExit: 1")
	}

	resp, err := doAICC(cmd.Context(), http.MethodDelete, "/sessions/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s\n\nHint: session may already be stopped or not found\nExit: 1", resp.StatusCode, string(raw))
	}

	fmt.Printf("Session %s stopped.\n", id)
	return nil
}

// --- nself claw session attach ---

var clawSessionAttachCmd = &cobra.Command{
	Use:   "attach <id>",
	Short: "Attach to a Claude Code PTY session",
	Long: `Attach to a running PTY session via WebSocket.

The WebSocket connection uses ?token=<jwt> for authentication (CLI-only
pattern per OD-E1-04; browser clients use Authorization header).

Note: WebSocket attach requires the gorilla/websocket dependency (P5 migration
ticket). Until that ticket lands, attach prints connection details only.

Exit codes:
  0  Attached (or details printed)
  2  Authentication error
  3  Connection error`,
	Args: cobra.ExactArgs(1),
	RunE: runClawSessionAttach,
}

func runClawSessionAttach(cmd *cobra.Command, args []string) error {
	id := args[0]
	if id == "" {
		return fmt.Errorf("session ID required\n\nHint: use `nself claw session list` to see active sessions\nExit: 1")
	}

	af, err := auth.ReadAuthFile()
	if err != nil {
		return fmt.Errorf("not logged in\n\nHint: run `nself login` first\nExit: 2")
	}

	// WebSocket URL uses ?token= auth pattern (OD-E1-04).
	// TODO P5: add gorilla/websocket and open interactive WS session.
	wsURL := fmt.Sprintf("ws://localhost:%d/sessions/%s/attach?token=%s",
		ports.AICCPort, id, af.AccessToken)

	fmt.Printf("Session attach — WebSocket endpoint:\n  %s\n\n", wsURL[:len(wsURL)-len(af.AccessToken)]+"<token>")
	fmt.Println("Hint: interactive attach requires P5 WebSocket implementation.")
	fmt.Println("      Use your WebSocket client to connect to the URL above with your token.")
	return nil
}

// import_reader wraps []byte as io.Reader for http.NewRequest.
type import_reader struct {
	data []byte
	pos  int
}

func (r *import_reader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func init() {
	clawSessionCmd.AddCommand(clawSessionStartCmd)
	clawSessionCmd.AddCommand(clawSessionListCmd)
	clawSessionCmd.AddCommand(clawSessionStopCmd)
	clawSessionCmd.AddCommand(clawSessionAttachCmd)
}
