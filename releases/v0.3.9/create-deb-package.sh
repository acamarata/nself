#!/usr/bin/env bash
#
# Create Debian package for nself v0.3.9
#

set -e

VERSION="0.3.9"
ARCH="all"
PACKAGE_NAME="nself"
BUILD_DIR="/tmp/nself-deb-build"

echo "Creating Debian package for nself v${VERSION}..."

# Clean and create build directory
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR/DEBIAN"
mkdir -p "$BUILD_DIR/usr/local/nself"
mkdir -p "$BUILD_DIR/usr/local/bin"
mkdir -p "$BUILD_DIR/etc/bash_completion.d"

# Extract nself to package directory
cd "$BUILD_DIR/usr/local/nself"
tar -xzf /Users/admin/Sites/nself/releases/v${VERSION}/nself-v${VERSION}.tar.gz

# Create symlink
ln -sf /usr/local/nself/bin/nself "$BUILD_DIR/usr/local/bin/nself"

# Create control file
cat > "$BUILD_DIR/DEBIAN/control" << EOF
Package: nself
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Depends: docker.io | docker-ce, docker-compose | docker-compose-plugin, git, curl, jq, openssl, ca-certificates
Maintainer: nself.org <nself@nself.org>
Description: Self-hosted backend platform
 nself provides a complete self-hosted backend stack with PostgreSQL,
 Hasura GraphQL, authentication, storage, and more. All services run
 locally using Docker Compose.
Homepage: https://nself.org
EOF

# Create postinst script
cat > "$BUILD_DIR/DEBIAN/postinst" << 'EOF'
#!/bin/bash
set -e

# Make nself executable
chmod +x /usr/local/nself/bin/nself
chmod -R 755 /usr/local/nself

# Install mkcert if not present
if ! command -v mkcert &> /dev/null; then
    echo "Installing mkcert..."
    wget -qO /usr/local/bin/mkcert https://github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-linux-amd64
    chmod +x /usr/local/bin/mkcert
fi

echo "nself v0.3.9 installed successfully!"
echo "Get started with: nself help"
EOF
chmod 755 "$BUILD_DIR/DEBIAN/postinst"

# Create bash completion
cat > "$BUILD_DIR/etc/bash_completion.d/nself" << 'EOF'
_nself() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local commands="init build start stop restart status logs doctor db admin reset help version update"
    COMPREPLY=($(compgen -W "${commands}" -- ${cur}))
}
complete -F _nself nself
EOF

# Build the package
dpkg-deb --build "$BUILD_DIR" "nself_${VERSION}_${ARCH}.deb"

echo "✓ Debian package created: nself_${VERSION}_${ARCH}.deb"

# Calculate SHA256
SHA256=$(sha256sum "nself_${VERSION}_${ARCH}.deb" | cut -d' ' -f1)
echo "SHA256: $SHA256"

# Move to release directory
mv "nself_${VERSION}_${ARCH}.deb" /Users/admin/Sites/nself/releases/v${VERSION}/

# Clean up
rm -rf "$BUILD_DIR"

echo "✓ Package moved to releases directory"