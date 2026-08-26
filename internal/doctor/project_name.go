package doctor

// project_name.go — resolves the nSelf PROJECT_NAME for the --deep and
// hardening checks that shell out to a project's Postgres/nginx containers.
//
// Purpose: give every check in this package one place to turn a project
// directory into the container-name prefix nSelf compose actually uses,
// instead of each check hardcoding a literal container name.
// Inputs: projectDir string — the nSelf working directory (same value
// DeepChecks/HardeningChecks already receive).
// Outputs: the project's PROJECT_NAME, or legacyProjectName when it cannot
// be determined.
// Constraints: must never error out of a doctor check — config.Load failures
// degrade to the legacy default instead of surfacing as a check failure.
// SPORT: cli/internal/doctor — container name derivation (unblocks part of
// nself-org/web#127).

import "github.com/nself-org/cli/internal/config"

// legacyProjectName is the literal name several --deep/hardening checks
// hardcoded before they resolved PROJECT_NAME from config. It stays as the
// fallback when config.Load cannot determine a real project name (e.g.
// doctor invoked outside an nSelf project directory, or an unreadable/oversized
// env file), so that setups already relying on it — including this repo's own
// dogfood stack — keep working exactly as before.
const legacyProjectName = "nself"

// resolveProjectName returns the configured PROJECT_NAME for projectDir. It
// loads the same .env cascade the rest of the CLI uses (internal/config.Load),
// so a project with a custom PROJECT_NAME (e.g. nself-org/web's "nself-web")
// resolves to its real container prefix instead of the old hardcoded "nself".
//
// Falls back to legacyProjectName when config.Load errors or somehow yields
// an empty project name — config.ApplyDefaults normally fills an unset
// PROJECT_NAME with "myproject", so an empty result here only happens on a
// genuine load failure.
func resolveProjectName(projectDir string) string {
	cfg, err := config.Load(projectDir)
	if err != nil || cfg.ProjectName == "" {
		return legacyProjectName
	}
	return cfg.ProjectName
}
