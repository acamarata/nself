// Package nginxtopo resolves on-disk nginx directory topology for projects
// fronted by another stack's nginx (NGINX_FRONTED_BY / config.NginxConfig.
// FrontedBy). It has no dependency on internal/config, internal/build, or
// internal/doctor so any of them can import it without a cycle.
package nginxtopo

import "path/filepath"

// Purpose: FrontedBy names only a stack (e.g. "nself-web"), never a path —
// nothing in the CLI's config model turns a stack name into a filesystem
// location. ResolveFrontingDir is the one topology the CLI trusts to bridge
// that gap, first established for the SEC-HARDENING-06 doctor check
// (internal/doctor, cli#380/cli#371 staging regression) and reused as-is by
// the nginx site-conf generator (internal/build, cli#385) rather than
// inventing a second convention.
// Inputs: projectDir (a project's resolved nSelf root) and frontedBy
// (config.NginxConfig.FrontedBy, expected non-empty by callers).
// Outputs: the fronting stack's own project directory and ok=true when the
// convention is confirmed; "", false otherwise.
// Constraints: never guesses beyond the one confirmed convention — callers
// must refuse rather than fall back to a wrong directory when ok is false.

// ResolveFrontingDir resolves the directory of the stack named by frontedBy
// when projectDir is laid out as a subdirectory (conventionally "backend")
// directly under that stack's own directory.
//
// Example: projectDir ".../nself-web/backend" with frontedBy "nself-web"
// resolves to ".../nself-web" — the fronting stack's own root. Callers
// append their own "nginx/..." subpath (e.g. "nginx/sites") to that root.
func ResolveFrontingDir(projectDir, frontedBy string) (dir string, ok bool) {
	parent := filepath.Dir(projectDir)
	if parent == projectDir || filepath.Base(parent) != frontedBy {
		return "", false
	}
	return parent, true
}
