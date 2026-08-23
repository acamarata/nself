# nself gateway

<!-- BEGIN PROSE:summary -->
> Manage the ɳSelf AI gateway (nself-ai-gateway, port 3761).
<!-- END PROSE:summary -->

## Synopsis

```
nself gateway <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Manage the nSelf AI gateway and related AI services (E6 canonical trio).

## Usage

```
nself gateway <subcommand> [flags]
```

### `nself gateway status`
Health-check all three AI services: nself-ai-gateway (3761), nself-ai-cc (3760), and nself-ai-mcp (3762).
```
nself gateway status [flags]
```
**Flags:**
**Example:**
```
$ nself gateway status
✓ nself-ai-gateway (3761) — healthy
✓ nself-ai-cc (3760) — healthy
✓ nself-ai-mcp (3762) — healthy
3/3 services healthy
```
---
### `nself gateway keys list`
List all registered AI provider keys for the authenticated user. Key material is never returned — only metadata (id, provider, label, is_active, created_at).
```
nself gateway keys list [flags]
```
**Flags:**
---
### `nself gateway keys add`
Register a new AI provider API key. The key is encrypted at rest (AES-256-GCM via NSELF_GATEWAY_MASTER_KEY) before storage. The plain key is transmitted over TLS only and never logged.
```
nself gateway keys add --provider <name> --key <api-key> [flags]
```
**Required flags:**
**Optional flags:**
**Example:**
```
$ nself gateway keys add --provider anthropic --key sk-ant-... --label "prod key"
Key registered: id=a1b2c3d4 provider=anthropic label="prod key"
```
---
### `nself gateway keys remove`
Deactivate a provider key by ID. The key remains in storage for audit purposes but is excluded from routing.
```
nself gateway keys remove <id>
```
---
### `nself gateway quota`
Show current AI request quota usage (today, per provider/model).
```
nself gateway quota [flags]
```
**Flags:**
**Example:**
```
$ nself gateway quota
Provider    Model              Tokens In  Tokens Out  Requests  Date
anthropic   claude-opus-4-5    12,450     4,200       18        2026-06-25
openai      gpt-4o             0          0           0         2026-06-25
```
---
### `nself gateway routes`
List active routing rules (provider routing configuration).
```
nself gateway routes [flags]
```
**Flags:**
---
## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `NSELF_GATEWAY_URL` | No | Override gateway base URL (default: `http://localhost:3761`) |
| `NSELF_JWT` | Yes | Authenticated user JWT (sourced from `nself auth token`) |

## Related

- `nself claw session` — manage PTY sessions backed by nself-ai-cc (3760)
- `nself claw proxy` — proxy LLM requests via nself-ai-gateway (3761)
- SPORT `F08-SERVICE-INVENTORY.md` — canonical service registry
- SPORT `F10-PORT-REGISTRY.md` — port assignments
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `keys` | Manage AI provider keys |
| `quota` | Show AI request quota usage |
| `routes` | List gateway routing rules |
| `status` | Health-check all three AI services |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
<!-- TODO(docs): needs human prose -->

```bash
nself gateway
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
