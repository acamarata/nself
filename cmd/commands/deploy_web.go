// Package commands implements CLI commands for the nSelf binary.
// This file adds 'nself deploy web' which builds Vercel/web apps locally and
// deploys the prebuilt output — eliminating Vercel Build CPU charges (~$43/mo).
//
// Why prebuilt: 'vercel deploy --prebuilt' skips remote build compute entirely;
// Vercel just serves the already-built .vercel/output directory. Local or
// ops-box builds are free; Vercel Build CPU is not.
//
// Usage:
//
//	nself deploy web [app…]          # build+deploy all apps (or listed subset)
//	nself deploy web --prod          # promote to production
//	nself deploy web nchat org       # subset by name
//	nself deploy web --dry-run       # print commands without running
//	nself deploy web --token=<tok>   # explicit Vercel token (else VERCEL_TOKEN)
package commands

import (
	"github.com/spf13/cobra"
)

// webApps is the canonical list of deployable web apps in the nSelf Turborepo.
// Source of truth: web/pnpm-workspace.yaml (apps/* + named roots).
// Update this list when apps are added or removed from the monorepo.
var webApps = []string{
	"org",
	"docs",
	"nchat",
	"nclaw",
	"ntv",
	"clawde",
	"nfamily",
	"ntask",
	"ntask-marketing",
	"cloud",
	"base",
	"install",
	"status",
	"nsentry",
}

// deployWebCmd is the 'nself deploy web' subcommand.
var deployWebCmd = &cobra.Command{
	Use:   "web [app...]",
	Short: "Build web apps locally and deploy prebuilt output to Vercel",
	Long: `Build one or more web apps locally (vercel build) then deploy the
prebuilt output (vercel deploy --prebuilt), eliminating Vercel Build CPU charges.

Running 'vercel build' locally produces a .vercel/output directory.
'vercel deploy --prebuilt' uploads that directory directly — Vercel does
no build compute, so you pay only for hosting, not CPU minutes.

Apps are deployed sequentially. Pass app names to target a subset.

Examples:
  nself deploy web                # build+deploy all 14 apps
  nself deploy web org docs       # subset: org + docs only
  nself deploy web --prod         # deploy to production
  nself deploy web --dry-run      # print commands, do nothing
  nself deploy web nchat --prod   # prod deploy for nchat only

Environment:
  VERCEL_TOKEN    Vercel API token (required; also accepted via --token)
  VERCEL_ORG_ID   Vercel org/team ID (optional; read from .vercel/project.json)
  VERCEL_PROJECT_ID  Vercel project ID (optional; read from .vercel/project.json)`,
	Args: cobra.ArbitraryArgs,
	RunE: runDeployWeb,
}

func init() {
	f := deployWebCmd.Flags()
	f.Bool("prod", false, "Promote deploy to production (adds --prod to 'vercel deploy')")
	f.Bool("dry-run", false, "Print the vercel commands without executing")
	f.String("token", "", "Vercel API token (overrides VERCEL_TOKEN env var)")
	f.String("web-dir", "", "Path to the web Turborepo root (default: auto-detected sibling '../web')")

	deployCmd.AddCommand(deployWebCmd)
}
