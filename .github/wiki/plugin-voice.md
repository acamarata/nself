# Voice Plugin

> Speech-to-text, text-to-speech, and voicebot with multi-provider support. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install voice
```

## What It Does

Provides a unified REST API for voice processing: transcribes audio to text (STT via Whisper and other providers) and converts text to speech (TTS). Includes a voicebot mode that chains STT → AI reasoning → TTS for voice-interactive applications. Records and stores voice sessions.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `VOICE_PORT` | `3714` | Voice service port |
| `VOICE_STT_PROVIDER` | `whisper` | STT provider: `whisper`, `deepgram`, `assembly` |
| `VOICE_TTS_PROVIDER` | `openai` | TTS provider: `openai`, `elevenlabs`, `azure` |
| `OPENAI_API_KEY` | — | Required for OpenAI STT/TTS |
| `DEEPGRAM_API_KEY` | — | Deepgram API key (optional) |
| `ELEVENLABS_API_KEY` | — | ElevenLabs API key (optional) |
| `VOICE_RECORDING_ENABLED` | `true` | Store audio recordings |

## Ports

| Port | Purpose |
|------|---------|
| 3714 | Voice service REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_voice_sessions`, voice session records
- `np_voice_transcripts`, STT transcription results
- `np_voice_recordings`, stored audio references

## Nginx Routes

| Route | Target |
|-------|--------|
| `/voice/` | Voice processing API |
