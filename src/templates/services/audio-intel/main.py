"""audio-intel — Audio intelligence CS_N service for nself.

Transcribes audio from URLs and RSS feeds using OpenAI Whisper (local model),
then optionally analyzes the transcript via nself-ai plugin.

Endpoints:
  POST /transcribe         — download + transcribe + optionally analyze
  GET  /results            — paginated list of all results
  GET  /results/{id}       — single result with full transcript
  POST /rss                — process all new audio items from an RSS feed
  GET  /health             — health check
"""

import os
import re
import tempfile
import time
import urllib.request
from datetime import datetime, timezone
from typing import Optional

import feedparser
import httpx
import psycopg2
import psycopg2.extras
import uvicorn
import whisper
from fastapi import FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

DATABASE_URL: str = os.environ["DATABASE_URL"]
WHISPER_MODEL_NAME: str = os.getenv("WHISPER_MODEL", "base")
# Prompt sent to nself-ai after transcription.  Override per deployment.
AI_ANALYSIS_PROMPT: str = os.getenv(
    "AI_ANALYSIS_PROMPT",
    "Summarize the key points from this audio transcript in 3-5 bullet points. "
    "Identify the main topics, any action items, and notable insights.",
)
# Internal URL of the nself-ai plugin.  Must be set to enable analysis.
NSELF_AI_URL: str = os.getenv("NSELF_AI_URL", "")
# Maximum audio download size (bytes). Default 500 MB.
MAX_DOWNLOAD_BYTES: int = int(os.getenv("MAX_DOWNLOAD_BYTES", str(500 * 1024 * 1024)))

# ---------------------------------------------------------------------------
# Whisper model (loaded once at startup)
# ---------------------------------------------------------------------------

_model: Optional[whisper.Whisper] = None


def get_model() -> whisper.Whisper:
    global _model
    if _model is None:
        _model = whisper.load_model(WHISPER_MODEL_NAME)
    return _model


# ---------------------------------------------------------------------------
# Database helpers
# ---------------------------------------------------------------------------

def get_conn():
    return psycopg2.connect(DATABASE_URL)


def ensure_table() -> None:
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS np_audio_intel_results (
                    id               SERIAL PRIMARY KEY,
                    source_url       TEXT NOT NULL,
                    title            TEXT,
                    transcript       TEXT,
                    analysis         TEXT,
                    duration_seconds INTEGER,
                    model_used       TEXT,
                    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
                )
                """
            )
            cur.execute(
                "CREATE INDEX IF NOT EXISTS idx_np_audio_intel_source "
                "ON np_audio_intel_results (source_url)"
            )
        conn.commit()


def insert_result(
    source_url: str,
    title: Optional[str],
    transcript: str,
    analysis: Optional[str],
    duration_seconds: Optional[int],
    model_used: str,
) -> int:
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO np_audio_intel_results
                    (source_url, title, transcript, analysis, duration_seconds, model_used)
                VALUES (%s, %s, %s, %s, %s, %s)
                RETURNING id
                """,
                (source_url, title, transcript, analysis, duration_seconds, model_used),
            )
            row = cur.fetchone()
        conn.commit()
    return row[0]


def url_already_processed(source_url: str) -> bool:
    with get_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(
                "SELECT id FROM np_audio_intel_results WHERE source_url = %s LIMIT 1",
                (source_url,),
            )
            return cur.fetchone() is not None


# ---------------------------------------------------------------------------
# Audio download
# ---------------------------------------------------------------------------

def _is_direct_audio(url: str) -> bool:
    """Heuristic: does the URL look like a direct audio file?"""
    audio_exts = (".mp3", ".m4a", ".wav", ".ogg", ".flac", ".aac", ".opus", ".webm")
    path = url.split("?")[0].lower()
    return any(path.endswith(ext) for ext in audio_exts)


def download_audio(url: str, dest_dir: str) -> str:
    """Download audio to dest_dir.  Returns path to downloaded file.

    Uses yt-dlp for all URLs (handles direct audio, YouTube, podcasts, etc.).
    Falls back to urllib for simple direct audio URLs if yt-dlp is unavailable.
    """
    try:
        import yt_dlp  # noqa: PLC0415

        ydl_opts = {
            "format": "bestaudio/best",
            "outtmpl": os.path.join(dest_dir, "%(title)s.%(ext)s"),
            "quiet": True,
            "no_warnings": True,
            "max_filesize": MAX_DOWNLOAD_BYTES,
            "postprocessors": [
                {
                    "key": "FFmpegExtractAudio",
                    "preferredcodec": "mp3",
                    "preferredquality": "128",
                }
            ],
        }
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(url, download=True)
            # yt-dlp may have post-processed to .mp3
            title = info.get("title", "")
            filename = ydl.prepare_filename(info)
            # After post-processing the extension may have changed
            base = os.path.splitext(filename)[0]
            for ext in (".mp3", ".m4a", ".wav", ".ogg", ".flac", ".aac", ".opus", ".webm"):
                candidate = base + ext
                if os.path.exists(candidate):
                    return candidate, title
            # If we can't find the post-processed file, return original
            if os.path.exists(filename):
                return filename, title
            raise RuntimeError(f"yt-dlp download produced no file at {filename}")
    except ImportError:
        pass

    # Fallback: direct HTTP download
    if not _is_direct_audio(url):
        raise RuntimeError("yt-dlp not available and URL does not appear to be direct audio")
    dest_path = os.path.join(dest_dir, "audio.mp3")
    downloaded = 0
    req = urllib.request.Request(url, headers={"User-Agent": "audio-intel/1.0"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        with open(dest_path, "wb") as f:
            while True:
                chunk = resp.read(65536)
                if not chunk:
                    break
                downloaded += len(chunk)
                if downloaded > MAX_DOWNLOAD_BYTES:
                    raise RuntimeError("Audio file exceeds MAX_DOWNLOAD_BYTES limit")
                f.write(chunk)
    title = os.path.basename(url.split("?")[0])
    return dest_path, title


# ---------------------------------------------------------------------------
# Transcription
# ---------------------------------------------------------------------------

def transcribe(audio_path: str) -> dict:
    """Run Whisper transcription.  Returns dict with 'text' and 'duration'."""
    model = get_model()
    result = model.transcribe(audio_path, verbose=False)
    duration = None
    if result.get("segments"):
        last_seg = result["segments"][-1]
        duration = int(last_seg.get("end", 0))
    return {"text": result["text"].strip(), "duration": duration}


# ---------------------------------------------------------------------------
# Analysis via nself-ai
# ---------------------------------------------------------------------------

async def analyze_transcript(transcript: str) -> Optional[str]:
    if not NSELF_AI_URL:
        return None
    ai_url = NSELF_AI_URL.rstrip("/") + "/v1/chat/completions"
    payload = {
        "model": "default",
        "messages": [
            {"role": "system", "content": AI_ANALYSIS_PROMPT},
            {"role": "user", "content": transcript},
        ],
        "max_tokens": 512,
    }
    try:
        async with httpx.AsyncClient(timeout=60.0) as client:
            resp = await client.post(ai_url, json=payload)
            resp.raise_for_status()
            data = resp.json()
            return data["choices"][0]["message"]["content"]
    except Exception as exc:  # noqa: BLE001
        return f"[analysis failed: {exc}]"


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------

app = FastAPI(
    title="audio-intel",
    description="Audio intelligence: transcribe + analyze audio from URLs and RSS feeds",
    version="1.0.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.on_event("startup")
def startup_event():
    ensure_table()
    # Warm up the model at startup so the first request is not slow.
    get_model()


# ---------------------------------------------------------------------------
# Request / response models
# ---------------------------------------------------------------------------


class TranscribeRequest(BaseModel):
    url: str
    title: Optional[str] = None
    analyze: bool = True
    skip_if_exists: bool = True


class TranscribeResponse(BaseModel):
    id: int
    source_url: str
    title: Optional[str]
    transcript: str
    analysis: Optional[str]
    duration_seconds: Optional[int]
    model_used: str
    created_at: str


class RssRequest(BaseModel):
    feed_url: str
    analyze: bool = True
    max_items: int = 10


class RssResponse(BaseModel):
    processed: int
    skipped: int
    results: list[TranscribeResponse]


# ---------------------------------------------------------------------------
# Endpoints
# ---------------------------------------------------------------------------


@app.get("/health")
def health():
    return {"status": "healthy", "service": "audio-intel", "timestamp": datetime.now(timezone.utc).isoformat()}


@app.post("/transcribe", response_model=TranscribeResponse)
async def transcribe_url(req: TranscribeRequest):
    """Download, transcribe, and optionally analyze audio from a URL.

    - Supports direct audio URLs (mp3, m4a, wav, etc.) and any URL yt-dlp handles.
    - Set `analyze: true` to call nself-ai for a summary (requires NSELF_AI_URL).
    - Set `skip_if_exists: true` to skip re-processing already-processed URLs.
    """
    if req.skip_if_exists and url_already_processed(req.url):
        # Return the existing record
        with get_conn() as conn:
            with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
                cur.execute(
                    "SELECT * FROM np_audio_intel_results WHERE source_url = %s ORDER BY id DESC LIMIT 1",
                    (req.url,),
                )
                row = cur.fetchone()
        if row:
            return TranscribeResponse(
                id=row["id"],
                source_url=row["source_url"],
                title=row["title"],
                transcript=row["transcript"],
                analysis=row["analysis"],
                duration_seconds=row["duration_seconds"],
                model_used=row["model_used"],
                created_at=row["created_at"].isoformat(),
            )

    with tempfile.TemporaryDirectory() as tmpdir:
        try:
            audio_path, detected_title = download_audio(req.url, tmpdir)
        except Exception as exc:
            raise HTTPException(status_code=422, detail=f"Download failed: {exc}") from exc

        try:
            result = transcribe(audio_path)
        except Exception as exc:
            raise HTTPException(status_code=500, detail=f"Transcription failed: {exc}") from exc

    transcript_text = result["text"]
    duration = result["duration"]
    title = req.title or detected_title or None

    analysis = None
    if req.analyze:
        analysis = await analyze_transcript(transcript_text)

    row_id = insert_result(
        source_url=req.url,
        title=title,
        transcript=transcript_text,
        analysis=analysis,
        duration_seconds=duration,
        model_used=WHISPER_MODEL_NAME,
    )

    return TranscribeResponse(
        id=row_id,
        source_url=req.url,
        title=title,
        transcript=transcript_text,
        analysis=analysis,
        duration_seconds=duration,
        model_used=WHISPER_MODEL_NAME,
        created_at=datetime.now(timezone.utc).isoformat(),
    )


@app.get("/results", response_model=list[TranscribeResponse])
def list_results(
    limit: int = Query(default=20, ge=1, le=200),
    offset: int = Query(default=0, ge=0),
):
    """List all transcription results with pagination."""
    with get_conn() as conn:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(
                "SELECT id, source_url, title, transcript, analysis, duration_seconds, "
                "model_used, created_at "
                "FROM np_audio_intel_results "
                "ORDER BY created_at DESC "
                "LIMIT %s OFFSET %s",
                (limit, offset),
            )
            rows = cur.fetchall()

    return [
        TranscribeResponse(
            id=row["id"],
            source_url=row["source_url"],
            title=row["title"],
            transcript=row["transcript"],
            analysis=row["analysis"],
            duration_seconds=row["duration_seconds"],
            model_used=row["model_used"],
            created_at=row["created_at"].isoformat(),
        )
        for row in rows
    ]


@app.get("/results/{result_id}", response_model=TranscribeResponse)
def get_result(result_id: int):
    """Get a single transcription result by ID."""
    with get_conn() as conn:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(
                "SELECT id, source_url, title, transcript, analysis, duration_seconds, "
                "model_used, created_at "
                "FROM np_audio_intel_results WHERE id = %s",
                (result_id,),
            )
            row = cur.fetchone()

    if not row:
        raise HTTPException(status_code=404, detail="Result not found")

    return TranscribeResponse(
        id=row["id"],
        source_url=row["source_url"],
        title=row["title"],
        transcript=row["transcript"],
        analysis=row["analysis"],
        duration_seconds=row["duration_seconds"],
        model_used=row["model_used"],
        created_at=row["created_at"].isoformat(),
    )


@app.post("/rss", response_model=RssResponse)
async def process_rss(req: RssRequest):
    """Process audio items from an RSS/podcast feed.

    Fetches the feed, extracts audio enclosures (up to `max_items`),
    and transcribes any that have not been processed before.
    """
    try:
        feed = feedparser.parse(req.feed_url)
    except Exception as exc:
        raise HTTPException(status_code=422, detail=f"Failed to parse RSS feed: {exc}") from exc

    if feed.bozo and not feed.entries:
        raise HTTPException(status_code=422, detail="Invalid or empty RSS feed")

    processed_items: list[TranscribeResponse] = []
    skipped = 0
    count = 0

    for entry in feed.entries:
        if count >= req.max_items:
            break

        # Find audio enclosure
        audio_url = None
        entry_title = entry.get("title", "")
        for enc in entry.get("enclosures", []):
            mime = enc.get("type", "")
            if mime.startswith("audio/") or _is_direct_audio(enc.get("href", "")):
                audio_url = enc.get("href")
                break

        if not audio_url:
            # Some feeds put audio in media_content
            for media in entry.get("media_content", []):
                if media.get("medium") == "audio":
                    audio_url = media.get("url")
                    break

        if not audio_url:
            continue

        if url_already_processed(audio_url):
            skipped += 1
            continue

        # Transcribe this entry
        transcribe_req = TranscribeRequest(
            url=audio_url,
            title=entry_title or None,
            analyze=req.analyze,
            skip_if_exists=False,
        )
        try:
            result = await transcribe_url(transcribe_req)
            processed_items.append(result)
            count += 1
        except HTTPException:
            skipped += 1

    return RssResponse(
        processed=len(processed_items),
        skipped=skipped,
        results=processed_items,
    )


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
