#!/usr/bin/env bash
set -euo pipefail

# search.sh - Search service management commands

# Determine root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source required utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/utils/platform-compat.sh"

# Show help for search command
show_search_help() {
  echo "nself search - Search service management"
  echo ""
  echo "Usage: nself search <subcommand> [OPTIONS]"
  echo ""
  echo "Subcommands:"
  echo "  enable     Enable search with selected engine"
  echo "  disable    Disable search service"
  echo "  status     Show search service status"
  echo "  list       List available search engines"
  echo "  setup      Interactive search setup wizard"
  echo "  test       Test search functionality"
  echo "  reindex    Rebuild search index"
  echo "  config     Show current configuration"
  echo "  docs       Show search documentation"
  echo ""
  echo "Description:"
  echo "  Manages enterprise search functionality with support for multiple"
  echo "  search engines including PostgreSQL FTS, MeiliSearch, Typesense,"
  echo "  Elasticsearch, OpenSearch, and Sonic."
  echo ""
  echo "Examples:"
  echo "  nself search enable meilisearch  # Enable MeiliSearch"
  echo "  nself search setup                # Interactive setup"
  echo "  nself search test \"hello world\"   # Test search"
  echo "  nself search status               # Check status"
}

# Get search engine info
get_search_engine_info() {
  local engine="${1:-meilisearch}"
  
  case "$engine" in
  postgres)
    echo "PostgreSQL Full-Text Search"
    echo "  - Built into existing database (no extra container)"
    echo "  - Good for small to medium datasets"
    echo "  - Supports phrase search and ranking"
    echo "  - Resource efficient"
    ;;
  meilisearch)
    echo "Meilisearch"
    echo "  - Lightning fast, typo-tolerant search"
    echo "  - Best developer experience"
    echo "  - Instant search optimized"
    echo "  - Faceted search and filtering"
    echo "  - Recommended for most use cases"
    echo "  - Port: ${MEILISEARCH_PORT:-7700}"
    ;;
  typesense)
    echo "Typesense"
    echo "  - Fast, typo-tolerant search"
    echo "  - High performance alternative"
    echo "  - Good for production workloads"
    echo "  - Faceted search support"
    echo "  - Port: ${TYPESENSE_PORT:-8108}"
    ;;
  elasticsearch)
    echo "Elasticsearch"
    echo "  - Industry standard, most powerful"
    echo "  - Complex query DSL"
    echo "  - Aggregations and analytics"
    echo "  - Resource intensive (2GB+ RAM)"
    echo "  - Best for large-scale applications"
    echo "  - Port: ${ELASTICSEARCH_PORT:-9200}"
    ;;
  opensearch)
    echo "OpenSearch"
    echo "  - Open source Elasticsearch fork by AWS"
    echo "  - Elasticsearch compatible"
    echo "  - Security features included"
    echo "  - Includes OpenSearch Dashboards"
    echo "  - Good for AWS deployments"
    echo "  - Port: ${OPENSEARCH_PORT:-9200}"
    ;;
  zinc)
    echo "Zinc"
    echo "  - Lightweight Elasticsearch alternative"
    echo "  - Written in Go, minimal resources"
    echo "  - Simple REST API"
    echo "  - Good for logs and metrics"
    echo "  - Port: ${ZINC_PORT:-4080}"
    ;;
  sonic)
    echo "Sonic"
    echo "  - Ultra-lightweight and fast"
    echo "  - Minimal resource usage (<100MB RAM)"
    echo "  - Best for autocomplete"
    echo "  - Simple protocol"
    echo "  - Limited features"
    echo "  - Port: ${SONIC_PORT:-1491}"
    ;;
  *)
    echo "Unknown search engine: $engine"
    return 1
    ;;
  esac
}

# Get default port for search engine
get_search_engine_port() {
  local engine="${1:-meilisearch}"
  
  case "$engine" in
  postgres) echo "${POSTGRES_PORT:-5432}" ;;
  meilisearch) echo "${MEILISEARCH_PORT:-7700}" ;;
  typesense) echo "${TYPESENSE_PORT:-8108}" ;;
  elasticsearch) echo "${ELASTICSEARCH_PORT:-9200}" ;;
  opensearch) echo "${OPENSEARCH_PORT:-9200}" ;;
  zinc) echo "${ZINC_PORT:-4080}" ;;
  sonic) echo "${SONIC_PORT:-1491}" ;;
  *) echo "7700" ;;
  esac
}

# Enable search
search_enable() {
  local engine="${1:-}"
  
  show_command_header "nself search enable" "Enable search service"
  
  # Load environment
  load_env_with_priority
  
  # If no engine specified, prompt
  if [[ -z "$engine" ]]; then
    echo "Available search engines:"
    echo "  1) meilisearch  - Fast, typo-tolerant (RECOMMENDED)"
    echo "  2) typesense    - High-performance alternative"
    echo "  3) zinc         - Lightweight Elasticsearch alternative"
    echo "  4) elasticsearch - Most powerful (2GB+ RAM)"
    echo "  5) opensearch   - Open source Elasticsearch fork"
    echo "  6) sonic        - Ultra-lightweight autocomplete"
    echo "  7) postgres     - PostgreSQL FTS (no extra container)"
    echo ""
    echo -n "Select engine (1-7) [1]: "
    local choice
    read choice
    
    case "${choice:-1}" in
    1) engine="meilisearch" ;;
    2) engine="typesense" ;;
    3) engine="zinc" ;;
    4) engine="elasticsearch" ;;
    5) engine="opensearch" ;;
    6) engine="sonic" ;;
    7) engine="postgres" ;;
    *)
      log_error "Invalid choice"
      return 1
      ;;
    esac
  fi
  
  # Validate engine
  case "$engine" in
  postgres|meilisearch|typesense|elasticsearch|opensearch|zinc|sonic)
    log_info "Enabling search with $engine..."
    ;;
  *)
    log_error "Invalid search engine: $engine"
    echo "Valid options: meilisearch, typesense, zinc, elasticsearch, opensearch, sonic, postgres"
    return 1
    ;;
  esac
  
  # Update .env.local
  if grep -q "^SEARCH_ENABLED=" .env.local 2>/dev/null; then
    safe_sed_inline .env.local 's/^SEARCH_ENABLED=.*/SEARCH_ENABLED=true/'
  else
    echo "SEARCH_ENABLED=true" >> .env.local
  fi
  
  if grep -q "^SEARCH_ENGINE=" .env.local 2>/dev/null; then
    safe_sed_inline .env.local "s/^SEARCH_ENGINE=.*/SEARCH_ENGINE=$engine/"
  else
    echo "SEARCH_ENGINE=$engine" >> .env.local
  fi
  
  # Set default port
  local port=$(get_search_engine_port "$engine")
  if ! grep -q "^SEARCH_PORT=" .env.local 2>/dev/null; then
    echo "SEARCH_PORT=$port" >> .env.local
  fi
  
  # Set default host
  if ! grep -q "^SEARCH_HOST=" .env.local 2>/dev/null; then
    if [[ "$engine" == "postgres" ]]; then
      echo "SEARCH_HOST=postgres" >> .env.local
    else
      echo "SEARCH_HOST=search" >> .env.local
    fi
  fi
  
  # Generate API key for engines that need it
  if [[ "$engine" == "meilisearch" ]] || [[ "$engine" == "typesense" ]]; then
    if ! grep -q "^SEARCH_API_KEY=" .env.local 2>/dev/null || [[ -z "${SEARCH_API_KEY:-}" ]]; then
      local api_key=$(openssl rand -hex 32)
      if grep -q "^SEARCH_API_KEY=" .env.local 2>/dev/null; then
        safe_sed_inline .env.local "s/^SEARCH_API_KEY=.*/SEARCH_API_KEY=$api_key/"
      else
        echo "SEARCH_API_KEY=$api_key" >> .env.local
      fi
      log_info "Generated API key for $engine"
    fi
  fi
  
  log_success "Search enabled with $engine"
  echo ""
  get_search_engine_info "$engine"
  echo ""
  log_info "Run 'nself build' to generate configuration"
  log_info "Then 'nself start' to launch the search service"
}

# Disable search
search_disable() {
  show_command_header "nself search disable" "Disable search service"
  
  # Load environment
  load_env_with_priority
  
  log_info "Disabling search service..."
  
  # Update .env.local
  if grep -q "^SEARCH_ENABLED=" .env.local 2>/dev/null; then
    safe_sed_inline .env.local 's/^SEARCH_ENABLED=.*/SEARCH_ENABLED=false/'
  else
    echo "SEARCH_ENABLED=false" >> .env.local
  fi
  
  # Stop search container if running (except postgres)
  local engine="${SEARCH_ENGINE:-postgres}"
  if [[ "$engine" != "postgres" ]]; then
    if docker ps --format "{{.Names}}" | grep -q "nself-search"; then
      log_info "Stopping search container..."
      docker stop nself-search >/dev/null 2>&1
      docker rm nself-search >/dev/null 2>&1
    fi
  fi
  
  log_success "Search service disabled"
  log_info "Run 'nself build' to update configuration"
}

# Show search status
search_status() {
  show_command_header "nself search status" "Search service status"
  
  # Load environment
  load_env_with_priority
  
  local search_enabled="${SEARCH_ENABLED:-false}"
  local search_engine="${SEARCH_ENGINE:-postgres}"
  local search_host="${SEARCH_HOST:-search}"
  local search_port="${SEARCH_PORT:-$(get_search_engine_port "$search_engine")}"
  
  echo "Configuration:"
  echo "  Enabled:  $search_enabled"
  echo "  Engine:   $search_engine"
  echo "  Host:     $search_host"
  echo "  Port:     $search_port"
  
  if [[ -n "${SEARCH_API_KEY:-}" ]]; then
    echo "  API Key:  [SET]"
  else
    echo "  API Key:  [NOT SET]"
  fi
  
  if [[ -n "${SEARCH_INDEX_PREFIX:-}" ]]; then
    echo "  Index Prefix: $SEARCH_INDEX_PREFIX"
  fi
  
  echo "  Auto Index:   ${SEARCH_AUTO_INDEX:-true}"
  echo "  Language:     ${SEARCH_LANGUAGE:-en}"
  
  echo ""
  echo "Engine Details:"
  get_search_engine_info "$search_engine"
  
  if [[ "$search_enabled" == "true" ]]; then
    echo ""
    echo "Container Status:"
    
    if [[ "$search_engine" == "postgres" ]]; then
      # Check PostgreSQL container
      if docker ps --format "{{.Names}}" | grep -q "postgres"; then
        log_success "PostgreSQL is running (search uses existing database)"
      else
        log_warning "PostgreSQL is not running"
        log_info "Run 'nself start' to launch it"
      fi
    else
      # Check dedicated search container
      if docker ps --format "{{.Names}}" | grep -q "nself-search"; then
        log_success "Search container is running"
        
        # Show container info
        local container_info=$(docker ps --filter "name=nself-search" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1)
        echo "  $container_info"
      else
        log_warning "Search container is not running"
        log_info "Run 'nself start' to launch it"
      fi
    fi
  fi
}

# List available search engines
search_list() {
  show_command_header "nself search list" "Available search engines"
  
  echo "PostgreSQL FTS (postgres)"
  get_search_engine_info "postgres"
  echo ""
  
  echo "MeiliSearch (meilisearch) - RECOMMENDED"
  get_search_engine_info "meilisearch"
  echo ""
  
  echo "Typesense (typesense)"
  get_search_engine_info "typesense"
  echo ""
  
  echo "Elasticsearch (elasticsearch)"
  get_search_engine_info "elasticsearch"
  echo ""
  
  echo "OpenSearch (opensearch)"
  get_search_engine_info "opensearch"
  echo ""
  
  echo "Sonic (sonic)"
  get_search_engine_info "sonic"
}

# Interactive search setup
search_setup() {
  show_command_header "nself search setup" "Interactive search setup"
  
  # Load environment
  load_env_with_priority
  
  echo "This wizard will help you configure search for your application."
  echo ""
  
  # Ask about use case
  echo "What is your primary use case?"
  echo "  1) Product search (e-commerce)"
  echo "  2) Content search (blog/CMS)"
  echo "  3) User search (social/directory)"
  echo "  4) Log search (monitoring)"
  echo "  5) Autocomplete only"
  echo "  6) General purpose"
  echo ""
  echo -n "Select use case (1-6) [6]: "
  local use_case
  read use_case
  use_case="${use_case:-6}"
  
  # Recommend engine based on use case
  local recommended_engine="meilisearch"
  case "$use_case" in
  1|2|3|6)
    recommended_engine="meilisearch"
    echo ""
    log_info "Recommended: MeiliSearch (fast, typo-tolerant, faceted search)"
    ;;
  4)
    recommended_engine="elasticsearch"
    echo ""
    log_info "Recommended: Elasticsearch (powerful analytics and aggregations)"
    ;;
  5)
    recommended_engine="sonic"
    echo ""
    log_info "Recommended: Sonic (lightweight, fast autocomplete)"
    ;;
  esac
  
  echo ""
  echo "Available search engines:"
  echo "  1) postgres     - Built-in, no extra container"
  echo "  2) meilisearch  - Fast, typo-tolerant (recommended)"
  echo "  3) typesense    - Alternative to MeiliSearch"
  echo "  4) elasticsearch - Most powerful (2GB+ RAM)"
  echo "  5) opensearch   - Open source Elasticsearch"
  echo "  6) sonic        - Lightweight autocomplete"
  echo ""
  
  local default_choice=2
  case "$recommended_engine" in
  postgres) default_choice=1 ;;
  meilisearch) default_choice=2 ;;
  typesense) default_choice=3 ;;
  elasticsearch) default_choice=4 ;;
  opensearch) default_choice=5 ;;
  sonic) default_choice=6 ;;
  esac
  
  echo -n "Select engine (1-6) [$default_choice]: "
  local choice
  read choice
  choice="${choice:-$default_choice}"
  
  local engine
  case "$choice" in
  1) engine="postgres" ;;
  2) engine="meilisearch" ;;
  3) engine="typesense" ;;
  4) engine="elasticsearch" ;;
  5) engine="opensearch" ;;
  6) engine="sonic" ;;
  *)
    log_error "Invalid choice"
    return 1
    ;;
  esac
  
  echo ""
  log_info "Configuring $engine..."
  
  # Configure index prefix for multi-tenant
  echo ""
  echo -n "Index prefix (for multi-tenant apps) []: "
  local index_prefix
  read index_prefix
  
  if [[ -n "$index_prefix" ]]; then
    if grep -q "^SEARCH_INDEX_PREFIX=" .env.local 2>/dev/null; then
      safe_sed_inline .env.local "s/^SEARCH_INDEX_PREFIX=.*/SEARCH_INDEX_PREFIX=$index_prefix/"
    else
      echo "SEARCH_INDEX_PREFIX=$index_prefix" >> .env.local
    fi
  fi
  
  # Configure language
  echo ""
  echo -n "Default language (en, es, fr, de, etc.) [en]: "
  local language
  read language
  language="${language:-en}"
  
  if grep -q "^SEARCH_LANGUAGE=" .env.local 2>/dev/null; then
    safe_sed_inline .env.local "s/^SEARCH_LANGUAGE=.*/SEARCH_LANGUAGE=$language/"
  else
    echo "SEARCH_LANGUAGE=$language" >> .env.local
  fi
  
  # Enable search with selected engine
  echo ""
  search_enable "$engine"
}

# Test search functionality
search_test() {
  local query="${1:-test query}"
  
  show_command_header "nself search test" "Test search functionality"
  
  # Load environment
  load_env_with_priority
  
  local search_enabled="${SEARCH_ENABLED:-false}"
  local search_engine="${SEARCH_ENGINE:-postgres}"
  
  if [[ "$search_enabled" != "true" ]]; then
    log_error "Search is not enabled"
    log_info "Run 'nself search enable' first"
    return 1
  fi
  
  log_info "Testing search with query: \"$query\""
  echo ""
  
  case "$search_engine" in
  postgres)
    log_info "Testing PostgreSQL full-text search..."
    
    # Check if PostgreSQL is running
    if [[ "${POSTGRES_ENABLED:-false}" == "true" ]]; then
      local pg_host="${POSTGRES_HOST:-postgres}"
      local pg_port="${POSTGRES_PORT:-5432}"
      local pg_db="${POSTGRES_DB:-nself}"
      local pg_user="${POSTGRES_USER:-nself}"
      
      # Test with psql if available
      if command -v psql >/dev/null 2>&1; then
        if PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "$pg_host" -p "$pg_port" -U "$pg_user" -d "$pg_db" -c "SELECT 1;" >/dev/null 2>&1; then
          log_success "PostgreSQL connection successful"
          
          # Test full-text search capabilities
          local test_query="SELECT to_tsvector('english', 'The quick brown fox') @@ to_tsquery('english', 'quick & fox');"
          if PGPASSWORD="${POSTGRES_PASSWORD:-}" psql -h "$pg_host" -p "$pg_port" -U "$pg_user" -d "$pg_db" -c "$test_query" >/dev/null 2>&1; then
            log_success "PostgreSQL full-text search is working"
          else
            log_warning "PostgreSQL connected but full-text search test failed"
          fi
        else
          log_error "Could not connect to PostgreSQL"
          log_info "Check your database configuration and credentials"
        fi
      else
        # Try with docker exec if postgres container exists
        if docker ps --format '{{.Names}}' | grep -q postgres; then
          if docker exec postgres psql -U "$pg_user" -d "$pg_db" -c "SELECT 1;" >/dev/null 2>&1; then
            log_success "PostgreSQL connection successful (via Docker)"
            
            # Test full-text search
            local test_query="SELECT to_tsvector('english', 'The quick brown fox') @@ to_tsquery('english', 'quick & fox');"
            if docker exec postgres psql -U "$pg_user" -d "$pg_db" -c "$test_query" >/dev/null 2>&1; then
              log_success "PostgreSQL full-text search is working"
            else
              log_warning "PostgreSQL connected but full-text search test failed"
            fi
          else
            log_error "Could not connect to PostgreSQL container"
          fi
        else
          log_warning "psql not found and no PostgreSQL container running"
          log_info "Install psql or ensure PostgreSQL container is running"
        fi
      fi
    else
      log_warning "PostgreSQL is not enabled in configuration"
      log_info "Set POSTGRES_ENABLED=true to use PostgreSQL search"
    fi
    ;;
    
  meilisearch)
    log_info "Testing MeiliSearch..."
    local url="http://${SEARCH_HOST:-search}:${SEARCH_PORT:-7700}/health"
    
    if command -v curl >/dev/null 2>&1; then
      if curl -s "$url" >/dev/null 2>&1; then
        log_success "MeiliSearch is responding"
        
        # Try to create a test index and search
        if [[ -n "${SEARCH_API_KEY:-}" ]]; then
          # Create test index
          curl -s -X POST "http://${SEARCH_HOST:-search}:${SEARCH_PORT:-7700}/indexes" \
            -H "Authorization: Bearer ${SEARCH_API_KEY}" \
            -H "Content-Type: application/json" \
            -d '{"uid": "test_temp", "primaryKey": "id"}' >/dev/null 2>&1
          
          log_info "Test index created"
          
          # Clean up test index
          curl -s -X DELETE "http://${SEARCH_HOST:-search}:${SEARCH_PORT:-7700}/indexes/test_temp" \
            -H "Authorization: Bearer ${SEARCH_API_KEY}" >/dev/null 2>&1
          
          log_info "Test index cleaned up"
        fi
      else
        log_error "MeiliSearch is not responding"
        log_info "Check if the container is running with 'nself search status'"
      fi
    else
      log_warning "curl not found, cannot test connectivity"
    fi
    ;;
    
  typesense)
    log_info "Testing Typesense..."
    local url="http://${SEARCH_HOST:-search}:${SEARCH_PORT:-8108}/health"

    if command -v curl >/dev/null 2>&1; then
      local api_key="${SEARCH_API_KEY:-}"
      local health_response

      if [[ -n "$api_key" ]]; then
        health_response=$(curl -s -H "X-TYPESENSE-API-KEY: $api_key" "$url" 2>/dev/null)
      else
        health_response=$(curl -s "$url" 2>/dev/null)
      fi

      if echo "$health_response" | grep -q "ok"; then
        log_success "Typesense is responding and healthy"

        # Get cluster info
        local info_url="http://${SEARCH_HOST:-search}:${SEARCH_PORT:-8108}/debug"
        if [[ -n "$api_key" ]]; then
          curl -s -H "X-TYPESENSE-API-KEY: $api_key" "$info_url" 2>/dev/null | head -5
        fi
      else
        log_error "Typesense is not responding"
        log_info "Check if the container is running with 'nself search status'"
      fi
    else
      log_warning "curl not found, cannot test connectivity"
    fi
    ;;

  elasticsearch)
    log_info "Testing Elasticsearch..."
    local url="http://${SEARCH_HOST:-search}:${SEARCH_PORT:-9200}"

    if command -v curl >/dev/null 2>&1; then
      local health_response=$(curl -s "$url/_cluster/health" 2>/dev/null)

      if echo "$health_response" | grep -q "status"; then
        local status=$(echo "$health_response" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        if [[ "$status" == "green" ]]; then
          log_success "Elasticsearch cluster is healthy (green)"
        elif [[ "$status" == "yellow" ]]; then
          log_warning "Elasticsearch cluster status is yellow (some replicas unavailable)"
        else
          log_error "Elasticsearch cluster status is $status"
        fi

        # Show cluster info
        curl -s "$url" 2>/dev/null | grep -E '"name"|"cluster_name"|"version"' | head -5
      else
        log_error "Elasticsearch is not responding"
      fi
    else
      log_warning "curl not found, cannot test connectivity"
    fi
    ;;

  opensearch)
    log_info "Testing OpenSearch..."
    local url="http://${SEARCH_HOST:-search}:${SEARCH_PORT:-9200}"

    if command -v curl >/dev/null 2>&1; then
      local health_response=$(curl -s "$url/_cluster/health" 2>/dev/null)

      if echo "$health_response" | grep -q "status"; then
        local status=$(echo "$health_response" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        if [[ "$status" == "green" ]]; then
          log_success "OpenSearch cluster is healthy (green)"
        elif [[ "$status" == "yellow" ]]; then
          log_warning "OpenSearch cluster status is yellow (some replicas unavailable)"
        else
          log_error "OpenSearch cluster status is $status"
        fi

        # Show cluster info
        curl -s "$url" 2>/dev/null | grep -E '"name"|"cluster_name"|"version"' | head -5
      else
        log_error "OpenSearch is not responding"
      fi
    else
      log_warning "curl not found, cannot test connectivity"
    fi
    ;;

  sonic)
    log_info "Testing Sonic..."
    local host="${SEARCH_HOST:-search}"
    local port="${SEARCH_PORT:-1491}"

    # Sonic uses a TCP protocol, not HTTP
    if command -v nc >/dev/null 2>&1; then
      if nc -z "$host" "$port" 2>/dev/null; then
        log_success "Sonic is responding on port $port"
        log_info "Sonic uses a custom TCP protocol"
        log_info "Use the Sonic client library to interact with it"
      else
        log_error "Cannot connect to Sonic at $host:$port"
      fi
    else
      log_warning "nc not found, cannot test TCP connectivity"
      log_info "Try: nc -z $host $port"
    fi
    ;;

  *)
    log_info "Testing $search_engine..."
    log_warning "Automated test not implemented for $search_engine"
    log_info "Please test manually using the engine's API or client library"
    ;;
  esac

  echo ""
  log_info "Integration example for your application:"

  case "$search_engine" in
  postgres)
    echo "  // PostgreSQL full-text search"
    echo "  SELECT * FROM items"
    echo "  WHERE to_tsvector('english', title || ' ' || description)"
    echo "  @@ plainto_tsquery('english', '$query');"
    ;;

  meilisearch)
    echo "  // MeiliSearch JavaScript SDK"
    echo "  const { MeiliSearch } = require('meilisearch');"
    echo "  const client = new MeiliSearch({"
    echo "    host: 'http://${SEARCH_HOST:-search}:${SEARCH_PORT:-7700}',"
    echo "    apiKey: '${SEARCH_API_KEY:-YOUR_API_KEY}'"
    echo "  });"
    echo "  const results = await client.index('products').search('$query');"
    ;;

  typesense)
    echo "  // Typesense JavaScript SDK"
    echo "  const Typesense = require('typesense');"
    echo "  const client = new Typesense.Client({"
    echo "    nodes: [{ host: '${SEARCH_HOST:-search}', port: ${SEARCH_PORT:-8108}, protocol: 'http' }],"
    echo "    apiKey: '${SEARCH_API_KEY:-YOUR_API_KEY}'"
    echo "  });"
    echo "  const results = await client.collections('products').documents().search({ q: '$query' });"
    ;;

  elasticsearch|opensearch)
    echo "  // Elasticsearch/OpenSearch JavaScript SDK"
    echo "  const { Client } = require('@elastic/elasticsearch');"
    echo "  const client = new Client({ node: 'http://${SEARCH_HOST:-search}:${SEARCH_PORT:-9200}' });"
    echo "  const results = await client.search({"
    echo "    index: 'products',"
    echo "    body: { query: { match: { title: '$query' } } }"
    echo "  });"
    ;;

  sonic)
    echo "  // Sonic Node.js client"
    echo "  const Sonic = require('sonic-channel');"
    echo "  const search = new Sonic.Search({"
    echo "    host: '${SEARCH_HOST:-search}',"
    echo "    port: ${SEARCH_PORT:-1491},"
    echo "    auth: '${SEARCH_API_KEY:-SecretPassword}'"
    echo "  });"
    echo "  const results = await search.query('collection', 'bucket', '$query');"
    ;;

  *)
    echo "  // Check documentation for $search_engine integration"
    echo "  // Host: ${SEARCH_HOST:-search}"
    echo "  // Port: ${SEARCH_PORT:-$(get_search_engine_port "$search_engine")}"
    ;;
  esac
}

# Reindex search data
search_reindex() {
  show_command_header "nself search reindex" "Rebuild search index"
  
  # Load environment
  load_env_with_priority
  
  local search_enabled="${SEARCH_ENABLED:-false}"
  local search_engine="${SEARCH_ENGINE:-postgres}"
  
  if [[ "$search_enabled" != "true" ]]; then
    log_error "Search is not enabled"
    log_info "Run 'nself search enable' first"
    return 1
  fi
  
  log_info "Triggering reindex for $search_engine..."
  echo ""
  
  case "$search_engine" in
  postgres)
    log_info "PostgreSQL reindexing:"
    echo "  Run these commands in your database:"
    echo "  REINDEX INDEX your_fts_index;"
    echo "  VACUUM ANALYZE your_table;"
    ;;
    
  meilisearch)
    log_info "MeiliSearch reindexing:"
    echo "  MeiliSearch automatically indexes data on insertion."
    echo "  To force reindex:"
    echo "  1. Delete the index"
    echo "  2. Recreate it"
    echo "  3. Re-insert all documents"
    ;;
    
  elasticsearch|opensearch)
    log_info "$search_engine reindexing:"
    echo "  POST /_reindex"
    echo "  {"
    echo "    \"source\": {\"index\": \"old_index\"},"
    echo "    \"dest\": {\"index\": \"new_index\"}"
    echo "  }"
    ;;
    
  *)
    log_warning "Reindex procedure not defined for $search_engine"
    log_info "Please consult the $search_engine documentation"
    ;;
  esac
  
  echo ""
  log_info "Note: Actual reindexing must be triggered from your application"
}

# Show search configuration
search_config() {
  show_command_header "nself search config" "Current search configuration"
  
  # Load environment
  load_env_with_priority
  
  echo "Search Environment Variables:"
  echo ""
  
  # Show all SEARCH_ variables
  env | grep "^SEARCH_" | sort | while IFS='=' read -r key value; do
    if [[ "$key" == "SEARCH_API_KEY" ]] && [[ -n "$value" ]]; then
      echo "$key=[REDACTED]"
    else
      echo "$key=$value"
    fi
  done
  
  if ! env | grep -q "^SEARCH_"; then
    log_info "No search configuration found"
    log_info "Run 'nself search setup' to configure"
  fi
}

# Show search documentation
search_configure() {
  local engine="${1:-}"
  
  if [[ -z "$engine" ]]; then
    log_error "Search engine required"
    echo "Usage: nself search configure <engine>"
    echo "Engines: meilisearch, typesense, zinc, elasticsearch, opensearch, sonic"
    return 1
  fi
  
  show_command_header "Search" "Configuring search engine: $engine"
  
  load_env_with_priority
  
  # Update .env
  safe_sed_inline .env "s/^SEARCH_ENGINE=.*/SEARCH_ENGINE=$engine/" 2>/dev/null || echo "SEARCH_ENGINE=$engine" >> .env
  
  log_success "Search engine configured: $engine"
  log_info "Rebuild and restart to apply changes:"
  log_info "  nself build"
  log_info "  nself restart"
}

search_clear() {
  local index="${1:-}"
  
  show_command_header "Search" "Clearing search index"
  
  load_env_with_priority
  
  if [[ "${SEARCH_ENABLED:-false}" != "true" ]]; then
    log_error "Search is not enabled"
    return 1
  fi
  
  local engine="${SEARCH_ENGINE:-meilisearch}"
  local port=$(get_search_engine_port "$engine")
  
  case "$engine" in
    meilisearch)
      if [[ -z "$index" ]]; then
        curl -X DELETE "http://localhost:$port/indexes" \
          -H "Authorization: Bearer ${MEILISEARCH_MASTER_KEY}"
        log_success "All indexes cleared"
      else
        curl -X DELETE "http://localhost:$port/indexes/$index" \
          -H "Authorization: Bearer ${MEILISEARCH_MASTER_KEY}"
        log_success "Index '$index' cleared"
      fi
      ;;
    *)
      log_warning "Clear not implemented for $engine"
      ;;
  esac
}

search_import() {
  local file="${1:-}"
  
  if [[ -z "$file" ]] || [[ ! -f "$file" ]]; then
    log_error "File required: nself search import <file>"
    return 1
  fi
  
  show_command_header "Search" "Importing data from $file"
  
  load_env_with_priority
  
  if [[ "${SEARCH_ENABLED:-false}" != "true" ]]; then
    log_error "Search is not enabled"
    return 1
  fi
  
  local engine="${SEARCH_ENGINE:-meilisearch}"
  local port=$(get_search_engine_port "$engine")
  
  case "$engine" in
    meilisearch)
      local index="${2:-documents}"
      curl -X POST "http://localhost:$port/indexes/$index/documents" \
        -H "Authorization: Bearer ${MEILISEARCH_MASTER_KEY}" \
        -H "Content-Type: application/json" \
        --data-binary "@$file"
      log_success "Data imported to index '$index'"
      ;;
    *)
      log_warning "Import not implemented for $engine"
      ;;
  esac
}

search_export() {
  local output="${1:-search-export.json}"
  
  show_command_header "Search" "Exporting search data to $output"
  
  load_env_with_priority
  
  if [[ "${SEARCH_ENABLED:-false}" != "true" ]]; then
    log_error "Search is not enabled"
    return 1
  fi
  
  local engine="${SEARCH_ENGINE:-meilisearch}"
  local port=$(get_search_engine_port "$engine")
  
  case "$engine" in
    meilisearch)
      local indexes=$(curl -s "http://localhost:$port/indexes" \
        -H "Authorization: Bearer ${MEILISEARCH_MASTER_KEY}" | jq -r '.results[].uid')
      
      echo "{" > "$output"
      local first=true
      for index in $indexes; do
        if [[ "$first" != "true" ]]; then echo "," >> "$output"; fi
        first=false
        echo "  \"$index\": " >> "$output"
        curl -s "http://localhost:$port/indexes/$index/documents" \
          -H "Authorization: Bearer ${MEILISEARCH_MASTER_KEY}" >> "$output"
      done
      echo "}" >> "$output"
      log_success "Data exported to $output"
      ;;
    *)
      log_warning "Export not implemented for $engine"
      ;;
  esac
}

search_dashboard() {
  show_command_header "Search" "Opening search dashboard"
  
  load_env_with_priority
  
  if [[ "${SEARCH_ENABLED:-false}" != "true" ]]; then
    log_error "Search is not enabled"
    return 1
  fi
  
  local engine="${SEARCH_ENGINE:-meilisearch}"
  local port=$(get_search_engine_port "$engine")
  local url=""
  
  case "$engine" in
    meilisearch)
      url="http://localhost:$port"
      ;;
    zinc)
      url="http://localhost:$port"
      ;;
    opensearch)
      url="http://localhost:5601"  # OpenSearch Dashboards
      ;;
    *)
      log_warning "No dashboard available for $engine"
      return 1
      ;;
  esac
  
  if [[ -n "$url" ]]; then
    log_info "Opening dashboard: $url"
    if command -v open >/dev/null 2>&1; then
      open "$url"
    elif command -v xdg-open >/dev/null 2>&1; then
      xdg-open "$url"
    else
      log_info "Please open in browser: $url"
    fi
  fi
}

search_health() {
  show_command_header "Search" "Checking search service health"
  
  load_env_with_priority
  
  if [[ "${SEARCH_ENABLED:-false}" != "true" ]]; then
    log_error "Search is not enabled"
    return 1
  fi
  
  local engine="${SEARCH_ENGINE:-meilisearch}"
  local port=$(get_search_engine_port "$engine")
  
  case "$engine" in
    meilisearch)
      if curl -s "http://localhost:$port/health" | grep -q "available"; then
        log_success "Meilisearch is healthy"
      else
        log_error "Meilisearch is unhealthy"
      fi
      ;;
    elasticsearch|opensearch)
      if curl -s "http://localhost:$port/_cluster/health" | grep -q "green\|yellow"; then
        log_success "$engine cluster is healthy"
      else
        log_error "$engine cluster is unhealthy"
      fi
      ;;
    *)
      if curl -s "http://localhost:$port" >/dev/null 2>&1; then
        log_success "$engine is responding"
      else
        log_error "$engine is not responding"
      fi
      ;;
  esac
}

search_logs() {
  show_command_header "Search" "Viewing search service logs"
  
  load_env_with_priority
  
  if [[ "${SEARCH_ENABLED:-false}" != "true" ]]; then
    log_error "Search is not enabled"
    return 1
  fi
  
  local follow="${1:-}"
  local engine="${SEARCH_ENGINE:-meilisearch}"
  
  # Map engine to container name
  local container_name="search"
  case "$engine" in
    meilisearch) container_name="meilisearch" ;;
    typesense) container_name="typesense" ;;
    elasticsearch) container_name="elasticsearch" ;;
    opensearch) container_name="opensearch" ;;
    zinc) container_name="zinc" ;;
    sonic) container_name="sonic" ;;
  esac
  
  if [[ "$follow" == "-f" ]] || [[ "$follow" == "--follow" ]]; then
    compose logs -f "$container_name"
  else
    compose logs --tail=50 "$container_name"
  fi
}

search_docs() {
  local engine="${1:-}"
  
  show_command_header "nself search docs" "Search documentation"
  
  if [[ -z "$engine" ]]; then
    # General documentation
    echo "nself Search Integration Guide"
    echo "=============================="
    echo ""
    echo "Quick Start:"
    echo "  1. Enable search:    nself search enable [engine]"
    echo "  2. Build config:     nself build"
    echo "  3. Start services:   nself start"
    echo "  4. Test search:      nself search test"
    echo ""
    echo "Supported Engines:"
    echo "  - postgres:     Built-in PostgreSQL full-text search"
    echo "  - meilisearch:  Fast, typo-tolerant search (recommended)"
    echo "  - typesense:    Alternative to MeiliSearch"
    echo "  - elasticsearch: Industry standard, most powerful"
    echo "  - opensearch:   Open source Elasticsearch fork"
    echo "  - sonic:        Lightweight autocomplete"
    echo ""
    echo "For engine-specific docs: nself search docs <engine>"
  else
    # Engine-specific documentation
    echo "$engine Integration Guide"
    echo "=============================="
    echo ""
    
    case "$engine" in
    postgres)
      echo "PostgreSQL Full-Text Search"
      echo ""
      echo "Setup:"
      echo "  No additional container needed - uses existing PostgreSQL"
      echo ""
      echo "SQL Example:"
      echo "  -- Create text search column"
      echo "  ALTER TABLE products ADD COLUMN search_vector tsvector;"
      echo ""
      echo "  -- Create index"
      echo "  CREATE INDEX products_search_idx ON products USING GIN(search_vector);"
      echo ""
      echo "  -- Update search vector"
      echo "  UPDATE products SET search_vector ="
      echo "    to_tsvector('english', title || ' ' || description);"
      echo ""
      echo "  -- Search query"
      echo "  SELECT * FROM products"
      echo "  WHERE search_vector @@ plainto_tsquery('english', 'search term');"
      ;;
      
    meilisearch)
      echo "MeiliSearch Integration"
      echo ""
      echo "Docker Image: getmeili/meilisearch:latest"
      echo "Default Port: 7700"
      echo ""
      echo "JavaScript SDK:"
      echo "  npm install meilisearch"
      echo ""
      echo "  const { MeiliSearch } = require('meilisearch');"
      echo "  const client = new MeiliSearch({"
      echo "    host: 'http://search:7700',"
      echo "    apiKey: process.env.SEARCH_API_KEY"
      echo "  });"
      echo ""
      echo "  // Create index"
      echo "  await client.createIndex('products', { primaryKey: 'id' });"
      echo ""
      echo "  // Add documents"
      echo "  await client.index('products').addDocuments(products);"
      echo ""
      echo "  // Search"
      echo "  const results = await client.index('products').search('query');"
      echo ""
      echo "Features:"
      echo "  - Typo tolerance"
      echo "  - Faceted search"
      echo "  - Filtering"
      echo "  - Sorting"
      echo "  - Highlighting"
      ;;
      
    elasticsearch)
      echo "Elasticsearch Integration"
      echo ""
      echo "Docker Image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0"
      echo "Default Port: 9200"
      echo "Memory Required: 2GB+ RAM"
      echo ""
      echo "JavaScript Client:"
      echo "  npm install @elastic/elasticsearch"
      echo ""
      echo "  const { Client } = require('@elastic/elasticsearch');"
      echo "  const client = new Client({"
      echo "    node: 'http://search:9200'"
      echo "  });"
      echo ""
      echo "  // Index document"
      echo "  await client.index({"
      echo "    index: 'products',"
      echo "    document: { title: 'Product', description: '...' }"
      echo "  });"
      echo ""
      echo "  // Search"
      echo "  const result = await client.search({"
      echo "    index: 'products',"
      echo "    query: { match: { title: 'search term' } }"
      echo "  });"
      ;;
      
    *)
      log_error "Documentation not available for: $engine"
      echo "Please check the official $engine documentation"
      ;;
    esac
  fi
}

# Main command function
cmd_search() {
  local subcommand="${1:-}"
  shift || true
  
  case "$subcommand" in
  enable)
    search_enable "$@"
    ;;
  disable)
    search_disable "$@"
    ;;
  status)
    search_status "$@"
    ;;
  configure)
    search_configure "$@"
    ;;
  list)
    search_list "$@"
    ;;
  setup)
    search_setup "$@"
    ;;
  test)
    search_test "$@"
    ;;
  reindex)
    search_reindex "$@"
    ;;
  clear)
    search_clear "$@"
    ;;
  import)
    search_import "$@"
    ;;
  export)
    search_export "$@"
    ;;
  dashboard)
    search_dashboard "$@"
    ;;
  health)
    search_health "$@"
    ;;
  logs)
    search_logs "$@"
    ;;
  config)
    search_config "$@"
    ;;
  docs)
    search_docs "$@"
    ;;
  -h | --help | help | "")
    show_search_help
    ;;
  *)
    log_error "Unknown subcommand: $subcommand"
    echo ""
    show_search_help
    return 1
    ;;
  esac
}

# Export for use as library
export -f cmd_search

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_search "$@"
fi