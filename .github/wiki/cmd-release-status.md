# nself release-status

<!-- BEGIN PROSE:summary -->
> View the live deployment status of all release distribution surfaces.
<!-- END PROSE:summary -->

## Synopsis

```
nself release-status [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself release-status` polls six distribution surfaces and reports whether each
is serving the expected version: the CLI binary (GitHub Releases), the Admin Docker
image (Docker Hub), the Homebrew tap formula, the ping_api canary, the web/org
marketing site, and the overall Vercel deployment.

Each surface is reported as `fresh`, `stale`, or `unknown`. A `stale` result means
the surface has not yet propagated the latest release. An `unknown` result means
the surface did not respond within the timeout window.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output as JSON |
| `--timeout` | `30s` | HTTP timeout per artifact check |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Check release status across all surfaces
nself release-status

# Check with a longer timeout for slow networks
nself release-status --timeout 60s

# Emit JSON for scripting or CI dashboards
nself release-status --json
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-release]], run the full release cascade
- [[cmd-release-check]], pre-flight validation before releasing
- [[cmd-release-rollback]], revert distribution surfaces to a prior version
- [[cmd-status]], check the live stack status on this machine
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
