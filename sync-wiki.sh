#!/bin/bash
# Manual script to sync /docs to GitHub wiki

set -e

echo "🔄 Syncing /docs to GitHub wiki..."

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Clone the wiki
echo "📥 Cloning wiki repository..."
git clone git@github.com:acamarata/nself.wiki.git "$TEMP_DIR/wiki" 2>/dev/null || {
    echo "❌ Failed to clone wiki. Make sure:"
    echo "   1. The wiki is initialized (create at least one page on GitHub)"
    echo "   2. You have SSH access to the repository"
    echo ""
    echo "To initialize the wiki:"
    echo "   1. Go to https://github.com/acamarata/nself/wiki"
    echo "   2. Click 'Create the first page'"
    echo "   3. Save any content"
    echo "   4. Run this script again"
    exit 1
}

# Remove old content (except .git)
echo "🧹 Cleaning old wiki content..."
find "$TEMP_DIR/wiki" -mindepth 1 -maxdepth 1 ! -name '.git' -exec rm -rf {} +

# Copy docs to wiki
echo "📋 Copying documentation..."
cp -r docs/* "$TEMP_DIR/wiki/"

# Handle special files
if [ -f "$TEMP_DIR/wiki/README.md" ]; then
    echo "📝 Converting README.md to Home.md..."
    mv "$TEMP_DIR/wiki/README.md" "$TEMP_DIR/wiki/Home.md"
fi

# Ensure Home.md exists
if [ ! -f "$TEMP_DIR/wiki/Home.md" ] && [ -f "$TEMP_DIR/wiki/Home.md" ]; then
    cp "$TEMP_DIR/wiki/Home.md" "$TEMP_DIR/wiki/Home.md"
fi

# The sidebar is already in docs/_Sidebar.md, so it will be copied

# Commit and push
cd "$TEMP_DIR/wiki"
git add -A

if git diff --staged --quiet; then
    echo "✅ Wiki is already up to date!"
else
    echo "💾 Committing changes..."
    git config user.name "$(git config --global user.name || echo 'Wiki Sync')"
    git config user.email "$(git config --global user.email || echo 'noreply@example.com')"
    git commit -m "Sync documentation from main repository

Updated: $(date '+%Y-%m-%d %H:%M:%S')
Source: Manual sync from /docs"
    
    echo "📤 Pushing to GitHub..."
    git push origin master 2>/dev/null || git push origin main || {
        echo "❌ Failed to push. Trying to pull and merge..."
        git pull --rebase origin master 2>/dev/null || git pull --rebase origin main
        git push origin master 2>/dev/null || git push origin main
    }
    
    echo "✅ Wiki successfully synchronized!"
fi

echo ""
echo "📚 View the wiki at: https://github.com/acamarata/nself/wiki"