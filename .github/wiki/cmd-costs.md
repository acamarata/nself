# nself costs

<!-- BEGIN PROSE:summary -->
> Show an itemized breakdown of estimated monthly operational costs for your nSelf install.
<!-- END PROSE:summary -->

## Synopsis

```
nself costs [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself costs` calculates and displays an estimated monthly cost breakdown for a self-hosted nSelf deployment. It covers infrastructure, DNS/CDN, frontend hosting, payment processing, and plugin license fees.

The command detects your Hetzner server type from the `HETZNER_SERVER_TYPE` environment variable. If the variable is not set and no `--server-type` flag is provided, it falls back to `cx23` (the nSelf reference deployment default). The VPS cost table covers the full CX, CAX, and CCX series plus the CX23 reference instance. If an unrecognized server type is supplied, the VPS line shows `$0.00` with a note to check hetzner.com/cloud.

Cloudflare (DNS/DDoS/CDN) and Vercel (frontend hosting) are listed at $0.00 reflecting the free tier used by most self-hosted installs. Stripe payment processing fees are shown structurally (2.9% + $0.30 per card transaction; 0.5% + $0.15 ACH) plus a concrete example for 100 x $0.99 transactions. The example row is illustrative and excluded from the running total.

Installed paid plugin bundles are detected via the local plugin directory. Each unique bundle is counted once at $0.99/month. The final TOTAL line sums VPS cost plus paid bundle fees and excludes the volume-dependent Stripe example.

Output defaults to a tab-aligned table. Use `--format json` for machine-readable output.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--format`, `-f` | `table` | Output format: table\|json |
| `--server-type` | `""` | Hetzner server type (e.g. cx23, cax11). Auto-detected from HETZNER_SERVER_TYPE env var if unset. |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Auto-detect server type from environment
nself costs
```

```bash
# Specify a server type explicitly
nself costs --server-type cx23
```

```bash
# Use a dedicated vCPU instance
nself costs --server-type ccx13
```

```bash
# Output as JSON for scripting
nself costs --format json
nself costs -f json
```

```bash
# Combine server type and format flags
nself costs --server-type cax21 --format json
```

```bash
# Redirect table output to a file
nself costs > monthly-costs.txt
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-plugin.md](cmd-plugin.md) — install and manage plugins (affects bundle license cost lines)
- [cmd-license.md](cmd-license.md) — manage your nSelf license key
- [cmd-status.md](cmd-status.md) — view the running state of your nSelf install
- [cmd-doctor.md](cmd-doctor.md) — verify environment and configuration health
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
