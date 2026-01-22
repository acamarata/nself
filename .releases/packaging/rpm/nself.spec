Name:           nself
Version:        0.4.4
Release:        1%{?dist}
Summary:        Self-hosted infrastructure manager for developers

License:        MIT
URL:            https://github.com/acamarata/nself
Source0:        https://github.com/acamarata/nself/archive/v%{version}.tar.gz

BuildArch:      noarch
Requires:       bash, docker, docker-compose, curl, git

%description
nself is a comprehensive CLI tool for deploying and managing
self-hosted backend infrastructure. It provides 36 commands
for managing Docker-based services, SSL certificates, monitoring,
and more. Works on macOS, Linux, and WSL.

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
echo "nself v0.4.4 installed successfully!"
echo "Run 'nself help' to get started."

%preun
# Nothing to do

%changelog
* Wed Jan 22 2026 acamarata <contact@acamarata.com> - 0.4.4-1
- Release v0.4.4: Database Tools
- New db command with comprehensive database management
- DBML schema workflow (scaffold, import, apply)
- Environment-aware seeding and mock data generation
- Type generation for TypeScript, Go, Python

* Wed Jan 22 2026 acamarata <contact@acamarata.com> - 0.4.3-1
- Release v0.4.3: Deployment Pipeline
- New env command for environment management
- Enhanced deploy command with zero-downtime support
- New prod and staging shortcut commands
- Fixed nginx variable substitution and 16 Dockerfile templates

* Wed Jan 22 2026 acamarata <contact@acamarata.com> - 0.4.2-1
- Release v0.4.2: Service & Monitoring Management
- 6 new commands: email, search, functions, mlflow, metrics, monitor
- 92 unit tests, complete documentation

* Tue Jan 21 2026 acamarata <contact@acamarata.com> - 0.4.1-1
- Release v0.4.1: Platform compatibility fixes
- Fixed Bash 3.2 compatibility for macOS
- Fixed cross-platform sed, stat, and timeout commands
- 36 CLI commands for comprehensive infrastructure management

* Sun Oct 13 2025 acamarata <contact@acamarata.com> - 0.4.0-1
- Release v0.4.0: Production-ready release
- All core features complete and tested
- Enhanced cross-platform compatibility

* Sat Aug 17 2024 acamarata <contact@acamarata.com> - 0.3.8-1
- Release v0.3.8: Enterprise features and critical fixes
- Added backup systems, monitoring, and SSL management
- 33 CLI commands for comprehensive infrastructure management
