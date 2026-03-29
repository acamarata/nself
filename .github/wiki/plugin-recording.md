# Recording Plugin

> Recording orchestration and archive scheduling. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install recording
```

## What It Does

Orchestrates recording of live streams and scheduled broadcasts. Triggered by the `epg` plugin recording rules or manual API calls. Manages recording jobs, monitors progress, and archives completed recordings to MinIO. Provides hooks for post-processing (transcoding, thumbnail generation) after a recording completes.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `RECORDING_PORT` | `3092` | Recording service port |
| `RECORDING_OUTPUT_BUCKET` | — | MinIO bucket for recordings |
| `RECORDING_MAX_DURATION` | `14400` | Max recording duration in seconds |
| `RECORDING_AUTO_TRANSCODE` | `false` | Auto-transcode after recording |

## Ports

| Port | Purpose |
|------|---------|
| 3092 | Recording orchestration REST API |

## Database Tables

3 tables added to your Postgres database:
- `np_recording_jobs` — recording job queue and status
- `np_recording_archives` — completed recording archive index
- `np_recording_schedules` — scheduled recording configurations

## Nginx Routes

None — recording is an internal orchestration service.
