# CLI Reference — ClawDE-Specific Extensions

This document covers ClawDE-specific extensions to the nSelf CLI. For the full nSelf CLI command reference (84 commands), see `.github/wiki/COMMANDS.md` and `SPORT F02-COMMAND-INVENTORY.md`.

---

## PTY Allocation

**Scope:** ClawDE only (ADR-001 boundary). PTY allocation is not surfaced as an nSelf CLI flag. It is an internal `clawd` daemon primitive managed by the host-local Rust process.

**ADR references:**
- ADR-001 (`.claude/phases/current/p1/adrs/ADR-001-clawde-pivot-service-boundary.md`) — PTY bridge is ClawDE-native; not exposed to vanilla nSelf users.
- ADR-003 (`.claude/phases/current/p1/adrs/ADR-003-mcp-policy-and-trust-model.md`) — PTY sessions are NOT MCP tools; they sit below the MCP surface.

**Full specification:** `.claude/phases/current/p1/evidence/e4-pty-allocation-spec.md`

### Runtime Configuration

PTY allocation behavior is controlled by two environment variables read at `clawd` daemon startup:

| Variable | Default | Range | Description |
|---|---|---|---|
| `CLAWDE_PTY_MAX_SESSIONS` | `4` | `1–64` | Maximum concurrent PTY sessions. Exceeding this limit returns `AllocError::QuotaExceeded`. |
| `CLAWDE_PTY_IDLE_TIMEOUT_S` | `300` | `30–86400` | Idle timeout in seconds. Sessions idle beyond this value are reclaimed automatically. |

### Allocation API

```rust
pub fn pty_alloc(
    owner_id: &str,
    cols: u16,
    rows: u16,
) -> Result<(RawFd, String), AllocError>
```

**Error variants:**
- `AllocError::QuotaExceeded { current, limit }` — session limit reached
- `AllocError::SandboxInitFailed { reason }` — platform sandbox could not be applied
- `AllocError::TimeoutError { elapsed_ms }` — PTY device open stalled

### Platform Sandbox

| Platform | Mechanism | Network | File writes |
|---|---|---|---|
| Linux (amd64/arm64) | seccomp BPF, default action `SCMP_ACT_ERRNO(EPERM)` | Blocked (`socket`, `connect` denied) | Per-workspace only |
| macOS (arm64) | `sandbox-exec` profile | Blocked (`deny network*`) | Workspace root only (`deny file-write*` with workspace exception) |
| Windows | ConPTY (no seccomp) | N/A — platform-level ACL | N/A |

**Seccomp allowed syscalls (Linux, 30 total):** `read`, `write`, `open`, `openat`, `close`, `mmap`, `mmap2`, `mprotect`, `munmap`, `ioctl`, `select`, `poll`, `epoll_wait`, `epoll_ctl`, `epoll_create1`, `getpid`, `gettid`, `exit`, `exit_group`, `sigreturn`, `rt_sigreturn`, `rt_sigaction`, `rt_sigprocmask`, `futex`, `nanosleep` — plus PTY ioctls: `TIOCGWINSZ`, `TIOCSWINSZ`, `TIOCSPTLCK`, `TIOCGPTPEER`, `TIOCGPTN`.

### Session Schema

Each PTY session is tracked in-memory in `AppContext::pty_sessions`:

| Field | Type | Description |
|---|---|---|
| `session_id` | `String` (UUID v4) | Stable session identifier |
| `owner_id` | `String` | Allocating entity identifier |
| `fd` | `RawFd` | Master PTY file descriptor |
| `start_time` | `DateTime<Utc>` | Allocation wall-clock time |
| `last_activity` | `DateTime<Utc>` | Last I/O timestamp (idle-reaper input) |
| `window_cols` | `u16` | Terminal width (updated via TIOCSWINSZ) |
| `window_rows` | `u16` | Terminal height (updated via TIOCSWINSZ) |
| `sandbox_pid` | `u32` | Worker process PID (zombie-guard via `is_process_alive`) |
| `status` | `PtyStatus` | `active` / `idle` / `reclaimed` / `error` |

---

*For the full PTY allocation specification including lifecycle diagram, seccomp BPF table, and unit test enumeration, see `.claude/phases/current/p1/evidence/e4-pty-allocation-spec.md`.*

---

## PTY Session Pool and Recovery

Pool state machine, failure modes, and recovery procedures for PTY session management.

**Full spec:** `.claude/phases/current/p1/evidence/e4-pty-pool-spec.md`

**Key env var:** `CLAWDE_PTY_RECONNECT_WINDOW_S` (default 60) — reconnect window for disconnected sessions.

**Pool states:** `idle`, `leased`, `heartbeating`, `stale`, `reclaimed`, `disconnected`.

**Heartbeat:** 15s client ping, 45s server timeout (3 misses), message format `{session_id, timestamp_ms, client_version}`.

---

## Auto-Permission Policy and Audit Log

ClawDE PTY sessions operate under a four-level permission model. This section describes the policy, audit log, and dangerous command gating.

**Full spec:** `.claude/phases/current/p1/evidence/e4-permission-policy-spec.md`

**ADR references:**
- ADR-001 — Permission policy is ClawDE-private; not exposed via nSelf plugin or MCP surface.
- ADR-003 — Dangerous command gate aligns with MCP tool-level approval; MCP-triggered dangerous commands pass through the same gate.
- ADR-004 — Session-scoped workspace token provides actor identity for all audit log entries.

### Permission Levels

| Level | Description | Default |
|---|---|---|
| `pty:read-only` | Read terminal output only; no injection | No |
| `pty:interactive` | Standard interactive access; dangerous commands require confirmation | Yes (on allocation) |
| `pty:elevated` | All interactive ops plus run-sudo and full-shell; time-bounded; requires explicit user approval | No (elevation required) |
| `pty:restricted` | Allowlisted commands only; immediate deny for dangerous commands; no confirmation dialog | No |

### Default Grant

All PTY sessions start at `pty:interactive` on allocation. Elevation to `pty:elevated` requires explicit ClawDE UI approval.

### Time-Bounded Elevated Access

`pty:elevated` expires after `CLAWDE_PTY_ELEVATED_TTL_S` seconds (default 900s / 15 min). Expiry is checked on every PTY write. Auto-downgrade to `pty:interactive` on expiry. Audit `action: expire` emitted.

| Env var | Default | Range |
|---|---|---|
| `CLAWDE_PTY_ELEVATED_TTL_S` | `900` | [60, 86400] |

### Dangerous Command Denylist — 8 Entries

The following commands trigger a per-action confirmation gate (30s timeout) at `pty:interactive` and `pty:elevated`. At `pty:restricted`, they are denied immediately without a dialog.

| Command | Match condition |
|---|---|
| `sudo` | `argv[0] == "sudo"` |
| `rm -rf` | `argv[0] == "rm"` AND recursive+force flags present |
| `dd` | `argv[0] == "dd"` |
| `mkfs` / `mkfs.*` | `argv[0] == "mkfs"` or starts with `"mkfs."` |
| `shred` | `argv[0] == "shred"` |
| `wipefs` | `argv[0] == "wipefs"` |
| `passwd` | `argv[0] == "passwd"` |
| `chown -R /` | `argv[0] == "chown"` AND recursive flag AND root path argument |

Match strategy: shell-tokenize argv (POSIX split), check `argv[0]` exact match (path-normalized), then flag tokens. Substring matching is not used. Confirmation timeout: 30s. On timeout or user denial: write discarded + SIGHUP to `sandbox_pid`.

### Revocation Sequence — 6 Steps

An audit log entry is emitted at each step. The sequence is invoked by `policy_engine.revoke()`.

1. In-memory session state marked `revoked`.
2. DB write: `revoked_at`, `revoked_by`, `revoke_reason` set; `permission_level` set to `pty:read-only`.
3. Next write attempt: `PermissionRevoked` error returned; no bytes injected.
4. `SIGHUP` sent to `sandbox_pid`.
5. 5-second grace period.
6. If process still alive: `SIGKILL`.

### Audit Log — Append-Only

All permission events are written to `cd_pty_audit_log` (SQLite). The log is append-only — no UPDATE or DELETE operations are ever issued on audit rows.

**Action enum values:** `grant` · `revoke` · `expire` · `downgrade` · `dangerous-cmd-blocked` · `dangerous-cmd-confirmed`

**Log fields:** `session_id`, `action`, `permission_level`, `actor`, `timestamp_ms`, `outcome`, `reason` (max 256 chars)

**Source files (Build wave):**
- `cli/internal/pty/permissions.go` — policy engine
- `cli/internal/pty/audit.go` — audit log writer
