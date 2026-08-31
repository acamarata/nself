package commands

// Purpose: Interactive confirmation gate for `nself plugin install <url>`
//          (CLI-R16) — the command-layer half of the "official by name,
//          third-party by URL" contract. The download/extract/verify/install
//          mechanics live in internal/plugin/install_thirdparty.go; this file
//          owns only the user-facing warning + confirmation, so that package
//          doesn't need a terminal dependency.
// Inputs:  the cobra.Command (for stdin/stdout, testable via SetIn/SetOut),
//          the requested source URL, and the --yes flag value.
// Outputs: nil to proceed, or an error describing why installation was
//          refused (invalid URL, or the user/CI declined).
// Constraints: Must run BEFORE any network request for that URL — nothing is
//              downloaded or executed until this returns nil.

import (
	"fmt"

	"github.com/nself-org/cli/internal/plugin"

	"github.com/spf13/cobra"
)

// confirmThirdPartyInstall validates sourceURL and, unless yes is true, asks
// the user to confirm installing from it. It always prints the warning
// (naming the host) so the source is visible even in the --yes/non-interactive
// path, which CI logs capture.
func confirmThirdPartyInstall(cmd *cobra.Command, sourceURL string, yes bool) error {
	u, err := plugin.ValidateThirdPartyURL(sourceURL)
	if err != nil {
		return err
	}

	out := cmd.ErrOrStderr()
	_, _ = fmt.Fprintf(out, "\nwarning: %s is a THIRD-PARTY plugin source (host: %s).\n", sourceURL, u.Host)
	_, _ = fmt.Fprintf(out, "  It is not part of the official nself plugin registry: its author is not\n")
	_, _ = fmt.Fprintf(out, "  vetted and its tarball is NOT signature-verified. Checksum is verified\n")
	_, _ = fmt.Fprintf(out, "  only if the plugin's own manifest declares one.\n")

	if yes {
		_, _ = fmt.Fprintf(out, "  Proceeding without confirmation (--yes).\n")
		return nil
	}

	if !confirmPrompt(cmd, fmt.Sprintf("Install from %s anyway? [y/N] ", u.Host)) {
		return fmt.Errorf("installation from %s declined", u.Host)
	}
	return nil
}
