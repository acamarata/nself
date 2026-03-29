> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Watermark Plugin

> Image and video watermarking — text and image overlay with batch processing. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install watermark
```

## What It Does

Applies text and image watermarks to photos and videos. Supports configurable position (corner, center, tiled), opacity, and scaling. Batch processes entire MinIO buckets or applies watermarks on demand during media delivery. Works alongside the `thumb` and `transcoder` plugins in media pipelines.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `WATERMARK_PORT` | `3059` | Watermark service port |
| `WATERMARK_DEFAULT_POSITION` | `bottom-right` | Default watermark position |
| `WATERMARK_DEFAULT_OPACITY` | `0.7` | Default opacity (0-1) |
| `WATERMARK_LOGO_PATH` | — | Path to default watermark image |

## Ports

| Port | Purpose |
|------|---------|
| 3059 | Watermark REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_watermark_presets` — watermark style presets
- `np_watermark_jobs` — batch processing job queue
- `np_watermark_log` — watermark application log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/watermark/` | Watermark API |
