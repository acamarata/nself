#!/usr/bin/env bash

set -e

nself_update() {
	REPO_OWNER="acamarata"
	REPO_NAME="nself"
	SOURCE="${BASH_SOURCE[0]}"
	while [ -h "$SOURCE" ]; do
	  DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"
	  SOURCE="$(readlink "$SOURCE")"
	  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
	done
	SCRIPT_DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"

	# Updated path for new structure
	VERSION_FILE="$SCRIPT_DIR/../VERSION"
	GITHUB_API="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"

	if [[ -f "$VERSION_FILE" ]]; then
	    CURRENT_VERSION=$(cat "$VERSION_FILE")
	else
	    CURRENT_VERSION="0.0.0"
	fi

	echo "Current version: $CURRENT_VERSION"

	LATEST_JSON=$(curl -sL "$GITHUB_API")
	# Use sed instead of grep -P for macOS compatibility
	LATEST_VERSION=$(echo "$LATEST_JSON" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)

	echo "Latest version: $LATEST_VERSION"

	if [[ "$CURRENT_VERSION" == "$LATEST_VERSION" ]]; then
	    echo "Already up to date."
	    exit 0
	fi

	# Use sed instead of grep -P for macOS compatibility
	ASSET_URL=$(echo "$LATEST_JSON" | sed -n 's/.*"tarball_url":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
	if [[ -z "$ASSET_URL" ]]; then
	    echo "No .tar.gz asset found in the latest release."
	    exit 1
	fi

	TMP_DIR=$(mktemp -d)
	ARCHIVE_FILE="$TMP_DIR/nself_latest.tar.gz"
	EXTRACT_DIR="$TMP_DIR/extracted"

	echo "Downloading update..."
	curl -L "$ASSET_URL" -o "$ARCHIVE_FILE"

	echo "Extracting..."
	mkdir -p "$EXTRACT_DIR"
	tar -xzf "$ARCHIVE_FILE" -C "$EXTRACT_DIR" --strip-components=1

	echo "Updating..."
	# Get the parent directory to update the entire nself installation
	INSTALL_DIR="$(dirname "$SCRIPT_DIR")"
	rsync -a --delete "$EXTRACT_DIR/src/" "$INSTALL_DIR/"

	echo "$LATEST_VERSION" > "$VERSION_FILE"

	echo "Updated to version $LATEST_VERSION"
}

nself_update