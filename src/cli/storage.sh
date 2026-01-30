#!/usr/bin/env bash
# storage.sh - File storage and upload management CLI
# Part of nself storage system

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required libraries
source "${SCRIPT_DIR}/../lib/utils/display.sh"

# Compatibility aliases for output functions
output_info() { log_info "$@"; }
output_success() { log_success "$@"; }
output_error() { log_error "$@"; }
output_warning() { log_warning "$@"; }

source "${SCRIPT_DIR}/../lib/storage/upload-pipeline.sh"
source "${SCRIPT_DIR}/../lib/storage/graphql-integration.sh"

# Source load-env if it exists (optional for testing)
[[ -f "${SCRIPT_DIR}/../lib/config/load-env.sh" ]] && source "${SCRIPT_DIR}/../lib/config/load-env.sh" || true

#######################################
# Show storage command help
#######################################
show_storage_help() {
  cat << 'EOF'
nself storage - File storage and upload management

USAGE:
  nself storage <command> [options]

COMMANDS:
  upload <file>              Upload a file to storage
  list [prefix]              List uploaded files
  delete <path>              Delete an uploaded file
  config                     Configure upload pipeline
  status                     Show pipeline status
  test                       Test upload functionality
  init                       Initialize storage system
  graphql-setup              Generate GraphQL integration package

UPLOAD OPTIONS:
  --dest <path>              Destination path in storage
  --thumbnails               Generate image thumbnails
  --virus-scan               Scan file for viruses
  --compression              Compress large files
  --all-features             Enable all features

EXAMPLES:
  # Upload a file
  nself storage upload photo.jpg

  # Upload with thumbnails
  nself storage upload avatar.png --thumbnails

  # Upload with all features
  nself storage upload document.pdf --all-features

  # Upload to specific path
  nself storage upload file.txt --dest users/123/documents/

  # List all uploads
  nself storage list

  # List uploads in folder
  nself storage list users/123/

  # Delete a file
  nself storage delete users/123/file.txt

  # Show configuration
  nself storage config

  # Test upload system
  nself storage test

  # Generate GraphQL integration
  nself storage graphql-setup

CONFIGURATION:
  Set these in your .env file:

  # Storage backend
  STORAGE_BACKEND=minio              # Options: minio, s3, gcs
  MINIO_ENDPOINT=http://minio:9000
  MINIO_ACCESS_KEY=minioadmin
  MINIO_SECRET_KEY=minioadmin
  MINIO_BUCKET=uploads

  # Upload features
  UPLOAD_ENABLE_MULTIPART=true       # Enable multipart uploads
  UPLOAD_ENABLE_THUMBNAILS=false     # Generate thumbnails
  UPLOAD_ENABLE_VIRUS_SCAN=false     # Scan for viruses
  UPLOAD_ENABLE_COMPRESSION=true     # Compress large files

  # Thumbnail configuration
  UPLOAD_THUMBNAIL_SIZES=150x150,300x300,600x600
  UPLOAD_IMAGE_FORMATS=avif,webp,jpg

  # Public URL
  STORAGE_PUBLIC_URL=http://storage.localhost

For more information:
  https://docs.nself.org/storage/uploads

EOF
}

#######################################
# Upload file command
# Arguments:
#   $@ - Command arguments
#######################################
cmd_upload() {
  local file_path=""
  local dest_path=""
  local options=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dest)
        dest_path="$2"
        shift 2
        ;;
      --thumbnails)
        options="${options:+${options},}thumbnails"
        shift
        ;;
      --virus-scan)
        options="${options:+${options},}virus-scan"
        shift
        ;;
      --compression)
        options="${options:+${options},}compression"
        shift
        ;;
      --all-features)
        options="thumbnails,virus-scan,compression"
        shift
        ;;
      --help|-h)
        show_storage_help
        return 0
        ;;
      *)
        if [[ -z "${file_path}" ]]; then
          file_path="$1"
        else
          output_error "Unknown argument: $1"
          return 1
        fi
        shift
        ;;
    esac
  done

  # Validate file path provided
  if [[ -z "${file_path}" ]]; then
    output_error "File path required"
    printf "\nUsage: nself storage upload <file> [options]\n"
    printf "Run 'nself storage upload --help' for more information\n"
    return 1
  fi

  # Initialize upload pipeline
  if ! init_upload_pipeline; then
    output_error "Failed to initialize upload pipeline"
    return 1
  fi

  # Upload file
  if upload_file "${file_path}" "${dest_path}" "${options}"; then
    return 0
  else
    output_error "Upload failed"
    return 1
  fi
}

#######################################
# List files command
# Arguments:
#   $1 - Prefix (optional)
#######################################
cmd_list() {
  local prefix="${1:-}"

  # Initialize upload pipeline
  if ! init_upload_pipeline; then
    output_error "Failed to initialize upload pipeline"
    return 1
  fi

  output_info "Listing files in: ${MINIO_BUCKET}${prefix:+/${prefix}}"
  printf "\n"

  if list_uploads "${prefix}"; then
    return 0
  else
    output_error "Failed to list files"
    return 1
  fi
}

#######################################
# Delete file command
# Arguments:
#   $1 - File path
#######################################
cmd_delete() {
  local file_path="${1:-}"

  if [[ -z "${file_path}" ]]; then
    output_error "File path required"
    printf "\nUsage: nself storage delete <path>\n"
    return 1
  fi

  # Initialize upload pipeline
  if ! init_upload_pipeline; then
    output_error "Failed to initialize upload pipeline"
    return 1
  fi

  # Confirm deletion
  printf "Delete file: %s? [y/N] " "${file_path}"
  read -r response
  response=$(echo "$response" | tr '[:upper:]' '[:lower:]')

  if [[ "${response}" != "y" ]]; then
    output_info "Deletion cancelled"
    return 0
  fi

  if delete_upload "${file_path}"; then
    return 0
  else
    return 1
  fi
}

#######################################
# Show storage configuration
#######################################
cmd_config() {
  cat << EOF

Storage Configuration
====================

Backend Configuration:
  STORAGE_BACKEND        = ${STORAGE_BACKEND:-minio}
  MINIO_ENDPOINT         = ${MINIO_ENDPOINT:-http://minio:9000}
  MINIO_BUCKET           = ${MINIO_BUCKET:-uploads}
  STORAGE_PUBLIC_URL     = ${STORAGE_PUBLIC_URL:-http://storage.localhost}

Upload Features:
  UPLOAD_ENABLE_MULTIPART     = ${UPLOAD_ENABLE_MULTIPART:-true}
  UPLOAD_ENABLE_THUMBNAILS    = ${UPLOAD_ENABLE_THUMBNAILS:-false}
  UPLOAD_ENABLE_VIRUS_SCAN    = ${UPLOAD_ENABLE_VIRUS_SCAN:-false}
  UPLOAD_ENABLE_COMPRESSION   = ${UPLOAD_ENABLE_COMPRESSION:-true}

Thumbnail Configuration:
  UPLOAD_THUMBNAIL_SIZES = ${UPLOAD_THUMBNAIL_SIZES:-150x150,300x300,600x600}
  UPLOAD_IMAGE_FORMATS   = ${UPLOAD_IMAGE_FORMATS:-avif,webp,jpg}

To modify configuration, edit your .env file:
  vi .env.dev

Or set environment variables:
  export UPLOAD_ENABLE_THUMBNAILS=true
  nself storage upload photo.jpg

EOF
}

#######################################
# Show pipeline status
#######################################
cmd_status() {
  # Initialize upload pipeline
  if ! init_upload_pipeline; then
    output_error "Failed to initialize upload pipeline"
    return 1
  fi

  get_pipeline_status
}

#######################################
# Test upload functionality
#######################################
cmd_test() {
  output_info "Testing upload pipeline..."
  printf "\n"

  # Initialize upload pipeline
  if ! init_upload_pipeline; then
    output_error "Pipeline initialization failed"
    return 1
  fi

  output_success "Pipeline initialized"

  # Create test file
  local test_file="/tmp/nself_upload_test_$$.txt"
  printf "nself upload test - %s\n" "$(date)" > "${test_file}"

  output_info "Created test file: ${test_file}"

  # Test upload
  if upload_file "${test_file}" "test/upload_test.txt" ""; then
    output_success "Upload test passed"

    # Clean up
    delete_upload "test/upload_test.txt" >/dev/null 2>&1 || true
    rm -f "${test_file}"

    printf "\n"
    output_success "All tests passed!"
    return 0
  else
    output_error "Upload test failed"
    rm -f "${test_file}"
    return 1
  fi
}

#######################################
# Initialize storage system
#######################################
cmd_init() {
  output_info "Initializing storage system..."

  # Check if MinIO is enabled
  if [[ "${MINIO_ENABLED:-false}" != "true" ]]; then
    output_warning "MinIO is not enabled in your configuration"
    printf "\nTo enable MinIO, add to your .env file:\n"
    printf "  MINIO_ENABLED=true\n\n"
    printf "Then rebuild and restart:\n"
    printf "  nself build && nself start\n\n"
    return 1
  fi

  # Initialize upload pipeline
  if init_upload_pipeline; then
    output_success "Storage system initialized"

    printf "\n"
    get_pipeline_status

    printf "\nNext steps:\n"
    printf "  1. Upload a file: nself storage upload <file>\n"
    printf "  2. View configuration: nself storage config\n"
    printf "  3. Run tests: nself storage test\n\n"

    return 0
  else
    output_error "Failed to initialize storage system"
    return 1
  fi
}

#######################################
# Generate GraphQL integration package
#######################################
cmd_graphql_setup() {
  local output_dir="${1:-.backend/storage}"

  output_info "Generating GraphQL integration package..."
  printf "\n"

  # Create output directory
  if [[ ! -d "${output_dir}" ]]; then
    mkdir -p "${output_dir}"
    output_info "Created directory: ${output_dir}"
  fi

  # Generate package
  if generate_graphql_package "${output_dir}"; then
    output_success "GraphQL integration generated successfully!"

    printf "\nGenerated Files:\n"
    printf "  %s/migrations/     - Database migration\n" "${output_dir}"
    printf "  %s/metadata/       - Hasura permissions\n" "${output_dir}"
    printf "  %s/graphql/        - GraphQL operations\n" "${output_dir}"
    printf "  %s/types/          - TypeScript types\n" "${output_dir}"
    printf "  %s/hooks/          - React hooks\n" "${output_dir}"
    printf "  %s/README.md       - Integration guide\n" "${output_dir}"

    printf "\nNext Steps:\n"
    printf "  1. Review generated files in %s\n" "${output_dir}"
    printf "  2. Run migration:\n"
    printf "     psql \$DATABASE_URL < %s/migrations/*_create_files_table.sql\n" "${output_dir}"
    printf "  3. Apply Hasura metadata:\n"
    printf "     hasura metadata apply\n"
    printf "  4. Copy types and hooks to your frontend:\n"
    printf "     cp %s/types/files.ts src/types/\n" "${output_dir}"
    printf "     cp %s/hooks/useFiles.ts src/hooks/\n" "${output_dir}"
    printf "\n"

    return 0
  else
    output_error "Failed to generate GraphQL integration"
    return 1
  fi
}

#######################################
# Main storage command router
# Arguments:
#   $@ - Command arguments
#######################################
main() {
  local command="${1:-}"

  # Handle no command or help
  if [[ -z "${command}" ]] || [[ "${command}" == "help" ]] || [[ "${command}" == "--help" ]] || [[ "${command}" == "-h" ]]; then
    show_storage_help
    return 0
  fi

  shift

  # Route to command
  case "${command}" in
    upload)
      cmd_upload "$@"
      ;;
    list|ls)
      cmd_list "$@"
      ;;
    delete|rm)
      cmd_delete "$@"
      ;;
    config)
      cmd_config
      ;;
    status)
      cmd_status
      ;;
    test)
      cmd_test
      ;;
    init)
      cmd_init
      ;;
    graphql-setup)
      cmd_graphql_setup "$@"
      ;;
    *)
      output_error "Unknown command: ${command}"
      printf "\nRun 'nself storage help' for usage information\n"
      return 1
      ;;
  esac
}

# Run main function if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
