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
| `deprecation-check` | Check for deprecated API usage in this install (G6) |
| `changelog` | Show the deprecation calendar and sunset dates for a plugin (G9) |

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

Probes the plugin deprecation registry for deprecated API usage. Reports endpoints that are deprecated or approaching removal. Exits non-zero if any BREAKING entries are found (entries with no `deprecated_in` grace period), making it suitable for CI gates.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--plugin` | (all) | Filter check to a specific plugin name |
| `--strict` | false | Exit non-zero if any BREAKING entries found (used by CI gate) |
| `--json` | false | Print findings as JSON |
| `--help`, `-h` | — | Show help |

**Example:**

```bash
# Check all plugins for deprecated API usage
nself api deprecation-check

# Check a specific plugin
nself api deprecation-check --plugin ai

# CI-safe strict mode (exits 1 on any BREAKING entry)
nself api deprecation-check --strict --json

# JSON output for automated tooling
nself api deprecation-check --json
```

---

### nself api changelog

```
nself api changelog <plugin> [flags]
```

Shows the deprecation calendar for a plugin: all deprecated endpoints, when each was deprecated, when it will be removed, and any Sunset header dates. Use this to plan client migrations before removal dates.

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<plugin>` | Plugin name (e.g. `ai`, `mux`, `claw`) |

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Print output as JSON |
| `--help`, `-h` | — | Show help |

**Example:**

```bash
# Show deprecation calendar for the ai plugin
nself api changelog ai

# JSON output for automated tooling
nself api changelog ai --json

# Show calendar for the mux plugin
nself api changelog mux
```

**Output:**

```
Plugin: ai  (API v1.3.0)
Deprecated endpoints: 0
All endpoints are current. No action needed.
```

When deprecated endpoints exist:

```
Plugin: ai  (API v1.3.0)
Deprecated endpoints: 1

  Endpoint        deprecated_in  removed_in  sunset
  /v1/complete    v1.2.0         v2.0.0      Sat, 01 Jan 2027 00:00:00 GMT
  Replacement: POST /v2/complete
  Reason: Unified completion API with streaming support
```

## See Also

- [[cmd-version]] — CLI version information
- [[cmd-doctor]] — Comprehensive system diagnostics
- [[cmd-status]] — Live service health
