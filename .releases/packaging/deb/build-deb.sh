#!/bin/bash
set -e

VERSION="0.3.8"
PACKAGE_NAME="nself"
BUILD_DIR="/tmp/${PACKAGE_NAME}-deb-build"
DEB_DIR="${BUILD_DIR}/${PACKAGE_NAME}_${VERSION}"

echo "Building Debian package for nself v${VERSION}"

# Clean and create build directory
rm -rf "$BUILD_DIR"
mkdir -p "$DEB_DIR"

# Copy DEBIAN control files
cp -r packaging/deb/DEBIAN "$DEB_DIR/"
chmod 755 "$DEB_DIR/DEBIAN/postinst"
chmod 755 "$DEB_DIR/DEBIAN/prerm"

# Create directory structure
mkdir -p "$DEB_DIR/opt/nself"
mkdir -p "$DEB_DIR/usr/share/doc/nself"

# Copy nself files
cp -r bin src install.sh LICENSE README.md VERSION "$DEB_DIR/opt/nself/"

# Copy documentation
cp README.md "$DEB_DIR/usr/share/doc/nself/"
cp LICENSE "$DEB_DIR/usr/share/doc/nself/"

# Build the package
cd "$BUILD_DIR"
dpkg-deb --build "${PACKAGE_NAME}_${VERSION}"

# Move to packaging directory
mv "${PACKAGE_NAME}_${VERSION}.deb" "$(pwd)/packaging/deb/"

echo "✅ Debian package built: packaging/deb/${PACKAGE_NAME}_${VERSION}.deb"