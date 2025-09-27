#!/usr/bin/env bash
# docker-compose-simple.sh - Simplified docker compose startup

# Simple docker compose up that doesn't hang
simple_compose_up() {
  local project="${1:-nself}"
  local env_file="${2:-.env.runtime}"
  local verbose="${3:-false}"

  local compose_cmd=$(get_compose_command 2>/dev/null || echo "docker compose")

  printf "${COLOR_BLUE}⠋${COLOR_RESET} Starting Docker services (this may take a few minutes if images need downloading)...\n"

  # Start services without building - use pre-built images
  if [ "$verbose" = "true" ]; then
    $compose_cmd --project-name "$project" --env-file "$env_file" up -d --no-build
  else
    $compose_cmd --project-name "$project" --env-file "$env_file" up -d --no-build 2>&1 | \
      grep -E "(Pulling|Creating|Starting|Created|Started|Container|Network|Volume)" || true
  fi

  local result=$?

  # Give services a moment to stabilize
  sleep 3

  # Count running services
  local running_count=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')

  if [ "$running_count" -gt 0 ]; then
    printf "${COLOR_GREEN}✓${COLOR_RESET} Started %d services\n" "$running_count"
    return 0
  else
    printf "${COLOR_YELLOW}⚠${COLOR_RESET}  No services started - check docker-compose.yml and .env.runtime\n"
    return 1
  fi
}

export -f simple_compose_up