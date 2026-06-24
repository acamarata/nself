# Retrieval Plugin

> Hybrid semantic + keyword retrieval service with Reciprocal Rank Fusion (RRF). **ɳClaw bundle — Pro plugin.**

> **Requires:** ɳClaw bundle license or ɳSelf+ tier. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install plugin-retrieval
```

## What It Does

Runs a hybrid retrieval HTTP service on port 3825. Combines **pgvector** approximate
nearest-neighbour (ANN) cosine-similarity search with **tsvector** BM25 full-text
search, then fuses the two ranked result lists using **Reciprocal Rank Fusion (RRF)**.

This is the retrieval backend consumed by `nself-ai-mcp`'s `search` and `recall` tools.
ɳClaw's infinite memory feature depends on this plugin to retrieve relevant context
from the Postgres knowledge store.

### Why Hybrid Retrieval?

- **Semantic search alone** (pgvector) misses exact-match keyword queries and proper
  nouns that aren't well-represented in embedding space.
- **Keyword search alone** (tsvector) misses paraphrased queries and semantic
  similarities that don't share surface-level tokens.
- **RRF fusion** merges both ranked lists without requiring score calibration, producing
  a single, higher-quality ranking that outperforms either method alone.

## RRF Algorithm

```
RRF_score(d) = Σ_r  1 / (k + rank_r(d))
```

- `k = 60` (configurable via `RETRIEVAL_RRF_K`) — the standard rank constant.
- `rank_r(d)` is the 1-indexed rank of document `d` in list `r` (vector or keyword).
- A document at rank 1 in both lists scores `2 / (60+1) ≈ 0.0328`.
- A document at rank 1 in only one list scores `1 / (60+1) ≈ 0.0164`.
- Documents missing from a list contribute 0 from that list.
- Results are sorted descending by RRF score.

## Configuration

| Env Var                    | Default  | Description                               |
|----------------------------|----------|-------------------------------------------|
| `DATABASE_URL`             | —        | Postgres connection string (required)     |
| `RETRIEVAL_PORT`           | `3825`   | HTTP listen port (F10 registered)         |
| `RETRIEVAL_VECTOR_DIMS`    | `1536`   | Embedding vector dimension (ada-002)      |
| `RETRIEVAL_RRF_K`          | `60`     | RRF rank constant                         |
| `RETRIEVAL_ALLOWED_ORIGINS`| `""`     | CSV of CORS-allowed origins (empty = off) |

## Ports

| Port | Purpose               |
|------|-----------------------|
| 3825 | Retrieval REST API    |

## Database Tables

Three tables added to your Postgres schema:

| Table                         | Purpose                                      |
|-------------------------------|----------------------------------------------|
| `np_retrieval_documents`      | Indexed text chunks with generated tsvector  |
| `np_retrieval_embeddings`     | pgvector `vector(1536)` per (document, model)|
| `np_retrieval_index_config`   | Per-tenant fusion weights + RRF k override   |

All tables carry:
- `tenant_id UUID NOT NULL` — Cloud Multi-Tenancy with Hasura row filter.
- `source_account_id TEXT NOT NULL DEFAULT 'primary'` — Multi-App Isolation.

## API

```
GET  /health                   — Service + DB health check (no auth)
POST /search                   — Hybrid RRF search (requires X-Hasura-Tenant-Id)
POST /index                    — Index a document chunk + optional embedding
DELETE /index/{chunk_id}       — Remove a chunk and its embedding
```

### POST /search

Request:

```json
{
  "query": "golang context cancellation",
  "top_k": 5,
  "index_type": "hybrid",
  "query_embedding": [0.1, 0.2, ...]
}
```

`index_type` values:
- `"hybrid"` (default) — runs both pgvector and tsvector, then RRF fuses.
- `"vector"` — pgvector ANN only (requires `query_embedding`).
- `"keyword"` — tsvector BM25 only.

Response:

```json
{
  "results": [
    {
      "chunk_id": "doc-abc-001",
      "content": "The context package defines ...",
      "metadata": "{\"source\": \"golang_docs\"}",
      "rrf_score": 0.032786,
      "vector_score": 0.94,
      "keyword_score": 0.72
    }
  ],
  "top_k": 5,
  "query": "golang context cancellation",
  "tenant_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

### POST /index

Request:

```json
{
  "chunk_id": "doc-abc-001",
  "content": "The context package defines the Context type ...",
  "embedding": [0.1, 0.2, ...],
  "metadata": "{\"source\": \"golang_docs\", \"page\": 3}",
  "model": "text-embedding-ada-002"
}
```

- `embedding` is optional. When omitted, only tsvector keyword search works for this chunk.
- `embedding` length must equal `RETRIEVAL_VECTOR_DIMS` when provided.
- `chunk_id` must be stable across updates — upsert on `(chunk_id, tenant_id)`.

### DELETE /index/{chunk_id}

Removes the document and all associated embeddings (cascade). Idempotent.

## Tenant Isolation

Every SQL query includes `tenant_id = $N` in the WHERE clause.
Hasura additionally enforces `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}` row
filters on all tables and all roles. Cross-tenant data leaks are impossible at both
the Go and Hasura layers.

The `X-Hasura-Tenant-Id` header must be present on all authenticated endpoints.
Requests missing this header receive HTTP 401.

## Security

- **SSRF:** no outbound HTTP. All queries target the local Postgres instance only.
- **SQL injection:** all queries use parameterised `$N` placeholders.
- **License gate:** enforced by the CLI at install time (`requires_license: true`
  in `plugin.json`). The CLI validates against `ping.nself.org` before starting
  the service.

## Bundle

Part of the **ɳClaw bundle**. This plugin is a dependency of `nself-ai-mcp` — install
order: `plugin-retrieval` first, then `nself-ai-mcp`.

## Related

- `nself-ai-mcp` — MCP server that exposes `search` and `recall` tools backed by this service.
- `plugin-ai` — embedding generation via `POST /embed`; output is the `query_embedding` input here.
- [ɳClaw Bundle](https://nself.org/products/nclaw)
