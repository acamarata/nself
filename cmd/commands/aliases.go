package commands

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// normalizeInvokedBinary rewrites os.Args when the binary is invoked through
// a product-alias name, so a symlink gives a dedicated CLI for free:
//
//	ln -s $(which nself) /usr/local/bin/nsentry
//	nsentry monitors list   ≡   nself sentry monitors list
//
// Called from Execute() before any arg parsing (never from init(): init must
// only do cobra registration). Idempotent: skips when the subcommand is
// already present.
func normalizeInvokedBinary() {
	if len(os.Args) == 0 {
		return
	}
	base := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if base != "nsentry" {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "sentry" {
		return
	}
	rewritten := make([]string, 0, len(os.Args)+1)
	rewritten = append(rewritten, os.Args[0], "sentry")
	rewritten = append(rewritten, os.Args[1:]...)
	os.Args = rewritten
}

func init() {
	// TRAP-09: guard against duplicate registration if up/down are ever
	// added as first-class commands in the future.
	for _, c := range RootCmd.Commands() {
		if c.Name() == "up" || c.Name() == "down" {
			return
		}
	}

	upCmd := &cobra.Command{
		Use:    "up",
		Short:  "Alias for: nself start",
		Hidden: true,
		RunE:   startCmd.RunE,
	}

	downCmd := &cobra.Command{
		Use:    "down",
		Short:  "Alias for: nself stop",
		Hidden: true,
		RunE:   stopCmd.RunE,
	}

	RootCmd.AddCommand(upCmd, downCmd)
}
