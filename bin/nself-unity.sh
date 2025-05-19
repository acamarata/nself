#!/bin/bash

# nself-init.sh - Helper script to create project directory structure and initial files

set -e

# ----------------------------
# Resolve Script Directory
# ----------------------------
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
  DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
SCRIPT_DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"

UNITY_DIR="$SCRIPT_DIR/../unity"
TARGET_DIR="$PWD"               # Destination (current directory)

sync_unity(){
	echo "Checking files and directories from '$UNITY_DIR'..."
	
	if [ ! -d "$UNITY_DIR" ]; then
	    echo "❌ Error: Source directory '$UNITY_DIR' not found."
	    exit 1
	fi

	# Iterate over all files and directories in 'unity'
	shopt -s dotglob

	# Loop through all files and directories inside UNITY_DIR
	for item in "$UNITY_DIR"/* "$UNITY_DIR"/.*; do
	    # Skip special directories (.) and (..)
	    [[ $(basename "$item") == "." || $(basename "$item") == ".." ]] && continue
	    
	    rel_path="${item#$UNITY_DIR/}"  # Get relative path
	    target_path="$TARGET_DIR/$rel_path"  # Destination path in the current directory
	    echo $rel_path 
	    echo $target_path 
	    if [ -e "$target_path" ]; then
	        # If the file exists, ask the user whether to replace it
	        echo "⚠️ '$rel_path' already exists."
	        read -p "Do you want to replace it? (y/n): " choice
	        case "$choice" in
	            y|Y ) 
	                rm -rf "$target_path"
	                cp -r "$item" "$target_path"
	                echo "✅ Replaced '$rel_path'."
	                ;;
	            n|N ) 
	                echo "✅ Keeping existing '$rel_path'."
	                ;;
	            * ) 
	                echo "❌ Invalid choice, keeping existing '$rel_path'."
	                ;;
	        esac
	    else
	        # If the file does not exist, copy it
	        cp -r "$item" "$target_path"
	        echo "✅ Copied '$rel_path'."
	    fi
	done

	echo "🎉 Sync complete!"
}

sync_unity