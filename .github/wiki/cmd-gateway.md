# nself gateway

Manage the nSelf AI gateway service: provider key vault, quota tracking, routing rules, and service health.

## Usage

```bash
nself gateway <subcommand> [flags]
```

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `status` | Health-check all three AI services |
| `keys list` | List stored provider keys (no key material shown) |
| `keys add` | Add a provider API key |
| `keys remove <id>` | Remove a provider key by ID |
| `quota` | Show daily token and request usage |
| `routes` | List active routing rules |

---

## nself gateway status

Health-check `nself-ai-cc` (port 3760), `nself-ai-gateway` (port 3761), and `nself-ai-mcp` (port 3762) in parallel. Exits 0 only when all three services are healthy.

```bash
nself gateway status
```

Example output:

```
SERVICE            PORT   STATUS
-------            ----   ------
nself-ai-cc        3760   ok
nself-ai-gateway   3761   ok
nself-ai-mcp       3762   ok

3/3 services healthy
```

If any service is unreachable, exits 1 with an error message and hint.

---

## nself gateway keys list

List all provider keys stored in the nself-ai-gateway vault. Key material (actual API key values) is never displayed.

```bash
nself gateway keys list
```

Columns: `ID`, `PROVIDER`, `LABEL`, `ACTIVE`, `CREATED`

---

## nself gateway keys add

Add a provider API key to the vault. If `--key` is not provided, input is prompted with characters hidden.

```bash
nself gateway keys add --provider anthropic
nself gateway keys add --provider openai --label "production" --key sk-...
```

Supported providers: `anthropic`, `openai`, `google`, `custom`

Flags:

| Flag | Required | Description |
|------|----------|-------------|
| `--provider` | yes | Provider identifier |
| `--key` | no | API key value (hidden prompt if omitted) |
| `--label` | no | Optional label for the key |

---

## nself gateway keys remove

Remove a provider key by its UUID.

```bash
nself gateway keys remove 550e8400-e29b-41d4-a716-446655440000
```

Use `nself gateway keys list` to find the key ID.

---

## nself gateway quota

Display today's token and request usage from nself-ai-gateway. Filter by provider or model with flags.

```bash
nself gateway quota
nself gateway quota --provider anthropic
nself gateway quota --provider openai --model gpt-4o
```

Columns: `PROVIDER`, `MODEL`, `DATE`, `TOKENS_IN`, `TOKENS_OUT`, `REQUESTS`, `LIMIT`

---

## nself gateway routes

List the active routing rules that determine which provider and model handle each request.

```bash
nself gateway routes
```

Columns: `NAME`, `PROVIDER`, `MODEL`, `PRIORITY`, `ENABLED`

---

## Error format

All gateway commands follow the nSelf error format:

```
Error: <what went wrong>
Hint: <actionable suggestion>
Exit: <code>
```

Common exit codes:

| Code | Meaning |
|------|---------|
| 1 | Missing key or invalid argument |
| 2 | Quota exceeded |
| 3 | Gateway service unreachable |

---

## Related

- [[cmd-claw]] — nClaw session and proxy commands
- [[plugin-nself-ai-gateway]] — gateway service configuration
- [[Home]]
