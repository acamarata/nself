# Photos Plugin

> Photo album management with EXIF extraction, tagging, and MinIO-backed storage. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install photos
```

## What It Does

Provides photo album management with automatic EXIF metadata extraction, flexible tagging, and webhook events fired on upload. Stores originals in MinIO and generates configurable thumbnail sizes automatically. Albums and photos are queryable via the Hasura GraphQL API.

## Dependencies

Requires the `minio` plugin.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PHOTOS_PORT` | `3091` | Photos service port |
| `PHOTOS_STORAGE_BUCKET` | — | MinIO bucket name for photo storage |
| `PHOTOS_THUMB_SIZES` | `200x200,800x800` | Comma-separated thumbnail dimensions to generate |

## Ports

| Port | Purpose |
|------|---------|
| 3091 | Photos REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_photos_albums` — album records
- `np_photos_photos` — photo records with storage references
- `np_photos_tags` — tag definitions
- `np_photos_exif` — extracted EXIF metadata per photo

## Nginx Routes

| Route | Target |
|-------|--------|
| `/photos/` | Photos API |
