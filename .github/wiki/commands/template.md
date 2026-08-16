# nself template

> Browse, install, and publish full-stack app templates from the ɳSelf Template Marketplace.

## Synopsis

```
nself template <subcommand> [flags]
nself init --template <slug> [dest-dir]
```

## Description

Templates are full-stack app packages that include a Postgres schema, Hasura metadata, seed data, and an optional Flutter starter. They follow the same manifest format as `nself init` language scaffolds but are community-authored and distributed through the ɳSelf Template Marketplace at `nself.org/templates`.

Use `nself template list` to browse from the CLI, `nself template info <slug>` to inspect a specific template, and `nself init --template <slug>` to install one into a new project directory.

Authors publish via `nself template publish` and receive 80% revenue share on paid templates (requires a KYC-approved author account, see [B55 plugin-author-revenue-share](https://nself.org/docs/developers/revenue-share)).

## Subcommands

| Subcommand | Description |
|---|---|
| `list` | List all templates, with optional category and price filters |
| `info <slug>` | Show detail for a single template including required plugins |
| `publish` | Validate and submit a template to the registry |
| `update` | Apply incremental schema migrations for the installed template |

## Bundled Clone Templates

Six clone templates are embedded directly in the CLI binary. They work without any network access and are available on every install regardless of connectivity or license tier.

| Template | Version | Required plugins | Description |
|---|---|---|---|
| `airbnb-clone` | 1.0.0 | `auth`, `notify`, `photos` | Property rental marketplace with listings, bookings, and reviews |
| `discord-clone` | 1.0.0 | `chat`, `realtime`, `auth`, `moderation` | Real-time messaging platform with servers, channels, and roles |
| `notion-clone` | 1.0.0 | `cms`, `auth`, `realtime` | Collaborative note-taking with workspaces, pages, and blocks |
| `slack-clone` | 1.0.0 | `chat`, `livekit`, `realtime`, `auth`, `notify` | Team messaging with threads, reactions, and voice/video |
| `substack-clone` | 1.0.0 | `cms`, `notify`, `auth` | Newsletter platform with subscriber tiers and post publishing |
| `zoom-clone` | 1.0.0 | `livekit`, `recording`, `auth`, `notify` | Video meeting platform with lobby, recording, and participant management |

All bundled templates are free (no license check). Scaffold one with:

```bash
nself init --template <name> [dest-dir]
```

See [[nself init]] for the `--no-seed` and `--dry-run` flags.

## nself template list

```
nself template list [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--category` | | Filter by category: saas, marketplace, social, productivity, media, ecommerce |
| `--free` | false | Show free templates only |
| `--sort` | installs | Sort by: installs, rating, newest, price |
| `--json` | false | Output raw JSON |

**Examples:**

```bash
nself template list
nself template list --category saas
nself template list --free --sort rating
nself template list --json
```

## nself template info

```
nself template info <slug> [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--json` | false | Output raw JSON |

**Examples:**

```bash
nself template info saas-starter
nself template info media-server --json
```

## nself init --template (marketplace)

When the slug passed to `--template` is not a built-in language scaffold (express, fastapi, go, rust), it is resolved against the marketplace registry, downloaded, its SHA256 checksum verified, and extracted into the destination directory.

```bash
nself init --template <slug> [dest-dir]
```

| Flag | Default | Description |
|---|---|---|
| `--force` | false | Overwrite destination directory if non-empty |
| `--quiet` | false | Suppress output messages |

**Examples:**

```bash
nself init --template saas-starter ./my-saas
nself init --template media-server /srv/media --force
```

Paid templates require an active license key:

```bash
nself license set nself_pro_<your-key>
nself init --template marketplace-starter ./shop
```

After installation the CLI prints the required plugins list. Install them with:

```bash
nself plugin install auth notify cms
nself start
```

## nself template publish

```
nself template publish [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--tarball` | | Path to the compiled template archive (.tar.gz) , required |
| `--manifest` | template.yml | Path to the template manifest file |

The command validates the `template.yml` manifest, computes the SHA256 of the tarball, and prints submission instructions. Final upload and review happen through the author portal at `nself.org/developers/templates`.

**Manifest format (template.yml):**

```yaml
slug: saas-starter
display_name: SaaS Starter
description: Multi-tenant SaaS with billing, auth, and dashboard
version: "1.2.0"
price_usd: 0
category: saas
required_plugins:
  - auth
  - notify
  - cms
preview_url: https://saas-starter-demo.nself.org
cli_min_version: "1.0.9"
```

**Example:**

```bash
tar -czf dist/my-template.tar.gz schema/ metadata/ seed/ flutter/
nself template publish --tarball dist/my-template.tar.gz --manifest template.yml
```

## nself template update

```
nself template update [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--force` | false | Allow destructive migrations (DROP, TRUNCATE) , requires confirmation |
| `--dry-run` | false | Print pending migrations without applying them |

Runs incremental `.sql` files from the project's `migrations/` directory. Only additive changes are applied without flags. Destructive migrations require `--force` plus an explicit prompt.

**Examples:**

```bash
nself template update --dry-run
nself template update
nself template update --force
```

## Environment variables

| Variable | Description |
|---|---|
| `NSELF_TEMPLATE_REGISTRY_URL` | Override the default template registry URL (`https://nself.org/api/templates`) |
| `NSELF_LICENSE_KEY` | License key , required for paid template downloads |

## Related

- [[nself init]], Project initialisation including built-in language scaffolds
- [[nself plugin]], Plugin management
- [Template Marketplace](https://nself.org/templates), Browse templates in the browser
- [Publish a template](https://nself.org/docs/templates/publish-template), Author guide

[[Home]]
