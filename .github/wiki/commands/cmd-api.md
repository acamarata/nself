# nself api

**API versioning and deprecation tooling for operators.**

Operator commands for the ɳSelf API versioning baseline (v1.0.9).
The long-term support contract guarantees backward compatibility through 2027-04-17.

## Synopsis

```
nself api <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|-----------|-------------|
| `version` | Show API version for every surface observable from this install |
| `deprecation-check` | Check for deprecated API usage in this install |

---

## nself api version

Show the API version for every surface reachable from this install.

```
nself api version [--surface <name>] [--json] [--timeout <seconds>]
```

**What it probes:**
- `cli`, this binary's version
- `ping_api`, version probed from `ping.nself.org/version`
- `marketplace`, version probed from `plugins.nself.org/health` headers
- `sdk`, per-installed-plugin SDK version from `plugin.json` `apiVersion` field
- `hasura`, local Hasura running status (if stack is running)

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--surface` | (all) | Filter to a single surface: `cli`, `ping_api`, `marketplace`, `sdk`, `hasura` |
| `--json` | false | Output as JSON instead of table |
| `--timeout` | 5 | HTTP probe timeout in seconds |

### Examples

```bash
# Show all surfaces
nself api version

# Filter to ping_api surface only
nself api version --surface ping_api

# JSON output for scripting
nself api version --json
```

### Example output

```
Surface                        Version         Deprecated   EOL Date
------------------------------------------------------------------------
cli                            1.0.9           no           -
ping_api                       1.0.9           no           -
marketplace                    v1              no           -
hasura                         running (version via `nself status --json`)  no  -
```

---

## nself api deprecation-check

Walk installed plugins and CLI command tree to find any deprecated API usage.

```
nself api deprecation-check [--json]
```

Cross-references the central deprecation registry at `.claude/docs/api-deprecations.md`
(mirrored publicly at `docs.nself.org/api/deprecations/`).

At v1.0.9 baseline, the registry is empty. This command exits 0 with "0 deprecations
found" at baseline.

When future deprecations are added, this command will report:

```
[ping_api] /license/old-endpoint (deprecated since v1.1.0, EOL 2028-06-01)
  Migration: https://docs.nself.org/api/deprecations/ping-api-license-old-endpoint
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Output as JSON instead of human-readable |

### Examples

```bash
# Check for deprecated API usage
nself api deprecation-check

# JSON output
nself api deprecation-check --json
```

### Example output (v1.0.9 baseline)

```
0 deprecations found. Your install is clean against the v1.0.9 LTS baseline.

  Registry: .claude/docs/api-deprecations.md (no entries at v1.0.9)
  LTS window: 2026-04-17 → 2027-04-17
```

---

## long-term support Commitment

ɳSelf v1.0.9 is an long-term support release. Every surface listed in `nself api version` is
backward-compatible through 2027-04-17. Breaking changes during this window require:

1. A `BREAKING-CHANGE-OK` annotation in the PR
2. A maintainer (not PR author) approval label
3. An entry in the deprecation registry with at least 6 months notice

See [API Deprecations](https://docs.nself.org/api/deprecations/) and
`.claude/docs/operations/breaking-change-policy.md` for details.

---

[[Home]] | [[Commands]] | [[Changelog]]
