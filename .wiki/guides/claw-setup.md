# Setting Up nself-claw

nself-claw is a Max tier plugin that adds a persistent, memory-backed AI assistant to your backend. It requires nself-ai. Install and configure ai first before following this guide.

---

## Prerequisites

- nself v0.9.9+ installed
- Max license key (Max or Enterprise tier)
- nself-ai installed and running
- At least one AI provider configured in ai (Claude OAuth recommended for best results)

---

## Quick Start

Install the plugin:

```bash
nself plugin install ai claw
```

Register claw as a custom service in `.env`. Assign it the next available CS slot after ai:

```bash
CS_3=claw:express-ts:3102
CS_3_ROUTE=claw
CS_3_PUBLIC=false
CS_3_HEALTHCHECK=/health
CS_3_REPLICAS=1
CS_3_MEMORY=1G
CS_3_CPU=0.5
```

Copy service files and set the required token:

```bash
cp -r ~/.nself/plugins/claw/ts/ services/claw/

# Create a caller token for claw to talk to ai
nself ai token create --namespace claw --rate-limit 10000/hour
# Save the returned token:
# PLUGIN_CLAW_AI_TOKEN=nself_ai_tok_claw_xxxxx
```

Add the token to `.env`:

```bash
PLUGIN_CLAW_AI_TOKEN=nself_ai_tok_claw_xxxxxxxxxxxxx
```

Rebuild and start:

```bash
nself build
docker compose up -d claw
```

Verify:

```bash
curl -s http://127.0.0.1:3102/health
# {"status":"ok","plugin":"claw"}
```

---

## Thread Model

claw organizes conversations into a four-layer hierarchy:

```
Thread
  └── Messages       (individual exchanges)
        └── Memory Blocks   (compressed past conversation segments)
              └── Thread Core  (rolling summary of the whole thread)
```

A **thread** is one ongoing conversation — like a chat session that persists between app launches. Threads belong to a user.

**Messages** are the individual back-and-forth exchanges within a thread.

**Memory blocks** are compressed summaries of older message groups. When a thread grows long, claw compresses old messages into blocks so they stop consuming context tokens. The original messages are kept in the database, but claw loads blocks instead when building context.

The **thread core** is a ~400-token summary of the thread's overall arc — what the user is working on, their preferences, and recurring themes. It updates periodically as the thread evolves.

---

## 3-Layer Memory

When claw assembles context for a new message, it pulls from three layers:

### Layer 1: Recent Messages

The last ~800 tokens of raw messages. These are the most recent exchanges, included verbatim. No compression.

### Layer 2: Memory Blocks

Older conversation segments, compressed. When you need to recall something from earlier in the thread, claw runs a vector similarity search against all memory blocks and pulls the most relevant ones into context. Only the top-matching blocks are included — not all of them.

This lets threads grow indefinitely without hitting context limits. Past conversations become searchable rather than simply truncated.

### Layer 3: Thread Core

A rolling summary of the whole thread, always included. ~400 tokens. It covers the user's long-term goals, preferences, and anything claw should always remember about this particular thread.

### Context Budget

Total: **2,500 tokens**

| Layer | Allocation |
| --- | --- |
| Thread core | ~400 tokens |
| Relevant memory blocks | ~1,300 tokens (varies) |
| Recent messages | ~800 tokens |

The system prompt and persona instructions consume the remaining headroom before the model's full context window.

---

## Personas

A persona defines how claw behaves in a thread. Each persona has a name, a system prompt, and optional capability flags (e.g., whether it can use the browser tool).

### Built-In Personas

Three personas ship by default:

| Persona | Description | Default for |
| --- | --- | --- |
| `CamClaw` | General-purpose assistant. Technical, direct, minimal filler. | nself admin / internal |
| `nChat` | Friendly conversational assistant. Supports the nChat app. | chat.nself.org |
| `ChatIslam` | Islamic knowledge assistant. Trained on fiqh and hadith references. | chatislam.com |

### Creating a Custom Persona

```bash
nself claw persona create \
  --name "SupportBot" \
  --description "Customer support for acme.com" \
  --system-prompt "You are a support agent for Acme. Be concise and helpful. When you do not know the answer, say so and offer to escalate." \
  --enable-browser false \
  --enable-web-search false
```

List personas:

```bash
nself claw persona list
```

Assign a persona to a new thread via the API:

```json
{
  "user_id": "uuid",
  "persona_id": "supportbot",
  "title": "My first thread"
}
```

---

## CLI Usage

### Start an Interactive Session

```bash
nself claw chat
```

This opens a REPL connected to the running claw service. Threads persist between sessions.

### Thread Management

```bash
# List all threads for a user
nself claw threads list --user <user_id>

# Resume a specific thread
nself claw chat --thread <thread_id>

# Delete a thread (and all its messages and memory)
nself claw threads delete <thread_id>
```

### Memory Operations

```bash
# View memory blocks for a thread
nself claw memory list --thread <thread_id>

# Force a memory compaction (normally runs automatically)
nself claw memory compact --thread <thread_id>

# View the current thread core
nself claw memory core --thread <thread_id>
```

---

## API Reference

All endpoints require a caller token in `Authorization: Bearer <token>`.

### POST /claw/message

Send a message and get a reply.

```json
{
  "thread_id": "uuid",
  "user_id": "uuid",
  "content": "What did we discuss about the database schema?"
}
```

Response:

```json
{
  "message_id": "uuid",
  "content": "We discussed moving the users table to...",
  "thread_id": "uuid",
  "tokens_used": 1842
}
```

### POST /claw/threads

Create a new thread.

```json
{
  "user_id": "uuid",
  "persona_id": "CamClaw",
  "title": "Project planning"
}
```

### GET /claw/threads

List threads for a user.

Query params: `user_id` (required), `limit` (default 20), `offset` (default 0).

### GET /claw/threads/:thread_id

Get a thread with its recent messages and thread core.

### DELETE /claw/threads/:thread_id

Delete a thread and all associated data.

### GET /claw/personas

List available personas.

### POST /claw/personas

Create a persona.

---

## Migrating from v1

If you ran nself-claw v1 (session-based, no persistent memory), run the migration after updating:

```bash
nself plugin update claw
nself claw migrate-v1
nself restart
```

The migration converts v1 sessions to threads. Existing conversation history moves into the thread model. Embeddings for memory blocks regenerate in the background — this can take a few minutes for large histories.

See the [upgrade guide](./upgrading-to-max-v2.md) for full details.

---

## Troubleshooting

### "AI plugin not reachable"

claw calls the ai plugin internally. Confirm ai is running:

```bash
curl -s http://127.0.0.1:3101/health
```

Then confirm the claw service has the correct AI endpoint configured:

```bash
docker exec <claw_container> env | grep PLUGIN_CLAW_AI_TOKEN
```

### "Invalid or missing caller token"

claw needs `PLUGIN_CLAW_AI_TOKEN` set in `.env`. Create a token if you have not:

```bash
nself ai token create --namespace claw --rate-limit 10000/hour
```

Add the result to `.env` and restart claw.

### Thread context feels wrong or truncated

Check memory block health for the thread:

```bash
nself claw memory list --thread <thread_id>
```

If blocks are not being created, the compaction background job may have stalled. Force a compaction:

```bash
nself claw memory compact --thread <thread_id>
```

### "Persona not found"

The persona ID does not exist. List what is available:

```bash
nself claw persona list
```

Persona IDs are case-sensitive.

---

## Related

- [nself-ai Setup](./ai-setup.md)
- [nself-mux Setup](./mux-setup.md)
- [nself-voice Setup](./voice-setup.md)
- [Pro Plugin Setup](./pro-plugin-setup.md)
