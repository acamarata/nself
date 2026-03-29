# Media Processing Plugin

> FFmpeg-based video encoding, transcoding, and adaptive streaming pipeline. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install media-processing
```

## What It Does

Provides a Rust-based FFmpeg wrapper service for video encoding and transcoding. Produces HLS and DASH adaptive streams, MP4 outputs, and thumbnail images. Manages an upload pipeline with real-time progress tracking so clients can poll or subscribe to job status during long transcode operations.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MEDIA_PROCESSING_PORT` | `3088` | Media processing service port |
| `MEDIA_PROCESSING_WORKERS` | `4` | Number of concurrent transcode workers |
| `MEDIA_FFMPEG_PATH` | `/usr/bin/ffmpeg` | Path to the FFmpeg binary |
| `MEDIA_OUTPUT_PATH` | — | Filesystem path for transcoded output files |

## Ports

| Port | Purpose |
|------|---------|
| 3088 | Media processing REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_media_processing_jobs` — transcode job records
- `np_media_processing_profiles` — encoding profile definitions
- `np_media_processing_outputs` — completed output file records
- `np_media_processing_segments` — HLS/DASH segment metadata
- `np_media_processing_manifests` — adaptive streaming manifests
- `np_media_processing_thumbnails` — generated thumbnail records
- `np_media_processing_uploads` — upload pipeline state
- `np_media_processing_progress` — real-time job progress tracking
- `np_media_processing_errors` — transcode error log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/media/` | Media processing API |
