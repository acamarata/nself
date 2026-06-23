# plugin-retrieval

**Bundle:** ɳClaw | **Tier:** Pro | **Port:** 3825 | **License:** Required

Hybrid retrieval plugin for nSelf. Provides Reciprocal Rank Fusion (RRF) search over your
Postgres database using two complementary methods:

- **pgvector ANN** — approximate nearest-neighbor vector search using HNSW index (cosine distance)
- **tsvector BM25** — native Postgres full-text search using `websearch_to_tsquery`

Both result sets are fused using RRF scoring, producing a single ranked list that outperforms
either method in isolation.

---

## Table of Contents

1. [Installation](#installation)
2. [Configuration](#configuration)
3. [Database Schema](#database-schema)
4. [Hasura Tenant RLS](#hasura-tenant-rls)
5. [RRF Algorithm](#rrf-algorithm)
6. [API Reference](#api-reference)
   - [GET /health](#get-health)
   - [POST /search](#post-search)
   - [POST /index](#post-index)
   - [DELETE /index/{chunk\_id}](#delete-indexchunk_id)
7. [Tenant Isolation](#tenant-isolation)
8. [SSRF & Security](#ssrf--security)
9. [Troubleshooting](#troubleshooting)

---

## Installation

```bash
nself plugin install plugin-retrieval
```

Requires an active ɳSelf+ or ɳClaw bundle license:

```bash
nself license activate <your-license-key>
```

Verify the plugin is running:

```bash
nself plugin status plugin-retrieval
curl http://localhost:3825/health
```

---

## Configuration

| Environment Variable         | Required | Default | Description                                          |
|------------------------------|----------|---------|------------------------------------------------------|
| `DATABASE_URL`               | Yes      | —       | PostgreSQL DSN                                       |
| `RETRIEVAL_PORT`             | No       | `3825`  | HTTP listen port                                     |
| `RETRIEVAL_VECTOR_DIMS`      | No       | `1536`  | Embedding dimensions (must match your AI model)      |
| `RETRIEVAL_RRF_K`            | No       | `60`    | RRF constant k (rank smoothing; 60 is the standard)  |
| `RETRIEVAL_ALLOWED_ORIGINS`  | No       | —       | CSV of allowed CORS origins                          |

Set variables via `nself env set` or in your `.env` file (never commit credentials).

---

## Database Schema

The plugin creates three tables in the `public` schema, all prefixed `np_retrieval_`:

### `np_retrieval_indexes`

Logical index definitions. One index per use-case (e.g., "product-docs", "support-articles").

| Column        | Type        | Notes                        |
|---------------|-------------|------------------------------|
| `id`          | UUID PK     | auto-generated               |
| `tenant_id`   | UUID        | cloud multi-tenancy RLS key  |
| `name`        | TEXT        | unique per tenant            |
| `description` | TEXT        | optional                     |
| `vector_dims` | INT         | default 1536                 |
| `created_at`  | TIMESTAMPTZ |                              |
| `updated_at`  | TIMESTAMPTZ |                              |

### `np_retrieval_documents`

Top-level documents (e.g., a PDF, a wiki page, a support article).

| Column       | Type        | Notes                       |
|--------------|-------------|-----------------------------|
| `id`         | UUID PK     |                             |
| `tenant_id`  | UUID        | RLS key                     |
| `index_id`   | UUID FK     | references np_retrieval_indexes |
| `title`      | TEXT        |                             |
| `uri`        | TEXT        | optional source URL         |
| `metadata`   | JSONB       | arbitrary payload           |

### `np_retrieval_chunks`

Text chunks (typically 256–512 tokens each) with both vector and full-text columns.

| Column        | Type           | Notes                                      |
|---------------|----------------|--------------------------------------------|
| `id`          | UUID PK        |                                            |
| `tenant_id`   | UUID           | RLS key                                    |
| `document_id` | UUID FK        | references np_retrieval_documents          |
| `chunk_index` | INT            | sequential position within the document    |
| `chunk_text`  | TEXT           | raw text content                           |
| `chunk_vec`   | vector(1536)   | HNSW-indexed embedding (nullable)          |
| `chunk_tsv`   | tsvector       | GENERATED ALWAYS — auto-built from text    |
| `metadata`    | JSONB          |                                            |
| `created_at`  | TIMESTAMPTZ    |                                            |

Indexes created automatically:

- `idx_np_retrieval_chunks_vec` — HNSW cosine for pgvector ANN
- `idx_np_retrieval_chunks_tsv` — GIN for tsvector BM25

---

## Hasura Tenant RLS

All three tables have Hasura row-level permissions enforcing:

```json
{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}
```

This means every GraphQL query/mutation automatically filters to the calling tenant.
Cross-tenant data leakage is impossible at the GraphQL layer.

Apply metadata:

```bash
nself hasura sync --plugin plugin-retrieval
```

---

## RRF Algorithm

Reciprocal Rank Fusion (Cormack, Clarke & Buettcher, SIGIR 2009) merges ranked lists from
multiple retrieval methods into a single list without requiring score normalization.

### Formula

For each document `d` across retrieval methods `m₁, m₂, ...`:

```
RRF_score(d) = Σᵢ  1 / (k + rank_i(d))
```

Where:
- `rank_i(d)` is the 1-indexed rank of document `d` in list `i` (0 if absent)
- `k` is the smoothing constant (default: 60 — from the original paper)

### Why k=60?

The original paper found k=60 to be robust across diverse test collections.
Lower k values (e.g., k=10) amplify the importance of rank-1 results;
higher values (k=200+) dampen differences between ranks.

### Example

| Chunk | pgvector rank | tsvector rank | RRF score (k=60)       |
|-------|---------------|---------------|------------------------|
| A     | 1             | 3             | 1/61 + 1/63 = 0.03227  |
| B     | 2             | 2             | 1/62 + 1/62 = 0.03226  |
| C     | 3             | 1             | 1/63 + 1/61 = 0.03227  |
| D     | 4             | —             | 1/64         = 0.01563  |

A and C tie (as expected for symmetric ranks); B is very close. D, present only in
vector results, scores lower. The fusion ensures both semantic similarity and
keyword relevance contribute to the final ranking.

---

## API Reference

All endpoints (except `/health`) require the `X-Hasura-Tenant-Id` header, which is
automatically injected by Hasura from the caller's JWT session variables.

### GET /health

Returns service status. No authentication required.

**Response:**
```json
{"status": "ok", "plugin": "plugin-retrieval", "port": 3825, "timestamp": "2026-06-22T00:00:00Z"}
```

---

### POST /search

Runs hybrid pgvector ANN + tsvector BM25 + RRF fusion search.

**Request headers:**
- `Content-Type: application/json`
- `X-Hasura-Tenant-Id: <tenant-uuid>`

**Request body:**
```json
{
  "query": "how to configure authentication",
  "index_id": "550e8400-e29b-41d4-a716-446655440000",
  "query_vector": [0.12, -0.34, ...],
  "top_k": 10
}
```

| Field          | Type     | Required | Notes                                           |
|----------------|----------|----------|-------------------------------------------------|
| `query`        | string   | Yes      | Natural-language search query                   |
| `index_id`     | UUID     | Yes      | Scopes search to this index                     |
| `query_vector` | float[]  | No       | Embedding of the query; omit for text-only mode |
| `top_k`        | int      | No       | Number of results to return (default: 10)       |

**Response:**
```json
{
  "results": [
    {
      "chunk_id": "abc123...",
      "document_id": "def456...",
      "chunk_index": 2,
      "chunk_text": "Authentication can be configured via...",
      "rrf_score": 0.03226,
      "vec_rank": 1,
      "fts_rank": 3,
      "metadata": {}
    }
  ],
  "count": 10
}
```

If `query_vector` is omitted, only tsvector BM25 results are used (no vector retrieval).
`vec_rank` and `fts_rank` are 0 if the chunk was not found by that method.

---

### POST /index

Stores a text chunk with an optional embedding vector.

**Request body:**
```json
{
  "document_id": "def456...",
  "chunk_index": 0,
  "chunk_text": "Authentication can be configured via the auth plugin...",
  "chunk_vector": [0.12, -0.34, ...],
  "metadata": {"source": "docs/auth.md", "section": "quickstart"}
}
```

**Response (201 Created):**
```json
{"chunk_id": "abc123..."}
```

---

### DELETE /index/{chunk\_id}

Removes a chunk from the index. Enforces tenant isolation — tenants cannot delete
another tenant's chunks.

**Response:** `204 No Content` on success, `404` if not found or not owned by tenant.

---

## Tenant Isolation

All search and indexing operations are scoped to the tenant identified by
`X-Hasura-Tenant-Id`. The plugin enforces this at both the SQL layer (WHERE clause)
and the Hasura GraphQL layer (row-level filter).

To verify isolation during QA:

1. Create two tenants A and B.
2. Index 10 documents under tenant A, 10 under tenant B.
3. Search as tenant A — results must contain only tenant A documents.
4. Search as tenant B — results must contain only tenant B documents.

---

## SSRF & Security

**SSRF: N/A** — plugin-retrieval makes no outbound HTTP requests. All queries run
against local Postgres only. The plugin never fetches external URLs.

Security checklist:
- Tenant isolation: enforced at SQL + Hasura layers
- No secrets in logs
- No outbound connections
- TLS handled by nSelf reverse proxy (nginx)
- pgvector HNSW index does not leak cross-tenant embeddings (filtered by `tenant_id` WHERE clause before ANN)

---

## Troubleshooting

**pgvector extension missing:**
```
ERROR: type "vector" does not exist
```
Postgres needs the pgvector extension. On Hetzner/Docker managed by nSelf, it is
pre-installed. For custom Postgres: `apt install postgresql-15-pgvector`.

**No vector results (vec_rank always 0):**
Ensure you pass `query_vector` in the search request. Without it, the plugin falls
back to text-only mode.

**HNSW index not used (slow searches):**
Run `ANALYZE np_retrieval_chunks;` to update planner statistics after bulk inserts.

**Cross-tenant data in results:**
Check that your Hasura JWT includes `X-Hasura-Tenant-Id` as a session variable and
that the Hasura metadata has been applied (`nself hasura sync --plugin plugin-retrieval`).
