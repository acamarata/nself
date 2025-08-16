#!/usr/bin/env bash

set -e

VERSION="0.3.7"
ARCH=$(uname -m)
PACKAGE_NAME="nself"
DESCRIPTION="Deploy a feature-complete backend infrastructure in seconds"
MAINTAINER="Aric Camarata <support@nself.org>"
HOMEPAGE="https://nself.org"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Building nself packages v${VERSION}${NC}"

# Create build directory
BUILD_DIR=$(mktemp -d)
echo "Build directory: $BUILD_DIR"

# Prepare file structure
echo -e "${BLUE}Preparing file structure...${NC}"
mkdir -p "$BUILD_DIR/usr/local/bin"
mkdir -p "$BUILD_DIR/usr/local/lib/nself"
mkdir -p "$BUILD_DIR/usr/share/doc/nself"

# Copy files
cp -r src "$BUILD_DIR/usr/local/lib/nself/"
cp -r templates "$BUILD_DIR/usr/local/lib/nself/"
cp README.md LICENSE "$BUILD_DIR/usr/share/doc/nself/"
cp -r docs "$BUILD_DIR/usr/share/doc/nself/"

# Create wrapper script
cat > "$BUILD_DIR/usr/local/bin/nself" << 'EOF'
#!/usr/bin/env bash
exec /usr/local/lib/nself/src/cli/nself.sh "$@"
EOF
chmod +x "$BUILD_DIR/usr/local/bin/nself"

# Build .deb package
if command -v dpkg-deb >/dev/null 2>&1; then
    echo -e "${BLUE}Building .deb package...${NC}"
    
    DEB_DIR="$BUILD_DIR/DEBIAN"
    mkdir -p "$DEB_DIR"
    
    # Create control file
    cat > "$DEB_DIR/control" << EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Architecture: all
Maintainer: ${MAINTAINER}
Depends: docker.io | docker-ce, docker-compose
Description: ${DESCRIPTION}
Homepage: ${HOMEPAGE}
EOF
    
    # Create postinst script
    cat > "$DEB_DIR/postinst" << 'EOF'
#!/bin/bash
set -e

# Create .nself directory structure
mkdir -p /opt/nself
cp -r /usr/local/lib/nself/* /opt/nself/

echo "nself has been installed successfully!"
echo "Run 'nself init' to get started"
EOF
    chmod 755 "$DEB_DIR/postinst"
    
    # Build the package
    dpkg-deb --build "$BUILD_DIR" "nself_${VERSION}_all.deb"
    mkdir -p packaging/debian
    mv "nself_${VERSION}_all.deb" packaging/debian/
    echo -e "${GREEN}✓ Created packaging/debian/nself_${VERSION}_all.deb${NC}"
else
    echo -e "${RED}dpkg-deb not found, skipping .deb package${NC}"
fi

# Build .rpm package
if command -v rpmbuild >/dev/null 2>&1; then
    echo -e "${BLUE}Building .rpm package...${NC}"
    
    # Create RPM build structure
    RPM_BUILD_ROOT="$HOME/rpmbuild"
    mkdir -p "$RPM_BUILD_ROOT"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
    
    # Create tarball for RPM
    tar czf "$RPM_BUILD_ROOT/SOURCES/nself-${VERSION}.tar.gz" -C "$BUILD_DIR" .
    
    # Create spec file
    cat > "$RPM_BUILD_ROOT/SPECS/nself.spec" << EOF
Name:           nself
Version:        ${VERSION}
Release:        1%{?dist}
Summary:        ${DESCRIPTION}
License:        Source-Available
URL:            ${HOMEPAGE}
Source0:        nself-${VERSION}.tar.gz
BuildArch:      noarch
Requires:       docker, docker-compose

%description
${DESCRIPTION}

%prep
%setup -q -n .

%install
rm -rf \$RPM_BUILD_ROOT
mkdir -p \$RPM_BUILD_ROOT
cp -r * \$RPM_BUILD_ROOT/

%files
/usr/local/bin/nself
/usr/local/lib/nself/
/usr/share/doc/nself/

%post
mkdir -p /opt/nself
cp -r /usr/local/lib/nself/* /opt/nself/
echo "nself has been installed successfully!"
echo "Run 'nself init' to get started"

%changelog
* $(date "+%a %b %d %Y") ${MAINTAINER} - ${VERSION}-1
- Release ${VERSION}
EOF
    
    # Build RPM
    rpmbuild -bb "$RPM_BUILD_ROOT/SPECS/nself.spec"
    mkdir -p packaging/rpm
    cp "$RPM_BUILD_ROOT/RPMS/noarch/nself-${VERSION}-1"*.rpm packaging/rpm/
    echo -e "${GREEN}✓ Created packaging/rpm/nself-${VERSION}-1.noarch.rpm${NC}"
else
    echo -e "${RED}rpmbuild not found, skipping .rpm package${NC}"
fi

# Clean up
rm -rf "$BUILD_DIR"
echo -e "${GREEN}Package building complete!${NC}"