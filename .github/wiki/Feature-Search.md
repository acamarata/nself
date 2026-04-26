# Feature: Search

ɳSelf integrates MeiliSearch for fast, typo-tolerant full-text search.

## What's Included

| Capability | Description |
|-----------|-------------|
| Full-text search | Typo-tolerant search across your data |
| Filters and facets | Filter results by attributes |
| Custom ranking | Configure relevance rules |
| Instant search | Results in milliseconds |
| REST API | Standard MeiliSearch HTTP API |

## How to Enable

```bash
nself service enable search
nself build && nself restart
```

Or set in `.env`:
```env
SEARCH_ENABLED=true
```

## Key Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SEARCH_ENABLED` | Enable MeiliSearch | `false` |
| `SEARCH_PORT` | Internal port | `7700` |
| `SEARCH_AUTO_INDEX` | Auto-create indexes | `true` |
| `SEARCH_MASTER_KEY` | MeiliSearch master key | Auto-generated |

## Connecting to MeiliSearch

```
Endpoint: https://search.your-domain.com
Master Key: ${SEARCH_MASTER_KEY}
```

Use the [MeiliSearch SDK](https://www.meilisearch.com/docs/learn/getting_started/sdks) for your language of choice.

## Free Search Plugin

For extended search features (query analytics, synonym management, advanced faceting), install the free `search` plugin:

```bash
nself plugin install search
```

## See Also

- [[plugin-search]], search plugin reference
- [[cmd-service]], enable/disable search
- [[Config-Env-Vars]], full env var reference

---
← [[Home]] | [[_Sidebar]]
