# nself claw session

Manage Claude Code PTY sessions via nself-ai-cc (port 3760).

## Usage

```
nself claw session <subcommand> [flags]
```

PTY sessions allow you to start, attach to, and stop interactive Claude Code sessions. Each session is a managed subprocess (PTY) tracked by nself-ai-cc. Sessions persist until explicitly stopped or the service restarts.

## Subcommands

### `nself claw session start`

Start a new Claude Code PTY session.

```
nself claw session start [provider] [flags]
```

**Arguments:**
| Arg | Default | Description |
|---|---|---|
| `provider` | `claude` | AI provider: `claude`, `codex`, or a custom binary name |

**Flags:**
| Flag | Default | Description |
|---|---|---|
| `--binary` | _(PATH lookup)_ | Explicit binary path override (overrides `CLAUDE_BINARY_PATH` / `GEMINI_BINARY_PATH`) |
| `--json` | false | Output session details as JSON |

**Example:**
```
$ nself claw session start claude
Session started: id=sess_1234567890_1 status=pending
WebSocket: ws://localhost:3760/sessions/sess_1234567890_1/stream
```

The session starts in `pending` state and transitions to `active` once the PTY subprocess is ready. Use `nself claw session attach` to connect.

---

### `nself claw session list`

List active Claude Code PTY sessions for the authenticated user.

```
nself claw session list [flags]
```

**Flags:**
| Flag | Default | Description |
|---|---|---|
| `--json` | false | Output as JSON |
| `--all` | false | Include ended/error sessions |

**Example:**
```
$ nself claw session list
ID                        Provider  Status   Created
sess_1234567890_1         claude    active   2026-06-25T11:00:00Z
sess_1234567890_2         claude    pending  2026-06-25T11:05:00Z
```

---

### `nself claw session attach`

Attach to an active Claude Code PTY session. Streams output to stdout and forwards stdin to the session.

```
nself claw session attach <id>
```

**Arguments:**
| Arg | Description |
|---|---|
| `id` | Session ID from `nself claw session list` or `start` |

**Example:**
```
$ nself claw session attach sess_1234567890_1
Attached to session sess_1234567890_1 (claude) — Ctrl+C to detach
> ...
```

**Note:** In P4, the WS connection uses `?token=<jwt>` in the URL (CLI-only pattern). This will migrate to Authorization header in P5 (OD-E1-04).

---

### `nself claw session stop`

Terminate a Claude Code PTY session. Sends SIGTERM to the subprocess; SIGKILL after 5s grace period.

```
nself claw session stop <id> [flags]
```

**Arguments:**
| Arg | Description |
|---|---|
| `id` | Session ID to terminate |

**Flags:**
| Flag | Default | Description |
|---|---|---|
| `--force` | false | Skip grace period; send SIGKILL immediately |

**Example:**
```
$ nself claw session stop sess_1234567890_1
Session sess_1234567890_1 stopped.
```

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `NSELF_AICC_URL` | No | Override nself-ai-cc base URL (default: `http://localhost:3760`) |
| `CLAUDE_BINARY_PATH` | No | Absolute path to `claude` binary. Falls back to `exec.LookPath("claude")`. |
| `GEMINI_BINARY_PATH` | No | Absolute path to `gemini` binary. Falls back to `exec.LookPath("gemini")`. |
| `NSELF_JWT` | Yes | Authenticated user JWT (sourced from `nself auth token`) |

## Session Limits

A maximum of 10 concurrent active sessions per user is enforced by nself-ai-cc. Attempting to create an 11th active session returns a 429 error.

## Security

- PTY subprocess output is never inspected for API keys or tokens (DD-3 ToS boundary).
- WS `?token=` parameter is redacted in access logs (OD-E1-04 P4 pattern).

## Related

- `nself claw proxy` — proxy LLM requests via nself-ai-gateway (3761)
- `nself gateway status` — health-check all three AI services
- SPORT `F08-SERVICE-INVENTORY.md` — nself-ai-cc service entry
- SPORT `F10-PORT-REGISTRY.md` — port 3760 assignment
