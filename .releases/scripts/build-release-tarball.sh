#!/usr/bin/env bash
#
# Build minimal release tarball for nself
# Only includes files needed for runtime, excludes development/test files
#

set -e

# Configuration
VERSION="${1:-$(cat src/VERSION)}"
RELEASE_DIR=".releases/v${VERSION}"
TARBALL_NAME="nself-v${VERSION}.tar.gz"
TEMP_DIR=$(mktemp -d)
BUILD_DIR="${TEMP_DIR}/nself-${VERSION}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
RESET='\033[0m'

log_info() { echo -e "${BLUE}ℹ${RESET} $1"; }
log_success() { echo -e "${GREEN}✓${RESET} $1"; }
log_warning() { echo -e "${YELLOW}⚠${RESET} $1"; }
log_error() { echo -e "${RED}✗${RESET} $1" >&2; }

# Create build directory
log_info "Creating minimal release for nself v${VERSION}..."
mkdir -p "${BUILD_DIR}"
mkdir -p "${RELEASE_DIR}"

# Copy essential files and directories
log_info "Copying essential runtime files..."

# Core executable
cp -r bin "${BUILD_DIR}/"

# Core source (excluding tests and development files)
mkdir -p "${BUILD_DIR}/src"
cp src/VERSION "${BUILD_DIR}/src/"

# CLI commands (all needed for runtime)
cp -r src/cli "${BUILD_DIR}/src/"

# Libraries (all needed for runtime)
cp -r src/lib "${BUILD_DIR}/src/"

# Services configurations (needed for docker generation)
mkdir -p "${BUILD_DIR}/src/services"
cp -r src/services/docker "${BUILD_DIR}/src/services/"

# Templates (needed for project initialization)
mkdir -p "${BUILD_DIR}/src/templates"
# Copy only essential templates, not test/example files
find src/templates -type f \( \
    -name "*.template" -o \
    -name ".env.dev" -o \
    -name ".env.staging" -o \
    -name ".env.prod" -o \
    -name ".env.secrets" -o \
    -name "Dockerfile" -o \
    -name "docker-compose*.yml" -o \
    -name "*.conf" \
\) | while read -r file; do
    rel_path="${file#src/templates/}"
    mkdir -p "${BUILD_DIR}/src/templates/$(dirname "$rel_path")"
    cp "$file" "${BUILD_DIR}/src/templates/$rel_path"
done

# Copy service templates (needed for microservices)
cp -r src/templates/services "${BUILD_DIR}/src/templates/" 2>/dev/null || true

# Essential configs
cp -r src/templates/hasura "${BUILD_DIR}/src/templates/" 2>/dev/null || true
cp -r src/templates/certs "${BUILD_DIR}/src/templates/" 2>/dev/null || true
cp -r src/templates/nginx "${BUILD_DIR}/src/templates/" 2>/dev/null || true

# Documentation (minimal)
cp LICENSE "${BUILD_DIR}/" 2>/dev/null || true
cp README.md "${BUILD_DIR}/" 2>/dev/null || true

# Create minimal docs directory with only essential guides
mkdir -p "${BUILD_DIR}/docs"
for doc in COMMANDS.md QUICK_START.md CONFIGURATION.md; do
    [ -f "docs/${doc}" ] && cp "docs/${doc}" "${BUILD_DIR}/docs/"
done

# Get original directory before cd
ORIG_DIR="$(pwd)"

# Create tarball
log_info "Creating tarball: ${TARBALL_NAME}..."
cd "${TEMP_DIR}"
tar -czf "${TARBALL_NAME}" "nself-${VERSION}"

# Move to release directory
mv "${TARBALL_NAME}" "${ORIG_DIR}/${RELEASE_DIR}/"
cd "${ORIG_DIR}" > /dev/null

# Calculate sizes
FULL_SIZE=$(du -sh . | cut -f1)
TARBALL_SIZE=$(du -sh "${RELEASE_DIR}/${TARBALL_NAME}" | cut -f1)
FILE_COUNT=$(tar -tzf "${RELEASE_DIR}/${TARBALL_NAME}" | wc -l | tr -d ' ')

# Clean up
rm -rf "${TEMP_DIR}"

# Report
log_success "Release tarball created successfully!"
echo ""
echo "Release Info:"
echo "  Version:     v${VERSION}"
echo "  Tarball:     ${RELEASE_DIR}/${TARBALL_NAME}"
echo "  Size:        ${TARBALL_SIZE} (from ${FULL_SIZE} source)"
echo "  Files:       ${FILE_COUNT} files"
echo ""
echo "Excluded from release:"
echo "  ✗ Test files and fixtures"
echo "  ✗ Development scripts"
echo "  ✗ Git history and .github"
echo "  ✗ Release building tools (.releases)"
echo "  ✗ Example and demo files"
echo "  ✗ IDE configurations"
echo ""
echo "To test installation:"
echo "  tar -xzf ${RELEASE_DIR}/${TARBALL_NAME}"
echo "  cd nself-${VERSION}"
echo "  ./bin/nself version"