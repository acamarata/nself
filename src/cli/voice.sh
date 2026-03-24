#!/usr/bin/env bash
# voice.sh - Voice plugin management for nself
# Manages TTS, STT, phone calls, and wake-word detection via the nself-voice pro plugin.
#
# Commands:
#   nself voice status                                   Show voice service health
#   nself voice tts "<text>" [--provider=piper] [--output=<file>]
#                                                        Synthesize text to audio file
#   nself voice stt <audio_file>                         Transcribe audio to text
#   nself voice call <number> [--message="<msg>"]        Initiate a phone call
#   nself voice wake start                               Start wake-word detection
#   nself voice wake stop                                Stop wake-word detection
#   nself voice wake status                              Show wake-word detection status
#
# Usage: nself voice <subcommand> [options]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source display helpers
source "$NSELF_ROOT/src/lib/utils/cli-output.sh" 2>/dev/null || true
source "$NSELF_ROOT/src/lib/utils/display.sh" 2>/dev/null || true

# Fallbacks if display helpers didn't load
if ! type cli_error >/dev/null 2>&1; then
  cli_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi
if ! type log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! type log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi
if ! type log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi

# ============================================================================
# Usage
# ============================================================================

voice_usage() {
  printf "nself voice — voice plugin management\n\n"
  printf "Usage: nself voice <subcommand> [options]\n\n"
  printf "Subcommands:\n"
  printf "  status                                       Show voice service health and version\n"
  printf "  tts \"<text>\" [--provider=<p>] [--output=<f>] Synthesize text to audio (WAV)\n"
  printf "  stt <audio_file>                             Transcribe audio file to text\n"
  printf "  call <number> [--message=\"<msg>\"]            Initiate a phone call\n"
  printf "  wake start                                   Start wake-word detection\n"
  printf "  wake stop                                    Stop wake-word detection\n"
  printf "  wake status                                  Show wake-word detection status\n"
  printf "  routing [status]                             Show LLM routing config and TTFT estimate\n"
  printf "  routing set-tier <standard|realtime>         Configure voice LLM tier\n\n"
  printf "Environment:\n"
  printf "  NSELF_VOICE_URL         Voice plugin base URL (default: http://localhost:3714)\n"
  printf "  PLUGIN_INTERNAL_SECRET  Required for all commands\n\n"
  printf "Examples:\n"
  printf "  nself voice status\n"
  printf "  nself voice tts \"Hello from nself\"\n"
  printf "  nself voice tts \"Hello\" --provider=piper --output=/tmp/hello.wav\n"
  printf "  nself voice stt recording.wav\n"
  printf "  nself voice call +15551234567 --message=\"Your server is down\"\n"
  printf "  nself voice wake start\n"
  printf "  nself voice wake status\n"
  printf "  nself voice wake stop\n"
}

# ============================================================================
# Top-level dispatcher
# ============================================================================

cmd_voice() {
  local subcommand="${1:-}"

  if [ -z "$subcommand" ]; then
    voice_usage
    exit 0
  fi

  shift

  case "$subcommand" in
    status)
      cmd_voice_status "$@"
      ;;
    tts)
      cmd_voice_tts "$@"
      ;;
    stt)
      cmd_voice_stt "$@"
      ;;
    call)
      cmd_voice_call "$@"
      ;;
    wake)
      cmd_voice_wake "$@"
      ;;
    routing)
      cmd_voice_routing "$@"
      ;;
    help | --help | -h)
      voice_usage
      exit 0
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      printf "\n"
      voice_usage
      exit 1
      ;;
  esac
}

# ============================================================================
# T-0860: status — show voice service health
# ============================================================================

cmd_voice_status() {
  local voice_url="${NSELF_VOICE_URL:-http://localhost:3714}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local response=""
  response=$(curl -s \
    -H "x-internal-token: ${internal_secret}" \
    "${voice_url}/voice/health" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from voice service at ${voice_url}. Is nself-voice running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    printf "\n\033[1mVoice Service Status\033[0m\n"
    local status version tts_provider stt_provider
    status=$(printf '%s' "$response" | jq -r '.status // "unknown"' 2>/dev/null)
    version=$(printf '%s' "$response" | jq -r '.version // "unknown"' 2>/dev/null)
    tts_provider=$(printf '%s' "$response" | jq -r '.tts_provider // "piper"' 2>/dev/null)
    stt_provider=$(printf '%s' "$response" | jq -r '.stt_provider // "whisper"' 2>/dev/null)

    local color="\033[0;32m"
    if [ "$status" != "ok" ] && [ "$status" != "healthy" ]; then
      color="\033[0;31m"
    fi

    printf "  Status:       ${color}%s\033[0m\n" "$status"
    printf "  Version:      %s\n" "$version"
    printf "  TTS provider: %s\n" "$tts_provider"
    printf "  STT provider: %s\n" "$stt_provider"
    printf "\n"
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0860: tts — synthesize text to audio
# ============================================================================

cmd_voice_tts() {
  local text="" provider="" output_file=""
  local voice_url="${NSELF_VOICE_URL:-http://localhost:3714}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "${1:-}" in
    --help | -h | "")
      printf "Usage: nself voice tts \"<text>\" [--provider=<p>] [--output=<file>]\n\n"
      printf "  text           The text to synthesize\n"
      printf "  --provider     TTS provider (default: piper). Options: piper, espeak\n"
      printf "  --output       Output file path (default: /tmp/nself-tts-<timestamp>.wav)\n\n"
      printf "Examples:\n"
      printf "  nself voice tts \"Hello from nself\"\n"
      printf "  nself voice tts \"Hello\" --provider=piper --output=/tmp/hello.wav\n"
      return 0
      ;;
  esac

  # First positional arg is the text (unless it starts with --)
  if [ "${1:-}" != "" ] && [ "$(printf '%s' "${1:-}" | cut -c1-2)" != "--" ]; then
    text="$1"
    shift
  fi

  while [ $# -gt 0 ]; do
    case "$1" in
      --provider=*) provider="${1#--provider=}"; shift ;;
      --provider)   provider="$2"; shift 2 ;;
      --output=*)   output_file="${1#--output=}"; shift ;;
      --output)     output_file="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [ -z "$text" ]; then
    cli_error "Text required"
    printf "Usage: nself voice tts \"<text>\" [--provider=<p>] [--output=<file>]\n" >&2
    return 1
  fi

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  # Generate default output path if not specified
  if [ -z "$output_file" ]; then
    local ts=""
    ts=$(date +%s)
    output_file="/tmp/nself-tts-${ts}.wav"
  fi

  local escaped_text=""
  escaped_text=$(printf '%s' "$text" | sed 's/\\/\\\\/g; s/"/\\"/g')

  local body="{\"text\":\"${escaped_text}\""
  if [ -n "$provider" ]; then
    body="${body},\"provider\":\"${provider}\""
  fi
  body="${body}}"

  log_info "Synthesizing speech..."

  local http_code=""
  http_code=$(curl -s -o "$output_file" -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "x-internal-token: ${internal_secret}" \
    -d "$body" \
    "${voice_url}/tts" 2>/dev/null)

  if [ "$http_code" != "200" ]; then
    # Output file may contain error JSON
    local err_body=""
    err_body=$(cat "$output_file" 2>/dev/null || printf "")
    rm -f "$output_file" 2>/dev/null || true
    cli_error "TTS request failed (HTTP ${http_code}): ${err_body}"
    return 1
  fi

  log_success "Audio saved to: ${output_file}"

  # Offer to play on macOS (afplay) or Linux (aplay)
  if command -v afplay >/dev/null 2>&1; then
    printf "  Play: afplay %s\n" "$output_file"
  elif command -v aplay >/dev/null 2>&1; then
    printf "  Play: aplay %s\n" "$output_file"
  fi
}

# ============================================================================
# T-0860: stt — transcribe audio file to text
# ============================================================================

cmd_voice_stt() {
  local audio_file="${1:-}"
  local voice_url="${NSELF_VOICE_URL:-http://localhost:3714}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$audio_file" in
    --help | -h)
      printf "Usage: nself voice stt <audio_file>\n\n"
      printf "  audio_file   Path to WAV, OGG, MP3, or M4A audio file\n\n"
      printf "Examples:\n"
      printf "  nself voice stt recording.wav\n"
      return 0
      ;;
  esac

  if [ -z "$audio_file" ]; then
    cli_error "Audio file required"
    printf "Usage: nself voice stt <audio_file>\n" >&2
    return 1
  fi

  if [ ! -f "$audio_file" ]; then
    cli_error "File not found: ${audio_file}"
    return 1
  fi

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  log_info "Transcribing $(basename "$audio_file")..."

  local response=""
  response=$(curl -s -X POST \
    -H "x-internal-token: ${internal_secret}" \
    -F "audio=@${audio_file}" \
    "${voice_url}/stt" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from voice service at ${voice_url}. Is nself-voice running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local transcript duration
    transcript=$(printf '%s' "$response" | jq -r '.transcript // .text // empty' 2>/dev/null)
    duration=$(printf '%s' "$response" | jq -r '.duration_seconds // empty' 2>/dev/null)
    if [ -n "$transcript" ]; then
      printf '%s\n' "$transcript"
      if [ -n "$duration" ]; then
        printf "\n\033[2mDuration: %ss\033[0m\n" "$duration"
      fi
    else
      printf '%s\n' "$response"
    fi
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0860: call — initiate a phone call
# ============================================================================

cmd_voice_call() {
  local number="${1:-}"
  local message=""
  local voice_url="${NSELF_VOICE_URL:-http://localhost:3714}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$number" in
    --help | -h)
      printf "Usage: nself voice call <number> [--message=\"<msg>\"]\n\n"
      printf "  number     Phone number in E.164 format (e.g. +15551234567)\n"
      printf "  --message  Text to speak during the call\n\n"
      printf "Examples:\n"
      printf "  nself voice call +15551234567\n"
      printf "  nself voice call +15551234567 --message=\"Your server is down\"\n"
      return 0
      ;;
  esac

  if [ -z "$number" ]; then
    cli_error "Phone number required"
    printf "Usage: nself voice call <number> [--message=\"<msg>\"]\n" >&2
    return 1
  fi

  shift || true
  while [ $# -gt 0 ]; do
    case "$1" in
      --message=*) message="${1#--message=}"; shift ;;
      --message)   message="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local escaped_number=""
  escaped_number=$(printf '%s' "$number" | sed 's/\\/\\\\/g; s/"/\\"/g')

  local body="{\"to\":\"${escaped_number}\""
  if [ -n "$message" ]; then
    local escaped_message=""
    escaped_message=$(printf '%s' "$message" | sed 's/\\/\\\\/g; s/"/\\"/g')
    body="${body},\"message\":\"${escaped_message}\""
  fi
  body="${body}}"

  local response=""
  response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "x-internal-token: ${internal_secret}" \
    -d "$body" \
    "${voice_url}/voice/call" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from voice service at ${voice_url}. Is nself-voice running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local call_id status
    call_id=$(printf '%s' "$response" | jq -r '.call_id // .id // empty' 2>/dev/null)
    status=$(printf '%s' "$response" | jq -r '.status // "initiated"' 2>/dev/null)
    if [ -n "$call_id" ]; then
      log_success "Call ${status}: ${call_id}"
    else
      printf '%s\n' "$response"
    fi
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0860: wake — wake-word detection management
# ============================================================================

cmd_voice_wake() {
  local subcmd="${1:-status}"
  shift || true

  local voice_url="${NSELF_VOICE_URL:-http://localhost:3714}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  case "$subcmd" in

    start)
      local response=""
      response=$(curl -s -X POST \
        -H "x-internal-token: ${internal_secret}" \
        "${voice_url}/voice/wake/start" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from voice service at ${voice_url}. Is nself-voice running?"
        return 1
      fi

      log_success "Wake-word detection started."
      ;;

    stop)
      local response=""
      response=$(curl -s -X POST \
        -H "x-internal-token: ${internal_secret}" \
        "${voice_url}/voice/wake/stop" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from voice service at ${voice_url}. Is nself-voice running?"
        return 1
      fi

      log_success "Wake-word detection stopped."
      ;;

    status)
      local response=""
      response=$(curl -s \
        -H "x-internal-token: ${internal_secret}" \
        "${voice_url}/voice/wake/status" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from voice service at ${voice_url}. Is nself-voice running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        local active wake_word detections
        active=$(printf '%s' "$response" | jq -r '.active // false' 2>/dev/null)
        wake_word=$(printf '%s' "$response" | jq -r '.wake_word // "hey nself"' 2>/dev/null)
        detections=$(printf '%s' "$response" | jq -r '.detections_today // 0' 2>/dev/null)

        printf "\n\033[1mWake-Word Detection\033[0m\n"
        if [ "$active" = "true" ]; then
          printf "  Status:           \033[0;32mactive\033[0m\n"
        else
          printf "  Status:           \033[0;33minactive\033[0m\n"
        fi
        printf "  Wake word:        %s\n" "$wake_word"
        printf "  Detections today: %s\n" "$detections"
        printf "\n"
      else
        printf '%s\n' "$response"
      fi
      ;;

    help | --help | -h)
      printf "Usage: nself voice wake <start|stop|status>\n\n"
      printf "Subcommands:\n"
      printf "  start    Start wake-word detection\n"
      printf "  stop     Stop wake-word detection\n"
      printf "  status   Show current detection status\n\n"
      printf "Examples:\n"
      printf "  nself voice wake start\n"
      printf "  nself voice wake status\n"
      printf "  nself voice wake stop\n"
      ;;

    *)
      cli_error "Unknown wake action: $subcmd"
      printf "Actions: start, stop, status\n"
      exit 1
      ;;
  esac
}

# ============================================================================
# T-1407: routing — voice LLM routing management
# ============================================================================

# Helper: set or update an env var in .env (Bash 3.2 compatible)
_voice_set_env() {
  local key="$1" value="$2"
  if [ -f ".env" ] && grep -q "^${key}=" .env 2>/dev/null; then
    sed -i.bak "s|^${key}=.*|${key}=${value}|" .env && rm -f .env.bak
  else
    printf "\n%s=%s\n" "$key" "$value" >> .env
  fi
}

cmd_voice_routing() {
  local action="${1:-status}"

  case "$action" in
    status)
      local provider model mode
      provider=$(grep "^NSELF_VOICE_LLM_PROVIDER=" .env 2>/dev/null | cut -d= -f2- || printf "google")
      model=$(grep "^NSELF_VOICE_LLM_MODEL=" .env 2>/dev/null | cut -d= -f2- || printf "gemini-2.5-flash")
      mode=$(grep "^NSELF_VOICE_MODE=" .env 2>/dev/null | cut -d= -f2- || printf "standard")

      # Provide fallback values when env vars are missing
      provider="${provider:-google}"
      model="${model:-gemini-2.5-flash}"
      mode="${mode:-standard}"

      # Estimate TTFT based on provider
      local ttft_estimate
      case "$provider" in
        google)   ttft_estimate="~300ms (Gemini Flash)" ;;
        openai)   ttft_estimate="~150ms (GPT-4o Realtime)" ;;
        *)        ttft_estimate="unknown" ;;
      esac

      printf "\n\033[1mVoice Routing Status\033[0m\n"
      printf "  Provider:      %s\n" "$provider"
      printf "  Model:         %s\n" "$model"
      printf "  Mode:          %s\n" "$mode"
      printf "  Est. TTFT:     %s\n" "$ttft_estimate"
      printf "\n"
      ;;
    set-tier)
      local tier="${2:-standard}"
      case "$tier" in
        standard)
          _voice_set_env "NSELF_VOICE_LLM_PROVIDER" "google"
          _voice_set_env "NSELF_VOICE_LLM_MODEL" "gemini-2.5-flash"
          _voice_set_env "NSELF_VOICE_MODE" "standard"
          printf "Voice tier set to standard (Gemini 2.5 Flash, ~300ms TTFT)\n"
          ;;
        realtime)
          if ! grep -q "^OPENAI_API_KEY=" .env 2>/dev/null; then
            cli_error "OPENAI_API_KEY not found in .env"
            printf "Add OPENAI_API_KEY=sk-... to .env then retry.\n" >&2
            return 1
          fi
          _voice_set_env "NSELF_VOICE_LLM_PROVIDER" "openai"
          _voice_set_env "NSELF_VOICE_LLM_MODEL" "gpt-4o-realtime-preview"
          _voice_set_env "NSELF_VOICE_MODE" "realtime"
          printf "Voice tier set to realtime (GPT-4o Realtime, ~150ms TTFT)\n"
          printf "Note: NSELF_VOICE_MODE=realtime requires nself restart\n"
          ;;
        *)
          printf "Unknown tier '%s'. Use: standard | realtime\n" "$tier" >&2
          return 1
          ;;
      esac
      ;;
    help | --help | -h)
      printf "Usage: nself voice routing [status|set-tier <standard|realtime>]\n\n"
      printf "  status                Show current LLM routing config and TTFT estimate\n"
      printf "  set-tier standard     Use Gemini 2.5 Flash (~300ms TTFT)\n"
      printf "  set-tier realtime     Use GPT-4o Realtime (~150ms TTFT, requires OPENAI_API_KEY)\n\n"
      printf "Examples:\n"
      printf "  nself voice routing\n"
      printf "  nself voice routing set-tier standard\n"
      printf "  nself voice routing set-tier realtime\n"
      ;;
    *)
      printf "Usage: nself voice routing [status|set-tier <standard|realtime>]\n" >&2
      return 1
      ;;
  esac
}

# ============================================================================
# Entry point
# ============================================================================

cmd_voice "$@"
