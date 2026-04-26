# Knowledge Base Plugin

> Docs and FAQ with semantic search, versioning, and translations. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install knowledge-base
```

## What It Does

A structured knowledge base for documentation and FAQs. Organize content into categories and articles, maintain full version history, support multiple translations per article, and provide semantic search powered by the `search` plugin or the `ai` plugin for embedding-based retrieval. Exposes a GraphQL API for embedding the knowledge base in your app or support widget.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `KNOWLEDGE_BASE_PORT` | `3713` | Knowledge base service port |
| `KNOWLEDGE_BASE_SEMANTIC_SEARCH` | `false` | Use AI embeddings for search |
| `KNOWLEDGE_BASE_DEFAULT_LOCALE` | `en` | Default article locale |
| `KNOWLEDGE_BASE_VERSION_HISTORY` | `20` | Versions to retain per article |

## Ports

| Port | Purpose |
|------|---------|
| 3713 | Knowledge base REST API |

## Database Tables

8 tables added to your Postgres database:
- `np_knowledge_base_categories`, content categories
- `np_knowledge_base_articles`, article content
- `np_knowledge_base_versions`, article version history
- `np_knowledge_base_translations`, per-locale translations
- `np_knowledge_base_feedback`, article helpfulness ratings
- `np_knowledge_base_search_index`, search index metadata
- `np_knowledge_base_tags`, article tags
- `np_knowledge_base_related`, related article links

## Nginx Routes

| Route | Target |
|-------|--------|
| `/knowledge-base/` | Knowledge base API |
