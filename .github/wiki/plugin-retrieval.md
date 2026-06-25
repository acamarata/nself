# plugin-retrieval

> Hybrid retrieval plugin: pgvector ANN + tsvector BM25 merged with Reciprocal Rank Fusion (RRF). Provides the search backend for ɳClaw memory and nself-ai-mcp search/recall tools.
> **Pro plugin — requires license.**

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install plugin-retrieval
```

## What It Does

plugin-retrieval is a hybrid retrieval service that combines two complementary search strategies and merges them using Reciprocal Rank Fusion (RRF). It is the retrieval backend consumed by the `nself-ai-mcp` plugin's search and recall tools.

- **pgvector ANN**: Cosine-distance approximate nearest-neighbor search over 1536-dim embeddings (OpenAI ada-002 compatible).
- **tsvector BM25**: PostgreSQL full-text search using `ts_rank_cd` with `plainto_tsquery`.
- **RRF fusion**: Merges both ranked lists using the standard RRF formula (k=60, Cormack et al. 2009).

Embeddings are caller-provided — the plugin does **not** call any external AI provider. This keeps it provider-agnostic and eliminates SSRF risk.

## RRF Algorithm

RRF score for a document `d`:

```
score(d) = Σ_{list L} 1 / (k + rank_of_d_in_L)
```

where `k = 60` (smoothing constant). Documents appearing in both the vector list and the text list receive additive scores, promoting documents that are relevant by both measures. Documents appearing in only one list receive a single-list score.

## Endpoints

### `POST /search`

Hybrid search over indexed documents.

**Request:**
```json
{
  "query": "how to configure plugins",
  "embedding": [0.12, -0.34, ...],
  "top_k": 10,
  "source_account_id": "primary"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `query` | No | Full-text search query (tsvector) |
| `embedding` | No | Caller-provided float32 vector (pgvector ANN) |
| `top_k` | No | Max results (default 10) |
| `source_account_id` | No | Multi-app isolation key (default: `"primary"`) |

At least one of `query` or `embedding` must be provided. If both are provided, both searches run and results are merged via RRF.

**Response:**
```json
{
  "results": [
    {
      "id": "doc-abc123",
      "source_account_id": "primary",
      "title": "Plugin configuration guide",
      "content": "To configure plugins...",
      "rrf_score": 0.0316,
      "similarity": 0.87,
      "text_rank": 0.0
    }
  ],
  "count": 1
}
```

### `POST /index`

Upsert a document and optionally its embedding.

**Request:**
```json
{
  "id": "doc-abc123",
  "title": "Plugin configuration guide",
  "content": "To configure plugins, edit plugin.json...",
  "embedding": [0.12, -0.34, ...],
  "source_account_id": "primary"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique document ID (within `source_account_id`) |
| `content` | Yes | Plain-text document body |
| `title` | No | Optional title |
| `embedding` | No | Float32 vector; if absent, only text index updated |
| `source_account_id` | No | Multi-app isolation key (default: `"primary"`) |

**Response:** `201 Created`
```json
{"status": "indexed"}
```

### `GET /health`

Returns `{"status":"ok"}` when Postgres is reachable. Returns `503` otherwise.

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_retrieval_documents` | Text corpus: id, title, content, tsvector GIN index |
| `np_retrieval_embeddings` | pgvector ANN index: 1536-dim embeddings, IVFFlat (lists=100) |
| `np_embeddings` | BGE-M3 1024-dim embeddings (E7 extension) |
| `np_retrieval_index` | RRF metadata + tsvector for hybrid scoring |

All tables use `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `PORT` | `3825` | HTTP bind port |

## Port

| Port | Purpose |
|------|---------|
| 3825 | HTTP API (search, index, health) |

## Hasura Permissions

All tables have `source_account_id` row-level filters. `nself_user` role can only read/write documents belonging to their own `source_account_id`. The `embedding` column is not exposed via GraphQL (pgvector not yet supported by Hasura).

## Tenant Isolation

All queries include `WHERE source_account_id = $N`. The pgvector ANN query joins `np_retrieval_embeddings` back to `np_retrieval_documents` using both `document_id` **and** `source_account_id`, preventing cross-account embedding hits from returning another account's document content.

## Docker

```bash
docker pull nself/plugin-retrieval:latest
docker run -e DATABASE_URL=postgres://... -p 3825:3825 nself/plugin-retrieval:latest
```

## License

Requires `nclaw` bundle or ɳSelf+ subscription.

```bash
nself license set nself_pro_xxxxx...
nself plugin install plugin-retrieval
nself plugin status plugin-retrieval
```
