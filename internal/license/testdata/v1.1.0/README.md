# v1.1.0 License Cache Fixture

Backward-compatibility test fixture representing the license cache format used by
nself-cli v1.1.0, before the W2-T1 `PluginsAllowed` Ed25519 signing scheme was
tightened in v1.1.1.

## Files

- `license.json` — a v1.1.0-era CacheEntry with `signature_key_id: 1` and the
  pre-v1.1.1 mock signature bytes. The signature will not pass verification (it
  is mock data), but the file structure is representative of what a v1.1.0
  installation wrote to `~/.cache/nself/license.json`.

## BC strategy (S03.M7)

v1.1.1 reads v1.1.0 cache files via `ReadCache()`. If the signature fails
verification, `IsZeroPubKey()` is true in dev builds and the CLI falls back to
bare (unverified) mode. In production builds the failed-verify triggers a
fail-open re-fetch from `ping.nself.org` — the cache is regenerated transparently.
No user action required. This is the canonical "regen on bad sig" strategy.
