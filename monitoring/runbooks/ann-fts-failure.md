# Runbook: HybridRetrievalSlow / RerankerSlow / ANN + FTS Failure

**Alerts:** `HybridRetrievalSlow` (warning), `RerankerSlow` (warning)
**Components:** pgvector HNSW index (ANN), PostgreSQL full-text search (FTS), bge-reranker service

## What fired

- **HybridRetrievalSlow** — The p95 hybrid retrieval latency (ANN + FTS + RRF merge) has exceeded 250 ms for 5 minutes or more. The SLO target from G1-T16 is p95 < 250 ms.
- **RerankerSlow** — The p95 bge-reranker latency has exceeded 350 ms for 5 minutes. The SLO target from G1-T23 is p95 < 350 ms.

## Architecture overview

```
User query
    │
    ├── ANN search  (pgvector HNSW on embedding vectors)
    │       └── top-K candidate chunks
    │
    ├── FTS search  (PostgreSQL ts_rank on tsvector columns)
    │       └── top-K candidate chunks
    │
    └── RRF merge   (Reciprocal Rank Fusion — combine + deduplicate)
            └── top-N candidates → bge-reranker → final ranked list
```

## Immediate checks

```bash
# 1. Is the AI plugin container running?
nself status

# 2. Check AI plugin logs for slow query evidence
docker logs $(docker ps -qf name=nself-ai) --tail 100 | grep -i "retrieval\|rerank\|slow\|timeout"

# 3. Check Prometheus metric directly
curl -s http://localhost:9090/api/v1/query \
  --data-urlencode 'query=histogram_quantile(0.95, sum(rate(nself_hybrid_retrieval_p95_ms_bucket[5m])) by (le))'

# 4. Check reranker latency
curl -s http://localhost:9090/api/v1/query \
  --data-urlencode 'query=histogram_quantile(0.95, sum(rate(nself_rerank_p95_ms_bucket[5m])) by (le))'
```

## ANN (pgvector HNSW) failure modes

### HNSW index missing or not used

```sql
-- Verify index exists
\d np_memory_chunks

-- Check query plan uses index scan
EXPLAIN (ANALYZE, FORMAT TEXT)
SELECT id, content
  FROM np_memory_chunks
 ORDER BY embedding <-> '[...]'
 LIMIT 20;
```

If the plan shows `Seq Scan` instead of `Index Scan`, the HNSW index may be corrupted or the query is not using it. Rebuild:

```sql
REINDEX INDEX CONCURRENTLY np_memory_chunks_embedding_idx;
```

### HNSW query time increase (high m or ef_construction)

Check current index parameters:

```sql
SELECT indexname, indexdef
  FROM pg_indexes
 WHERE indexname LIKE '%embedding%';
```

If the index was built with high `m` or `ef_search`, queries will be slower but more accurate. Tune with `SET hnsw.ef_search = 40;` (default is 64 — lower = faster, less accurate).

### Vector dimension mismatch

If the embedding model was swapped (e.g., 768-dim → 1536-dim), existing index entries will fail to match. Check:

```bash
docker logs $(docker ps -qf name=nself-ai) | grep -i "dim\|dimension"
```

## FTS (full-text search) failure modes

### Missing tsvector GIN index

```sql
\d np_memory_chunks

-- Rebuild if missing
CREATE INDEX CONCURRENTLY np_memory_chunks_fts_idx
  ON np_memory_chunks
  USING GIN (to_tsvector('english', content));
```

### Slow ts_rank due to table bloat

```sql
-- Check table bloat
SELECT schemaname, tablename, n_dead_tup, n_live_tup
  FROM pg_stat_user_tables
 WHERE tablename = 'np_memory_chunks';

-- Vacuum if dead tuple ratio > 20%
VACUUM ANALYZE np_memory_chunks;
```

## RRF merge failure modes

The Reciprocal Rank Fusion merge is pure Go (no I/O). If it is slow:

1. The candidate list from ANN or FTS is much larger than expected. Check query parameters (`top_k`).
2. The deduplication step is encountering hash collisions. Check AI plugin logs for `rrf_dedup_collision`.

## Reranker (bge-reranker) failure modes

### bge-reranker container resource exhaustion

```bash
docker stats $(docker ps -qf name=nself-ai-reranker)
```

If CPU is pegged at 100%:

- Reduce `reranker_batch_size` in the AI plugin config (default: 16; try 8).
- Add CPU limits if not already set.

### Model cache cold start

On container restart, the bge-reranker model loads from disk. This causes the first ~10 requests to be slow (1–3 s). This is expected behavior — the alert has a 5-minute `for:` window specifically to ignore cold starts.

### FTS fallback procedure

If ANN index is unavailable, the AI plugin can fall back to FTS-only retrieval:

```bash
# Set in AI plugin env (restart required)
NSELF_AI_RETRIEVAL_MODE=fts_only

# Or via CLI config
nself config set ai.retrieval_mode fts_only
nself plugin restart ai
```

FTS-only mode: higher latency (typically 80–150 ms vs 50–200 ms hybrid), lower recall precision. Only use as emergency fallback while ANN index is rebuilt.

## Escalation

If neither ANN nor FTS fix resolves the alert within 30 minutes:

1. File a bug via `pci-send nself reranker-incident high bug "Reranker SLO breach — runbook escalation"`
2. Switch to `fts_only` mode to restore partial functionality
3. Post incident summary in `.claude/memory/lessons.md` with root cause and resolution

## See also

- G6-T09 spec: AI observability metrics (Amendment 4 Decision #22)
- G1-T16: Hybrid retrieval SLO definition
- G1-T23: Reranker SLO definition
- `cli/monitoring/rules/ai-alerts.yml` — alert rule definitions
- `cli/monitoring/grafana/dashboards/ai-observability.json` — Grafana dashboard
