# Documents Plugin

> Document management with templates, versioning, and share tokens. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install documents
```

## What It Does

Manages documents stored in MinIO with metadata, versioning, and access control. Create document templates for consistent formatting, generate share links with configurable expiry and permissions, and track document access via share tokens. Supports PDF, DOCX, and plain text formats.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DOCUMENTS_PORT` | `3106` | Documents service port |
| `DOCUMENTS_STORAGE_BUCKET` | — | MinIO bucket for documents |
| `DOCUMENTS_SHARE_TTL` | `604800` | Default share link TTL (7 days) |
| `DOCUMENTS_MAX_FILE_SIZE` | `50MB` | Maximum document size |

## Ports

| Port | Purpose |
|------|---------|
| 3106 | Documents REST API |

## Database Tables

5 tables added to your Postgres database:
- `np_documents_documents` — document metadata
- `np_documents_versions` — version history
- `np_documents_templates` — document templates
- `np_documents_shares` — share token records
- `np_documents_access_log` — document access log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/documents/` | Document management API |
| `/documents/share/{token}` | Share link access |
