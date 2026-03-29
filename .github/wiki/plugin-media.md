# Media Plugin

> Media library with MinIO storage, image resizing, and video transcoding integration. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install media
```

## What It Does

Provides a managed media library backed by MinIO. Upload images and videos, generate thumbnails on demand, extract EXIF metadata, and manage file organization. Integrates with the `cms` and `blog` plugins as their media backend. Images are served with on-demand resizing via URL parameters.

## Dependencies

Requires MinIO (`MINIO_ENABLED=true`).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MEDIA_PORT` | `3047` | Media service port |
| `MEDIA_MAX_FILE_SIZE` | `100MB` | Maximum upload file size |
| `MEDIA_IMAGE_FORMATS` | `webp,jpeg,png` | Supported output formats |
| `MEDIA_THUMB_SIZES` | `150,300,600` | Auto-generated thumbnail widths |
| `MEDIA_CDN_URL` | — | CDN base URL for asset delivery |

## Ports

| Port | Purpose |
|------|---------|
| 3047 | Media library REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_media_files` — file metadata and storage references
- `np_media_folders` — folder organization
- `np_media_variants` — generated image variants
- `np_media_usage` — which content uses which media

## Nginx Routes

| Route | Target |
|-------|--------|
| `/media/` | Media management API |
| `/media/upload` | Upload endpoint |
| `/media/resize` | On-demand image resizing |
