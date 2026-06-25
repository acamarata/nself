package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/nself-org/cli/internal/health"
	"github.com/nself-org/cli/internal/ui"
	"github.com/nself-org/cli/internal/watchdog"

	"github.com/spf13/cobra"
)

var watchdogCmd = &cobra.Command{
	Use:   "watchdog",
	Short: "Self-healing container watchdog with circuit breaker",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var watchdogStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show watchdog status and circuit breaker states",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")

		// Create a read-only watchdog to check file-based state
		cfg := watchdog.DefaultConfig()
		docker := health.NewShellDockerClient("nself")
		wd := watchdog.New(cfg, docker)
		status := wd.GetStatus()

		if jsonOut {
			data, err := json.MarshalIndent(status, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		ui.CommandHeader("Watchdog Status", "")

		if status.Running {
			ui.Success("Watchdog is running")
		} else {
			ui.Warn("Watchdog is not running")
		}

		openCircuits := 0
		if len(status.Circuits) == 0 {
			ui.Bullet("No circuit breakers tracked")
		} else {
			ui.Section("Circuit Breakers")
			for _, c := range status.Circuits {
				state := string(c.State)
				switch c.State {
				case watchdog.CircuitPermanentOpen:
					openCircuits++
					fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s: %s (since %s, %d consecutive open windows) — run: nself watchdog reset %s\n",
						ui.C(ui.Red, ui.IconFailure), c.Service, state,
						c.PermanentOpenAt.Format(time.RFC3339), c.ConsecutiveOpenWindows, c.Service)
				case watchdog.CircuitOpen:
					openCircuits++
					fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s: %s (tripped at %s, %d attempts)\n",
						ui.C(ui.Red, ui.IconFailure), c.Service, state,
						c.TrippedAt.Format(time.RFC3339), c.Attempts)
				default:
					fmt.Printf("  %s %s: %s (%d attempts)\n",
						ui.C(ui.Green, ui.IconSuccess), c.Service, state, c.Attempts)
				}
			}
		}

		// Exit 2 when any circuit is open (including PERMANENT_OPEN) so scripts can react.
		if openCircuits > 0 {
			cmd.Root().SetContext(context.WithValue(cmd.Root().Context(), exitCodeKey, 2))
		}
		return nil
	},
}

var watchdogResetCmd = &cobra.Command{
	Use:   "reset-breakers",
	Short: "Reset all tripped circuit breakers to closed state",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := watchdog.DefaultConfig()
		docker := health.NewShellDockerClient("nself")
		wd := watchdog.New(cfg, docker)
		count := wd.ResetBreakers()
		if count == 0 {
			ui.Bullet("No circuit breakers were tripped")
		} else {
			ui.Success(fmt.Sprintf("Reset %d circuit breaker(s)", count))
		}
		return nil
	},
}

// watchdogResetServiceCmd resets a single service's circuit breaker, including PERMANENT_OPEN.
// Requires --force in production to prevent accidental resets without operator review.
var watchdogResetServiceCmd = &cobra.Command{
	Use:   "reset <service>",
	Short: "Reset a specific service circuit breaker (including PERMANENT_OPEN)",
	Long: `Reset a named service's circuit breaker to CLOSED state.

This is the only supported recovery path for a service stuck in PERMANENT_OPEN.
In production (NSELF_ENV=prod) the --force flag is required to prevent accidental resets.

After a reset, the watchdog will resume automated restart attempts on the next poll cycle.
If the service continues to fail, it will open the circuit breaker again and eventually
return to PERMANENT_OPEN after WATCHDOG_PERMANENT_OPEN_THRESHOLD consecutive open windows.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		service := args[0]
		force, _ := cmd.Flags().GetBool("force")

		nSelfEnv := os.Getenv("NSELF_ENV")
		if nSelfEnv == "prod" && !force {
			return fmt.Errorf("resetting circuit breaker in production requires --force flag; review the runbook first: .claude/docs/operations/watchdog-runbook.md")
		}

		op := operatorIdentity()
		cfg := watchdog.DefaultConfig()
		docker := health.NewShellDockerClient("nself")
		wd := watchdog.New(cfg, docker)

		ok := wd.ResetService(service, op)
		if !ok {
			return fmt.Errorf("service %q has no tracked circuit breaker", service)
		}
		ui.Success(fmt.Sprintf("Circuit breaker for %q reset to CLOSED by %s", service, op))
		return nil
	},
}

// operatorIdentity returns the current OS user for audit logging.
func operatorIdentity() string {
	u, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return u.Username
}

var watchdogHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show watchdog event history",
	RunE: func(cmd *cobra.Command, args []string) error {
		sinceStr, _ := cmd.Flags().GetString("since")
		jsonOut, _ := cmd.Flags().GetBool("json")

		var since time.Duration
		if sinceStr != "" {
			var err error
			since, err = time.ParseDuration(sinceStr)
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", sinceStr, err)
			}
		}

		cfg := watchdog.DefaultConfig()
		docker := health.NewShellDockerClient("nself")
		wd := watchdog.New(cfg, docker)
		events := wd.GetHistory(since)

		if jsonOut {
			data, err := json.MarshalIndent(events, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		if len(events) == 0 {
			ui.Bullet("No watchdog events recorded")
			return nil
		}

		ui.CommandHeader("Watchdog History", fmt.Sprintf("%d events", len(events)))
		for _, e := range events {
			fmt.Printf("  %s  %-20s  %-15s  %s\n",
				e.Timestamp.Format("2006-01-02 15:04:05"),
				e.Service, e.Action, e.Detail)
		}
		return nil
	},
}

var watchdogTestAlertCmd = &cobra.Command{
	Use:   "test-alert",
	Short: "Send a test alert through all configured channels (TG + email)",
	RunE: func(cmd *cobra.Command, args []string) error {
		service, _ := cmd.Flags().GetString("service")
		severity, _ := cmd.Flags().GetString("severity")

		ui.CommandHeader("Watchdog Test Alert", "")
		delivered, errs := watchdog.TestAlert(service, severity)

		for _, ch := range delivered {
			ui.Success(fmt.Sprintf("Delivered to %s", ch))
		}
		for _, err := range errs {
			ui.Warn(fmt.Sprintf("Channel error: %v", err))
		}

		if len(delivered) == 0 {
			return fmt.Errorf("no alerts delivered — check WATCHDOG_TG_BOT_TOKEN and SMTP env vars")
		}
		return nil
	},
}

func init() {
	watchdogStatusCmd.Flags().Bool("json", false, "JSON output")
	watchdogHistoryCmd.Flags().String("since", "24h", "Show events since duration (e.g. 24h, 7d)")
	watchdogHistoryCmd.Flags().Bool("json", false, "JSON output")
	watchdogTestAlertCmd.Flags().String("service", "test-service", "Service name for the test alert")
	watchdogTestAlertCmd.Flags().String("severity", "critical", "Severity level (warning, critical)")
	watchdogResetServiceCmd.Flags().Bool("force", false, "Required when NSELF_ENV=prod")

	watchdogCmd.AddCommand(
		watchdogStatusCmd,
		watchdogResetCmd,
		watchdogResetServiceCmd,
		watchdogHistoryCmd,
		watchdogTestAlertCmd,
	)
	RootCmd.AddCommand(watchdogCmd)
}
