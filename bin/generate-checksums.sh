#!/bin/bash

# generate-checksums.sh - Generate SHA256 checksums for NSELF files

set -e

echo "Generating checksums for NSELF files..."

# Change to script directory
cd "$(dirname "$0")"

# Generate checksums for bin files
{
  cd bin
  sha256sum nself.sh build.sh compose.sh success.sh VERSION
  cd templates
  sha256sum .env.example | sed 's|^|templates/|'
} > checksums.sha256

echo "Checksums generated in checksums.sha256"
echo
echo "Contents:"
cat checksums.sha256