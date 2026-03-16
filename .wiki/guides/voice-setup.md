# Setting Up nself-voice

nself-voice adds text-to-speech, speech-to-text, telephony, and wake word detection to your backend. It can work entirely offline with Piper TTS and whisper.cpp, or connect to ElevenLabs and Twilio for higher quality and telephony support.

---

## Prerequisites

- nself v0.9.9+ installed
- Max license key (Max or Enterprise tier)
- Docker running on the host

---

## Quick Start with Piper (Offline TTS)

Piper is a fast, local TTS engine. No API key needed.

### 1. Install Piper

Download the binary for your platform from [github.com/rhasspy/piper/releases](https://github.com/rhasspy/piper/releases). Place it somewhere on your PATH:

```bash
# Linux
sudo mv piper /usr/local/bin/piper
chmod +x /usr/local/bin/piper
```

Download a voice model. The `en_US-lessac-medium` model is a good starting point:

```bash
mkdir -p /opt/piper/models
cd /opt/piper/models

# Download model + config
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json
```

### 2. Configure

```bash
# In .env
PLUGIN_VOICE_PIPER_BINARY=/usr/local/bin/piper
PLUGIN_VOICE_PIPER_MODEL_PATH=/opt/piper/models/en_US-lessac-medium.onnx
```

### 3. Install and Start the Plugin

```bash
nself plugin install voice
```

Register as a custom service:

```bash
CS_4=voice:express-ts:3103
CS_4_ROUTE=voice
CS_4_PUBLIC=false
CS_4_HEALTHCHECK=/health
CS_4_REPLICAS=1
CS_4_MEMORY=512M
CS_4_CPU=0.5
```

```bash
cp -r ~/.nself/plugins/voice/ts/ services/voice/
nself build
docker compose up -d voice
```

### 4. Test

```bash
nself voice tts "Hello from nself voice"
# Plays audio or saves to voice_output.wav
```

---

## ElevenLabs Setup

ElevenLabs provides higher quality voices with more expressiveness.

```bash
# In .env
PLUGIN_VOICE_ELEVENLABS_API_KEY=sk_...
PLUGIN_VOICE_ELEVENLABS_VOICE_ID=21m00Tcm4TlvDq8ikWAM
```

Find voice IDs in the ElevenLabs dashboard under Voices. The ID above is the default "Rachel" voice.

When both Piper and ElevenLabs are configured, ElevenLabs takes priority. To use Piper for a specific request, pass `provider: "piper"` in the API call.

Restart the service after setting keys:

```bash
docker compose up -d --force-recreate voice
```

---

## Speech-to-Text with whisper.cpp

whisper.cpp runs OpenAI's Whisper model locally — no cloud API needed.

### 1. Install whisper.cpp

```bash
git clone https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
make

# Download a model (base is fast and accurate enough for most use cases)
bash models/download-ggml-model.sh base.en
```

### 2. Configure

```bash
# In .env
PLUGIN_VOICE_WHISPER_BINARY=/path/to/whisper.cpp/main
PLUGIN_VOICE_WHISPER_MODEL=/path/to/whisper.cpp/models/ggml-base.en.bin
```

### 3. Test

```bash
nself voice stt audio.wav
# Outputs transcribed text
```

Supported input formats: WAV, MP3, OGG, FLAC. Files must be 16kHz mono for best results. nself-voice converts automatically if ffmpeg is available.

---

## Twilio Telephony

Twilio integration lets your backend initiate and receive phone calls.

### Required Environment Variables

```bash
# In .env
PLUGIN_VOICE_TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
PLUGIN_VOICE_TWILIO_AUTH_TOKEN=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
PLUGIN_VOICE_TWILIO_PHONE_NUMBER=+15551234567
```

### Webhook Configuration

Twilio needs a publicly accessible URL to send call events to. Set the voice webhook on your Twilio phone number to:

```
https://api.yourdomain.com/voice/twilio/webhook
```

In the Twilio dashboard: Phone Numbers > Manage > Active Numbers > click your number > Voice Configuration > set the webhook URL above.

For local development, use ngrok or a similar tunnel:

```bash
ngrok http 3103
# Set the ngrok URL as the webhook in Twilio
```

### Making a Call

```bash
curl -X POST http://127.0.0.1:3103/voice/call \
  -H 'Authorization: Bearer <caller_token>' \
  -H 'Content-Type: application/json' \
  -d '{"to": "+15559876543", "message": "Your order has shipped."}'
```

---

## Wake Word Detection

Wake word detection uses [rustpotter](https://github.com/GiviMAD/rustpotter) to listen for a trigger phrase and activate the voice pipeline.

### Configuration

```bash
# In .env
PLUGIN_VOICE_WAKE_WORD=hey nself
PLUGIN_VOICE_WAKE_WORD_THRESHOLD=0.5
```

### Start and Stop

```bash
# Start listening for the wake word
nself voice wake start

# Stop wake word detection
nself voice wake stop

# Check status
nself voice wake status
```

When the wake word is detected, nself-voice activates the STT pipeline, processes the spoken request, and sends the transcript to the claw plugin (if configured) for a response. The response plays back via TTS.

Custom wake word models in `.rpw` format are also supported. Place the model file at `PLUGIN_VOICE_WAKE_WORD_MODEL_PATH`.

---

## AI Voicebot

When nself-voice, nself-ai, and nself-claw are all running, they work together as a conversational voice interface:

1. Wake word activates listening
2. whisper.cpp transcribes the spoken input
3. The transcript goes to nself-claw as a message
4. claw replies with text
5. nself-voice plays the reply via TTS

No extra configuration needed beyond having all three plugins running. The voice plugin discovers claw automatically via the `PLUGIN_CLAW_TOKEN` env var.

```bash
# In .env — required for voicebot integration
PLUGIN_VOICE_CLAW_TOKEN=nself_ai_tok_voice_xxxxx
PLUGIN_VOICE_CLAW_THREAD_ID=uuid  # optional: route all voice input to a specific thread
```

---

## Troubleshooting

### "Piper binary not found"

Check the path in `PLUGIN_VOICE_PIPER_BINARY`:

```bash
docker exec <voice_container> env | grep PIPER
which piper  # on the host
```

If running in Docker, the binary path must be mounted into the container. Add a volume bind in `conf.d` or use `CS_4_VOLUME=/usr/local/bin/piper:/usr/local/bin/piper`.

### "Model file not found"

Confirm the model path and that the file exists:

```bash
ls -lh /opt/piper/models/
```

Both the `.onnx` file and the `.onnx.json` config file must be present.

### "ElevenLabs quota exceeded"

You have run out of ElevenLabs character credits for the month. nself-voice falls back to Piper automatically if Piper is configured. Otherwise TTS requests fail until the quota resets.

### STT output is garbled

The audio may not be 16kHz mono. If ffmpeg is installed on the host, nself-voice converts automatically. To check:

```bash
docker exec <voice_container> ffmpeg -version 2>/dev/null | head -1
```

If ffmpeg is missing, convert audio manually before sending:

```bash
ffmpeg -i input.mp3 -ar 16000 -ac 1 output.wav
nself voice stt output.wav
```

### Twilio webhook returning 403

The voice plugin is not receiving webhook requests. Confirm:
1. The webhook URL is publicly accessible
2. The URL path is exactly `/voice/twilio/webhook`
3. `PLUGIN_VOICE_TWILIO_AUTH_TOKEN` is correct

---

## Related

- [nself-ai Setup](./ai-setup.md)
- [nself-claw Setup](./claw-setup.md)
- [Custom Services Reference](../configuration/custom-services.md)
- [Pro Plugin Setup](./pro-plugin-setup.md)
