# nself license

> Manage your nSelf Pro membership license key.

## Synopsis

```
nself license <subcommand>
```

## Description

`nself license` manages the Pro membership license key used to install paid plugins. The key is stored locally at `~/.nself/license/key` with permissions `0600` (readable only by the current user).

Set your key once with `nself license set` and all subsequent `nself plugin install` commands for Pro plugins will use it automatically. The key is validated server-side against `ping.nself.org` — the CLI does not decode or verify the key locally.

**Key format:** One of four accepted prefixes followed by 32 or more characters:
- `nself_pro_*` — Pro tier
- `nself_max_*` — Pro tier
- `nself_ent_*` — Enterprise tier
- `nself_owner_*` — Owner tier

Keys are validated with a POST request to `https://ping.nself.org/license/validate`.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `set <key>` | Save a Pro license key to `~/.nself/license/key` |
| `show` | Display the saved key (masked) and tier |
| `validate` | Validate the saved key against `ping.nself.org` |
| `clear` | Remove the saved license key |
| `upgrade` | Open the pricing page in your browser |
| `tail` | Stream live license validation events from ping_api |

---

## nself license tail

Stream live license validation events from `ping.nself.org` in real time.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--filter <field=value>` | — | Filter events (repeatable) |
| `--ping-url <url>` | `$NSELF_PING_URL` or `https://ping.nself.org` | ping_api base URL |

### Filters

| Filter | Example | Description |
|---|---|---|
| `result=<value>` | `result=denied` | Show only `allow`, `deny`, or `rate_limit` events |
| `key=<prefix>` | `key=nself_pro_abc` | Show events for a specific key (first 12 chars matched) |
| `plugin=<name>` | `plugin=ai` | Show events for a specific plugin |

### Color coding

| Color | Meaning |
|---|---|
| Green | `allow` |
| Red | `deny` |
| Yellow | `rate_limit` |

Colors are only shown when output is a terminal. Piped output is plain text.

### Behavior

- Starts streaming within 2s of invocation
- Reconnects automatically on connection loss (exponential backoff, max 30s)
- Ctrl-C exits cleanly with no goroutine leak
- License key values are never shown in full — only the first 12 characters (key prefix) are displayed

### Examples

```bash
# Stream all validation events
nself license tail

# Show only denied validations
nself license tail --filter result=denied

# Show events for a specific key prefix
nself license tail --filter key=nself_pro_abc123

# Combine filters
nself license tail --filter result=denied --filter plugin=ai
```

## Examples

```bash
# Save your license key
nself license set nself_pro_xxxxx...

# Check what key is saved and its tier
nself license show

# Validate the key is still active
nself license validate

# Remove the saved key
nself license clear

# Open the pricing page to upgrade
nself license upgrade
```

**Sample `show` output:**

```
License key: nself_pro_****************************xxxx
Tier:        Pro
Status:      ✓ active
Stored at:   ~/.nself/license/key
```

← [[Commands]] | [[Home]] →
