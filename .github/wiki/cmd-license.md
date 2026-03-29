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
