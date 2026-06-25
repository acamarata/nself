# CRDT Plugin

> Offline-first CRDT sync server with Yjs (y-websocket protocol) and Automerge, Postgres persistence, and multi-tenant isolation. **Pro plugin, requires license.**

> **Requires:** Pro license or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install crdt
nself build
```

The license is validated against `ping.nself.org/license/validate`. An insufficient tier returns an error and the purchase URL.

## What It Does

Runs a self-hosted collaborative sync server as a drop-in replacement for Liveblocks or PartyKit — no extra infrastructure required. Two sync engines are available per document:

- **Yjs** (`y-websocket` protocol) — WebSocket-based real-time collaboration for text editors, whiteboards, and structured data. Compatible with the official `yjs` npm package and `@yjs/provider-websocket`.
- **Automerge** — conflict-free merge using the Automerge CRDT algorithm. Useful for offline-first mobile apps that sync periodically.

All document state and update logs are persisted to Postgres in the `np_crdt_documents` and `np_crdt_updates` tables.

## Yjs Setup

Connect your client-side Yjs provider to the plugin's WebSocket endpoint:

```typescript
import * as Y from 'yjs'
import { WebsocketProvider } from 'y-websocket'

const ydoc = new Y.Doc()
const wsUrl = 'ws://your-nself-host:8211'
const provider = new WebsocketProvider(wsUrl, 'my-document-id', ydoc)

provider.on('status', ({ status }) => {
  console.log('sync status:', status) // 'connecting' | 'connected' | 'disconnected'
})
```

The `my-document-id` string is stored as `doc_id` in `np_crdt_documents`. Each unique document ID creates a separate collaboration room.

## Automerge Setup

For Automerge documents, use the HTTP sync endpoint:

```typescript
import * as Automerge from '@automerge/automerge'

// Initialize a document
let doc = Automerge.init()
doc = Automerge.change(doc, d => { d.title = 'Hello' })

// Serialize and sync to server
const state = Automerge.save(doc)
await fetch('http://your-nself-host:8211/crdt/automerge/my-doc-id', {
  method: 'PUT',
  body: state,
  headers: { 'Content-Type': 'application/octet-stream' }
})
```

Note: Automerge v2.x uses `Automerge.save()` / `Automerge.load()` for state serialization. The sync state machine (`initSyncState`, `generateSyncMessage`) is used for peer-to-peer sync, not required for server persistence.

## Postgres Schema

The plugin manages two tables:

| Table | Purpose |
|-------|---------|
| `np_crdt_documents` | Current document state (engine, binary state blob, update count) |
| `np_crdt_updates` | Raw CRDT update log entries for compaction |

Both tables include `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation (Convention A).

## Compaction

Update rows accumulate as clients collaborate. Run compaction periodically to merge updates into the document state and free storage:

```bash
curl -X POST http://your-nself-host:8211/crdt/compact/my-document-id
```

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `CRDT_PORT` | No | `8211` | Service port |
| `CRDT_MAX_CONNECTIONS` | No | `100` | Max concurrent WebSocket connections |

## Routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/crdt/documents` | List tracked documents |
| `PUT` | `/crdt/automerge/:docId` | Upload Automerge state |
| `GET` | `/crdt/automerge/:docId` | Download Automerge state |
| `POST` | `/crdt/compact/:docId` | Compact update log into document state |
| `DELETE` | `/crdt/documents/:docId` | Delete document and all updates |
| `WS` | `/:docId` | Yjs WebSocket sync endpoint |

## Multi-Tenant Isolation

All document rows are scoped to `source_account_id`. Each nSelf app instance has its own isolated view of documents. Yjs rooms are further isolated by document ID, which should include a tenant prefix for multi-tenant deployments (e.g. `tenant-abc/document-123`).

## Bundle

This plugin is not included in a named bundle. Access is granted with any Pro license tier or ɳSelf+.

## See Also

- [Architecture: Microkernel](Architecture-Microkernel.md)
- [Plugin Reference](Plugin-Reference.md)
