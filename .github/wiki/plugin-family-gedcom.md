# Plugin: family-gedcom

**Status: PLANNED — not yet installable**

Generic GEDCOM file importer for the ɳFamily plugin. Accepts any GEDCOM 5.5.1 or 7.0 file exported from popular genealogy apps (Ancestry, FamilySearch, MyHeritage, MacFamilyTree, etc.) and imports the data into the ɳFamily database schema.

## Why it's planned, not available

`family-gedcom` requires the `family` plugin as a hard dependency. The `family` plugin is part of the **ɳFamily bundle** (paid, $0.99/mo), which ships in v1.1.0. Until the `family` plugin is available:

- `nself plugin install family-gedcom` returns a clear error explaining the dependency
- The plugin is registered in the free registry to reserve the name and document the integration pattern
- No data will be lost or left in a partial state — the install gate is checked before any migration runs

## Planned capabilities

When available, `family-gedcom` will:

- Parse GEDCOM 5.5.1 and 7.0 files with full UTF-8 support
- Import individuals, families, events, and relationships into `np_family_*` tables
- Accept an optional photo folder and upload images to the `object-storage` plugin
- Register uploaded photos against individuals via the `photos` plugin
- Support incremental re-import (idempotent — re-importing the same GEDCOM updates rather than duplicates)
- Run as a one-shot import job, not a persistent service

## Dependencies

| Plugin | Bundle | Required | Notes |
|---|---|---|---|
| `family` | ɳFamily ($0.99/mo) | Hard | Provides `np_family_*` tables and Hasura schema |
| `object-storage` | Free | Hard | Required for photo upload |
| `photos` | Free | Hard | Required to link imported photos to individuals |

## Planned tables

These tables are defined by the `family` plugin. `family-gedcom` writes to them during import:

- `np_family_individuals` — persons in the tree
- `np_family_families` — family units (parent-child, spouse relationships)
- `np_family_events` — birth, death, marriage, and other life events
- `np_family_relationships` — explicit relationship links

## Planned env vars

| Env var | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `FAMILY_GEDCOM_BATCH_SIZE` | No | Records per transaction (default: 100) |
| `FAMILY_GEDCOM_SKIP_PHOTOS` | No | Set to `true` to skip photo upload |

## When will it be available?

The ɳFamily bundle is planned for v1.1.0. Once the `family` plugin ships:

1. `family-gedcom` will be updated to `status: stable` and `installable: true`
2. This wiki page will be updated with full usage instructions
3. The registry entry will include working install commands

## Importing GEDCOM files (planned usage)

```bash
# Install (once family plugin ships)
nself plugin install family-gedcom

# Run import
nself run family-gedcom import --file my-tree.ged

# Import with photos
nself run family-gedcom import --file my-tree.ged --photos-dir ./family-photos/
```

## See also

- [ɳFamily bundle](https://nself.org/plugins/family) — the paid bundle that unlocks this plugin
- [plugin-photos](plugin-photos.md) — photo management plugin (free)
- [plugin-object-storage](plugin-object-storage.md) — S3-compatible storage plugin (free)
