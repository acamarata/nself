# Transcoder Plugin

> FFmpeg-based video transcoding — HLS, DASH, and MP4 output with MinIO storage. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install transcoder
```

## What It Does

Transcodes video files using FFmpeg into adaptive streaming formats (HLS, DASH) and standard MP4. Manages an upload pipeline from MinIO (or direct upload), queues encoding jobs, and writes output back to MinIO. Integrates with the `subtitle-manager` plugin for subtitle embedding and the `tokens` plugin for content protection.

## Dependencies

Requires MinIO (`MINIO_ENABLED=true`). Maps to `media-processing` in the plugin registry (port 3088, 9 tables).

## Implementation Details

- **Language:** Rust
- **Port:** 3088
- **Tables:** 9

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `TRANSCODER_PORT` | `3088` | Transcoder service port |
| `TRANSCODER_WORKERS` | `2` | Concurrent encoding workers |
| `TRANSCODER_HLS_SEGMENT_DURATION` | `6` | HLS segment length in seconds |
| `TRANSCODER_OUTPUT_BUCKET` | — | MinIO bucket for encoded output |
| `TRANSCODER_PRESET` | `h264_medium` | Default encoding preset |

## Ports

| Port | Purpose |
|------|---------|
| 3088 | Transcoder REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_transcoder_jobs` — encoding job queue
- `np_transcoder_presets` — encoding preset definitions
- `np_transcoder_outputs` — transcoding output catalog
- `np_transcoder_hls_manifests` — HLS manifest records
- `np_transcoder_segments` — HLS segment index
- `np_transcoder_thumbnails` — generated thumbnail records
- `np_transcoder_subtitle_tracks` — embedded subtitle tracks
- `np_transcoder_quality_metrics` — encoding quality stats
- `np_transcoder_upload_sessions` — direct upload sessions

## Nginx Routes

| Route | Target |
|-------|--------|
| `/transcoder/` | Transcoder API |
| `/transcoder/upload` | Direct upload endpoint |
