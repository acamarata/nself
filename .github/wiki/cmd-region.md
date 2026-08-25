# nself region

**This command moved to a plugin.**

`nself region` is no longer part of the CLI core. Multi-region replication is
not part of the single-instance backend lifecycle the core covers, so it ships
as a plugin.

## Install

```bash
nself install region
```

Once installed, `nself region ...` works exactly as before — the CLI proxies the
command to the plugin.

## Commands

```bash
nself region add --region hel1 --pg-url postgres://...
nself region list
nself region status
nself region promote --region hel1
```

## Requirements

Multi-region is gated behind the `multi_region_enabled` feature flag, which the
plugin reads from the feature-flags plugin. Enable it with:

```bash
nself flags set multi_region_enabled true
```

---

← [[Commands]] · [[Plugin-Overview]] · [[Home]]
