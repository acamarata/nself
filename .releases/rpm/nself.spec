Name:           nself
Version:        0.4.0
Release:        1%{?dist}
Summary:        Production-ready self-hosted backend infrastructure

License:        Source-Available
URL:            https://nself.org
Source0:        https://github.com/acamarata/nself/archive/refs/tags/v%{version}.tar.gz

BuildArch:      noarch
Requires:       bash >= 3.2
Requires:       docker
Requires:       docker-compose

%description
nself is a production-ready infrastructure manager that helps developers
deploy backend services including PostgreSQL, Hasura GraphQL, authentication,
storage, email services, monitoring, and more with a single command. Includes
40+ service templates, automated SSL, and comprehensive monitoring stack.

%prep
%autosetup -n %{name}-%{version}

%build
# Nothing to build - shell scripts

%install
# Create directories
mkdir -p %{buildroot}%{_datadir}/%{name}
mkdir -p %{buildroot}%{_bindir}

# Copy source files
cp -r src %{buildroot}%{_datadir}/%{name}/
cp -r bin %{buildroot}%{_datadir}/%{name}/

# Create wrapper script
cat > %{buildroot}%{_bindir}/%{name} << 'EOF'
#!/bin/bash
exec %{_datadir}/%{name}/bin/nself "$@"
EOF
chmod 755 %{buildroot}%{_bindir}/%{name}

# Set permissions
find %{buildroot}%{_datadir}/%{name} -type f -name "*.sh" -exec chmod 755 {} \;

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_datadir}/%{name}

%changelog
* Sun Oct 13 2025 Aric Camarata <aric.camarata@gmail.com> - 0.4.0-1
- Production-ready release v0.4.0
- Fixed critical bugs (unbound variables, Bash 4+ compatibility)
- Enhanced cross-platform support (Bash 3.2+)
- All core features complete and tested
- 12/12 CI tests passing across all platforms
- Improved stability and error handling