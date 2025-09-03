#!/usr/bin/env bash
#
# Create RPM package for nself v0.3.9
#

set -e

VERSION="0.3.9"
RELEASE="1"
ARCH="noarch"
PACKAGE_NAME="nself"

echo "Creating RPM package for nself v${VERSION}..."

# Create RPM build structure
mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Copy source tarball
cp /Users/admin/Sites/nself/releases/v${VERSION}/nself-v${VERSION}.tar.gz ~/rpmbuild/SOURCES/

# Create spec file
cat > ~/rpmbuild/SPECS/nself.spec << EOF
Name:           nself
Version:        ${VERSION}
Release:        ${RELEASE}%{?dist}
Summary:        Self-hosted backend platform

License:        MIT
URL:            https://nself.org
Source0:        nself-v%{version}.tar.gz

BuildArch:      noarch
Requires:       docker docker-compose git curl jq openssl ca-certificates wget

%description
nself provides a complete self-hosted backend stack with PostgreSQL,
Hasura GraphQL, authentication, storage, and more. All services run
locally using Docker Compose.

%prep
%setup -q -c

%install
mkdir -p %{buildroot}/usr/local/nself
cp -a * %{buildroot}/usr/local/nself/
mkdir -p %{buildroot}/usr/local/bin
ln -sf /usr/local/nself/bin/nself %{buildroot}/usr/local/bin/nself

# Bash completion
mkdir -p %{buildroot}/etc/bash_completion.d
cat > %{buildroot}/etc/bash_completion.d/nself << 'COMPLETION'
_nself() {
    local cur="\${COMP_WORDS[COMP_CWORD]}"
    local commands="init build start stop restart status logs doctor db admin reset help version update"
    COMPREPLY=(\$(compgen -W "\${commands}" -- \${cur}))
}
complete -F _nself nself
COMPLETION

%post
# Make nself executable
chmod +x /usr/local/nself/bin/nself
chmod -R 755 /usr/local/nself

# Install mkcert if not present
if ! command -v mkcert &> /dev/null; then
    echo "Installing mkcert..."
    wget -qO /usr/local/bin/mkcert https://github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-linux-amd64
    chmod +x /usr/local/bin/mkcert
fi

echo "nself v%{version} installed successfully!"
echo "Get started with: nself help"

%files
/usr/local/nself
/usr/local/bin/nself
/etc/bash_completion.d/nself

%changelog
* Mon Sep 02 2025 nself.org <nself@nself.org> - 0.3.9-1
- Production ready release
- Admin UI integration
- Bug fixes and stability improvements
- 35+ commands available
EOF

# Build the RPM
cd ~/rpmbuild
rpmbuild -ba SPECS/nself.spec

# Copy to release directory
cp ~/rpmbuild/RPMS/noarch/nself-${VERSION}-${RELEASE}*.rpm /Users/admin/Sites/nself/releases/v${VERSION}/nself-${VERSION}-${RELEASE}.noarch.rpm

echo "✓ RPM package created: nself-${VERSION}-${RELEASE}.noarch.rpm"

# Calculate SHA256
SHA256=$(sha256sum /Users/admin/Sites/nself/releases/v${VERSION}/nself-${VERSION}-${RELEASE}.noarch.rpm | cut -d' ' -f1)
echo "SHA256: $SHA256"