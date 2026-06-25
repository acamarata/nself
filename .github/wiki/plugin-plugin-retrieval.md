# plugin-retrieval

Hybrid retrieval plugin for nSelf. Combines pgvector ANN and tsvector BM25 using
Reciprocal Rank Fusion (RRF) to produce ranked results that outperform either method alone.
Powers the search and recall tools in nself-ai-mcp.

**Port:** 3825 | **License:** Pro | **Bundle:** ɳClaw (AI-CP)

---

## What Is RRF?

Reciprocal Rank Fusion merges multiple ranked lists into one. The score for each document:

```
rrf_score = Σ 1 / (k + rank)   for each list containing the document
```

Where `k = 60` (standard smoothing constant, Cormack et al. 2009).

| Scenario | Score |
|----------|-------|
| Document at rank 1 in one list | 1/(60+1) = 0.01639 |
| Document at rank 1 in both lists | 2 × 0.01639 = 0.03279 |
| Document at rank 5 in one list | 1/(60+5) = 0.01538 |

Documents found by both retrieval methods rank higher. Documents found by only one method
still appear in the merged list. The fusion consistently outperforms single-method retrieval
for mixed semantic + keyword queries.

---

## Two Retrieval Methods

### pgvector ANN (Approximate Nearest Neighbor)

- Index type: IVFFlat with cosine distance (`<=>` operator)
- Dimensions: 1536 (compatible with OpenAI ada-002, Anthropic embeddings)
- Embedding provided by caller — plugin does NOT call any AI provider (SSRF N/A)
- Returns documents sorted by cosine similarity descending

### tsvector BM25 (Full-Text Search)

- Uses Postgres built-in `to_tsvector` + `plainto_tsquery` with English dictionary
- Ranked by `ts_rank_cd` (document coverage weighting)
- No external dependencies — pure Postgres

---

## API Reference

### POST /search

Run hybrid search. Provide `embedding` for vector, `query` for text, or both for full hybrid.

**Request:**

```json
{
  "query": "semantic memory retrieval",
  "embedding": [0.1, 0.2, 0.3],
  "top_k": 10,
  "source_account_id": "primary"
}
```

**Response:**

```json
{
  "results": [
    {
      "id": "doc-abc",
      "source_account_id": "primary",
      "title": "Memory retrieval architecture",
      "content": "...",
      "similarity": 0.92,
      "text_rank": 0.15,
      "rrf_score": 0.032
    }
  ],
  "count": 5
}
```

Fields:
- `similarity` — pgvector cosine similarity (1 = identical), present if vector search ran
- `text_rank` — tsvector ts_rank_cd score, present if text search ran
- `rrf_score` — final merged score (higher = better match)

Results are always sorted by `rrf_score` descending.

---

### POST /index

Upsert a document and its embedding into the retrieval index.

**Request:**

```json
{
  "id": "memory-2026-01-15",
  "title": "Meeting notes Jan 15",
  "content": "Discussed retrieval architecture and RRF fusion approach...",
  "embedding": [0.1, 0.2, 0.3],
  "source_account_id": "primary"
}
```

`embedding` is optional. If omitted, only text search will find this document.

**Response:** `{"status":"indexed"}` (201 Created)

Uses `ON CONFLICT (id, source_account_id) DO UPDATE` — safe to call repeatedly.

---

### GET /health

```
GET /health
→ {"status": "ok"}          200 — DB reachable
→ {"status": "unhealthy"}   503 — DB unavailable
```

---

## Tenant Isolation

Every query and index operation requires `source_account_id`. If not provided, defaults to
`"primary"`. This enforces Multi-App Isolation: tenants never see each other's documents.

Hasura row filters on both `np_retrieval_documents` and `np_retrieval_embeddings` enforce
`source_account_id = X-Hasura-Source-Account-Id` for `nself_user` role.

---

## Quick Start

```bash
nself license set <your-key>
nself plugin install plugin-retrieval
```

Set required environment variables:

```bash
nself plugin env set plugin-retrieval NSELF_DB_URL=postgres://...
nself plugin env set plugin-retrieval NSELF_LICENSE_KEY=<your-key>
```

Postgres must have the pgvector extension available:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

The plugin runs migrations automatically on startup.

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NSELF_DB_URL` | Yes | — | PostgreSQL connection string |
| `NSELF_LICENSE_KEY` | Yes | — | nSelf Pro license key |
| `NSELF_RETRIEVAL_PORT` | No | `3825` | HTTP server port |

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_retrieval_documents` | Full text corpus with tsvector GIN index |
| `np_retrieval_embeddings` | Embedding vectors with IVFFlat cosine index |

Both tables use composite primary key `(id, source_account_id)` for clean tenant isolation.
The FK from `np_retrieval_embeddings` to `np_retrieval_documents` cascades on delete.

---

## Performance Notes

- IVFFlat `lists=100` is tuned for up to ~1M vectors per account
- For >1M vectors per account, increase `lists` to `sqrt(row_count)` and run `REINDEX`
- Text search performance scales with `content` length; chunk at ~512 tokens for best results
- RRF fusion runs in memory (Go) — no additional DB round-trip

---

## License

Requires an active nSelf Pro license. Purchase at [nself.org/pro](https://nself.org/pro).

---

## Related

- [[plugin-nself-ai-mcp]] — uses this plugin for search and recall tools
- [[plugin-nself-ai-gateway]] — AI key pool for embedding generation
- [[Architecture]]
- [[Home]]
