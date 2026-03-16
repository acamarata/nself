# nself Changelog

## v0.9.9+max (Max tier plugin suite)

### nself-ai v2

- Multi-provider task routing: Classify→Phi (local), Summarize/FAQ→Gemini Flash, Chat/Code→OpenAI GPT-4o OAuth, Sensitive/Legal/Medical→Claude OAuth, LongContext→Gemini 2.5 Pro
- Caller token system: per-namespace authentication + rate limiting
- Gemini key pool: 3-key atomic round-robin with quota exhaustion tracking
- OAuth subscription tokens: AES-256-GCM encrypted storage for Claude Max + ChatGPT Plus
- External OpenAI-compatible API on port 18900 (opt-in via PLUGIN_AI_EXTERNAL_API=true)
- Priority queue per provider (4 levels: critical/high/normal/low)
- New CLI commands: `nself ai tokens create/list/remove/test`, `nself ai status`, `nself ai providers`

### nself-claw v2

- Hierarchical thread architecture: threads, messages, memory blocks, thread core
- 3-layer memory: recent messages (800 tokens), compressed memory blocks (600 tokens), thread core summary (400 tokens)
- pgvector similarity search for memory retrieval (0.7 cosine threshold)
- Persona management: CamClaw (full memory), nChat (session-only), ChatIslam (audience modes + escalation)
- Phi-4 orchestration pipeline: deterministic heuristics for compression + memory storage triggers
- POST /claw/message: full conversation flow with context assembly
- POST /claw/classify: content classification for mux integration
- New CLI commands: `nself claw chat`, `nself claw threads`, `nself claw persona`, `nself claw memory`

### nself-mux (updated)

- CompanionNotify action: send cards to ɳClaw companion app
- VoiceCall action: initiate Twilio calls from rules
- VoiceTts action: play TTS messages from rules
- use_claw_classify flag: route classification to nself-claw
- DLQ retry with exponential backoff (5min, 30min, 2hr)

### nself-voice (new)

- Piper local TTS: no API key required, subprocess-based
- ElevenLabs TTS: professional voice quality
- whisper.cpp STT: local speech-to-text
- Twilio telephony: inbound/outbound calls
- Wake word detection via rustpotter
- AI voicebot: voice + nself-claw + nself-ai pipeline
- New CLI commands: `nself voice tts`, `nself voice stt`, `nself voice call`, `nself voice wake`

### nself-browser (new)

- Playwright automation via Node.js worker (JSON IPC)
- Screenshot, PDF, scrape, JavaScript execution
- URL security policy: blocks private IPs, RFC-1918, SSRF vectors
- Stealth mode: anti-detection via playwright-extra
- ɳClaw BrowserResearch tool integration
- New CLI commands: `nself browser screenshot`, `nself browser scrape`, `nself browser pdf`, `nself browser allowlist`

---

## v0.9.9

Initial v0.9.9 release. See GitHub releases for full notes.
