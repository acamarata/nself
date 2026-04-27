# License Verification (Offline Mode)

ɳSelf is offline-first. License verification works on disconnected, intermittent, or air-gapped networks. The CLI caches a signed license artifact and validates it locally on every command.

## Fail-Open Policy

The CLI prefers availability over strict revocation. The cache TTL ladder governs behavior when remote validation is unreachable.

| Cache age | Behavior | User signal |
|---|---|---|
| 0 to 7 days | FAIL-OPEN, silent | None |
| 7 to 14 days | FAIL-OPEN, warning to stderr | `warning: license cache is N days old; refresh when online` |
| Over 14 days | Fail-closed, command refuses to run | `error: license cache expired; run 'nself license refresh'` |
| Bad signature, any age | Fail-closed, always | `error: license signature invalid` |

A bad signature is never accepted. Cache age cannot bypass cryptographic verification.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `NSELF_LICENSE_OFFLINE_MAX_DAYS` | `14` | Hard cap before fail-closed. Set lower for stricter posture. |
| `NSELF_LICENSE_CACHE_PATH` | `~/.nself/license/cache.json` | Cache file location. Override for shared CI runners or air-gapped hosts. |

## Manual Refresh

```bash
nself license refresh
```

Pulls a fresh signed artifact from `ping.nself.org/license/validate`, replaces the cache, resets the age clock. Run after extended offline periods, after key rotation, or when the warning fires.

## Air-Gapped Mode

For hosts that never reach the public internet:

1. Request an offline license key from `cloud.nself.org/account/offline-license`
2. Save the signed `.nself-offline-license` file
3. Place the file at `~/.nself/license/offline.key`
4. Set `NSELF_LICENSE_OFFLINE=1`

The CLI validates the signature locally and skips remote checks. Offline keys carry an embedded expiration (typically 1 year). Renew before expiration through the same workflow.

## Troubleshooting

**Cache corruption.** Delete `~/.nself/license/cache.json` and run `nself license refresh`. The cache is regenerable.

**Signature invalid after CLI upgrade.** Run `nself license refresh`. Major version bumps may rotate signing keys.

**Manual signature verification.** Inspect the cache contents and verify against the embedded public key:

```bash
nself license verify --offline ~/.nself/license/cache.json
```

**Stuck in fail-closed.** Connect to the internet and run `nself license refresh`. If the network is restricted, request an offline license key.

## See Also

- [License Management](license-management.md)
- [Plugin Installation](plugins.md)
- [nself license commands](commands/license.md)
