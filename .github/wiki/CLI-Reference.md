# CLI Reference — E6 AI Gateway Trio

This page documents the ten commands added or updated in E6 (Gateway Unification).
Full command reference: [[COMMANDS]] | Architecture: [[Architecture-Microkernel]]

---

## nself gateway

Manage the nSelf AI gateway (nself-ai-gateway, port 3761).

| Subcommand | Description |
|---|---|
| `nself gateway status` | Health-check all three AI services (3760, 3761, 3762) |
| `nself gateway keys list` | List registered provider keys (metadata only — no key material returned) |
| `nself gateway keys add` | Register a new AI provider key (AES-256-GCM encrypted at rest) |
| `nself gateway keys remove <id>` | Deactivate a provider key by ID |
| `nself gateway quota` | Show current quota usage (today, per provider/model) |
| `nself gateway routes` | List active provider routing rules |

Full reference: [[commands/cmd-gateway]]

---

## nself claw session

Manage Claude Code PTY sessions via nself-ai-cc (port 3760).

| Subcommand | Description |
|---|---|
| `nself claw session start [provider]` | Start a new PTY session (default: `claude`) |
| `nself claw session list` | List active sessions for the authenticated user |
| `nself claw session attach <id>` | Attach to an active session, stream output to stdout |
| `nself claw session stop <id>` | Terminate a PTY session (SIGTERM then SIGKILL after 5s) |

Full reference: [[commands/cmd-claw-session]]

---

## Canonical Ports (E6 trio)

| Service | Port | Purpose |
|---|---|---|
| nself-ai-cc | 3760 | PTY session relay |
| nself-ai-gateway | 3761 | AI provider key vault + request router |
| nself-ai-mcp | 3762 | MCP tool server (stdio + SSE) |

All three ports are centralized in `cli/internal/ports/ports.go` (constants `AICCPort`, `AIGatewayPort`, `AIMCPPort`).

---

## Related

- [[commands/cmd-gateway]] — full gateway command reference
- [[commands/cmd-claw-session]] — full claw session reference
- [[Config-Env-Vars]] — environment variables reference (see § AI Gateway Trio)
- [[COMMANDS]] — complete command index

[[Home]]
