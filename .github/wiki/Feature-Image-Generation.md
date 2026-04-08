# Feature: Image Generation

nClaw supports multi-provider image generation. You can generate images from text prompts through the chat interface or the API, and browse results in the image gallery.

**Route:** `/gallery` in claw-web
**API:** `POST /claw/image/generate`

---

## Supported Providers

| Provider | Env Var | Notes |
|----------|---------|-------|
| DALL-E (default) | `OPENAI_API_KEY` | OpenAI's image model |
| Stable Diffusion | `SD_API_URL` | Self-hosted or API |
| Midjourney | `MJ_API_KEY` | Via proxy API |

Set `CLAW_IMAGE_PROVIDER` in your `.env` to choose the default provider. Users can override per-request.

## How It Works

1. Send a prompt via chat or `POST /claw/image/generate`
2. The request enters the job queue (`np_claw.image_jobs` table)
3. The selected provider processes the request
4. Results appear in the gallery and in the chat thread

## API

### Generate Image

```
POST /claw/image/generate
Content-Type: application/json
Authorization: Bearer {token}

{
  "prompt": "A sunset over mountains in watercolor style",
  "provider": "dall-e",
  "size": "1024x1024"
}
```

### List Jobs

```
GET /claw/image/jobs
```

Returns all generation jobs with status (queued, processing, complete, failed).

### Get Job

```
GET /claw/image/jobs/{id}
```

Returns job details including the generated image URL when complete.

## Gallery

The `/gallery` route in claw-web shows all generated images as cards. Each card displays the prompt, provider, timestamp, and a zoomable preview.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CLAW_IMAGE_PROVIDER` | `dall-e` | Default provider |
| `OPENAI_API_KEY` | -- | Required for DALL-E |
| `SD_API_URL` | -- | Stable Diffusion endpoint |
| `MJ_API_KEY` | -- | Midjourney proxy key |

## Related

- [[Feature-nClaw]] -- nClaw overview
- [[Feature-Agent-Dashboard]] -- Agent metrics
