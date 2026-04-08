# API Reference

This page documents the REST API endpoints exposed by nSelf plugins. These endpoints are internal to your nSelf stack and accessed via Nginx reverse proxy. All endpoints require authentication unless noted otherwise.

**Base URL:** `https://api.yourdomain.com` (or `http://localhost` in development)

---

## Table of Contents

- [Authentication](#authentication)
- [nself-claw (AI Assistant)](#nself-claw-ai-assistant)
- [nself-ai (LLM Routing)](#nself-ai-llm-routing)
- [nself-mux (Email Pipeline)](#nself-mux-email-pipeline)
- [nself-claw-budget (Budget Management)](#nself-claw-budget-budget-management)
- [nself-voice (TTS/STT/Calls)](#nself-voice-ttssttcalls)
- [nself-browser (Web Automation)](#nself-browser-web-automation)
- [nself-google (Google Services)](#nself-google-google-services)
- [nself-notify (Notifications)](#nself-notify-notifications)
- [nself-cron (Scheduled Jobs)](#nself-cron-scheduled-jobs)
- [nself-post (Social Publishing)](#nself-post-social-publishing)
- [Internal Plugin-to-Plugin](#internal-plugin-to-plugin)

---

## Authentication

Plugin REST endpoints use two auth mechanisms:

| Method | Header | When Used |
|--------|--------|-----------|
| JWT Bearer | `Authorization: Bearer {token}` | User-facing endpoints |
| Internal Token | `X-Internal-Token: {secret}` | Plugin-to-plugin communication |

Internal tokens are configured via `PLUGIN_INTERNAL_SECRET` in `.env.secrets`. Plugin-to-plugin calls (e.g., mux calling claw) use `MUX_CLAW_SHARED_SECRET` or the generic internal token.

---

## nself-claw (AI Assistant)

### Chat

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/claw/message` | Send a message to the AI assistant. Returns AI response with tool results. |
| `GET` | `/claw/threads` | List conversation threads for the authenticated user. |
| `GET` | `/claw/threads/{id}` | Get a specific thread with messages. |
| `DELETE` | `/claw/threads/{id}` | Delete a thread and its messages. |

### Memory

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/memories` | List stored memories. Supports type filter (`fact`, `preference`, `decision`, `pattern`, `relationship`). |
| `POST` | `/claw/memories` | Create a memory entry manually. |
| `DELETE` | `/claw/memories/{id}` | Delete a specific memory. |
| `POST` | `/claw/memories/bulk-delete` | Delete multiple memories by ID. |
| `GET` | `/claw/memories/export` | Export all memories as JSON. |
| `POST` | `/claw/memories/import` | Import memories from JSON. |
| `POST` | `/claw/memories/search` | Semantic search over memories (pgvector cosine similarity). |

### Personas

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/personas` | List all personas. |
| `POST` | `/claw/personas` | Create a new persona. |
| `PUT` | `/claw/personas/{id}` | Update a persona. |
| `DELETE` | `/claw/personas/{id}` | Delete a persona. |

### Tools

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/tools` | List all registered tools (native + manifest + integration). |

### Budget

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/budget` | Get current budget status (daily/monthly limits, current spend). |
| `POST` | `/claw/budget` | Set or update budget limits. |

### Email Integration

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/email/priorities` | Get email priority classifications (action_required, time_sensitive, fyi, junk). |

### Contacts

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/contacts` | List contacts. |
| `POST` | `/claw/contacts` | Create a contact. |
| `PUT` | `/claw/contacts/{id}` | Update a contact. |
| `DELETE` | `/claw/contacts/{id}` | Delete a contact. |
| `GET` | `/claw/contacts/{id}/vcard` | Export contact as vCard 3.0. |

### Integrations

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/integrations` | List configured integrations. |
| `POST` | `/claw/integrations` | Create an integration (with SSRF validation on base_url). |
| `PUT` | `/claw/integrations/{id}` | Update an integration. |
| `DELETE` | `/claw/integrations/{id}` | Delete an integration. |
| `POST` | `/claw/integrations/{id}/test` | Test integration health. |

### Transactions

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/transactions` | Query financial transactions. Filters: keyword, category, date range, user_id. |
| `GET` | `/claw/transactions/summary` | 6-month totals, top 5 vendors (30d), category breakdown. |

### Proactive

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/proactive/log` | Get proactive notification history. |

### Goals

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/goals` | List goals. |
| `POST` | `/claw/goals` | Create a goal. |
| `PUT` | `/claw/goals/{id}` | Update a goal. |
| `DELETE` | `/claw/goals/{id}` | Delete a goal. |

### Health & Monitoring

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/health/monitoring` | Check Prometheus, Loki, Alertmanager health. |

### Pairing

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/claw/pair/open` | Open pairing window (internal auth via X-Internal-Secret). |
| `GET` | `/claw/pair/status` | Check pairing window status. |
| `GET` | `/claw/pair` | Get pairing code (404 when window closed). |
| `POST` | `/claw/pair/direct` | Direct pairing with HMAC-SHA256 4-word phrase. |
| `GET` | `/claw/pair/direct/status` | Check direct pairing status. |

### Onboarding

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/onboarding/status` | Check onboarding completion. |
| `POST` | `/claw/onboarding/answer` | Submit an onboarding answer. |
| `POST` | `/claw/onboarding/restart` | Restart onboarding flow. |

### Usage

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/usage` | 30-day AI usage by model (tokens, cost, routed_by). |

### Telegram Webhook

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/claw/telegram/webhook` | Telegram bot webhook handler. Gated by CLAW_TG_ALLOWED_USERS. |

### Memory Rooms (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/memory/rooms` | List memory rooms with counts and health scores. |
| `POST` | `/claw/memory/rooms` | Create a named memory room. |
| `PUT` | `/claw/memory/rooms/{id}` | Update room name or ordering. |
| `DELETE` | `/claw/memory/rooms/{id}` | Delete room (memories return to default). |
| `POST` | `/claw/memory/rooms/{id}/assign` | Assign memory IDs to a room. |

### Knowledge (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/knowledge/health` | Brain health check report (stale, orphan, duplicate detection). |
| `GET` | `/claw/knowledge/stats` | Knowledge graph statistics (node count, edges, avg connections). |
| `GET` | `/claw/knowledge/export` | Export knowledge graph. Query param `format=obsidian` or `format=json`. |
| `GET` | `/claw/ingestion/queue` | List ingestion pipeline jobs. |
| `POST` | `/claw/ingestion/start` | Start ingestion from URL or file. |
| `DELETE` | `/claw/ingestion/{id}` | Cancel an ingestion job. |

### Decisions (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/decisions` | List recorded decisions. |
| `POST` | `/claw/decisions` | Record a new decision with context. |
| `PUT` | `/claw/decisions/{id}` | Update decision outcome. |

### Routing Rules (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/routing/rules` | List smart routing rules. |
| `POST` | `/claw/routing/rules` | Create a routing rule. |
| `PUT` | `/claw/routing/rules/{id}` | Update a routing rule. |
| `DELETE` | `/claw/routing/rules/{id}` | Delete a routing rule. |

### Image Generation (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/claw/image/generate` | Generate image from text prompt. Supports provider override. |
| `GET` | `/claw/image/jobs` | List image generation jobs and statuses. |
| `GET` | `/claw/image/jobs/{id}` | Get specific job result with image URL. |

### Model & Sessions (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/model/recommend` | Get model recommendation for a given query type. |
| `POST` | `/claw/sessions/{id}/features` | Toggle session feature flags. |
| `POST` | `/claw/sessions/{id}/summarize` | Compress and summarize session context. |

### Marketplace (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/marketplace/personas` | Browse community personas (no auth required). |
| `POST` | `/claw/marketplace/personas` | Publish a persona to the marketplace. |
| `POST` | `/claw/marketplace/personas/{id}/install` | Install a marketplace persona. |

### Agent Dashboard (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/agents/dashboard` | Agent metrics with sparkline data for all personas. |
| `GET` | `/claw/agents/status` | Live per-agent status (active, idle, error). |
| `GET` | `/claw/crews` | List agent crew configurations. |
| `POST` | `/claw/crews` | Create an agent crew. |

### Habits & Career (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/habits` | List habit definitions. |
| `POST` | `/claw/habits` | Create a habit. |
| `PUT` | `/claw/habits/{id}` | Update habit settings. |
| `POST` | `/claw/habits/{id}/log` | Log daily habit completion. |
| `GET` | `/claw/tutor/cards` | List tutor flashcards. |
| `POST` | `/claw/tutor/quiz` | Start a quiz session. |
| `GET` | `/claw/career/cv` | Get structured CV data. |
| `PUT` | `/claw/career/cv` | Update CV data. |
| `GET` | `/claw/career/suggestions` | Get career suggestions. |

### Health Coach (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/health/checkin` | Health coach check-in. |
| `GET` | `/claw/health/summary` | Health metrics summary. |

### Reflections (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/reflections` | List daily reflections. |
| `POST` | `/claw/reflections` | Create a reflection entry. |

### Prompts & Workflows (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/prompts` | List prompt library templates. |
| `POST` | `/claw/prompts` | Create a prompt template. |
| `POST` | `/claw/prompts/{id}/chain` | Chain prompts for sequential execution. |
| `GET` | `/claw/workflows/templates` | List workflow templates. |
| `POST` | `/claw/workflows/templates` | Create a workflow template. |
| `POST` | `/claw/workflows/run` | Execute a workflow. |

### Project Files (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/projects/files` | List project files. |
| `POST` | `/claw/projects/files` | Upload a project file. |
| `DELETE` | `/claw/projects/files/{id}` | Delete a project file. |

### History & Gallery (Phase 79)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw/history` | Conversation history with search and date filters. |
| `GET` | `/claw/gallery` | Image gallery with generated images. |

---

## nself-ai (LLM Routing)

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/ai/complete` | Route a completion request to the best model (4-tier routing). |
| `POST` | `/ai/embeddings` | Generate vector embeddings for text. |
| `GET` | `/ai/status` | Provider health and routing status. |
| `GET` | `/ai/providers` | List configured AI providers and their status. |
| `GET` | `/ai/tokens` | List caller tokens (per-namespace auth). |
| `POST` | `/ai/tokens` | Create a caller token. |
| `DELETE` | `/ai/tokens/{id}` | Revoke a caller token. |
| `GET` | `/ai/accounts` | List OAuth AI accounts. |
| `POST` | `/ai/accounts/refresh` | Force refresh OAuth tokens. |
| `POST` | `/ai/email-event` | Receive forwarded email metadata from mux rules. |

**External API (port 18900):** OpenAI-compatible API for third-party integrations. Uses caller tokens for auth.

---

## nself-mux (Email Pipeline)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/mux/messages` | Query email history. Filters: ILIKE search on from/subject/body, date range, account. Internal auth. |
| `POST` | `/mux/tokens/import` | Import OAuth refresh token for a Gmail account. Internal auth. |
| `GET` | `/mux/auto-reply/audit` | Auto-reply audit log. Filter by rule. |
| `POST` | `/mux/pending-replies/{id}/approve` | Approve a pending auto-reply draft. |
| `POST` | `/mux/pending-replies/{id}/discard` | Discard a pending auto-reply draft. |

---

## nself-claw-budget (Budget Management)

The claw-budget plugin provides standalone budget management that works independently of the claw plugin:

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/claw-budget/status` | Current spend vs limits (daily + monthly). |
| `POST` | `/claw-budget/limits` | Set daily/monthly token budget limits. |
| `GET` | `/claw-budget/history` | Spend history over time. |

---

## nself-voice (TTS/STT/Calls)

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/voice/tts` | Text-to-speech. Providers: Piper (local), ElevenLabs. |
| `POST` | `/voice/stt` | Speech-to-text via whisper.cpp. |
| `POST` | `/voice/call/outbound` | Initiate outbound call via Twilio. |
| `GET` | `/voice/call/status` | Check call status. |
| `GET` | `/voice/routing/status` | Current LLM routing tier (standard vs realtime). |
| `POST` | `/voice/routing/set-tier` | Set voice LLM routing tier. |

---

## nself-browser (Web Automation)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/browser/tabs` | List open browser tabs. |
| `POST` | `/browser/action` | Execute browser action. Actions: navigate, click, type, extract, screenshot, wait_for, evaluate, get_cookies. |
| `GET` | `/browser/allowlist` | List allowed URLs. |
| `POST` | `/browser/allowlist` | Add URL to allowlist. |

SSRF protection blocks all RFC-1918 and private IP addresses.

---

## nself-google (Google Services)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/google/gmail/messages` | Search Gmail messages. |
| `GET` | `/google/gmail/message/{id}` | Get full email content. |
| `GET` | `/google/calendar/events` | List calendar events. |
| `POST` | `/google/calendar/events` | Create a calendar event. |
| `GET` | `/google/calendar/freebusy` | Find free time slots. |
| `GET` | `/google/drive/files` | List Drive files. |
| `POST` | `/google/sheets/append` | Append row to a Google Sheet. |
| `GET` | `/google/sheets/read` | Read data from a Google Sheet. |

Primary account resolution: `GOOGLE_PRIMARY_USER_ID` env var or first database row.

---

## nself-notify (Notifications)

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/notify/send` | Send notification via configured channels. |
| `GET` | `/notify/channels` | List notification channels. |
| `POST` | `/notify/channels` | Create a channel (Telegram, webhook). |
| `DELETE` | `/notify/channels/{id}` | Delete a channel. |

---

## nself-cron (Scheduled Jobs)

| Method | Path | Description |
|--------|------|------------|
| `GET` | `/cron/jobs` | List scheduled jobs. |
| `POST` | `/cron/jobs` | Create a cron job (timezone-aware, DB persistence). |
| `DELETE` | `/cron/jobs/{id}` | Delete a job. |
| `POST` | `/cron/jobs/{id}/run` | Trigger a job immediately. |

---

## nself-post (Social Publishing)

| Method | Path | Description |
|--------|------|------------|
| `POST` | `/post/publish` | Publish to one or more platforms (immediate or scheduled). |
| `GET` | `/post/accounts` | List connected publishing accounts. |
| `GET` | `/post/queue` | Get scheduled post queue. |

Supported platforms: WordPress, Ghost, Twitter, Telegram channel, Dev.to, Hashnode.

---

## Internal Plugin-to-Plugin

These endpoints use `X-Internal-Token` auth and are not meant for direct user access.

| Method | Path | Plugin | Description |
|--------|------|--------|------------|
| `POST` | `/internal/classify` | claw | Classify a message (used by mux). Auth: MUX_CLAW_SHARED_SECRET. |
| `GET` | `/internal/health` | all | Plugin health check. Returns 200 if healthy. |

### Auto-Generated Internal URLs

When plugins are installed, the CLI auto-generates `PLUGIN_{NAME}_INTERNAL_URL` environment variables:

| Variable | Default |
|----------|---------|
| `PLUGIN_AI_INTERNAL_URL` | `http://nself-ai:18900` |
| `PLUGIN_CLAW_INTERNAL_URL` | `http://nself-claw:18901` |
| `PLUGIN_MUX_INTERNAL_URL` | `http://nself-mux:18902` |
| `PLUGIN_VOICE_INTERNAL_URL` | `http://nself-voice:18903` |
| `PLUGIN_BROWSER_INTERNAL_URL` | `http://nself-browser:18904` |
| `PLUGIN_GOOGLE_INTERNAL_URL` | `http://nself-google:18905` |
| `PLUGIN_NOTIFY_INTERNAL_URL` | `http://nself-notify:18906` |
| `PLUGIN_CRON_INTERNAL_URL` | `http://nself-cron:18907` |

Plugins use these URLs to communicate with each other over the Docker bridge network.

---

## Error Responses

All endpoints return standard HTTP status codes:

| Code | Meaning |
|------|---------|
| 200 | Success |
| 400 | Bad request (validation error) |
| 401 | Unauthorized (missing or invalid token) |
| 403 | Forbidden (insufficient permissions or tier) |
| 404 | Not found |
| 429 | Rate limited |
| 500 | Internal server error |

Error body format:

```json
{
  "error": "human-readable message",
  "code": "MACHINE_READABLE_CODE"
}
```

---

## Related Pages

- [[Feature-nClaw]] -- nClaw feature overview
- [[Plugin-Architecture]] -- how plugins are structured
- [[Security-Architecture]] -- auth model and rate limiting
- [[Config-Env-Vars]] -- environment variable reference

---

← [[Home]] | [[_Sidebar]]
