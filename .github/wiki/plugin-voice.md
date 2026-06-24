# Voice Plugin

TTS, STT, telephony, wake word, and voicebot. Part of the ɳClaw bundle.

**Requires:** `nself_max_` license tier or higher. `nself license add nself_max_xxxxx...`

## Install

```bash
nself license add nself_max_xxxxx...
nself plugin install voice
```

## What It Does

Provides a unified REST API for voice processing:

- TTS: ElevenLabs, Piper (local), OpenAI
- STT: whisper.cpp (local), OpenAI Whisper
- Telephony: Twilio outbound calls, TwiML webhooks
- Voicebot: inbound call handler with AI reasoning
- Wake word: keyword detection (rustpotter, default "hey nself")
- nself-ai integration: chains voice input/output with the AI plugin

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PLUGIN_VOICE_PORT` | `3714` | Voice service port |
| `PLUGIN_VOICE_TTS_PROVIDER` | `piper` | TTS provider: `piper`, `elevenlabs`, `openai` |
| `PLUGIN_VOICE_STT_PROVIDER` | `whisper` | STT provider: `whisper`, `openai` |
| `PLUGIN_VOICE_PIPER_PATH` | — | Path to piper binary |
| `PLUGIN_VOICE_PIPER_MODEL_PATH` | — | Path to piper model file |
| `PLUGIN_VOICE_ELEVENLABS_API_KEY` | — | ElevenLabs API key |
| `PLUGIN_VOICE_ELEVENLABS_VOICE_ID` | — | ElevenLabs voice ID |
| `PLUGIN_VOICE_OPENAI_API_KEY` | — | OpenAI API key (TTS/STT) |
| `PLUGIN_VOICE_WHISPER_PATH` | — | Path to whisper.cpp binary |
| `PLUGIN_VOICE_WHISPER_MODEL` | — | Path to whisper model file |
| `PLUGIN_VOICE_TWILIO_ACCOUNT_SID` | — | Twilio account SID |
| `PLUGIN_VOICE_TWILIO_AUTH_TOKEN` | — | Twilio auth token |
| `PLUGIN_VOICE_TWILIO_DEFAULT_FROM` | — | Default caller ID |
| `PLUGIN_VOICE_BASE_URL` | — | Public base URL for TwiML webhooks |
| `PLUGIN_VOICE_BOT_MODE` | — | Voicebot mode: `ivr`, `ai`, etc. |
| `PLUGIN_VOICE_WAKE_WORD_KEYWORD` | `hey nself` | Wake word keyword |
| `DATABASE_URL` | — | Postgres connection string (optional; enables call tracking) |

## Port

| Port | Purpose |
|------|---------|
| 3714 | Voice service REST API |

## Database Tables

3 tables added to your Postgres database when `DATABASE_URL` is set:

| Table | Purpose |
|-------|---------|
| `np_voice_sessions` | Voice session records (TTS, STT, calls, wake word) |
| `np_voice_recordings` | Stored audio file references and transcripts |
| `np_voice_tts_cache` | TTS audio cache (content-addressed by text hash) |

All `np_voice_sessions` rows are isolated by `source_account_id` via Hasura row-level security.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/voice/` | Voice processing API (port 3714) |

## Endpoints (summary)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/tts` | Convert text to speech |
| POST | `/stt` | Transcribe audio to text |
| POST | `/calls` | Initiate outbound call |
| GET | `/calls/{sid}` | Get call status |
| DELETE | `/calls/{sid}` | Hang up call |
| POST | `/voicebot/handle` | Handle inbound TwiML webhook |
| POST | `/wake/start` | Start wake word listener |
| POST | `/wake/stop` | Stop wake word listener |
| GET | `/wake/status` | Wake word listener status |
| GET | `/health` | Health check |

---

[[Home]] | [[Plugin-Overview]] | [[bundle-claw]]
