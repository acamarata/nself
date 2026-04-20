# Plugin Permissions

Plugins declare required permissions in the `permissions` array of their `plugin.json` manifest. The nSelf CLI validates every declared permission against a canonical allowlist at install time and rejects unknown strings with a descriptive error (fail-closed semantics).

## Viewing Plugin Permissions

```bash
nself plugin info <name>
```

The output includes a `Permissions:` block listing each declared permission with a risk prefix:

```
Permissions:
  [low]  db:read
  [low]  db:write
  [med]  network:internet
  (v1.0.9: informational only; v1.1.0 will require explicit confirmation)
```

## Tier Roadmap

| Version | Tier | Behaviour |
|---------|------|-----------|
| **v1.0.9** | Tier 1 | Declaration validated at install; granted set written to audit log. No runtime enforcement. |
| **v1.1.0** | Tier 2 | Install-time confirmation prompt per dangerous permission; `nself plugin revoke` command. |
| **v1.2.0** | Tier 3 | Runtime egress enforcement via transparent HTTP proxy; `outbound_hosts` enforced at network layer. |

## Canonical Permissions

### Network

| Permission | Risk | Description |
|-----------|------|-------------|
| `network:internet` | medium | Unrestricted outbound internet access. |
| `network:plugin:<name>` | low | Calls another installed plugin's HTTP API. |

### Database

| Permission | Risk | Description |
|-----------|------|-------------|
| `db:read` | low | Read access to plugin-owned tables. |
| `db:write` | low | Write access to plugin-owned tables. |

### Secrets

| Permission | Risk | Description |
|-----------|------|-------------|
| `secrets:env:<VAR_NAME>` | medium | Read a specific environment variable at runtime. |

### Filesystem

| Permission | Risk | Description |
|-----------|------|-------------|
| `fs:write:<volume>` | medium | Write access to a named Docker volume. |

### System

| Permission | Risk | Description |
|-----------|------|-------------|
| `system:exec` | high | Execute arbitrary subprocess commands. |

### AI

| Permission | Risk | Description |
|-----------|------|-------------|
| `ai:provider:<name>` | medium | Call a named AI provider (e.g. `ai:provider:openai`). |

## For Plugin Authors

Declare all permissions your plugin uses in `plugin.json`:

```json
{
  "permissions": ["db:read", "network:internet"],
  "outbound_hosts": ["api.openai.com"]
}
```

Declare `outbound_hosts` now (even though not enforced until v1.2.0) so your manifest is forward-compatible with Tier 3 enforcement.

Using an undeclared or unknown permission string causes `nself plugin install` to fail with:

```
unknown permission: your-perm: plugin manifest error
```

---

[[Plugin-Install]] | [[Plugin-Overview]] | [[Home]]
