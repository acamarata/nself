package commands

// Purpose: nself ci serve — self-hosted CI webhook listener daemon.
//   Receives GitHub push/pull_request webhooks, verifies HMAC-SHA256 signatures,
//   and runs the nself-ci gate in an ephemeral Docker container per job.
//   Replaces GitHub Actions compute while keeping GitHub as the SCM and status
//   target. Webhooks are free on GitHub; only Actions minutes are billed.
// Inputs:  flags (--addr, --secret, --concurrency, --workdir, --timeout)
// Outputs: HTTP server on :3845; /healthz; /; GitHub commit status posted per job
// Constraints: Requires gh CLI (OAuth), Docker daemon, nself-ci binary on PATH
//   or adjacent plugin source. SPORT: CLI-CMD-CI-SERVE-001
// SPORT: CLI-CMD-CI-SERVE-001

import (
	nscicmd "github.com/nself-org/cli/internal/cmd/ci"
	"github.com/spf13/cobra"
)

var ciServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the nself-ci webhook listener daemon (port 3845)",
	Long: `Start an HTTP server that receives GitHub push and pull_request webhooks,
verifies the HMAC-SHA256 signature, and runs the nself-ci gate suite for each
event in an ephemeral Docker container.

GitHub webhooks are free — only Actions compute minutes are billed. This daemon
moves the compute to your own ops box while keeping GitHub as the SCM and
commit-status target.

The server posts a "nself-ci" GitHub commit status (pending → success/failure)
via gh OAuth, identical to the nself ci one-shot command.

Environment variables:
  GITHUB_WEBHOOK_SECRET  HMAC secret (same value configured in GitHub webhook settings)
  NSELF_CI_EVENT_SINK    Optional URL to POST completion events (JSON) to — seam for
                         the nself-cron-monitor plugin on port 3839.

Examples:
  nself ci serve
  nself ci serve --addr :3845 --concurrency 4
  nself ci serve --secret "$GITHUB_WEBHOOK_SECRET" --workdir /var/ci/workdirs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		secret, _ := cmd.Flags().GetString("secret")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		workdir, _ := cmd.Flags().GetString("workdir")
		jobTimeout, _ := cmd.Flags().GetInt("timeout")
		verbose, _ := cmd.Flags().GetBool("verbose")

		return nscicmd.RunServe(nscicmd.ServeConfig{
			Addr:        addr,
			Secret:      secret,
			Concurrency: concurrency,
			WorkDir:     workdir,
			JobTimeout:  jobTimeout,
			Verbose:     verbose,
		})
	},
}

func init() {
	ciServeCmd.Flags().String("addr", ":3845", "Listen address (host:port). Port 3845 = nself-ci-serve per F10 port registry")
	ciServeCmd.Flags().String("secret", "", "HMAC-SHA256 webhook secret (overrides GITHUB_WEBHOOK_SECRET env)")
	ciServeCmd.Flags().Int("concurrency", 2, "Max concurrent CI jobs")
	ciServeCmd.Flags().String("workdir", "/tmp/nself-ci-workdirs", "Base directory for ephemeral checkout workdirs")
	ciServeCmd.Flags().Int("timeout", 600, "Per-job timeout in seconds")
	ciServeCmd.Flags().BoolP("verbose", "v", false, "Verbose gate output")

	ciCmd.AddCommand(ciServeCmd)
}
