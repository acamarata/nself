#!/bin/bash
# nself v0.9.9 bootstrap script
# This file is a v0.9 artifact. In v1.0.0+, use the nself binary directly.
set -e

NSELF_VERSION="0.9.9"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${1:-}" in
  start)
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" up -d
    ;;
  stop)
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" down
    ;;
  status)
    docker compose -f "$SCRIPT_DIR/docker-compose.yml" ps
    ;;
  *)
    echo "Usage: $0 {start|stop|status}"
    exit 1
    ;;
esac
