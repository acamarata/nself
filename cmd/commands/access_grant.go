package commands

// Purpose: `nself access grant` handler — validates flags, loads the public
// key, and delegates to access.Grant, then reports the fingerprint and any
// backup path back to the operator.
// Inputs: --host, --identity, --user, --key, --sudo, --docker, --expires,
// --dry-run.
// Outputs: printed confirmation (or dry-run diff) plus the granted key's
// fingerprint; a non-nil error on any validation or transport failure.

import (
	"fmt"
	"os"

	"github.com/nself-org/cli/internal/access"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

func runAccessGrant(cmd *cobra.Command, args []string) error {
	user, _ := cmd.Flags().GetString("user")
	if user == "" {
		return fmt.Errorf("--user is required")
	}
	keyArg, _ := cmd.Flags().GetString("key")
	if keyArg == "" {
		return fmt.Errorf("--key is required, e.g. --key @teammate.pub")
	}
	sudo, _ := cmd.Flags().GetBool("sudo")
	docker, _ := cmd.Flags().GetBool("docker")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	expiresRaw, _ := cmd.Flags().GetString("expires")

	expires, err := parseExpiry(expiresRaw)
	if err != nil {
		return err
	}

	key, err := access.LoadPublicKeyArg(keyArg)
	if err != nil {
		return fmt.Errorf("invalid --key: %w", err)
	}

	t, err := newAccessTransport(cmd)
	if err != nil {
		return err
	}

	ui.CommandHeader("nself access grant", t.Describe())

	result, err := access.Grant(cmd.Context(), t, access.GrantRequest{
		User: user, Key: key, Sudo: sudo, Docker: docker, Expires: expires, DryRun: dryRun,
	})
	if err != nil {
		return fmt.Errorf("grant access for %s on %s: %w", user, t.Describe(), err)
	}

	if dryRun {
		ui.Info("Dry run: no changes made. Resulting authorized_keys diff:")
		fmt.Print(result.Diff)
		return nil
	}

	warnHetznerMismatch(cmd, t)

	if result.AlreadyGranted {
		ui.Success(fmt.Sprintf("%s already has this exact key granted (%s)", user, result.Fingerprint))
		return nil
	}

	if result.BackupPath != "" {
		ui.Info("Backed up authorized_keys to " + result.BackupPath)
	}
	ui.Success(fmt.Sprintf("Granted %s access to %s", user, t.Describe()))
	ui.Info("Fingerprint: " + result.Fingerprint)
	if sudo {
		ui.Warn("--sudo is recorded as metadata only; this command does not add " + user + " to the sudo group")
	}
	if docker {
		ui.Warn("--docker is recorded as metadata only; this command does not add " + user + " to the docker group")
	}
	if expires != nil {
		ui.Info("Expires: " + expires.Format("2006-01-02"))
	}
	return nil
}

// warnHetznerMismatch runs the best-effort Hetzner-project-vs-running-server
// key check (issue #238) after a grant and prints one ui.Warn per mismatch.
// It is silent (not a command failure) when HETZNER_NSELF_TOKEN is unset or
// the API call fails — this is an advisory check, never a gate on grant.
func warnHetznerMismatch(cmd *cobra.Command, t access.Transport) {
	token := os.Getenv("HETZNER_NSELF_TOKEN")
	if token == "" {
		return
	}
	warnings, err := access.HetznerMismatchWarnings(cmd.Context(), token, t)
	if err != nil {
		ui.Info("Hetzner project-key mismatch check skipped: " + err.Error())
		return
	}
	for _, w := range warnings {
		ui.Warn(w)
	}
}
