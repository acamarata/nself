# Upgrading to Pro Plugin Suite v2

This guide covers upgrading nself-ai, nself-claw, nself-mux, nself-voice, and nself-browser from v1 to v2.

---

## What Changed

| Plugin | v1 | v2 |
| --- | --- | --- |
| nself-ai | Single provider, no authentication | Multi-provider, caller tokens, task class routing |
| nself-claw | Session-based conversations, no memory persistence | Thread-based, 3-layer memory, personas |
| nself-mux | Basic rule actions (forward, tag, webhook, classify) | + CompanionNotify, VoiceCall, VoiceTts, claw classifier routing |
| nself-voice | Basic TTS and STT (ElevenLabs, Piper) | + Twilio telephony, wake word, whisper.cpp, voicebot integration |
| nself-browser | Basic scraping and screenshots | + Stealth mode, claw BrowserResearch tool, browser pool |

---

## Upgrade Steps

### nself-ai

No database migration needed. New tables added by v2 are additive — existing data is untouched.

```bash
nself plugin update ai
nself restart
```

After restart, create caller tokens for any services that call ai:

```bash
nself ai token create --namespace mux --rate-limit 2000/hour
nself ai token create --namespace claw --rate-limit 10000/hour
```

Store the tokens in the relevant service `.env` files before restarting those services.

New env vars to add to `.env`:

```bash
# Multi-provider keys (add whichever you have)
GEMINI_FREE_KEY_1=AIza...
GEMINI_FREE_KEY_2=AIza...
GEMINI_FREE_KEY_3=AIza...
GEMINI_API_KEY_OPENCLAW=AIza...           # paid, for LongContext tasks
PLUGIN_AI_ANTHROPIC_OAUTH_TOKEN_1=...    # Claude OAuth
PLUGIN_AI_ANTHROPIC_OAUTH_TOKEN_2=...
PLUGIN_AI_OPENAI_OAUTH_TOKEN=...         # OpenAI OAuth
```

See [ai-setup.md](./ai-setup.md) for full provider configuration.

---

### nself-claw

claw v2 changes the data model from sessions to threads. The migration converts v1 sessions automatically.

**Warning:** Embeddings for memory blocks regenerate after migration. This runs in the background and can take several minutes for large conversation histories. claw remains available during this time, but memory search may be incomplete until regeneration finishes.

```bash
nself plugin update claw
nself claw migrate-v1
nself restart
```

`nself claw migrate-v1` converts v1 session records to threads, preserving all message history. The migration is one-way — there is no automatic rollback for the data migration step.

New env var required:

```bash
PLUGIN_CLAW_AI_TOKEN=nself_ai_tok_claw_xxxxx  # from: nself ai token create --namespace claw
```

---

### nself-mux

No migration needed. v2 adds new action types — existing rules continue to work unchanged.

```bash
nself plugin update mux
nself restart
```

New env var required for AI-dependent actions:

```bash
PLUGIN_MUX_AI_TOKEN=nself_ai_tok_mux_xxxxx  # from: nself ai token create --namespace mux
```

Without this token, `ai_classify`, `CompanionNotify`, `VoiceCall`, and `VoiceTts` actions log an error and are skipped. Other rule actions continue to work.

---

### nself-voice

nself-voice is a new plugin in v2 — there is no v1 to migrate from. Install it fresh.

```bash
nself plugin install voice
```

Register it as a custom service and configure Piper, ElevenLabs, and Twilio as needed. See [voice-setup.md](./voice-setup.md) for full configuration.

---

### nself-browser

nself-browser is also a new plugin in v2. Install fresh.

```bash
nself plugin install browser
```

See [browser-setup.md](./browser-setup.md) for configuration. Docker is required.

---

## Rollback

If you need to revert a plugin to its previous version:

```bash
nself plugin rollback ai
nself plugin rollback claw
nself plugin rollback mux
```

`nself plugin rollback` restores the plugin code to the previously installed version and restarts the service.

**Note for claw rollback:** The data migration (`nself claw migrate-v1`) is not reversed by rollback. If you roll back claw after running the migration, v1 code runs against a v2 schema. This may cause errors. To fully roll back claw to v1 state, restore your database from a backup taken before the migration.

To create a backup before upgrading claw:

```bash
nself db backup
```

---

## Expected Downtime

Each plugin restart takes 30–60 seconds. During restart, the service is unavailable. The rest of the nself stack (Postgres, Hasura, Auth, Nginx) continues running.

For minimum disruption, upgrade plugins one at a time rather than all at once.

Recommended order:
1. `ai` — other plugins depend on it
2. `claw` — depends on ai
3. `mux` — depends on ai and optionally voice
4. `voice` — standalone
5. `browser` — standalone

---

## Post-Upgrade Verification

After all upgrades, verify each plugin is healthy:

```bash
# Check all custom services are running
nself status

# Check each plugin health endpoint
curl -s http://127.0.0.1:3101/health  # ai
curl -s http://127.0.0.1:3102/health  # claw
curl -s http://127.0.0.1:3102/health  # mux (adjust port if different CS slot)
curl -s http://127.0.0.1:3103/health  # voice
curl -s http://127.0.0.1:3104/health  # browser
```

Each should return `{"status":"ok","plugin":"<name>"}`.

Check caller tokens are working:

```bash
nself ai token list
```

Check claw threads are accessible:

```bash
nself claw threads list --user <your_user_id>
```

If any service returns an error, check its logs:

```bash
docker logs <container_name> --tail 50
```

---

## Related

- [nself-ai Setup](./ai-setup.md)
- [nself-claw Setup](./claw-setup.md)
- [nself-mux Setup](./mux-setup.md)
- [nself-voice Setup](./voice-setup.md)
- [nself-browser Setup](./browser-setup.md)
- [Pro Plugin Setup](./pro-plugin-setup.md)
