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
| `OBJECT_STORAGE_PORT` | `3301` | Port the Object Storage plugin service listens on |
| `OBJECT_STORAGE_PROVIDER` | — | Storage provider: `minio`, `s3`, `r2`, `b2`, `gcs`, `azure` |
| `OBJECT_STORAGE_BUCKET` | — | Default bucket name |
| `OBJECT_STORAGE_REGION` | — | Storage region (provider-dependent) |

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
