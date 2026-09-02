// Package database — drill_critical_tables.go: the drill's critical-table
// list and its per-project resolution. Split out of drill.go (file-size
// ratchet, internal/repoqa) — this is the self-contained piece of the drill
// smoke gate that needed its own home once it grew a resolution function and
// tests, per the same drill_*.go concern-split convention already used for
// restore_drill.go.
package database

import (
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// DefaultCriticalTables is the canonical np_*-prefixed critical-table list
// used when a project has not set BACKUP_CRITICAL_TABLES. The five tables
// here are load-bearing for the most common recovery scenario on an
// np_-prefixed (multi-app-isolation, per the Multi-Tenant Convention Wall)
// schema: license validation + signup pipeline + audit log must survive a
// restore.
//
// Decision (2026-09-01, closes the drill-critical-tables-naming bug note,
// option (c)): the list is project-configurable rather than a single hardcoded
// convention, because real deployed schemas disagree on naming — this
// project's own staging nself_web_db uses unprefixed names (users, licenses,
// audit_logs, plugins) with no np_ prefix and no billing-equivalent table at
// all. Hardcoding either convention would make the drill permanently
// wrong-shaped for whichever projects don't use it. ResolveCriticalTables
// reads the per-project override; this var stays the zero-config fallback so
// existing np_-prefixed deployments (and every test in this package) see no
// behavior change.
var DefaultCriticalTables = []string{
	"np_users",
	"np_licenses",
	"np_audit_log",
	"np_plugins",
	"np_billing",
}

// ResolveCriticalTables returns the critical-table list to smoke-check for
// this project: cfg.Backup.CriticalTables (BACKUP_CRITICAL_TABLES,
// comma-separated, whitespace-trimmed, empty entries dropped) when set, else
// DefaultCriticalTables. A nil cfg (unit tests exercising smokeCheck/
// verifyCriticalTables directly) also falls back to the default.
func ResolveCriticalTables(cfg *config.Config) []string {
	if cfg == nil || strings.TrimSpace(cfg.Backup.CriticalTables) == "" {
		return DefaultCriticalTables
	}
	var out []string
	for _, name := range strings.Split(cfg.Backup.CriticalTables, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return DefaultCriticalTables
	}
	return out
}
