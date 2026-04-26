# LiveKit Plugin

> Video and audio conferencing with recording, egress, and quality monitoring. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install livekit
```

## What It Does

Integrates a LiveKit server into your ɳSelf stack for WebRTC video and audio conferencing. Create rooms, generate participant tokens, record sessions to MinIO or S3, stream egress to RTMP endpoints, and monitor per-participant quality metrics. Used by the ɳSelf Chat and Claw client apps for voice and video calling.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `LIVEKIT_PORT` | `3107` | LiveKit management API port |
| `LIVEKIT_API_KEY` | *(auto-generated)* | LiveKit API key |
| `LIVEKIT_API_SECRET` | *(auto-generated)* | LiveKit API secret |
| `LIVEKIT_RECORDING_ENABLED` | `true` | Enable session recording |
| `LIVEKIT_RECORDING_BUCKET` | — | S3/MinIO bucket for recordings |
| `LIVEKIT_EGRESS_RTMP_URL` | — | RTMP URL for live streaming egress |

## Ports

| Port | Purpose |
|------|---------|
| 3107 | LiveKit management API |
| 7880 | LiveKit server WebRTC |
| 7881 | LiveKit server TCP fallback |
| 7882/UDP | LiveKit TURN server |

## Database Tables

6 tables added to your Postgres database:
- `np_livekit_rooms`, room definitions and settings
- `np_livekit_participants`, participant session records
- `np_livekit_recordings`, recording archives
- `np_livekit_egress`, egress stream configurations
- `np_livekit_quality_metrics`, per-session quality data
- `np_livekit_tokens`, issued participant tokens

## Nginx Routes

| Route | Target |
|-------|--------|
| `/livekit/` | LiveKit management API |
| `/rtc/` | LiveKit WebRTC proxy |
