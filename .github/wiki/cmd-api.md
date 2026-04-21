# nself api

> API versioning and deprecation tooling for operators.

## Synopsis

```
nself api <subcommand> [flags]
```

## Description

`nself api` provides tooling to inspect and validate API version state across all observable surfaces in a running nSelf install. Operators use it to check API surface versions before upgrades and to detect deprecated API usage before clients are affected.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `version` | Show API version for every surface observable from this install |
| `deprecation-check` | Check for deprecated API usage in this install |

---

### nself api version

```
nself api version [flags]
```

Reports the current API version for every observable surface (Hasura, Auth, Storage, Functions, etc.) in the running nSelf stack. Use this after an upgrade to confirm all surfaces are on the expected version.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--surface` | (all) | Filter output to a specific surface name |
| `--json` | false | Print output as JSON |
| `--timeout` | `10s` | HTTP timeout for surface probes |
| `--help`, `-h` | — | Show help |

**Example:**

```bash
# Check all API versions
nself api version

# JSON output for CI
nself api version --json

# Check only the Hasura surface
nself api version --surface hasura
```

---

### nself api deprecation-check

```
nself api deprecation-check [flags]
```

Probes the running stack for deprecated API usage — deprecated Hasura metadata fields, deprecated auth endpoints, deprecated storage paths, or deprecated plugin hook signatures. Exits non-zero if deprecations are found, making it CI-safe.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Print findings as JSON |
| `--help`, `-h` | — | Show help |

**Example:**

```bash
# Check for deprecated API usage
nself api deprecation-check

# JSON output for automated tooling
nself api deprecation-check --json
```

## See Also

- [[cmd-version]] — CLI version information
- [[cmd-doctor]] — Comprehensive system diagnostics
- [[cmd-status]] — Live service health
