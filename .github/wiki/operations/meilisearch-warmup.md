# MeiliSearch Index Warm-Up

nSelf can issue representative search queries immediately after `nself start` brings
MeiliSearch online, populating the query-result cache before real user traffic arrives.

Warm-up is **opt-in** and **non-blocking**. It runs in a background goroutine and never
delays the startup sequence. Errors are collected and surfaced only in verbose mode.

---

## Configuration

Set `MEILISEARCH_WARMUP_QUERIES` in your `.env.local` or `.env.secrets` to a
comma-separated list of query strings:

```bash
MEILISEARCH_WARMUP_QUERIES="users,posts,settings"
```

Leave the variable unset (or empty) to skip warm-up entirely. That is the default.

### Environment variable reference

| Variable | Default | Description |
|----------|---------|-------------|
| `MEILISEARCH_WARMUP_QUERIES` | _(empty — warm-up disabled)_ | Comma-separated list of query strings issued to MeiliSearch on startup |

---

## How it works

1. After health checks pass, nSelf reads `MEILISEARCH_WARMUP_QUERIES`.
2. When the variable is non-empty and the configured search engine is `meilisearch`,
   a background goroutine fires the warm-up.
3. Each query is sent to the MeiliSearch
   [multi-search endpoint](https://www.meilisearch.com/docs/reference/api/multi_search)
   (`POST /multi-search`) with `limit: 1`.
4. A 404 response (index not yet created on a fresh instance) is treated as success
   because the goal is request-path warm-up, not data validation.
5. The goroutine times out after 10 seconds regardless of progress.

Running `nself start --verbose` prints a one-line summary:

```
  MeiliSearch warm-up: 3/3 queries succeeded
```

---

## Choosing warm-up queries

Pick queries that reflect your application's most common search patterns. Two to five
queries is usually sufficient:

- Core entity names your users search for most often (for example `users`, `posts`)
- Short, common prefixes that stress-test the prefix-matching path
- A wildcard or empty string if your app uses auto-complete with no initial input

Avoid queries that return millions of results — `limit: 1` is used, but the query
parser still evaluates the full index.

---

## Disabling warm-up

Remove or unset `MEILISEARCH_WARMUP_QUERIES`:

```bash
unset MEILISEARCH_WARMUP_QUERIES
# or remove the line from .env.local
```

---

## Related

- [[Architecture]]
- [[cmd-start]]
- [[Home]]
