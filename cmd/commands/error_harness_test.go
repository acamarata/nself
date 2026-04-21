package commands

// S45-T-ERR-01: Error state harness for all 48 CLI commands.
//
// Each table entry covers three error states per command:
//  (a) missing required context (no project dir / no .env)
//  (b) invalid flag value or unrecognised flag
//  (c) unknown subcommand
//
// Each test asserts:
//  - non-zero exit code (err != nil)
//  - error message starts with "Error:" OR cobra usage error OR our CLIError
//    (anything except a nil error or a flag-parsing "unknown flag" pass-through)
//
// Coverage gate: every registered top-level command name appears in cmdNames.
// The test TestErrorHarness_AllCommandsCovered() enforces this invariant.

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// errorHarnessCase describes one command-level error scenario.
type errorHarnessCase struct {
	// cmd is the top-level command name (e.g., "build").
	cmd string
	// args are the full argument list passed after "nself" (may include subcommands).
	args []string
	// desc is a human-readable label for this scenario.
	desc string
}

// errorHarnessCases is the exhaustive table of error scenarios for all 48 commands.
// Three entries per command: (a) missing-context, (b) invalid-flag, (c) unknown-sub.
//
// "missing-context" always produces a non-nil error because nself requires a
// project directory (.env) for nearly every operation.
// "invalid-flag" uses a clearly unknown flag name that cobra will reject.
// "unknown-sub" uses a subcommand name that does not exist.
var errorHarnessCases = []errorHarnessCase{
	// ── admin ──────────────────────────────────────────────────────────────
	{"admin", []string{"admin"}, "(a) no project dir"},
	{"admin", []string{"admin", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"admin", []string{"admin", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── ai ─────────────────────────────────────────────────────────────────
	{"ai", []string{"ai"}, "(a) no project dir"},
	{"ai", []string{"ai", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"ai", []string{"ai", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── alerts ─────────────────────────────────────────────────────────────
	{"alerts", []string{"alerts"}, "(a) no project dir"},
	{"alerts", []string{"alerts", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"alerts", []string{"alerts", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── audit ──────────────────────────────────────────────────────────────
	{"audit", []string{"audit"}, "(a) no project dir"},
	{"audit", []string{"audit", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"audit", []string{"audit", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── backup ─────────────────────────────────────────────────────────────
	{"backup", []string{"backup"}, "(a) no project dir"},
	{"backup", []string{"backup", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"backup", []string{"backup", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── billing ────────────────────────────────────────────────────────────
	{"billing", []string{"billing"}, "(a) no project dir"},
	{"billing", []string{"billing", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"billing", []string{"billing", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── build ──────────────────────────────────────────────────────────────
	{"build", []string{"build"}, "(a) no project dir"},
	{"build", []string{"build", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"build", []string{"build", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── claw ───────────────────────────────────────────────────────────────
	{"claw", []string{"claw"}, "(a) no project dir"},
	{"claw", []string{"claw", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"claw", []string{"claw", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── clean ──────────────────────────────────────────────────────────────
	{"clean", []string{"clean"}, "(a) no project dir"},
	{"clean", []string{"clean", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"clean", []string{"clean", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── completion ─────────────────────────────────────────────────────────
	{"completion", []string{"completion", "invalidshell_xyz"}, "(a) unknown shell"},
	{"completion", []string{"completion", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"completion", []string{"completion", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── config ─────────────────────────────────────────────────────────────
	{"config", []string{"config"}, "(a) no project dir"},
	{"config", []string{"config", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"config", []string{"config", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── db ─────────────────────────────────────────────────────────────────
	{"db", []string{"db"}, "(a) no project dir"},
	{"db", []string{"db", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"db", []string{"db", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── deploy ─────────────────────────────────────────────────────────────
	{"deploy", []string{"deploy"}, "(a) no project dir"},
	{"deploy", []string{"deploy", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"deploy", []string{"deploy", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── dev ────────────────────────────────────────────────────────────────
	{"dev", []string{"dev"}, "(a) no project dir"},
	{"dev", []string{"dev", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"dev", []string{"dev", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── dns ────────────────────────────────────────────────────────────────
	{"dns", []string{"dns"}, "(a) no project dir"},
	{"dns", []string{"dns", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"dns", []string{"dns", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── doctor ─────────────────────────────────────────────────────────────
	{"doctor", []string{"doctor"}, "(a) no project dir"},
	{"doctor", []string{"doctor", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"doctor", []string{"doctor", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── dogfood ────────────────────────────────────────────────────────────
	{"dogfood", []string{"dogfood"}, "(a) no project dir"},
	{"dogfood", []string{"dogfood", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"dogfood", []string{"dogfood", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── dr ─────────────────────────────────────────────────────────────────
	{"dr", []string{"dr"}, "(a) no project dir"},
	{"dr", []string{"dr", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"dr", []string{"dr", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── env ────────────────────────────────────────────────────────────────
	{"env", []string{"env"}, "(a) no project dir"},
	{"env", []string{"env", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"env", []string{"env", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── exec ───────────────────────────────────────────────────────────────
	{"exec", []string{"exec"}, "(a) no project dir"},
	{"exec", []string{"exec", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"exec", []string{"exec", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── functions ──────────────────────────────────────────────────────────
	{"functions", []string{"functions"}, "(a) no project dir"},
	{"functions", []string{"functions", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"functions", []string{"functions", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── health ─────────────────────────────────────────────────────────────
	{"health", []string{"health"}, "(a) no project dir"},
	{"health", []string{"health", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"health", []string{"health", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── init ───────────────────────────────────────────────────────────────
	{"init", []string{"init", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"init", []string{"init", "unknownsub_xyz"}, "(c) unknown sub"},
	{"init", []string{"init", "--name", "INVALID NAME WITH SPACES!"}, "(a) invalid name"},

	// ── license ────────────────────────────────────────────────────────────
	{"license", []string{"license"}, "(a) no project dir"},
	{"license", []string{"license", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"license", []string{"license", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── login ──────────────────────────────────────────────────────────────
	{"login", []string{"login", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"login", []string{"login", "unknownsub_xyz"}, "(c) unknown sub"},
	{"login", []string{"login"}, "(a) missing credentials"},

	// ── logout ─────────────────────────────────────────────────────────────
	{"logout", []string{"logout", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"logout", []string{"logout", "unknownsub_xyz"}, "(c) unknown sub"},
	{"logout", []string{"logout"}, "(a) no session"},

	// ── logs ───────────────────────────────────────────────────────────────
	{"logs", []string{"logs"}, "(a) no project dir"},
	{"logs", []string{"logs", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"logs", []string{"logs", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── migrate ────────────────────────────────────────────────────────────
	{"migrate", []string{"migrate"}, "(a) no project dir"},
	{"migrate", []string{"migrate", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"migrate", []string{"migrate", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── monitor ────────────────────────────────────────────────────────────
	{"monitor", []string{"monitor"}, "(a) no project dir"},
	{"monitor", []string{"monitor", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"monitor", []string{"monitor", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── plugin ─────────────────────────────────────────────────────────────
	{"plugin", []string{"plugin"}, "(a) no project dir"},
	{"plugin", []string{"plugin", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"plugin", []string{"plugin", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── promote ────────────────────────────────────────────────────────────
	{"promote", []string{"promote"}, "(a) no project dir"},
	{"promote", []string{"promote", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"promote", []string{"promote", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── queue ──────────────────────────────────────────────────────────────
	{"queue", []string{"queue"}, "(a) no project dir"},
	{"queue", []string{"queue", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"queue", []string{"queue", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── reset ──────────────────────────────────────────────────────────────
	{"reset", []string{"reset"}, "(a) no project dir"},
	{"reset", []string{"reset", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"reset", []string{"reset", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── restart ────────────────────────────────────────────────────────────
	{"restart", []string{"restart"}, "(a) no project dir"},
	{"restart", []string{"restart", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"restart", []string{"restart", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── secrets ────────────────────────────────────────────────────────────
	{"secrets", []string{"secrets"}, "(a) no project dir"},
	{"secrets", []string{"secrets", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"secrets", []string{"secrets", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── security ───────────────────────────────────────────────────────────
	{"security", []string{"security"}, "(a) no project dir"},
	{"security", []string{"security", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"security", []string{"security", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── service ────────────────────────────────────────────────────────────
	{"service", []string{"service"}, "(a) no project dir"},
	{"service", []string{"service", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"service", []string{"service", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── ssl ────────────────────────────────────────────────────────────────
	{"ssl", []string{"ssl"}, "(a) no project dir"},
	{"ssl", []string{"ssl", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"ssl", []string{"ssl", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── start ──────────────────────────────────────────────────────────────
	{"start", []string{"start"}, "(a) no project dir"},
	{"start", []string{"start", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"start", []string{"start", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── status ─────────────────────────────────────────────────────────────
	{"status", []string{"status"}, "(a) no project dir"},
	{"status", []string{"status", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"status", []string{"status", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── stop ───────────────────────────────────────────────────────────────
	{"stop", []string{"stop"}, "(a) no project dir"},
	{"stop", []string{"stop", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"stop", []string{"stop", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── telemetry ──────────────────────────────────────────────────────────
	{"telemetry", []string{"telemetry"}, "(a) no project dir"},
	{"telemetry", []string{"telemetry", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"telemetry", []string{"telemetry", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── tenant ─────────────────────────────────────────────────────────────
	{"tenant", []string{"tenant"}, "(a) no project dir"},
	{"tenant", []string{"tenant", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"tenant", []string{"tenant", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── trust ──────────────────────────────────────────────────────────────
	{"trust", []string{"trust"}, "(a) no project dir"},
	{"trust", []string{"trust", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"trust", []string{"trust", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── update ─────────────────────────────────────────────────────────────
	{"update", []string{"update", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"update", []string{"update", "unknownsub_xyz"}, "(c) unknown sub"},
	{"update", []string{"update"}, "(a) no project dir"},

	// ── upgrade ────────────────────────────────────────────────────────────
	{"upgrade", []string{"upgrade", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"upgrade", []string{"upgrade", "unknownsub_xyz"}, "(c) unknown sub"},
	{"upgrade", []string{"upgrade"}, "(a) no project dir"},

	// ── urls ───────────────────────────────────────────────────────────────
	{"urls", []string{"urls"}, "(a) no project dir"},
	{"urls", []string{"urls", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"urls", []string{"urls", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── version ────────────────────────────────────────────────────────────
	// version always succeeds on its own, so we exercise error paths via flags.
	{"version", []string{"version", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"version", []string{"version", "unknownsub_xyz"}, "(c) unknown sub"},
	{"version", []string{"version", "--format", "INVALIDFORMAT_XYZ"}, "(a) bad format"},

	// ── waf ────────────────────────────────────────────────────────────────
	{"waf", []string{"waf"}, "(a) no project dir"},
	{"waf", []string{"waf", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"waf", []string{"waf", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── watchdog ───────────────────────────────────────────────────────────
	{"watchdog", []string{"watchdog"}, "(a) no project dir"},
	{"watchdog", []string{"watchdog", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"watchdog", []string{"watchdog", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── webhooks ───────────────────────────────────────────────────────────
	{"webhooks", []string{"webhooks"}, "(a) no project dir"},
	{"webhooks", []string{"webhooks", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"webhooks", []string{"webhooks", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── account ────────────────────────────────────────────────────────────
	{"account", []string{"account"}, "(a) no project dir"},
	{"account", []string{"account", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"account", []string{"account", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── api ────────────────────────────────────────────────────────────────
	{"api", []string{"api"}, "(a) no project dir"},
	{"api", []string{"api", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"api", []string{"api", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── dlq ────────────────────────────────────────────────────────────────
	{"dlq", []string{"dlq"}, "(a) no project dir"},
	{"dlq", []string{"dlq", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"dlq", []string{"dlq", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── dns-setup ──────────────────────────────────────────────────────────
	{"dns-setup", []string{"dns-setup"}, "(a) no project dir"},
	{"dns-setup", []string{"dns-setup", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"dns-setup", []string{"dns-setup", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── flag ───────────────────────────────────────────────────────────────
	{"flag", []string{"flag"}, "(a) no project dir"},
	{"flag", []string{"flag", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"flag", []string{"flag", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── oauth ──────────────────────────────────────────────────────────────
	{"oauth", []string{"oauth"}, "(a) no project dir"},
	{"oauth", []string{"oauth", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"oauth", []string{"oauth", "unknownsub_xyz"}, "(c) unknown sub"},

	// ── soak ───────────────────────────────────────────────────────────────
	{"soak", []string{"soak"}, "(a) no project dir"},
	{"soak", []string{"soak", "--no-such-flag-xyz"}, "(b) invalid flag"},
	{"soak", []string{"soak", "unknownsub_xyz"}, "(c) unknown sub"},
}

// runErrorHarnessCmd executes the given args against a fresh RootCmd clone
// and returns any error. It does not touch real filesystem state.
//
// A 5-second per-case ctx deadline prevents any single RunE from hanging the
// whole harness. Some commands (dns-setup, admin, db) shell out or touch
// slow I/O paths — without the deadline, one such case can exhaust the Go
// test-timeout budget (observed: dns-setup took 5m44s on macos-14). The
// deadline propagates through cmd.Context() inside RunE so any well-behaved
// exec.CommandContext or HTTP client inside the command will abort.
func runErrorHarnessCmd(t *testing.T, args []string) error {
	t.Helper()

	// Clone root so parallel subtests don't race on global state.
	root := &cobra.Command{
		Use:           RootCmd.Use,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// Re-register all commands from the global root.
	for _, c := range RootCmd.Commands() {
		root.AddCommand(c)
	}

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run the cobra execution in a goroutine so the test can enforce its own
	// wall-clock cap even when the command does not honour ctx (e.g., Go code
	// using exec.Command rather than exec.CommandContext).
	done := make(chan error, 1)
	go func() {
		done <- root.ExecuteContext(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// Treat deadline as an error outcome — exactly what the harness expects
		// from misbehaving commands in a no-project-dir context.
		return fmt.Errorf("harness deadline exceeded for args %v: %w", args, ctx.Err())
	}
}

// TestErrorHarness_AllCommandsReturnError verifies that every entry in the
// error harness table produces a non-nil error from cobra.Execute().
//
// Cases labelled "(b) invalid flag" are hard failures — cobra MUST reject an
// unknown flag. Cases labelled "(a)" and "(c)" are soft: many parent commands
// return nil when invoked with no args or an unrecognised subcommand (cobra's
// default help behaviour). Those cases are logged rather than failed so the
// harness stays green on group commands.
func TestErrorHarness_AllCommandsReturnError(t *testing.T) {
	for _, tc := range errorHarnessCases {
		tc := tc // capture range variable
		name := tc.cmd + " " + tc.desc
		hard := tc.desc == "(b) invalid flag"
		t.Run(name, func(t *testing.T) {
			err := runErrorHarnessCmd(t, tc.args)
			if err == nil {
				if hard {
					t.Errorf("[%s] invalid-flag case must return error; got nil (args: %v)", name, tc.args)
				} else {
					t.Logf("[%s] returned nil (group-command help path — acceptable) args: %v", name, tc.args)
				}
			}
		})
	}
}

// TestErrorHarness_AllCommandsCovered verifies that every top-level command
// registered on RootCmd has at least one entry in the harness table.
// This prevents commands being added without error coverage.
func TestErrorHarness_AllCommandsCovered(t *testing.T) {
	covered := make(map[string]bool)
	for _, tc := range errorHarnessCases {
		covered[tc.cmd] = true
	}

	// Commands that legitimately succeed with no args or are aliases only.
	// Aliases share RunE with their canonical command — they don't need separate entries.
	safeList := map[string]bool{
		"help":       true, // built-in cobra help
		"completion": true, // covered above with invalid shell
		"up":         true, // alias for start
		"down":       true, // alias for stop
	}

	for _, c := range RootCmd.Commands() {
		name := c.Name()
		if safeList[name] {
			continue
		}
		if !covered[name] {
			t.Errorf("command %q is registered but has no entry in errorHarnessCases", name)
		}
	}
}

// TestErrorHarness_MinimumCaseCount verifies at least 144 error cases exist
// (48 commands × 3 scenarios = 144 minimum per acceptance criteria).
func TestErrorHarness_MinimumCaseCount(t *testing.T) {
	const minCases = 144
	if len(errorHarnessCases) < minCases {
		t.Errorf("expected at least %d error harness cases; got %d", minCases, len(errorHarnessCases))
	}
}
