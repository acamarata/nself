# Streaming Plugin

> RTMP/HLS live streaming with viewer analytics, DVR, and moderation. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install streaming
```

## What It Does

Adds a live streaming platform to your nSelf stack. Accepts RTMP ingest from OBS, mobile apps, or encoders, and distributes as HLS to viewers. Includes DVR (time-shifting / replay), real-time viewer analytics, chat moderation integration, and stream key management per user or channel.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `STREAMING_PORT` | `3084` | Streaming management API port |
| `STREAMING_RTMP_PORT` | `1935` | RTMP ingest port |
| `STREAMING_HLS_SEGMENT_DURATION` | `2` | HLS segment duration in seconds |
| `STREAMING_DVR_ENABLED` | `true` | Enable DVR/replay |
| `STREAMING_DVR_DURATION` | `7200` | DVR window in seconds |
| `STREAMING_RECORDING_BUCKET` | — | MinIO bucket for recordings |

## Ports

| Port | Purpose |
|------|---------|
| 3084 | Streaming management REST API |
| 1935 | RTMP ingest |

## Database Tables

10 tables added to your Postgres database:
- `np_streaming_streams` — stream channel definitions
- `np_streaming_stream_keys` — per-user stream keys
- `np_streaming_sessions` — live session records
- `np_streaming_viewers` — concurrent viewer tracking
- `np_streaming_analytics` — viewer event analytics
- `np_streaming_dvr_segments` — DVR segment index
- `np_streaming_recordings` — VOD archive records
- `np_streaming_thumbnails` — stream preview thumbnails
- `np_streaming_moderation` — moderation actions
- `np_streaming_chat_links` — linked chat room associations

## Nginx Routes

| Route | Target |
|-------|--------|
| `/streaming/` | Stream management API |
| `/hls/` | HLS manifest and segment delivery |
