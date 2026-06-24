# CRDT Plugin

> Self-hosted offline-first sync via Yjs and automerge, backed by Postgres. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | Yes |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** Any paid bundle or ɳSelf+ subscription (tier: pro per F07-PRICING-TIERS).

## Bundle membership

The crdt plugin is not currently part of a named bundle. It is available to any user with a bundle-level or ɳSelf+ subscription.

Or get all bundles + all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install crdt
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## Description

The crdt plugin provides a self-hosted CRDT sync server that supports both Yjs (via the y-websocket protocol) and automerge HTTP sync. It is a drop-in alternative to hosted collaboration services such as Liveblocks and PartyKit, with no extra infrastructure required beyond a running nSelf stack and Postgres.

Document state is persisted immediately on every update to the `np_crdt_documents` table. Update history is stored in `np_crdt_updates` and compacted automatically in the background based on the configured retention window. Service restarts are transparent to clients: they reconnect and resume syncing without data loss.

Yjs and automerge clients can share the same server. The `CRDT_ENGINE` variable controls which protocols are active (`yjs`, `automerge`, or `both`). Multi-instance deployments can coordinate awareness events via an optional Redis sidecar (`CRDT_REDIS_URL`).

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | Postgres connection string (required) |
| `CRDT_ENGINE` | `both` | Active sync engine: `yjs`, `automerge`, or `both` |
| `CRDT_MAX_DOC_SIZE_MB` | `10` | Maximum document state size in MB |
| `CRDT_RETENTION_DAYS` | `90` | Days to retain update history before compaction |
| `CRDT_PORT` | `8211` | HTTP and WebSocket listen port |
| `CRDT_REDIS_URL` | — | Optional Redis URL for multi-instance awareness sync |
| `LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |

## Ports

| Port | Protocol | Description |
|------|----------|-------------|
| `8211` | HTTP + WebSocket | Yjs WebSocket endpoint and automerge HTTP sync |

The port is configurable via `CRDT_PORT`. Port `8211` is registered in F10-PORT-REGISTRY.

## Database Schema

The plugin creates two tables on first start via an embedded migration.

**`np_crdt_documents`** — current state of each document:

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID | Primary key |
| `doc_id` | TEXT | Unique document identifier |
| `engine` | TEXT | `yjs` or `automerge` |
| `state` | BYTEA | Current encoded document state |
| `updates_count` | INT | Number of updates applied |
| `source_account_id` | TEXT | Multi-app isolation (default: `primary`) |
| `last_modified` | TIMESTAMPTZ | Last update timestamp |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

**`np_crdt_updates`** — append-only update log:

| Column | Type | Notes |
|--------|------|-------|
| `id` | BIGSERIAL | Primary key |
| `doc_id` | TEXT | Foreign key to `np_crdt_documents` |
| `update_data` | BYTEA | Encoded CRDT update delta |
| `client_id` | TEXT | Optional client identifier |
| `applied_at` | TIMESTAMPTZ | When the update was applied |

## REST API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `WS` | `/crdt/yjs/:doc_id` | bearer | Yjs y-websocket connection |
| `POST` | `/crdt/sync/:doc_id` | bearer | automerge binary sync message |
| `GET` | `/crdt/doc/:doc_id` | bearer | Retrieve current document state |
| `DELETE` | `/crdt/doc/:doc_id` | bearer | Delete a document and its history |
| `GET` | `/crdt/docs` | bearer | List all documents |
| `POST` | `/crdt/compact/:doc_id` | bearer | Force history compaction for a document |
| `GET` | `/health` | — | Health check endpoint |

## Examples

### Yjs client (JavaScript)

```bash
# Install the y-websocket client in your project
npm install yjs y-websocket
```

```javascript
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'

const doc = new Y.Doc()
const provider = new WebsocketProvider(
  'wss://api.example.com',  // your nSelf API hostname
  'my-document-id',
  doc
)

// Use shared types as normal
const text = doc.getText('content')
text.insert(0, 'Hello from client A')
```

### automerge client (JavaScript)

```javascript
import * as Automerge from '@automerge/automerge'

// Generate a sync message from local state
const [newDoc, msg] = Automerge.generateSyncMessage(doc, syncState)

if (msg) {
  const res = await fetch('/crdt/sync/my-doc', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/octet-stream',
      'Authorization': 'Bearer <token>',
    },
    body: msg,
  })
  const reply = new Uint8Array(await res.arrayBuffer())
  // Apply server reply to local document
  const [updated, nextState] = Automerge.receiveSyncMessage(doc, syncState, reply)
}
```

### Docker Hub image

```bash
# Pull the published image directly (server-side deployment)
docker pull nself/plugin-crdt:latest
```

## Source

Source-available (license required to run): [`plugins-pro/paid/crdt/`](https://github.com/nself-org/plugins-pro/tree/main/paid/crdt)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-realtime]] — WebSocket presence and pub/sub
- [[plugin-auth]] — JWT authentication for CRDT connections
- [[Pricing]] — tier comparison
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
