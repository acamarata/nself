package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/soak"

	"github.com/spf13/cobra"
)

var soakCmd = &cobra.Command{
	Use:   "soak",
	Short: "Manage soak testing lifecycle",
	Long: `Manage soak testing lifecycle for nSelf environments.

Subcommands:
  abort    Abort an active soak and optionally roll back to a prior version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var soakAbortCmd = &cobra.Command{
	Use:   "abort --rollback <version>",
	Short: "Abort a running soak and roll back to a prior version",
	Long: `Abort an active soak on a target environment and roll back to a specified version.

The command:
  1. Validates environment gate (production requires --prod-i-mean-it)
  2. Detects idempotent state (already on target version is a no-op)
  3. Prompts for confirmation (skip with --yes)
  4. Tags current state for post-mortem
  5. Executes deploy rollback to the target version
  6. Posts a notify event with the rollback reason

Per the RISK doctrine, this command never executes without explicit user
confirmation unless --yes is passed.

Examples:
  nself soak abort --rollback v1.0.8
  nself soak abort --rollback v1.0.8 --dry-run
  nself soak abort --rollback v1.0.8 --yes
  nself soak abort --rollback v1.0.8 --env staging
  nself soak abort --rollback v1.0.8 --env prod --prod-i-mean-it --yes`,
	RunE: runSoakAbort,
}

func init() {
	f := soakAbortCmd.Flags()
	f.String("rollback", "", "Version to roll back to (required)")
	f.Bool("dry-run", false, "Print exact rollback steps without executing")
	f.Bool("yes", false, "Skip interactive confirmation prompt")
	f.Bool("prod-i-mean-it", false, "Required to run soak abort against a production environment")
	f.String("env", "staging", "Target environment: local|staging|prod")

	_ = soakAbortCmd.MarkFlagRequired("rollback")

	soakCmd.AddCommand(soakAbortCmd)
	RootCmd.AddCommand(soakCmd)
}

func runSoakAbort(cmd *cobra.Command, _ []string) error {
	version, _ := cmd.Flags().GetString("rollback")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	prodFlag, _ := cmd.Flags().GetBool("prod-i-mean-it")
	env, _ := cmd.Flags().GetString("env")

	opts := soak.AbortOptions{
		Version:     version,
		DryRun:      dryRun,
		SkipConfirm: yes,
		AllowProd:   prodFlag,
		Env:         env,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Stdin:       os.Stdin,
	}

	err := soak.Abort(cmd.Context(), opts)
	if errors.Is(err, soak.ErrAbortCanceled) {
		fmt.Fprintln(os.Stderr, "Soak abort canceled.")
		return nil
	}
	if errors.Is(err, soak.ErrAlreadyRolledBack) {
		// Idempotent no-op — not an error to the shell.
		return nil
	}
	return err
}
