# Thumb Plugin

> On-demand image thumbnail generation — resize, crop, and format conversion. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install thumb
```

## What It Does

Generates image thumbnails on demand via URL parameters. Resize, crop, convert format (WebP, JPEG, PNG, AVIF), and apply quality settings without pre-generating variants. Results are cached to MinIO or local storage. Maps to `file-processing` in the plugin registry (port 3089, 3 tables).

## Dependencies

Requires MinIO (`MINIO_ENABLED=true`).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `THUMB_PORT` | `3089` | Thumb service port |
| `THUMB_CACHE_TTL` | `86400` | Cached thumbnail TTL in seconds |
| `THUMB_MAX_WIDTH` | `4096` | Maximum output width |
| `THUMB_MAX_HEIGHT` | `4096` | Maximum output height |
| `THUMB_DEFAULT_FORMAT` | `webp` | Default output format |
| `THUMB_QUALITY` | `85` | Default JPEG/WebP quality (0-100) |

## Ports

| Port | Purpose |
|------|---------|
| 3089 | Thumb service REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_thumb_requests` — thumbnail request log
- `np_thumb_cache` — cached thumbnail references
- `np_thumb_presets` — named size/format presets

## Nginx Routes

| Route | Target |
|-------|--------|
| `/thumb/` | On-demand image processing |

## URL Format

```
/thumb/{width}x{height}/{format}/{source_path}
/thumb/preset/{preset_name}/{source_path}
```

Example: `/thumb/300x300/webp/uploads/photo.jpg`
