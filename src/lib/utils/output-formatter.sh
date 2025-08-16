#!/bin/bash

OUTPUT_FORMATTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${OUTPUT_FORMATTER_DIR}/display.sh"

OUTPUT_BUFFER=""
CAPTURE_MODE=false
DOCKER_PULL_ACTIVE=false
LAST_LINE=""
PROGRESS_ACTIVE=false

capture_output() {
  CAPTURE_MODE=true
  OUTPUT_BUFFER=""
}

stop_capture() {
  CAPTURE_MODE=false
  echo "$OUTPUT_BUFFER"
  OUTPUT_BUFFER=""
}

format_docker_output() {
  local line="$1"

  if [[ "$line" =~ "Pulling from" ]]; then
    DOCKER_PULL_ACTIVE=true
    echo -e "${BLUE}📦${RESET} Pulling Docker images..."
    return
  fi

  if [[ "$line" =~ "Pull complete" ]] || [[ "$line" =~ "Already exists" ]]; then
    if [[ "$DOCKER_PULL_ACTIVE" == true ]]; then
      printf "."
    fi
    return
  fi

  if [[ "$line" =~ "Downloaded newer image" ]] || [[ "$line" =~ "Image is up to date" ]]; then
    if [[ "$DOCKER_PULL_ACTIVE" == true ]]; then
      echo -e "\n${GREEN}✓${RESET} Docker images ready"
      DOCKER_PULL_ACTIVE=false
    fi
    return
  fi

  if [[ "$line" =~ "Creating" ]]; then
    local container=$(echo "$line" | sed 's/.*Creating //' | sed 's/ .*//')
    echo -e "${BLUE}🔨${RESET} Creating container: ${BOLD}$container${RESET}"
    return
  fi

  if [[ "$line" =~ "Started" ]]; then
    local container=$(echo "$line" | sed 's/.*Started //' | sed 's/ .*//')
    echo -e "${GREEN}✓${RESET} Started: ${BOLD}$container${RESET}"
    return
  fi

  if [[ "$line" =~ "Error" ]] || [[ "$line" =~ "ERROR" ]]; then
    echo -e "${RED}✗${RESET} $line"
    return
  fi

  if [[ "$line" =~ "Warning" ]] || [[ "$line" =~ "WARNING" ]]; then
    echo -e "${YELLOW}⚠${RESET} $line"
    return
  fi
}

format_build_output() {
  local line="$1"

  if [[ "$line" =~ "Generating" ]]; then
    local file=$(echo "$line" | sed 's/.*Generating //' | sed 's/ .*//')
    echo -e "${BLUE}📝${RESET} Generating: ${BOLD}$file${RESET}"
    return
  fi

  if [[ "$line" =~ "Created" ]]; then
    local file=$(echo "$line" | sed 's/.*Created //' | sed 's/ .*//')
    echo -e "${GREEN}✓${RESET} Created: ${BOLD}$file${RESET}"
    return
  fi

  if [[ "$line" =~ "Skipping" ]]; then
    return
  fi

  if [[ "$line" =~ "Error" ]] || [[ "$line" =~ "ERROR" ]]; then
    echo -e "${RED}✗${RESET} $line"
    return
  fi
}

show_progress() {
  local current=$1
  local total=$2
  local message=$3

  local percent=$((current * 100 / total))
  local filled=$((percent / 2))
  local empty=$((50 - filled))

  printf "\r${BLUE}▶${RESET} $message ["
  printf "%${filled}s" | tr ' ' '█'
  printf "%${empty}s" | tr ' ' '░'
  printf "] ${BOLD}%d%%${RESET}" "$percent"

  if [[ $current -eq $total ]]; then
    printf "\n"
  fi
}

format_test_output() {
  local line="$1"

  if [[ "$line" =~ "PASS" ]]; then
    echo -e "${GREEN}✓${RESET} Test passed"
    return
  fi

  if [[ "$line" =~ "FAIL" ]]; then
    echo -e "${RED}✗${RESET} Test failed"
    return
  fi

  if [[ "$line" =~ "SKIP" ]]; then
    echo -e "${YELLOW}⊘${RESET} Test skipped"
    return
  fi
}

format_validation_output() {
  local line="$1"

  if [[ "$line" =~ "Validating" ]]; then
    local item=$(echo "$line" | sed 's/.*Validating //' | sed 's/ .*//')
    echo -e "${BLUE}🔍${RESET} Validating: ${BOLD}$item${RESET}"
    return
  fi

  if [[ "$line" =~ "Valid" ]]; then
    echo -e "${GREEN}✓${RESET} Validation passed"
    return
  fi

  if [[ "$line" =~ "Invalid" ]]; then
    echo -e "${RED}✗${RESET} Validation failed: $line"
    return
  fi
}

filter_output() {
  local context="${1:-general}"

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue

    [[ "$line" =~ ^[[:space:]]*$ ]] && continue

    [[ "$line" == "$LAST_LINE" ]] && continue
    LAST_LINE="$line"

    case "$context" in
    docker)
      format_docker_output "$line"
      ;;
    build)
      format_build_output "$line"
      ;;
    test)
      format_test_output "$line"
      ;;
    validation)
      format_validation_output "$line"
      ;;
    *)
      [[ "$line" =~ "npm WARN" ]] && continue
      [[ "$line" =~ "npm notice" ]] && continue
      [[ "$line" =~ "found 0 vulnerabilities" ]] && continue

      echo "$line"
      ;;
    esac
  done
}

show_spinner() {
  local message="$1"
  local pid=$2
  local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  local i=0

  while kill -0 $pid 2>/dev/null; do
    printf "\r${BLUE}${spin:i++%${#spin}:1}${RESET} $message"
    sleep 0.1
  done
  printf "\r${GREEN}✓${RESET} $message\n"
}

format_error() {
  local error="$1"
  local suggestion="${2:-}"

  echo -e "\n${RED}═══ ERROR ═══${RESET}"
  echo -e "${RED}✗${RESET} $error"

  if [[ -n "$suggestion" ]]; then
    echo -e "${YELLOW}💡${RESET} Suggestion: $suggestion"
  fi

  echo -e "${RED}═════════════${RESET}\n"
}

format_warning() {
  local warning="$1"
  local suggestion="${2:-}"

  echo -e "\n${YELLOW}⚠ WARNING${RESET}"
  echo -e "$warning"

  if [[ -n "$suggestion" ]]; then
    echo -e "${DIM}Suggestion: $suggestion${RESET}"
  fi
}

format_success() {
  local message="$1"
  echo -e "${GREEN}✓${RESET} ${BOLD}$message${RESET}"
}

format_info() {
  local message="$1"
  echo -e "${BLUE}ℹ${RESET} $message"
}

format_step() {
  local step_num="$1"
  local total="$2"
  local message="$3"

  echo -e "\n${BOLD}[$step_num/$total]${RESET} $message"
}

format_section() {
  local title="$1"
  local width=${2:-50}

  local padding=$(((width - ${#title} - 2) / 2))
  local line=$(printf '%*s' "$width" | tr ' ' '─')

  echo -e "\n${BLUE}$line${RESET}"
  printf "${BLUE}│${RESET}%*s${BOLD}%s${RESET}%*s${BLUE}│${RESET}\n" \
    "$padding" "" "$title" "$padding" ""
  echo -e "${BLUE}$line${RESET}\n"
}

format_summary() {
  local title="$1"
  shift
  local items=("$@")

  echo -e "\n${BOLD}📊 $title${RESET}"
  echo -e "${DIM}$(printf '%.0s─' {1..40})${RESET}"

  for item in "${items[@]}"; do
    echo -e "  • $item"
  done

  echo -e "${DIM}$(printf '%.0s─' {1..40})${RESET}\n"
}
