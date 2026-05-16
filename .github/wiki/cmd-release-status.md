# nself release-status

> View the live deployment status of all release distribution surfaces.

## Synopsis

```bash
nself release-status [flags]
```

## Description

`nself release-status` polls six distribution surfaces and reports whether each
is serving the expected version: the CLI binary (GitHub Releases), the Admin Docker
image (Docker Hub), the Homebrew tap formula, the ping_api canary, the web/org
marketing site, and the overall Vercel deployment.

Each surface is reported as `fresh`, `stale`, or `unknown`. A `stale` result means
the surface has not yet propagated the latest release. An `unknown` result means
the surface did not respond within the timeout window.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | `–` | bool | `false` | Emit structured JSON output instead of human-readable table |
| `--timeout` | `–` | duration | `30s` | Maximum time to wait for each surface to respond |

## Examples

```bash
# Check release status across all surfaces
nself release-status

# Check with a longer timeout for slow networks
nself release-status --timeout 60s

# Emit JSON for scripting or CI dashboards
nself release-status --json
```

## See Also

- [[cmd-release]], run the full release cascade
- [[cmd-release-check]], pre-flight validation before releasing
- [[cmd-release-rollback]], revert distribution surfaces to a prior version
- [[cmd-status]], check the live stack status on this machine

← [[Commands]] | [[Home]] →
