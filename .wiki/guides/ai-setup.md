# Setting Up nself-ai

nself-ai is a Pro tier plugin that provides a multi-provider AI gateway for your self-hosted backend. It routes requests to the best available model based on the task type, with fallback support and caller-level access control.

---

## Prerequisites

- nself v0.9.9+ installed
- Max license key (`nself_pro_` prefix, Max or Enterprise tier)
- At least one AI provider configured (see below)

---

## Quick Start

Install the plugin:

```bash
nself plugin install ai
```

Register it as a custom service in `.env`:

```bash
CS_2=ai:express-ts:3101
CS_2_ROUTE=ai
CS_2_PUBLIC=false
CS_2_HEALTHCHECK=/health
CS_2_REPLICAS=1
CS_2_MEMORY=2G
CS_2_CPU=1.0
```

Copy the service files and rebuild:

```bash
cp -r ~/.nself/plugins/ai/ts/ services/ai/
nself build
docker compose up -d ai
```

Verify it started:

```bash
curl -s http://127.0.0.1:3101/health
# {"status":"ok","plugin":"ai","providers":["gemini","openai"]}
```

---

## Provider Configuration

nself-ai supports four provider types. You do not need all of them. Configure what you have.

### Gemini Free Pool

Three free Gemini API keys share the load on Classify, Summarize, and FAQ tasks. This keeps costs near zero for high-volume categorization work.

```bash
# In .env
GEMINI_FREE_KEY_1=AIza...
GEMINI_FREE_KEY_2=AIza...
GEMINI_FREE_KEY_3=AIza...
```

Get free keys at [aistudio.google.com](https://aistudio.google.com). Each key has its own rate limit, so three keys give you roughly 3x the throughput.

If only one key is set, all Gemini free-tier tasks route to that key.

### Gemini Pro

For tasks that need a long context window (up to 1M tokens), set a paid Gemini key:

```bash
GEMINI_API_KEY_OPENCLAW=AIza...
```

This key handles `LongContext` tasks only. It does not compete with the free pool.

### Claude OAuth

Claude Max subscription tokens allow you to use Claude without paying per-token. Two tokens share load:

```bash
PLUGIN_AI_ANTHROPIC_OAUTH_TOKEN_1=...
PLUGIN_AI_ANTHROPIC_OAUTH_TOKEN_2=...
```

These handle `Sensitive` tasks — content moderation, safety classification, and anything that should not go to a third-party API by default.

To get a Claude OAuth token: open Claude.ai, go to Settings, and generate an API token under your Max plan.

### OpenAI OAuth

A ChatGPT Plus subscription token for Chat tasks:

```bash
PLUGIN_AI_OPENAI_OAUTH_TOKEN=...
```

This routes `Chat` tasks to GPT-4o. If this token is not set, chat falls back to Gemini Flash.

---

## Task Class Routing

Each request carries a `task_class` field. The plugin routes it to the appropriate model:

| Task Class | Default Model | Fallback |
| --- | --- | --- |
| `Classify` | Phi (local, if available) | Gemini Flash (free pool) |
| `Summarize` | Gemini Flash (free pool) | Gemini Flash (paid) |
| `FAQ` | Gemini Flash (free pool) | Gemini Flash (paid) |
| `Chat` | OpenAI GPT-4o | Gemini Flash |
| `Sensitive` | Claude (OAuth) | Gemini Flash |
| `LongContext` | Gemini 2.5 Pro (paid) | None — fails if key missing |
| `Embed` | OpenAI text-embedding-3-small | None — requires OpenAI key |

Phi local inference is optional. If `PLUGIN_AI_PHI_MODEL_PATH` is not set, Classify routes to the free Gemini pool.

To override routing for a specific request, pass `provider` explicitly:

```json
{
  "task_class": "Chat",
  "provider": "anthropic",
  "messages": [{"role": "user", "content": "Hello"}]
}
```

---

## Caller Token System

Caller tokens let you control which services can call the AI plugin and what they can do. Each token has a namespace and an optional rate limit.

### What They Are

A caller token is a bearer token that your other services pass in `Authorization: Bearer <token>`. Without a valid token, the AI plugin returns 401.

Tokens belong to a namespace (e.g., `mux`, `claw`, `browser`). The namespace is used for rate limiting and audit logging.

### Creating Tokens

```bash
nself ai token create --namespace mux --rate-limit 1000/hour
# Returns: nself_ai_tok_mux_xxxxxxxxxxxxx
```

Store the token in the calling service's `.env`:

```bash
# In the mux service .env
PLUGIN_MUX_AI_TOKEN=nself_ai_tok_mux_xxxxxxxxxxxxx
```

### Listing Tokens

```bash
nself ai token list
```

### Revoking a Token

```bash
nself ai token revoke nself_ai_tok_mux_xxxxxxxxxxxxx
```

Revoked tokens are rejected immediately — no restart required.

---

## External OpenAI-Compatible API

nself-ai exposes an OpenAI-compatible API on port 18900. Any client that works with the OpenAI API can point to this endpoint instead.

Enable it:

```bash
# In .env
PLUGIN_AI_EXTERNAL_API=true
PLUGIN_AI_EXTERNAL_API_PORT=18900
```

Rebuild and restart:

```bash
nself build
docker compose up -d ai
```

Client usage (Python example):

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://your-server:18900/v1",
    api_key="nself_ai_tok_yourtoken"
)

response = client.chat.completions.create(
    model="auto",  # nself-ai picks the model based on task_class
    messages=[{"role": "user", "content": "Summarize this article..."}]
)
```

The `model` field accepts `auto` (let nself-ai route) or a specific provider name (`gemini-flash`, `gpt-4o`, `claude-sonnet`).

Port 18900 is only accessible from the host machine by default. To expose it through nginx, add a `conf.d` rule — see [nginx Configuration](../configuration/nginx.md).

---

## Troubleshooting

### "No provider available for task class X"

The required API key is not set for that task class. Check which provider handles the task class in the routing table above, then add the missing key to `.env` and restart the service:

```bash
docker compose up -d --force-recreate ai
```

### "Invalid caller token"

The caller token is missing, malformed, or revoked. Check that the calling service has the correct token in its environment:

```bash
docker exec <service_container> env | grep AI_TOKEN
```

If the token is correct, check that it has not been revoked:

```bash
nself ai token list
```

### "Rate limit exceeded for namespace X"

The namespace has hit its hourly or daily limit. Either wait for the window to reset, or increase the limit:

```bash
nself ai token update nself_ai_tok_mux_xxx --rate-limit 5000/hour
```

### "LongContext task failed — no provider"

`LongContext` tasks require `GEMINI_API_KEY_OPENCLAW` (a paid Gemini key). Free keys do not support the 1M-token context window. Set the key and restart the service.

### Service starts but health check fails

Memory may be too low. The AI service loads model routing config at startup and needs at least 512MB. Set `CS_2_MEMORY=2G` in `.env`, then:

```bash
nself build
docker compose up -d ai
```

---

## Related

- [nself-claw Setup](./claw-setup.md)
- [nself-mux Setup](./mux-setup.md)
- [Pro Plugin Setup](./pro-plugin-setup.md)
- [Custom Services Reference](../configuration/custom-services.md)
