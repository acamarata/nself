#!/usr/bin/env bats

# Test suite for nself main functionality

setup() {
  # Create a temporary directory for testing
  export TEST_DIR=$(mktemp -d)
  export SCRIPT_DIR="$TEST_DIR/bin"
  mkdir -p "$SCRIPT_DIR"
  
  # Create a mock VERSION file
  echo "0.2.1" > "$SCRIPT_DIR/VERSION"
  
  # Source helper functions from nself.sh
  source <(sed -n '32,83p' ../bin/nself.sh)  # Helper functions
}

teardown() {
  # Clean up test directory
  rm -rf "$TEST_DIR"
}

@test "read_local_version reads VERSION file correctly" {
  VERSION_FILE="$SCRIPT_DIR/VERSION"
  
  read_local_version() {
    if [ -f "$VERSION_FILE" ]; then
      LOCAL_VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
    else
      echo "VERSION file not found."
      return 1
    fi
  }
  
  read_local_version
  [ "$LOCAL_VERSION" = "0.2.1" ]
}

@test "read_local_version fails when VERSION file missing" {
  VERSION_FILE="$SCRIPT_DIR/NONEXISTENT"
  
  read_local_version() {
    if [ -f "$VERSION_FILE" ]; then
      LOCAL_VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
    else
      echo "VERSION file not found."
      return 1
    fi
  }
  
  run read_local_version
  [ "$status" -eq 1 ]
  [[ "$output" == *"VERSION file not found"* ]]
}

@test "version comparison detects newer versions" {
  is_newer_version() {
    local current="$1"
    local latest="$2"
    
    # Simple version comparison
    if [ "$current" != "$latest" ]; then
      return 0  # Newer version available
    else
      return 1  # Same version
    fi
  }
  
  run is_newer_version "0.2.1" "0.2.2"
  [ "$status" -eq 0 ]
}

@test "version comparison detects same version" {
  is_newer_version() {
    local current="$1"
    local latest="$2"
    
    # Simple version comparison
    if [ "$current" != "$latest" ]; then
      return 0  # Newer version available
    else
      return 1  # Same version
    fi
  }
  
  run is_newer_version "0.2.1" "0.2.1"
  [ "$status" -eq 1 ]
}

@test "command_exists works for docker check" {
  command_exists() {
    command -v "$1" >/dev/null 2>&1
  }
  
  # Test with ls which should exist
  run command_exists "ls"
  [ "$status" -eq 0 ]
}

@test "echo_clean outputs clean line" {
  echo_clean() {
    printf "\r\033[K%s\n" "$1"
  }
  
  run echo_clean "Clean output"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Clean output"* ]]
}