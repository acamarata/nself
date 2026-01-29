#!/usr/bin/env bash
# nself perf - Performance optimization
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/../lib/utils/output.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"

perf_optimize() {
    info "Optimizing database..."
    docker exec -i postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "ANALYZE"
    docker exec -i postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "VACUUM ANALYZE"
    success "Optimization complete"
}

perf_slow_queries() {
    docker exec -i postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
    "SELECT * FROM performance.get_slow_queries(1000, 20)"
}

case "${1:-help}" in
    optimize) perf_optimize ;;
    slow-queries) perf_slow_queries ;;
    *) printf "Usage: nself perf {optimize|slow-queries}\n" ;;
esac
