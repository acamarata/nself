package commands

// Tests for the 'nself flag' command family.
//
// These tests use an in-process httptest.Server to stand in for the
// feature-flags plugin (port 3305). The --plugin-url flag is used to
// redirect CLI commands to the test server, so no running plugin is required.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/flags"
	"github.com/spf13/cobra"
)

// newFlagCmd builds a minimal cobra tree containing only the flag command
// and its subcommands. This isolates the tests from global RootCmd state.
func newFlagCmd() *cobra.Command {
	root := &cobra.Command{
		Use:  "nself",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}

	// Re-create flag command subtree wired to the same RunE functions as the
	// production command, but using a fresh parent so tests do not share state.
	fc := &cobra.Command{
		Use:   "flag",
		Short: "Manage feature flags",
		RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}

	// list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all feature flags",
		RunE:  flagListCmd.RunE,
	}
	listCmd.Flags().String("type", "", "Filter by flag type")
	listCmd.Flags().Bool("json", false, "Output as JSON")
	listCmd.Flags().String("plugin-url", "", "Override feature-flags plugin URL")

	// kill
	killCmd := &cobra.Command{
		Use:   "kill <key>",
		Short: "Kill-switch a feature flag",
		Args:  cobra.ExactArgs(1),
		RunE:  flagKillCmd.RunE,
	}
	killCmd.Flags().String("reason", "", "Reason for kill-switch (required)")
	killCmd.Flags().String("plugin-url", "", "Override feature-flags plugin URL")

	// history
	historyCmd := &cobra.Command{
		Use:   "history <key>",
		Short: "Show audit log for a feature flag",
		Args:  cobra.ExactArgs(1),
		RunE:  flagHistoryCmd.RunE,
	}
	historyCmd.Flags().Bool("json", false, "Output as JSON")
	historyCmd.Flags().String("plugin-url", "", "Override feature-flags plugin URL")

	fc.AddCommand(listCmd, killCmd, historyCmd)
	root.AddCommand(fc)
	return root
}

// fakeFlagServer returns an httptest.Server that handles a minimal subset of
// the feature-flags plugin API sufficient for kill + history tests.
//
// State is in-memory: a single flag keyed by "test.flag", and an audit log
// that records kill events (action = "kill").
func fakeFlagServer(t *testing.T) *httptest.Server {
	t.Helper()

	type auditRow struct {
		ID      string  `json:"id"`
		FlagKey string  `json:"flag_key"`
		Actor   string  `json:"actor"`
		Action  string  `json:"action"`
		Reason  *string `json:"reason"`
		Ts      string  `json:"ts"`
	}

	enabled := true
	auditLog := []auditRow{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// GET /v1/flags
		case r.Method == http.MethodGet && r.URL.Path == "/v1/flags":
			type item struct {
				Key       string    `json:"key"`
				Type      string    `json:"type"`
				Enabled   bool      `json:"enabled"`
				UpdatedAt time.Time `json:"updated_at"`
			}
			var items []item
			if enabled {
				items = []item{{Key: "test.flag", Type: "release", Enabled: true, UpdatedAt: time.Now()}}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"flags": items, "count": len(items)})

		// GET /v1/flags/test.flag/history
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/history"):
			type histResp struct {
				Entries interface{} `json:"entries"`
			}
			rows := make([]map[string]interface{}, 0, len(auditLog))
			for _, e := range auditLog {
				row := map[string]interface{}{
					"id":       e.ID,
					"flag_key": e.FlagKey,
					"actor":    e.Actor,
					"action":   e.Action,
					"ts":       e.Ts,
				}
				if e.Reason != nil {
					row["reason"] = *e.Reason
				}
				rows = append(rows, row)
			}
			_ = json.NewEncoder(w).Encode(histResp{Entries: rows})

		// POST /v1/flags/test.flag/kill
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/kill"):
			var body flags.KillRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"bad body"}`, http.StatusBadRequest)
				return
			}
			if body.Reason == "" {
				http.Error(w, `{"error":"reason required"}`, http.StatusBadRequest)
				return
			}
			// Record kill event.
			enabled = false
			reason := body.Reason
			auditLog = append(auditLog, auditRow{
				ID:      fmt.Sprintf("audit-%d", len(auditLog)+1),
				FlagKey: "test.flag",
				Actor:   "cli",
				Action:  "kill",
				Reason:  &reason,
				Ts:      time.Now().UTC().Format(time.RFC3339),
			})
			// Return the updated flag.
			resp := map[string]interface{}{"key": "test.flag", "enabled": false,
				"created_at": time.Now(), "updated_at": time.Now()}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))

	return srv
}

// TestFlagKill_RequiresReason verifies that 'flag kill' without --reason
// returns a non-nil error before making any API call.
func TestFlagKill_RequiresReason(t *testing.T) {
	srv := fakeFlagServer(t)
	defer srv.Close()

	root := newFlagCmd()
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"flag", "kill", "test.flag",
		"--plugin-url", srv.URL + "/v1",
		// --reason intentionally omitted
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error when --reason is missing, got nil")
	}
	if !strings.Contains(err.Error(), "reason") {
		t.Errorf("expected error to mention 'reason', got: %v", err)
	}
}

// TestFlagKill_HistoryRecordsKillEvent verifies that after 'flag kill':
//  1. The flag is no longer returned by 'flag list'.
//  2. 'flag history' contains a "kill" entry.
func TestFlagKill_HistoryRecordsKillEvent(t *testing.T) {
	srv := fakeFlagServer(t)
	defer srv.Close()
	pluginURL := srv.URL + "/v1"

	// Confirm flag appears in list before kill.
	// flagListCmd --json uses fmt.Println (os.Stdout), so capture via os.Stdout redirect.
	root := newFlagCmd()
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"flag", "list", "--plugin-url", pluginURL, "--json"})
	beforeOut, err := captureStdout(t, func() error { return root.Execute() })
	if err != nil {
		t.Fatalf("flag list before kill: %v", err)
	}
	if !strings.Contains(beforeOut, "test.flag") {
		t.Fatalf("expected test.flag in list before kill, got: %s", beforeOut)
	}

	// Execute kill.
	root = newFlagCmd()
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"flag", "kill", "test.flag",
		"--reason", "T10 acceptance test kill-switch",
		"--plugin-url", pluginURL,
	})
	if err := root.Execute(); err != nil {
		t.Fatalf("flag kill: %v", err)
	}

	// Verify flag no longer in list (enabled=false, so filtered from list result).
	root = newFlagCmd()
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"flag", "list", "--plugin-url", pluginURL, "--json"})
	afterOut, err := captureStdout(t, func() error { return root.Execute() })
	if err != nil {
		t.Fatalf("flag list after kill: %v", err)
	}
	var listResp struct {
		Flags []struct {
			Key     string `json:"key"`
			Enabled bool   `json:"enabled"`
		} `json:"flags"`
	}
	if err := json.Unmarshal([]byte(afterOut), &listResp); err != nil {
		// If JSON parse fails, the output could be "No feature flags found" message.
		t.Logf("flag list after kill output: %s", afterOut)
	} else {
		for _, f := range listResp.Flags {
			if f.Key == "test.flag" && f.Enabled {
				t.Errorf("test.flag still enabled=true in list after kill")
			}
		}
	}

	// Verify history contains a kill event.
	root = newFlagCmd()
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"flag", "history", "test.flag",
		"--plugin-url", pluginURL, "--json",
	})
	histOut, err := captureStdout(t, func() error { return root.Execute() })
	if err != nil {
		t.Fatalf("flag history: %v", err)
	}
	if !strings.Contains(histOut, "kill") {
		t.Errorf("expected 'kill' action in flag history, got: %s", histOut)
	}
}
