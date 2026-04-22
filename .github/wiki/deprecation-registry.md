# Deprecation Registry

This page lists all CLI commands, flags, and env vars that are currently in Phase 1 or Phase 2
of the deprecation cycle.

## Global Deprecation Flags

The following persistent flags are available on all `nself` commands:

| Flag | Description |
|------|-------------|
| `--no-deprecation-warnings` | Suppress all deprecation warnings (use in scripts to avoid noisy stderr) |

Deprecation warnings are always written to **stderr**, never stdout, so they never break piped output.
Use `--no-deprecation-warnings` only in scripted environments where the warning is expected and irrelevant.

## Currently Deprecated Items

### v1.0.9 LTS (April 2026)

| Item | Type | Since | Replacement | Phase | EOL |
|------|------|-------|-------------|-------|-----|
| `nMedia` bundle | bundle name | v1.0.9 | `nTV` bundle | 1 | v1.2.0 |
| `nself_sessions.user_agent` (Hasura) | schema column | v1.0.9 | `device_type` + `device_name` | 1 | 2027-04-17 |

No CLI commands or flags are deprecated at v1.0.9.

## Deprecation Cycle

nSelf uses a 3-phase deprecation cycle across all surfaces:

**Phase 1 — Warning:** Item works. Every invocation prints:
```
[DEPRECATED] 'nself old-cmd' (since v1.0.9) → use 'nself new-cmd'. Docs: https://docs.nself.org/...
```

**Phase 2 — Deprecated:** Item still works. Warning includes removal timeline.

**Phase 3 — Removed:** Item gone. Clear error with migration guide link.

Emergency removals (security CVEs only) skip phases. See the full doctrine at
`docs.nself.org/deprecation-cycle`.

## Adding a Deprecation

To deprecate a CLI command or flag:

1. Add an entry to `cli/internal/deprecation/registry.yaml`.
2. Open a PR — the `deprecation-check.yml` CI gate verifies the entry is present.
3. Update this page.

```yaml
# registry.yaml
items:
  - name: "nself old-cmd"
    type: command
    since: "v1.1.0"
    replacement: "nself new-cmd"
    docs_url: "https://docs.nself.org/migrate/v1-1-0"
    phase: 1
```

[[Home]]
