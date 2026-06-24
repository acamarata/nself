# Object Storage Plugin

> Multi-provider S3-compatible object storage API with presigned URL generation, multipart uploads, and access logging. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install object-storage
```

## What It Does

Provides a unified S3-compatible object storage API across multiple providers including MinIO, AWS S3, Cloudflare R2, Backblaze B2, Google Cloud Storage, and Azure Blob Storage. Supports presigned URL generation for client-side uploads and downloads, multipart upload orchestration for large files, and full access logging to Postgres. Switch providers by changing a single env var with no application code changes.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `OS_PLUGIN_PORT` | `3301` | Port the Object Storage plugin service listens on |
| `OS_DEFAULT_PROVIDER` | `local` | Storage provider: `minio`, `s3`, `r2`, `b2`, `gcs`, `azure`, `local` |
| `OS_S3_ENDPOINT` | — | S3-compatible endpoint URL (MinIO, R2, etc.) |
| `OS_S3_ACCESS_KEY` | — | S3 access key |
| `OS_S3_SECRET_KEY` | — | S3 secret key |
| `OS_STORAGE_BASE_PATH` | `/data/object-storage` | Local storage path for the `local` provider |
| `OS_API_KEY` | — | Plugin API authentication key |
| `OS_MAX_UPLOAD_SIZE` | `1073741824` | Max upload size in bytes (1 GB default) |
| `DATABASE_URL` | — | Postgres connection string for audit logs |

## Ports

| Port | Purpose |
|------|---------|
| `3301` | Object Storage plugin HTTP service |

## Database Tables

5 tables added to your Postgres database.

- `np_object_storage_buckets`, Bucket metadata and configuration
- `np_object_storage_objects`, Object index and metadata
- `np_object_storage_presigned`, Presigned URL issuance log
- `np_object_storage_multipart`, In-progress and completed multipart uploads
- `np_object_storage_access_log`, Object access and operation audit log

## Nginx Routes

| Route | Description |
|-------|-------------|
| `/storage/` | Proxied to Object Storage plugin service on port 3301 |

## Presigned URLs

Generate temporary, expiring URLs for client-side uploads and downloads without exposing credentials. Useful for direct browser-to-storage transfers, mobile apps, and CDN offloading. The plugin tracks every issued URL in `np_object_storage_presigned` for audit purposes.

```bash
# Example: request a presigned upload URL via the plugin API
POST /storage/buckets/{bucket}/presign
{"key": "uploads/photo.jpg", "expires_in": 3600, "operation": "put"}
```

## Multipart Uploads

Files larger than the configured part size are split into chunks and uploaded in parallel parts. The plugin manages part tracking in `np_object_storage_multipart` and assembles them on the provider side. All S3-compatible providers support multipart upload; local storage also supports it via the plugin's chunked write implementation.

## Provider Configuration

Each provider needs its own credentials. Set `OBJECT_STORAGE_PROVIDER` to one of:

| Provider | Value | Notes |
|----------|-------|-------|
| Local filesystem | `local` | Default. No external credentials needed. |
| AWS S3 | `s3` | Set region, access key, secret key. |
| MinIO | `minio` | Self-hosted S3-compatible. Provide endpoint URL. |
| Cloudflare R2 | `r2` | No egress fees. Uses S3-compatible API. |
| Backblaze B2 | `b2` | Low-cost cold storage. S3-compatible API. |
| Google Cloud Storage | `gcs` | Set project and HMAC credentials. |
| Azure Blob Storage | `azure` | Set account name and key. |

## Not the Same As

- **ɳTV media plugins** handle video/audio stream processing and IPTV playlist management. Use object-storage if you want raw file storage independent of media playback.
- **Storage-transform plugins** (J-section) apply image/video transformations on read or write. Object-storage is the backing store; pair it with a transform plugin when you need on-the-fly resizing or transcoding.

---

[[Plugin-Overview]] | [[Home]]
