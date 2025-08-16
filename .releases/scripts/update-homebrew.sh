#!/bin/bash

# update-homebrew.sh - Update Homebrew formula for nself
# Usage: ./scripts/update-homebrew.sh <version>

set -e

VERSION="${1:-}"
TAP_REPO="acamarata/homebrew-nself"
FORMULA_NAME="nself"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# Remove 'v' prefix for version number
VERSION_NUM="${VERSION#v}"

echo "📦 Updating Homebrew formula for nself ${VERSION}"

# Create temp directory
TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

# Download release tarball to calculate SHA256
echo "⬇️  Downloading release tarball..."
wget -q "https://github.com/acamarata/nself/archive/refs/tags/${VERSION}.tar.gz"
SHA256=$(shasum -a 256 "${VERSION}.tar.gz" | cut -d' ' -f1)
echo "✓ SHA256: ${SHA256}"

# Clone or update tap repository
if [[ -d "$HOME/homebrew-nself" ]]; then
    echo "📂 Using existing tap repository..."
    cd "$HOME/homebrew-nself"
    git pull
else
    echo "📂 Cloning tap repository..."
    git clone "https://github.com/${TAP_REPO}.git" "$HOME/homebrew-nself"
    cd "$HOME/homebrew-nself"
fi

# Create Formula directory if it doesn't exist
mkdir -p Formula

# Note: The actual formula template is in .releases/homebrew/
# Generate formula
echo "📝 Generating formula..."
cat > "Formula/${FORMULA_NAME}.rb" << EOF
class Nself < Formula
  desc "Self-hosted infrastructure manager for developers"
  homepage "https://nself.org"
  url "https://github.com/acamarata/nself/archive/refs/tags/${VERSION}.tar.gz"
  sha256 "${SHA256}"
  license "MIT"
  head "https://github.com/acamarata/nself.git", branch: "main"

  depends_on "docker"
  depends_on "bash" if OS.linux?

  def install
    # Install all files to libexec
    libexec.install Dir["*"]
    
    # Create wrapper script
    (bin/"nself").write <<~EOS
      #!/bin/bash
      export NSELF_HOME="#{libexec}"
      export PATH="#{libexec}/bin:\$PATH"
      exec "#{libexec}/bin/nself" "\$@"
    EOS
    
    chmod 0755, bin/"nself"
    
    # Install completions
    bash_completion.install "#{libexec}/completions/nself.bash" if File.exist?("#{libexec}/completions/nself.bash")
    zsh_completion.install "#{libexec}/completions/_nself" if File.exist?("#{libexec}/completions/_nself")
  end

  def caveats
    <<~EOS
      nself has been installed!
      
      To get started:
        mkdir my-project && cd my-project
        nself init
        nself build
        nself start
      
      Documentation: https://github.com/acamarata/nself
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/nself version")
    assert_match "init", shell_output("#{bin}/nself help")
  end
end
EOF

echo "✓ Formula generated"

# Test the formula locally
echo "🧪 Testing formula locally..."
if command -v brew &> /dev/null; then
    brew install --build-from-source "Formula/${FORMULA_NAME}.rb" || true
    brew test "${FORMULA_NAME}" || true
fi

# Commit and push
echo "📤 Pushing to GitHub..."
git add "Formula/${FORMULA_NAME}.rb"
git commit -m "Update ${FORMULA_NAME} to ${VERSION}"
git push

# Clean up
rm -rf "$TEMP_DIR"

echo "✅ Homebrew formula updated successfully!"
echo ""
echo "Users can now install with:"
echo "  brew tap ${TAP_REPO}"
echo "  brew install ${FORMULA_NAME}"
echo ""
echo "Or upgrade with:"
echo "  brew upgrade ${FORMULA_NAME}"