# Search Plugin

> Full-text search with PostgreSQL FTS and MeiliSearch. **Free, MIT licensed.**

## Install

```bash
nself plugin install search
```

## What It Does

Adds full-text search to your ɳSelf backend with two engines: built-in PostgreSQL FTS for simple search, and MeiliSearch for typo-tolerant, faceted search. Provides an index management API to define which Postgres tables and columns to index, and a unified search API that queries either engine.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SEARCH_PORT` | `3302` | Search service port |
| `SEARCH_ENGINE` | `meilisearch` | Engine: `meilisearch` or `postgres` |
| `SEARCH_AUTO_INDEX` | `true` | Auto-index on data changes |
| `MEILISEARCH_HOST` | `http://meilisearch:7700` | MeiliSearch endpoint |
| `MEILISEARCH_MASTER_KEY` | — | MeiliSearch master key |

## Ports

| Port | Purpose |
|------|---------|
| 3302 | Search service REST API |
| 7700 | MeiliSearch (if SEARCH_ENGINE=meilisearch) |

## Database Tables

5 tables added to your Postgres database:
- `np_search_indices`, index definitions
- `np_search_index_fields`, indexed fields per index
- `np_search_documents`, cached document store
- `np_search_queries`, query analytics
- `np_search_synonyms`, search synonym mappings

## Nginx Routes

| Route | Target |
|-------|--------|
| `/search/` | Search API |

## API

```
GET  /health              — Health check
POST /indices             — Create an index
POST /indices/{name}/sync — Sync Postgres table to index
GET  /search?q={query}    — Search across all indices
GET  /search/{index}?q=   — Search a specific index
```
