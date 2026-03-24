#!/usr/bin/env bash
# docker-prune.sh - Docker image and resource cleanup
# Part of nself SysOps - Bash 3.2 compatible

set -euo pipefail

# Get the directory where this script is located
_PRUNE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_PRUNE_LIB_ROOT="$(dirname "$_PRUNE_DIR")"

# Source dependencies
source "$_PRUNE_LIB_ROOT/utils/display.sh" 2>/dev/null || true
source "$_PRUNE_LIB_ROOT/utils/platform-compat.sh" 2>/dev/null || true

# Prune old Docker images (older than 7 days by default)
docker_prune_images() {
  local age="${1:-168h}"  # 168h = 7 days
  local dry_run="${2:-false}"

  if ! command -v docker >/dev/null 2>&1; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "Docker not found" >&2
    return 1
  fi

  printf "\033[34m%s\033[0m Pruning Docker images older than %s...\n" "INFO" "$age"

  if [[ "$dry_run" == "true" ]]; then
    printf "\033[33m%s\033[0m DRY RUN - would remove:\n" "WARNING"
    docker image ls --filter "dangling=true" --format "  {{.Repository}}:{{.Tag}} ({{.Size}}, created {{.CreatedSince}})" 2>/dev/null || true
    docker image ls --filter "until=${age}" --format "  {{.Repository}}:{{.Tag}} ({{.Size}}, created {{.CreatedSince}})" 2>/dev/null || true
    return 0
  fi

  # Prune dangling images first
  local dangling_result
  dangling_result=$(docker image prune -f 2>/dev/null || true)
  if [[ -n "$dangling_result" ]]; then
    printf "%s\n" "$dangling_result"
  fi

  # Prune images older than specified age
  local age_result
  age_result=$(docker image prune -f --filter "until=${age}" 2>/dev/null || true)
  if [[ -n "$age_result" ]]; then
    printf "%s\n" "$age_result"
  fi

  printf "\033[32m%s\033[0m Docker image prune complete\n" "SUCCESS"
}

# Prune all unused Docker resources (images, containers, networks, build cache)
docker_prune_all() {
  local dry_run="${1:-false}"

  if ! command -v docker >/dev/null 2>&1; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "Docker not found" >&2
    return 1
  fi

  printf "\033[34m%s\033[0m Pruning all unused Docker resources...\n" "INFO"

  if [[ "$dry_run" == "true" ]]; then
    printf "\033[33m%s\033[0m DRY RUN - would run: docker system prune -f --filter 'until=168h'\n" "WARNING"
    docker system df 2>/dev/null || true
    return 0
  fi

  # System prune with age filter
  docker system prune -f --filter "until=168h" 2>/dev/null || true

  # Show reclaimed space
  printf "\n\033[34m%s\033[0m Current Docker disk usage:\n" "INFO"
  docker system df 2>/dev/null || true

  printf "\033[32m%s\033[0m Docker prune complete\n" "SUCCESS"
}

# Prune Docker build cache
docker_prune_buildcache() {
  printf "\033[34m%s\033[0m Pruning Docker build cache...\n" "INFO"
  docker builder prune -f --filter "until=168h" 2>/dev/null || true
  printf "\033[32m%s\033[0m Build cache pruned\n" "SUCCESS"
}

# Main dispatch
docker_prune() {
  local target="${1:-images}"
  shift 2>/dev/null || true

  case "$target" in
    images)
      docker_prune_images "$@"
      ;;
    all)
      docker_prune_all "$@"
      ;;
    buildcache|cache)
      docker_prune_buildcache "$@"
      ;;
    --help|help|-h)
      printf "nself maintenance docker-prune - Docker cleanup\n\n"
      printf "USAGE:\n"
      printf "  docker_prune images [age] [--dry-run]  Prune old images (default: 168h)\n"
      printf "  docker_prune all [--dry-run]           Prune all unused resources\n"
      printf "  docker_prune buildcache                Prune build cache\n"
      ;;
    *)
      printf "\033[33m%s\033[0m Unknown target: %s\n" "WARNING" "$target" >&2
      return 1
      ;;
  esac
}

# Export functions
export -f docker_prune 2>/dev/null || true
export -f docker_prune_images 2>/dev/null || true
export -f docker_prune_all 2>/dev/null || true
