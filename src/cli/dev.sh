#!/usr/bin/env bash
# dev.sh - Developer experience tools CLI
# Part of nself v0.7.0 - Sprint 19: Developer Experience Tools

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$(dirname "$SCRIPT_DIR")/lib"

# Source utilities
source "$LIB_DIR/dev/sdk-generator.sh"
source "$LIB_DIR/dev/docs-generator.sh"
source "$LIB_DIR/dev/test-helpers.sh"

# Show usage
show_usage() {
  cat << 'EOF'
nself dev - Developer Experience Tools

USAGE:
  nself dev <command> [options]

COMMANDS:
  SDK Generation:
    sdk generate <language> [output]   Generate SDK from GraphQL schema
                                       Languages: typescript, python
                                       Default output: ./sdk/<language>

    Example:
      nself dev sdk generate typescript
      nself dev sdk generate python ./my-sdk

  Documentation:
    docs generate [output]             Generate API documentation
                                       Default output: ./docs/api

    docs openapi [output]              Generate OpenAPI specification
                                       Default output: ./docs/api/openapi.yaml

    Example:
      nself dev docs generate
      nself dev docs openapi ./openapi.yaml

  Testing:
    test init [dir]                    Initialize test environment
                                       Default: .nself/test

    test fixtures <entity> [count]     Generate test fixtures
                                       Default count: 10

    test factory <entity> [output]     Generate mock data factory
                                       Default output: .nself/test/factories

    test snapshot create <name>        Create database snapshot
    test snapshot restore <name>       Restore database snapshot

    test run [dir]                     Run integration tests
                                       Default: .nself/test/integration

    Example:
      nself dev test init
      nself dev test fixtures users 50
      nself dev test factory users
      nself dev test snapshot create baseline
      nself dev test run

  Mock Data:
    mock <entity> <count>              Generate mock data
                                       Entities: users, posts, etc.

    Example:
      nself dev mock users 100

GLOBAL OPTIONS:
  -h, --help                          Show this help message
  -v, --verbose                       Verbose output
  --debug                             Debug mode

EXAMPLES:
  # Generate TypeScript SDK
  nself dev sdk generate typescript

  # Generate Python SDK to custom location
  nself dev sdk generate python ./backend/sdk

  # Generate API documentation
  nself dev docs generate

  # Initialize test environment
  nself dev test init

  # Generate 50 user fixtures
  nself dev test fixtures users 50

  # Create test snapshot
  nself dev test snapshot create clean-state

  # Generate mock users
  nself dev mock users 100

ENVIRONMENT VARIABLES:
  HASURA_GRAPHQL_ADMIN_SECRET        Admin secret for GraphQL API
  PROJECT_NAME                       Project name
  BASE_DOMAIN                        Base domain

For more information, visit: https://docs.nself.org
EOF
}

# Main command handler
main() {
  if [[ $# -eq 0 ]]; then
    show_usage
    exit 0
  fi

  local command="$1"
  shift

  case "$command" in
    sdk)
      handle_sdk_command "$@"
      ;;

    docs)
      handle_docs_command "$@"
      ;;

    test)
      handle_test_command "$@"
      ;;

    mock)
      handle_mock_command "$@"
      ;;

    -h|--help|help)
      show_usage
      exit 0
      ;;

    *)
      printf "Error: Unknown command '%s'\n\n" "$command" >&2
      show_usage
      exit 1
      ;;
  esac
}

# Handle SDK commands
handle_sdk_command() {
  if [[ $# -eq 0 ]]; then
    printf "Error: Missing SDK subcommand\n\n" >&2
    printf "Usage: nself dev sdk generate <language> [output]\n" >&2
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    generate)
      if [[ $# -eq 0 ]]; then
        printf "Error: Missing language parameter\n" >&2
        printf "Usage: nself dev sdk generate <language> [output]\n" >&2
        printf "Languages: typescript, python\n" >&2
        exit 1
      fi

      local language="$1"
      local output="${2:-./sdk/$language}"

      printf "Generating %s SDK...\n" "$language"
      generate_sdk "$language" "$output"
      ;;

    *)
      printf "Error: Unknown SDK subcommand '%s'\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

# Handle docs commands
handle_docs_command() {
  if [[ $# -eq 0 ]]; then
    printf "Error: Missing docs subcommand\n\n" >&2
    printf "Usage: nself dev docs <generate|openapi> [output]\n" >&2
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    generate)
      local output="${1:-./docs/api}"
      printf "Generating API documentation...\n"
      generate_docs "$output"
      ;;

    openapi)
      local output="${1:-./docs/api/openapi.yaml}"
      printf "Generating OpenAPI specification...\n"
      generate_openapi_spec "$output"
      ;;

    *)
      printf "Error: Unknown docs subcommand '%s'\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

# Handle test commands
handle_test_command() {
  if [[ $# -eq 0 ]]; then
    printf "Error: Missing test subcommand\n\n" >&2
    printf "Usage: nself dev test <init|fixtures|factory|snapshot|run> [options]\n" >&2
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    init)
      local test_dir="${1:-.nself/test}"
      init_test_environment "$test_dir"
      ;;

    fixtures)
      if [[ $# -eq 0 ]]; then
        printf "Error: Missing entity parameter\n" >&2
        printf "Usage: nself dev test fixtures <entity> [count] [output]\n" >&2
        exit 1
      fi

      local entity="$1"
      local count="${2:-10}"
      local output="${3:-.nself/test/fixtures/${entity}.json}"

      generate_fixtures "$entity" "$count" "$output"
      ;;

    factory)
      if [[ $# -eq 0 ]]; then
        printf "Error: Missing entity parameter\n" >&2
        printf "Usage: nself dev test factory <entity> [output]\n" >&2
        exit 1
      fi

      local entity="$1"
      local output="${2:-.nself/test/factories}"

      generate_mock_factory "$entity" "$output"
      ;;

    snapshot)
      if [[ $# -eq 0 ]]; then
        printf "Error: Missing snapshot action\n" >&2
        printf "Usage: nself dev test snapshot <create|restore> <name>\n" >&2
        exit 1
      fi

      local action="$1"
      shift

      case "$action" in
        create)
          local name="${1:-test-snapshot}"
          create_test_snapshot "$name"
          ;;

        restore)
          local name="${1:-test-snapshot}"
          restore_test_snapshot "$name"
          ;;

        *)
          printf "Error: Unknown snapshot action '%s'\n" "$action" >&2
          printf "Valid actions: create, restore\n" >&2
          exit 1
          ;;
      esac
      ;;

    run)
      local test_dir="${1:-.nself/test/integration}"
      run_integration_tests "$test_dir"
      ;;

    *)
      printf "Error: Unknown test subcommand '%s'\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

# Handle mock commands
handle_mock_command() {
  if [[ $# -lt 2 ]]; then
    printf "Error: Missing parameters\n" >&2
    printf "Usage: nself dev mock <entity> <count>\n" >&2
    exit 1
  fi

  local entity="$1"
  local count="$2"

  printf "Generating %s mock %s...\n" "$count" "$entity"

  # Generate mock data and output as JSON
  case "$entity" in
    users)
      local fixtures="["
      for i in $(seq 1 "$count"); do
        if [[ $i -gt 1 ]]; then
          fixtures+=","
        fi
        fixtures+="
  {
    \"id\": \"$(uuidgen | tr '[:upper:]' '[:lower:]')\",
    \"email\": \"user${i}@example.com\",
    \"displayName\": \"User ${i}\",
    \"createdAt\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"
  }"
      done
      fixtures+="
]"
      printf '%s\n' "$fixtures"
      ;;

    posts)
      local fixtures="["
      for i in $(seq 1 "$count"); do
        if [[ $i -gt 1 ]]; then
          fixtures+=","
        fi
        fixtures+="
  {
    \"id\": \"$(uuidgen | tr '[:upper:]' '[:lower:]')\",
    \"title\": \"Post ${i}\",
    \"content\": \"This is the content for post ${i}\",
    \"createdAt\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"
  }"
      done
      fixtures+="
]"
      printf '%s\n' "$fixtures"
      ;;

    *)
      printf "Error: Unknown entity '%s'\n" "$entity" >&2
      printf "Supported entities: users, posts\n" >&2
      exit 1
      ;;
  esac
}

# Run main
main "$@"
