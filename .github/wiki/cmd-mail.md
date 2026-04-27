# nself mail

> Send transactional and broadcast email through the nSelf stack.

## Synopsis

```
nself mail <subcommand>
```

## Description

`nself mail` wraps the mux + Postmark plugins. ping_api proxies each call to the running stack, so the Postmark plugin must be installed and a valid license key must be configured.

The command is license-gated. The Postmark plugin ships with the nClaw bundle and ɳSelf+. Without a configured key, the command exits with code `2` and points at `nself license add`.

**Configuration**

| Variable | Default | Purpose |
|---|---|---|
| `NSELF_PING_API_URL` | `https://ping.nself.org` | ping_api base URL |
| `NSELF_LICENSE_KEY` (or `NSELF_PLUGIN_LICENSE_KEY`, `NSELF_LICENSE_KEY_1`..`NSELF_LICENSE_KEY_10`) | — | License token sent as `Authorization: Bearer <key>` |

## Subcommands

| Subcommand | Description |
|---|---|
| `send` | Send a single transactional email via the mux pipeline |
| `broadcast` | Send a broadcast to a list using a saved Postmark template |
| `status` | Query delivery status for a previously sent message |
| `templates list` | List Postmark templates registered with nSelf |
| `dkim verify` | Verify the DKIM record for a domain |

All subcommands accept `--json` for machine-readable output.

---

## nself mail send

Send a single transactional email through the mux pipeline.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--to <addr>` | — | Recipient email address (required) |
| `--subject <s>` | — | Email subject (required) |
| `--body <text>` | — | Email body (inline) |
| `--body-file <path>` | — | Read body from file (`-` for stdin) |
| `--body-type <type>` | `text` | `text` or `html` |
| `--json` | `false` | Output as JSON |

`--body` and `--body-file` are mutually exclusive. One of the two is required.

### Example

```bash
nself mail send \
  --to user@example.com \
  --subject "Welcome" \
  --body "Thanks for signing up."
```

```
✓ Email queued for user@example.com
  Message ID: 7c0f...
  Accepted:   true
```

JSON form:

```bash
nself mail send --to user@example.com --subject "Welcome" --body "Hi" --json
```

```json
{
  "message_id": "7c0f...",
  "accepted": true,
  "to": "user@example.com"
}
```

---

## nself mail broadcast

Queue a broadcast email job against a saved list using a Postmark template.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--list <list-id>` | — | Mailing list ID (required) |
| `--template <tpl-id>` | — | Postmark template ID (required) |
| `--json` | `false` | Output as JSON |

### Example

```bash
nself mail broadcast --list customers --template welcome
```

```
✓ Broadcast queued: batch batch-9
  Recipients: 1284
  Queued:     true
```

---

## nself mail status

Query the delivery status of a previously sent message.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--message-id <id>` | — | Message ID returned by `nself mail send` (required) |
| `--json` | `false` | Output as JSON |

### Example

```bash
nself mail status --message-id 7c0f-abc-123
```

```
Message ID: 7c0f-abc-123
Status:     delivered
To:         user@example.com
Delivered:  2026-04-27T12:34:56Z
```

Possible status values: `queued`, `sent`, `delivered`, `bounced`, `spam_complaint`.

---

## nself mail templates list

List Postmark templates registered with the nSelf stack.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--json` | `false` | Output as JSON |

### Example

```bash
nself mail templates list
```

```
ID      Name      Subject              Provider   Updated
welcome Welcome   Welcome to nSelf     postmark   2026-04-20
receipt Receipt   Order confirmation   postmark   2026-04-22
```

---

## nself mail dkim verify

Check that a DKIM record is present and valid for the supplied domain.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--domain <d>` | — | Domain to verify (required) |
| `--json` | `false` | Output as JSON |

### Example

```bash
nself mail dkim verify --domain example.com
```

```
Domain:   example.com
✓ DKIM record valid
Selector: pm._domainkey
Record:   v=DKIM1; k=rsa; p=...
```

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Generic error (network, server, validation) |
| `2` | No license key configured (Postmark plugin not entitled) |

## Errors

| Error | Cause | Resolution |
|---|---|---|
| `nself mail requires nSelf+ or nClaw bundle` | No license key configured | `nself license add <key>` |
| `license rejected by ping.nself.org` | Key invalid or expired | `nself license validate` |
| `ping.nself.org unreachable` | Network failure or service outage | Check connectivity, retry |
| `ping_api server error` | 5xx from ping_api | Check `https://status.nself.org` and retry |

## Related

- [[cmd-license]] — Manage license keys for paid bundles
- [docs.nself.org/mail](https://docs.nself.org/mail) — Full mail user guide
- [[Home]]
