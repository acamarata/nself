Name:           nself
Version:        0.3.7
Release:        1%{?dist}
Summary:        Deploy feature-complete backend infrastructure in seconds

License:        Source-Available
URL:            https://nself.org
Source0:        https://github.com/acamarata/nself/archive/refs/tags/v%{version}.tar.gz

BuildArch:      noarch
Requires:       bash >= 4.0
Requires:       docker
Requires:       docker-compose

%description
nself is a self-hosted infrastructure manager that helps developers
deploy production-ready backend services including PostgreSQL, Hasura,
authentication, email services, and more with a single command.

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
* Fri Aug 16 2024 Aric Camarata <aric.camarata@gmail.com> - 0.3.7-1
- Release v0.3.7
- Improved update command with loading spinner
- Enhanced version command with standard flags
- Fixed CI integration tests
- Updated installation process