# nself release

> Orchestrate a full versioned release across all nSelf distribution surfaces.

## Synopsis

```bash
nself release <version> [flags]
```

## Description

`nself release` runs the 12-step release cascade that publishes a new nSelf version
to every distribution surface in sequence. Each step is verified before the next
begins. Pass `--dry-run` to rehearse the full cascade without making any external
changes.

The cascade covers the CLI binary, Pro plugins archive, the Admin Docker image,
the Homebrew tap formula, the ping_api canary deploy, all 11 Vercel web subapps,
README badge sync across 10 repositories, and the SPORT / changelog PR. It then
starts the 48-hour soak timer and dispatches post-release PCI messages.

Run `nself release-check <version>` first to validate pre-flight conditions, or
pass `--skip-check` to bypass that gate.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dry-run` | `–` | bool | `false` | Run all steps in simulation mode without external mutations |
| `--json` | `–` | bool | `false` | Emit structured JSON output instead of human-readable progress |
| `--skip-check` | `–` | bool | `false` | Skip the release-check pre-flight gate (step 0) |
| `--skip-plugins-pro` | `–` | bool | `false` | Skip the plugins-pro tagging step (step 2) |
| `--release-notes` | `–` | string | `""` | Path to a markdown file used as the GitHub Release body |

## Release Cascade

The release runs 13 ordered steps (0-12):

| Step | Action |
|------|--------|
| 0 | Pre-flight: runs `nself release-check <version>` |
| 1 | CLI: creates git tag, pushes, opens GitHub Release |
| 2 | plugins-pro: creates matching version tag |
| 3 | Admin: builds and pushes Docker image to Docker Hub |
| 4 | Homebrew: opens pull request on `homebrew-nself` tap |
| 5 | ping_api: deploys canary build |
| 6 | Artifact verification: polls all distribution URLs until live |
| 7 | web/\*: bumps version strings in 11 web subapp sources |
| 8 | Vercel: triggers deploy for all 11 subapps and polls for green |
| 9 | README badges: updates version badge in 10 repositories |
| 10 | SPORT regen and changelog PR against `main` |
| 11 | 48-hour soak timer starts; release is marked active |
| 12 | Post-release PCIs dispatched to plugin and web agents |

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `release-check` | [Pre-flight validation](cmd-release-check.md) |
| `release-rollback` | [Roll back to a prior release](cmd-release-rollback.md) |
| `release-status` | [View release pipeline status](cmd-release-status.md) |

## Examples

```bash
# Rehearse a release without touching any external services
nself release v1.2.0 --dry-run

# Full release using a custom release notes file
nself release v1.2.0 --release-notes ./notes/v1.2.0.md

# Release, skipping the pre-flight gate (run check separately first)
nself release v1.2.0 --skip-check
```

## See Also

- [[cmd-release-check]], pre-flight validation before releasing
- [[cmd-release-status]], check which surfaces are live after release
- [[cmd-release-rollback]], revert distribution surfaces to a prior version
- [[cmd-soak]], manage the 48-hour post-release soak window
- [[cmd-deploy]], deploy individual environments outside the release cascade

← [[Commands]] | [[Home]] →
