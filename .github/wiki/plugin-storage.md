# plugin-storage

Core file storage plugin for nSelf — bucket management, object CRUD, and tenant-isolated S3-compatible API.

**Port:** 9007  
**Bundle:** none (core system plugin, requires pro license)  
**Language:** Go  
**Tables:** `np_storage_buckets`, `np_storage_objects`

---

## Overview

The `storage` plugin provides a multi-tenant bucket and object storage API backed by your Postgres database. Objects are stored by reference (metadata in DB); backend storage is pluggable (local filesystem or S3-compatible endpoint).

All routes are scoped to `source_account_id`, enforcing strict tenant isolation. The API uses bearer token authentication — every `/api/v1` request must include the `STORAGE_API_KEY` token.

---

## Install

```bash
nself plugin install storage
```

---

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `STORAGE_API_KEY` | Yes | — | Bearer token for all `/api/v1` routes |
| `STORAGE_PLUGIN_PORT` | No | `9007` | TCP port to listen on |
| `STORAGE_MAX_UPLOAD_BYTES` | No | `104857600` (100 MiB) | Maximum upload body size |
| `STORAGE_BASE_PATH` | No | `/data/storage` | Filesystem path for local-mode storage |
| `STORAGE_S3_ENDPOINT` | No | — | S3-compatible endpoint URL (config-only, not per-request) |
| `STORAGE_S3_REGION` | No | `us-east-1` | S3 region |
| `STORAGE_S3_ACCESS_KEY` | No | — | S3 access key |
| `STORAGE_S3_SECRET_KEY` | No | — | S3 secret key |
| `STORAGE_S3_BUCKET` | No | — | Default S3 bucket name |

**Security note:** `STORAGE_S3_ENDPOINT` is read from config only — users cannot override it per request. This prevents server-side request forgery (SSRF).

---

## Authentication

All `/api/v1` routes require a valid bearer token:

```bash
curl -H "Authorization: Bearer $STORAGE_API_KEY" http://localhost:9007/api/v1/buckets
# Or via header:
curl -H "X-API-Key: $STORAGE_API_KEY" http://localhost:9007/api/v1/buckets
```

The `/health` endpoint is exempt from auth (used by Docker HEALTHCHECK and `nself doctor`).

---

## Tenant Isolation

Each request is scoped to a source account via the `X-Source-Account` header (default: `primary`). The API enforces isolation at the database level:

- Every query filters by `source_account_id`
- Bucket ownership is verified before any object read or write
- `ON CONFLICT` upserts are scoped to `(source_account_id, bucket_id, key)` so one tenant can never overwrite another tenant's objects

---

## API Reference

### Health

```
GET /health
```

Returns plugin status and DB connectivity. No auth required.

**Response (200):**
```json
{"status": "ok", "plugin": "storage", "port": "9007"}
```

**Response (503):**
```json
{"error": "database unavailable"}
```

---

### Buckets

#### List buckets

```
GET /api/v1/buckets
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "source_account_id": "primary",
    "name": "uploads",
    "provider": "local",
    "public": false,
    "max_file_size_bytes": null,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

#### Create bucket

```
POST /api/v1/buckets
Content-Type: application/json

{
  "name": "uploads",
  "provider": "local",
  "public": false,
  "max_file_size_bytes": 10485760
}
```

**Response (201):**
```json
{"id": "uuid"}
```

#### Get bucket

```
GET /api/v1/buckets/{id}
```

#### Update bucket

```
PUT /api/v1/buckets/{id}
Content-Type: application/json

{
  "public": true,
  "max_file_size_bytes": 52428800
}
```

#### Delete bucket

```
DELETE /api/v1/buckets/{id}
```

Cascades to all objects in the bucket.

---

### Objects

#### List objects

```
GET /api/v1/buckets/{id}/objects
```

**Response (200):**
```json
[
  {
    "id": "uuid",
    "source_account_id": "primary",
    "bucket_id": "uuid",
    "key": "images/photo.jpg",
    "content_type": "image/jpeg",
    "size_bytes": 204800,
    "etag": "abc123",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

#### Upload object (PUT)

```
PUT /api/v1/buckets/{id}/objects/{key}
Content-Type: image/jpeg
<binary body>
```

Object keys support path separators (e.g. `images/2024/photo.jpg`). Keys with `..` are rejected. Upload body is capped at `STORAGE_MAX_UPLOAD_BYTES`.

**Response (200):**
```json
{
  "id": "uuid",
  "key": "images/photo.jpg",
  "size_bytes": 204800,
  "etag": "abc123",
  "content_type": "image/jpeg",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Get object metadata

```
GET /api/v1/buckets/{id}/objects/{key}
```

Returns object metadata (not binary content).

#### Delete object

```
DELETE /api/v1/buckets/{id}/objects/{key}
```

---

## Database Schema

### np_storage_buckets

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `source_account_id` | TEXT | Tenant isolation (default `primary`) |
| `name` | TEXT | Bucket name, unique per account |
| `provider` | TEXT | Storage backend (`local`, `s3`) |
| `public` | BOOLEAN | Whether objects are publicly readable |
| `max_file_size_bytes` | BIGINT | Per-bucket upload limit (optional) |
| `created_at` | TIMESTAMPTZ | Created timestamp |
| `updated_at` | TIMESTAMPTZ | Updated timestamp |

### np_storage_objects

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `source_account_id` | TEXT | Tenant isolation (default `primary`) |
| `bucket_id` | UUID | FK → np_storage_buckets.id (ON DELETE CASCADE) |
| `key` | TEXT | Object key (path, e.g. `images/photo.jpg`) |
| `content_type` | TEXT | MIME type |
| `size_bytes` | BIGINT | Object size in bytes |
| `etag` | TEXT | MD5 checksum (S3 ETag convention) |
| `created_at` | TIMESTAMPTZ | Created timestamp |
| `updated_at` | TIMESTAMPTZ | Updated timestamp |

---

## Hasura Row-Level Security

The plugin ships Hasura metadata (`migrations/hasura_rls.yaml`) that adds `source_account_id` row filters on all roles for both tables.

Row filter applied to every permission:
```yaml
filter:
  source_account_id:
    _eq: X-Hasura-Source-Account-Id
```

Apply via:
```bash
hasura metadata apply
```

---

## Security Notes

- **Authentication:** `STORAGE_API_KEY` must be non-empty; the server refuses to start without it.
- **Tenant isolation:** bucket ownership is verified before any object write — cross-tenant injection is blocked at the DB query level.
- **Upload size:** `http.MaxBytesReader` caps the body before reading — `Content-Length: -1` (chunked / streaming) cannot bypass the limit.
- **Path traversal:** object keys containing `..` are rejected.
- **SSRF:** S3 endpoint is set via `STORAGE_S3_ENDPOINT` config only — not overridable per request.

---

## License

Requires a valid nSelf pro license. Install with:

```bash
nself license activate <your-key>
nself plugin install storage
```
