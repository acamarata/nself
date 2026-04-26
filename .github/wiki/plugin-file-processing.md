# File Processing Plugin

> FFmpeg-based thumbnail generation for files across multiple storage providers. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install file-processing
```

## What It Does

Generates thumbnails for images and videos stored in MinIO, S3, GCS, Cloudflare R2, Backblaze B2, or Azure Blob Storage. Processes assets automatically via event triggers on upload, writing thumbnail outputs back to the same storage provider. Built in Rust for low overhead and high throughput.

## Dependencies

Requires `minio` or a compatible storage provider configured via `FILE_PROCESSING_STORAGE_PROVIDER`.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `FILE_PROCESSING_PORT` | `3089` | File processing service port |
| `FILE_PROCESSING_STORAGE_PROVIDER` | `minio` | Storage backend: `minio`, `s3`, `r2`, `b2`, `gcs`, or `azure` |
| `FILE_PROCESSING_THUMB_SIZES` | `200x200,400x400` | Comma-separated list of thumbnail dimensions to generate |

## Ports

| Port | Purpose |
|------|---------|
| 3089 | File processing REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_file_processing_jobs`, thumbnail generation job records
- `np_file_processing_outputs`, generated thumbnail output records
- `np_file_processing_errors`, processing error log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/file-processing/` | File processing API |
