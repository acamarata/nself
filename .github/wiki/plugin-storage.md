# Storage Plugin

> S3-compatible file storage with tenant isolation: PUT/GET/DELETE/LIST objects, bucket management, object metadata, and SSRF-guarded backend config. **Pro plugin.**

> **Requires:** Pro license tier. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install storage
```

Verify the plugin is running:

```bash
nself plugin status storage
curl http://localhost:9007/health
```

## What It Does

Provides core file storage for nSelf deployments: create buckets, upload objects (PUT), retrieve objects (GET), delete objects (DELETE), and list bucket contents (LIST). Runs on port 9007 and connects to your nSelf Postgres database. The `storage` plugin is the system-level file store — for multi-provider S3/MinIO/R2/GCS workflows, use the `object-storage` plugin instead.

Key features:

- S3-compatible REST API (PUT/GET/DELETE/LIST objects and bucket CRUD)
- Tenant isolation via `source_account_id` on all database tables and API queries
- SSRF guard: S3 endpoint is read from operator env only — never from user request data; RFC1918/loopback/link-local addresses blocked at startup
- Path traversal protection on all object keys (`..` and absolute paths rejected)
- Soft-delete for objects (lifecycle rules retain history)
- Automatic bucket counter sync (used_bytes, object_count) on every write
- DB schema: `np_storage_buckets`, `np_storage_objects`, `np_storage_metadata`

## Port

| Port | Purpose |
|------|---------|
| 9007 | HTTP API (health + REST) |

## Configuration

All config is set via environment variables. The S3 endpoint (if used) is **operator-only** — it cannot be overridden per HTTP request (SSRF protection).

| Env Var | Default | Description |
|---------|---------|-------------|
| `STORAGE_PORT` | `9007` | HTTP listen port |
| `DATABASE_URL` | *(required)* | Postgres connection string |
| `STORAGE_DEFAULT_BACKEND` | `local` | Default storage backend: `local` or `s3` |
| `STORAGE_BASE_PATH` | `/data/storage` | Local filesystem base path for stored files |
| `STORAGE_S3_ENDPOINT` | *(none)* | S3-compatible endpoint URL (HTTPS only; SSRF-validated at startup) |
| `STORAGE_S3_REGION` | `us-east-1` | AWS/S3 region |
| `STORAGE_S3_ACCESS_KEY` | *(none)* | S3 access key |
| `STORAGE_S3_SECRET_KEY` | *(none)* | S3 secret key |
| `STORAGE_S3_BUCKET_PREFIX` | *(none)* | Optional prefix for all S3 bucket names |
| `STORAGE_API_KEY` | *(none)* | Optional API key for bearer auth |
| `STORAGE_MAX_OBJECT_BYTES` | `1073741824` | Max object size in bytes (1 GiB default) |
| `STORAGE_RATE_LIMIT_MAX` | *(none)* | Max requests per window (optional rate limiting) |
| `STORAGE_RATE_LIMIT_WINDOW_MS` | *(none)* | Rate limit window in milliseconds |

## API Reference

All endpoints require the `X-Source-Account` header for tenant isolation (defaults to `primary` if omitted). Set to the account/tenant identifier managed by nSelf auth.

### Health

```
GET /health
```

Returns plugin status and DB reachability.

```json
{"status": "ok", "plugin": "storage", "port": "9007"}
```

### Buckets

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/buckets` | List all buckets for current account |
| `POST` | `/api/v1/buckets` | Create a new bucket |
| `GET` | `/api/v1/buckets/{id}` | Get bucket by ID |
| `PUT` | `/api/v1/buckets/{id}` | Update bucket settings |
| `DELETE` | `/api/v1/buckets/{id}` | Delete bucket (cascades to objects) |

**Create bucket:**

```bash
curl -X POST http://localhost:9007/api/v1/buckets \
  -H "Content-Type: application/json" \
  -H "X-Source-Account: primary" \
  -d '{"name": "uploads", "backend": "local", "public_read": false}'
```

**Bucket object:**

```json
{
  "id": "a1b2c3d4-...",
  "source_account_id": "primary",
  "name": "uploads",
  "backend": "local",
  "backend_config": {},
  "public_read": false,
  "max_object_bytes": null,
  "allowed_mime_types": [],
  "quota_bytes": null,
  "used_bytes": 0,
  "object_count": 0,
  "created_at": "2026-06-21T00:00:00Z",
  "updated_at": "2026-06-21T00:00:00Z"
}
```

### Objects

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/buckets/{id}/objects` | List objects in bucket |
| `PUT` | `/api/v1/buckets/{id}/objects/{key}` | Create or replace an object |
| `GET` | `/api/v1/buckets/{id}/objects/{key}` | Get object metadata |
| `DELETE` | `/api/v1/buckets/{id}/objects/{key}` | Soft-delete object |

Object keys are path-like strings (e.g. `images/avatar.jpg`). Rules:

- Must not start with `/`
- Must not contain `..` (path traversal blocked)
- Must not contain null bytes

**Put object:**

```bash
curl -X PUT "http://localhost:9007/api/v1/buckets/<bucket-id>/objects/images/photo.jpg" \
  -H "Content-Type: application/json" \
  -H "X-Source-Account: primary" \
  -d '{
    "content_type": "image/jpeg",
    "size_bytes": 204800,
    "etag": "abc123",
    "storage_path": "local://data/storage/primary/images/photo.jpg"
  }'
```

**Object response:**

```json
{
  "id": "e5f6g7h8-...",
  "source_account_id": "primary",
  "bucket_id": "a1b2c3d4-...",
  "key": "images/photo.jpg",
  "content_type": "image/jpeg",
  "size_bytes": 204800,
  "etag": "abc123",
  "is_public": false,
  "version": 1,
  "created_at": "2026-06-21T00:00:00Z",
  "updated_at": "2026-06-21T00:00:00Z"
}
```

## Database Schema

Three tables under the `np_storage_*` prefix:

```sql
-- Buckets: one per logical storage container
np_storage_buckets (
  id, source_account_id, name, backend, backend_config,
  public_read, max_object_bytes, allowed_mime_types,
  quota_bytes, used_bytes, object_count,
  created_at, updated_at
)

-- Objects: one per stored file; keys are unique per bucket
np_storage_objects (
  id, source_account_id, bucket_id, key, content_type,
  size_bytes, etag, checksum_sha256, storage_path,
  is_public, version, deleted_at, created_at, updated_at
)

-- Metadata: arbitrary key-value pairs per object
np_storage_metadata (
  id, source_account_id, object_id, key, value,
  created_at, updated_at
)
```

Migrations are in `plugins-pro/paid/storage/migrations/`:

- `001_up.sql` — create tables + triggers
- `001_down.sql` — drop tables + triggers

Run migrations:

```bash
# Apply
psql "$DATABASE_URL" -f plugins-pro/paid/storage/migrations/001_up.sql

# Rollback
psql "$DATABASE_URL" -f plugins-pro/paid/storage/migrations/001_down.sql
```

## Hasura Row-Level Security

Tenant row-filters are defined in `migrations/hasura_rls.yaml`. Apply via Hasura Console or metadata API. All three tables enforce `source_account_id = X-Hasura-Source-Account`.

## Security

| Concern | Mitigation |
|---------|-----------|
| SSRF via S3 endpoint | `STORAGE_S3_ENDPOINT` is operator env only; RFC1918/loopback/link-local rejected at startup |
| Path traversal on object keys | Keys are rejected if they contain `..`, start with `/`, or contain null bytes |
| Tenant data leakage | All SQL queries scope to `source_account_id`; bucket ownership verified before object writes |
| Auth | Bearer token via `STORAGE_API_KEY`; set by nSelf auth middleware |

## Troubleshooting

**Plugin won't start — SSRF error:**

```
config: STORAGE_S3_ENDPOINT SSRF guard: scheme must be https
```

The endpoint must be HTTPS and must not resolve to an internal IP. Check `STORAGE_S3_ENDPOINT`.

**404 on object GET:**

Object may be soft-deleted (`deleted_at` is set). Use `GET /api/v1/buckets/{id}/objects` to list — soft-deleted objects are excluded from listing. Restore by re-uploading with `PUT`.

**Bucket not found on object write:**

The bucket `id` must belong to the same `X-Source-Account` as the object write. Cross-tenant bucket writes return 404, not 403, to avoid enumeration.

## Docker

```bash
docker run -d \
  --name nself-storage \
  -p 9007:9007 \
  -e DATABASE_URL="postgres://..." \
  -e STORAGE_DEFAULT_BACKEND=local \
  -e STORAGE_BASE_PATH=/data/storage \
  -v /your/data:/data/storage \
  nself/plugin-storage:latest
```

The image includes a HEALTHCHECK at `/health` with 30s interval, 5s timeout, 10s start delay.

## Related Plugins

| Plugin | Description |
|--------|-------------|
| `object-storage` | Multi-provider S3/MinIO/R2/GCS/Azure/B2 with multipart uploads and presigned URLs |
| `storage-transform` | On-the-fly image transformation (resize, crop, WebP/AVIF) |
| `cdn` | CDN integration for serving stored objects (planned) |
