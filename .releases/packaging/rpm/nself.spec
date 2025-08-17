Name:           nself
Version:        0.3.8
Release:        1%{?dist}
Summary:        Self-hosted infrastructure manager for developers

License:        MIT
URL:            https://github.com/acamarata/nself
Source0:        https://github.com/acamarata/nself/archive/v%{version}.tar.gz

BuildArch:      noarch
Requires:       bash, docker, docker-compose, curl, git

%description
nself is a comprehensive CLI tool for deploying and managing
self-hosted backend infrastructure. It provides 33 commands
for managing Docker-based services, SSL certificates, monitoring,
and more.

%prep
%setup -q

%build
# Nothing to build

%install
rm -rf $RPM_BUILD_ROOT

# Install to /opt
mkdir -p $RPM_BUILD_ROOT/opt/nself
cp -r * $RPM_BUILD_ROOT/opt/nself/

# Create symlink
mkdir -p $RPM_BUILD_ROOT/usr/bin
ln -s /opt/nself/bin/nself $RPM_BUILD_ROOT/usr/bin/nself

# Install documentation
mkdir -p $RPM_BUILD_ROOT/usr/share/doc/nself
cp README.md $RPM_BUILD_ROOT/usr/share/doc/nself/
cp LICENSE $RPM_BUILD_ROOT/usr/share/doc/nself/

%files
%doc usr/share/doc/nself/README.md
%license usr/share/doc/nself/LICENSE
/opt/nself/
/usr/bin/nself

%post
chmod +x /opt/nself/bin/nself
echo "nself v0.3.8 installed successfully!"
echo "Run 'nself help' to get started."

%preun
# Nothing to do

%changelog
* Sat Aug 17 2024 acamarata <contact@acamarata.com> - 0.3.8-1
- Release v0.3.8: Enterprise features and critical fixes
- Added backup systems, monitoring, and SSL management
- 33 CLI commands for comprehensive infrastructure management