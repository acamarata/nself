# nself bundle

**Inspect and manage nSelf plugin bundles.**

Bundles group related pro plugins into a single purchase. Each paid bundle costs
$0.99/mo or $9.99/yr. ɳSelf+ ($3.99/mo or $39.99/yr) includes all six paid bundles.

Activate a bundle by setting a license key:

```bash
nself license set <key>
```

---

## nself bundle list

List all available bundles with pricing and plugin counts.

```
nself bundle list [--installed] [--json]
```

### Flags

| Flag | Description |
|------|-------------|
| `--installed` | Show only bundles with at least one active plugin on this machine |
| `--json` | Output a machine-readable JSON array (used by the admin/ marketplace UI) |

### Examples

```bash
# Show all bundles
nself bundle list

# Show only bundles with plugins already installed
nself bundle list --installed

# Machine-readable output for scripting
nself bundle list --json
```

### Output columns (table mode)

| Column | Description |
|--------|-------------|
| Slug | CLI identifier used in `bundle info / install / remove` |
| Name | Display name (ɳClaw, ɳChat, etc.) |
| Price | $0.99/mo or $9.99/yr (paid); FREE (ɳTask); $3.99/mo (ɳSelf+) |
| Plugin count | Number of plugins in the bundle |

---

## nself bundle info \<name\>

Show full details for one bundle: plugin membership, pricing, and your current
license status.

```
nself bundle info <name>
```

### Arguments

| Argument | Description |
|----------|-------------|
| `name` | Bundle slug (nclaw, nchat, ntv, nfamily, clawde, nsentry, ntask, nself-plus) |

### Examples

```bash
nself bundle info nclaw
nself bundle info nsentry
nself bundle info nself-plus
```

---

## nself bundle install \<name\>

Install every plugin in a bundle in a single transaction. License validation runs
for each plugin before any filesystem change. On failure, all plugins installed in
this invocation are rolled back.

```
nself bundle install <name> [--dry-run] [--force] [--strict] [--channel <channel>]
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Print planned actions without installing |
| `--force` | Bypass license validation (logged as license-bypass) |
| `--strict` | Fail if any plugin in the bundle is missing from the registry |
| `--channel` | Release channel: stable (default), beta, or canary |

### Examples

```bash
nself bundle install nclaw
nself bundle install nsentry --channel beta
nself bundle install nchat --dry-run
```

---

## nself bundle remove \<name\>

Remove every plugin in a bundle. Plugin data (schema/tables) is dropped by
default; pass `--keep-data` to preserve it for later reinstall.

```
nself bundle remove <name> [--dry-run] [--keep-data]
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Print planned actions without removing |
| `--keep-data` | Preserve plugin schema and tables on remove |

### Examples

```bash
nself bundle remove nclaw
nself bundle remove ntv --keep-data
nself bundle remove nchat --dry-run
```

---

## Bundle catalog

| Slug | Display name | Price | Status |
|------|-------------|-------|--------|
| `nclaw` | ɳClaw | $0.99/mo or $9.99/yr | active |
| `nchat` | ɳChat | $0.99/mo or $9.99/yr | active |
| `ntv` | ɳTV | $0.99/mo or $9.99/yr | active |
| `nfamily` | ɳFamily | $0.99/mo or $9.99/yr | planned v1.1.0 |
| `clawde` | ClawDE | $0.99/mo or $9.99/yr | planned v1.1.0 |
| `nsentry` | ɳSentry | $0.99/mo or $9.99/yr | planned v1.1.0 |
| `ntask` | ɳTask | FREE | active |
| `nself-plus` | ɳSelf+ | $3.99/mo or $39.99/yr | active |

Buy or manage subscriptions at: https://nself.org/pricing

---

[[Commands]] | [[Plugin-Overview]] | [[Home]]
