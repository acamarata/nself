#!/bin/bash
# Debug script for nself build issues

echo "=== nself Build Debug Script ==="
echo "Date: $(date)"
echo "User: $(whoami)"
echo "PWD: $(pwd)"
echo "OS: $(uname -a)"
echo ""

echo "=== Environment Check ==="
echo "PATH: $PATH"
echo "SHELL: $SHELL"
echo "BASH_VERSION: $BASH_VERSION"
echo ""

echo "=== nself Installation Check ==="
echo "which nself: $(which nself)"
echo "nself location: $(readlink -f $(which nself))"
NSELF_BIN=$(which nself)
if [[ -n "$NSELF_BIN" ]]; then
  NSELF_ROOT=$(dirname $(dirname $(readlink -f "$NSELF_BIN")))
  echo "nself root: $NSELF_ROOT"
  echo "nself version: $(cat "$NSELF_ROOT/src/VERSION" 2>/dev/null || echo 'not found')"
fi
echo ""

echo "=== Directory Contents ==="
echo "Files in current directory:"
ls -la
echo ""

echo "=== .env File Check ==="
if [[ -f ".env" ]]; then
  echo ".env exists ($(wc -l < .env) lines)"
  echo "First 5 non-comment lines:"
  grep -v "^#" .env | grep -v "^$" | head -5
elif [[ -f ".env.local" ]]; then
  echo ".env.local exists ($(wc -l < .env.local) lines)"
  echo "First 5 non-comment lines:"
  grep -v "^#" .env.local | grep -v "^$" | head -5
else
  echo "No .env or .env.local file found"
fi
echo ""

echo "=== Docker Check ==="
echo "Docker version: $(docker --version 2>&1)"
echo "Docker info: $(docker info --format 'Containers: {{.Containers}}, Images: {{.Images}}' 2>&1)"
echo "Docker compose version: $(docker compose version 2>&1)"
echo ""

echo "=== Direct Script Execution Test ==="
if [[ -n "$NSELF_ROOT" ]]; then
  echo "Attempting to source build.sh directly..."

  # Try to source required files
  export SCRIPT_DIR="$NSELF_ROOT/src/cli"

  # Source display utilities
  if [[ -f "$SCRIPT_DIR/../lib/utils/display.sh" ]]; then
    source "$SCRIPT_DIR/../lib/utils/display.sh"
    echo "✓ Loaded display.sh"
  else
    echo "✗ Cannot load display.sh from $SCRIPT_DIR/../lib/utils/display.sh"
  fi

  # Check if build.sh exists
  if [[ -f "$SCRIPT_DIR/build.sh" ]]; then
    echo "✓ Found build.sh at $SCRIPT_DIR/build.sh"

    # Check if cmd_build function exists
    if grep -q "^cmd_build()" "$SCRIPT_DIR/build.sh"; then
      echo "✓ cmd_build function exists in build.sh"
    else
      echo "✗ cmd_build function not found in build.sh"
    fi
  else
    echo "✗ build.sh not found at $SCRIPT_DIR/build.sh"
  fi
fi
echo ""

echo "=== Running Build with Debug ==="
export DEBUG=true
echo "Executing: DEBUG=true nself build"
DEBUG=true timeout 10 nself build 2>&1 | head -30
echo ""

echo "=== Post-Build Check ==="
if [[ -f "docker-compose.yml" ]]; then
  echo "✓ docker-compose.yml created ($(wc -l < docker-compose.yml) lines)"
else
  echo "✗ docker-compose.yml NOT created"
fi
echo ""

echo "=== Debug Complete ==="
echo "Please share this output for troubleshooting"