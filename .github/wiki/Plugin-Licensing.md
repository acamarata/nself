# Plugin Licensing

Pro plugins (59 total) require a valid nSelf license key. Free plugins (25 total) are MIT licensed and require no key.

## Key Format

License keys use the prefix format:

```
nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

Accepted prefixes:
- `nself_pro_` — Pro tier and above
- `nself_max_` — Max tier
- `nself_ent_` — Enterprise tier
- `nself_owner_` — Owner/self-hosted tier

Keys must be at least 32 characters total.

## Set Your License Key

```bash
nself license set nself_pro_xxxxx...
```

The key is stored at `~/.nself/license/key` (chmod 600) and also read from the `NSELF_PLUGIN_LICENSE_KEY` environment variable.

## Check License Status

```bash
nself license status
```

Shows your current tier, expiry, and which plugin categories are available.

## Remove Your Key

```bash
nself license remove
```

## Install Pro Plugins

After setting your key:

```bash
nself plugin install ai
nself plugin install claw livekit chat
nself build
```

Tier entitlement is checked server-side at `ping.nself.org/license/validate`. The CLI never makes tier decisions locally.

## Validation

License validation sends:

```
POST https://ping.nself.org/license/validate
{"license_key": "...", "product": "plugins-pro"}
```

Response codes:
- `200` — Valid, allowed
- `401/403/404` — Invalid or insufficient tier

Validation results are cached locally for 24 hours at `~/.nself/license/cache`.

## Offline Mode

To skip remote validation (e.g., air-gapped environments):

```bash
NSELF_LICENSE_SKIP_VERIFY=1 nself plugin install {name}
```

## Tier Entitlements

| Tier | Monthly | Annual | Plugin Access |
|------|---------|--------|--------------|
| Free | $0 | $0 | 16 free plugins only |
| Basic | $0.99 | $9.99 | All 59 pro plugins |
| Pro | $1.99 | $19.99 | Basic + full AI suite |
| Elite | $4.99 | $49.99 | Pro + email support |
| Business | $9.99 | $99.99 | Elite + 24h support |
| Business+ | $49.99 | $499.99 | Business + dedicated channel |
| Enterprise | $99.99 | $999.99 | Business+ + managed DevOps |

Existing $9.99/yr keys are grandfathered to the Basic tier.

## Related Pages

- [[Plugin-Overview]] — Tier comparison and plugin categories
- [[Plugin-Install]] — How to install plugins
