package commands

// Purpose: shared flag handling for the `nself access` subcommands — turning
// --host/--identity into an access.Transport, and --expires into a
// *time.Time. Split out of access.go so the cobra wiring stays a plain
// command tree.
// Inputs: a *cobra.Command carrying --host/--identity flags.
// Outputs: an access.Transport (always SSHTransport in the real CLI —
// LocalFileTransport is exercised only by tests in internal/access and
// access_test.go) or a parse error.
// Constraints: --host has no default; a hardcoded IP here would be exactly
// the kind of accidental-production-target footgun this command exists to
// prevent, so the operator must always name the host explicitly.

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/access"

	"github.com/spf13/cobra"
)

// newAccessTransport is a package-level indirection so access_test.go can
// substitute a fixture Transport (LocalFileTransport over a temp dir)
// without any of the grant/revoke/list handlers knowing the difference. The
// real CLI never reassigns it — every live invocation resolves through
// buildAccessTransport to SSHTransport.
var newAccessTransport = buildAccessTransport

// buildAccessTransport resolves --host/--identity into a Transport. It
// never invokes ssh itself; SSHTransport only runs commands when Grant,
// Revoke, or List actually call Read/Backup/Write.
func buildAccessTransport(cmd *cobra.Command) (access.Transport, error) {
	host, _ := cmd.Flags().GetString("host")
	if host == "" {
		return nil, fmt.Errorf("--host is required, e.g. --host root@203.0.113.5")
	}

	identity, _ := cmd.Flags().GetString("identity")
	if identity == "" {
		identity = defaultIdentityPath()
	}

	return &access.SSHTransport{Host: host, IdentityPath: identity}, nil
}

// defaultIdentityPath mirrors the fallback used by 'nself deploy' for its own
// SSH key: NSELF_DEPLOY_KEY_PATH, then ~/.ssh/id_ed25519.
func defaultIdentityPath() string {
	if k := os.Getenv("NSELF_DEPLOY_KEY_PATH"); k != "" {
		return k
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "id_ed25519"
	}
	return filepath.Join(home, ".ssh", "id_ed25519")
}

// parseExpiry parses --expires (YYYY-MM-DD), returning nil for an empty flag.
func parseExpiry(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, fmt.Errorf("--expires must be YYYY-MM-DD, got %q: %w", raw, err)
	}
	return &t, nil
}
