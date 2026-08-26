package commands

// Purpose: `nself access revoke` handler — delegates to access.Revoke and
// surfaces the ErrLastKey lockout guard as an actionable error rather than a
// raw wrapped message.
// Inputs: --host, --identity, --user, --force, --dry-run.
// Outputs: printed confirmation (or dry-run diff) plus the revoked key's
// fingerprint; a non-nil error when the user is unknown or the lockout guard
// trips.

import (
	"errors"
	"fmt"

	"github.com/nself-org/cli/internal/access"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

func runAccessRevoke(cmd *cobra.Command, args []string) error {
	user, _ := cmd.Flags().GetString("user")
	if user == "" {
		return fmt.Errorf("--user is required")
	}
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	t, err := newAccessTransport(cmd)
	if err != nil {
		return err
	}

	ui.CommandHeader("nself access revoke", t.Describe())

	result, err := access.Revoke(cmd.Context(), t, access.RevokeRequest{
		User: user, Force: force, DryRun: dryRun,
	})
	if err != nil {
		if errors.Is(err, access.ErrLastKey) {
			ui.Error(err.Error())
			return fmt.Errorf("revoke access for %s on %s: last key on host", user, t.Describe())
		}
		return fmt.Errorf("revoke access for %s on %s: %w", user, t.Describe(), err)
	}

	if dryRun {
		ui.Info("Dry run: no changes made. Resulting authorized_keys diff:")
		fmt.Print(result.Diff)
		return nil
	}

	if result.BackupPath != "" {
		ui.Info("Backed up authorized_keys to " + result.BackupPath)
	}
	ui.Success(fmt.Sprintf("Revoked %s's access to %s", user, t.Describe()))
	ui.Info("Fingerprint: " + result.Fingerprint)
	return nil
}
