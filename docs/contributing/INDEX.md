# Contributing to nself

**Welcome!** Thank you for your interest in contributing to nself. This guide will help you get started with development, understand our standards, and make meaningful contributions.

## 📚 Table of Contents

- [Quick Start for Contributors](#quick-start-for-contributors)
- [Development Documentation](#development-documentation)
- [Compatibility Requirements](#compatibility-requirements)
- [Development Workflow](#development-workflow)
- [Testing Your Changes](#testing-your-changes)
- [Submitting Changes](#submitting-changes)
- [Getting Help](#getting-help)

---

## Quick Start for Contributors

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/nself.git
cd nself
```

### 2. Create Test Directory

**⚠️ IMPORTANT**: Never run nself commands in the nself repository itself!

```bash
# Create a separate test directory
mkdir ~/test-nself && cd ~/test-nself

# Initialize test project
/path/to/nself/bin/nself init --demo

# Build and start
/path/to/nself/bin/nself build
/path/to/nself/bin/nself start
```

### 3. Make Changes

```bash
# Create a feature branch
cd /path/to/nself
git checkout -b feature/my-contribution

# Make your changes
# Edit files in src/, docs/, etc.
```

### 4. Test Changes

```bash
# Test in your test directory
cd ~/test-nself
/path/to/nself/bin/nself build --force
/path/to/nself/bin/nself start

# Verify everything works
/path/to/nself/bin/nself status
/path/to/nself/bin/nself doctor
```

### 5. Submit Pull Request

```bash
# Commit your changes
cd /path/to/nself
git add .
git commit -m "feat: Add my contribution"
git push origin feature/my-contribution

# Open PR on GitHub
```

---

## Development Documentation

### Essential Reading

Before contributing, please read these documents:

#### 1. **[DEVELOPMENT.md](DEVELOPMENT.md)** - Development Guide
Complete development setup, architecture, coding standards, and patterns.

**Read this for:**
- Repository structure
- Coding standards
- Common patterns
- Adding new commands
- Shell script guidelines

#### 2. **[CROSS-PLATFORM-COMPATIBILITY.md](CROSS-PLATFORM-COMPATIBILITY.md)** - Compatibility Rules
Critical compatibility rules for Bash 3.2+, POSIX compliance, cross-platform support.

**Read this for:**
- Bash 3.2+ requirements
- POSIX compliance rules
- Platform-specific command differences
- Pre-commit checklist
- Common failure patterns

**⚠️ CRITICAL**: This is the most important document for contributors. Follow these rules to avoid CI failures.

#### 3. **[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)** - Community Guidelines
Standards for community interaction and contribution.

**Read this for:**
- Community standards
- Expected behavior
- Technical standards
- AI agent guidelines

---

## Compatibility Requirements

### Why Compatibility Matters

nself runs on macOS, Linux (all distros), and WSL. We support:
- ✅ Bash 3.2+ (macOS default)
- ✅ POSIX compliance
- ✅ BSD and GNU tools
- ✅ All major platforms

### Critical Rules

#### ❌ NEVER Use These

```bash
# Bash 4+ features - NOT SUPPORTED
${var,,}              # Lowercase expansion
${var^^}              # Uppercase expansion
declare -A            # Associative arrays
mapfile / readarray   # Array reading
&>>                   # Redirect both stdout/stderr

# Non-POSIX commands
echo -e "text"        # Use printf instead
```

#### ✅ ALWAYS Use These

```bash
# POSIX-compliant alternatives
printf "%s\n" "text"                    # Instead of echo -e
tr '[:upper:]' '[:lower:]'              # Instead of ${var,,}
safe_stat_perms() { ... }               # Platform-safe wrappers
```

### Pre-Commit Checklist

Before committing, **run these commands**:

```bash
# Check for echo -e usage
grep -r "echo -e" src/ && echo "❌ FAIL: Found echo -e" || echo "✅ PASS"

# Check for Bash 4+ lowercase
grep -r '\${[^}]*,,}' src/ && echo "❌ FAIL: Found \${var,,}" || echo "✅ PASS"

# Check for Bash 4+ uppercase
grep -r '\${[^}]*\^\^}' src/ && echo "❌ FAIL: Found \${var^^}" || echo "✅ PASS"

# Check for declare -A
grep -r "declare -A" src/ && echo "❌ FAIL: Found declare -A" || echo "✅ PASS"

# Check for &>>
grep -r "&>>" src/ && echo "❌ FAIL: Found &>>" || echo "✅ PASS"
```

**All checks must PASS before submitting a PR.**

---

## Development Workflow

### Repository Structure

```
nself/
├── bin/nself                   # Main executable (shim)
├── src/
│   ├── cli/                    # Command implementations
│   │   ├── init.sh
│   │   ├── build.sh
│   │   ├── start.sh
│   │   └── ...
│   ├── lib/                    # Shared libraries
│   │   ├── init/               # Init command logic
│   │   ├── build/              # Build system
│   │   ├── services/           # Service generation
│   │   └── utils/              # Utility functions
│   └── templates/              # Service templates
├── docs/                       # Documentation (auto-publishes to Wiki)
│   ├── Home.md                 # Wiki homepage
│   ├── _Sidebar.md             # Wiki navigation
│   ├── commands/               # Command documentation
│   ├── guides/                 # User guides
│   ├── services/               # Service documentation
│   ├── architecture/           # System architecture
│   ├── configuration/          # Configuration docs
│   └── contributing/           # This directory
├── .github/
│   └── workflows/              # CI/CD workflows
└── install.sh                  # Installation script
```

### Common Contribution Types

#### Adding a New Command

1. Create `/src/cli/mycommand.sh`
2. Implement `cmd_mycommand()` function
3. Add help text in help.sh
4. Update documentation
5. Add tests
6. Test on multiple platforms

See [DEVELOPMENT.md](DEVELOPMENT.md#adding-a-new-command) for detailed instructions.

#### Adding a Service Template

1. Create template directory in `/src/templates/services/`
2. Add template files with `{{PLACEHOLDERS}}`
3. Document in `/docs/services/`
4. Test template generation
5. Update service documentation

#### Fixing a Bug

1. Create issue if one doesn't exist
2. Create branch: `fix/issue-description`
3. Write test that reproduces bug
4. Fix the bug
5. Verify test passes
6. Submit PR referencing issue

#### Improving Documentation

1. Documentation in `/docs/` auto-publishes to GitHub Wiki
2. Follow markdown formatting
3. Keep version references accurate (v0.3.9 current)
4. Update _Sidebar.md if adding new pages
5. Verify internal links work

---

## Testing Your Changes

### Manual Testing

```bash
# Always test in a separate directory
cd ~/test-nself

# Test your changes
/path/to/nself/bin/nself init --demo
/path/to/nself/bin/nself build --force
/path/to/nself/bin/nself start --verbose

# Verify services
/path/to/nself/bin/nself status
/path/to/nself/bin/nself urls
/path/to/nself/bin/nself logs

# Test specific service
/path/to/nself/bin/nself logs postgres
/path/to/nself/bin/nself exec postgres psql -U postgres -c "SELECT version();"

# Clean up
/path/to/nself/bin/nself stop --volumes
```

### CI Tests

Our CI runs 12 test jobs:

1. **ShellCheck Linting** - Static analysis (error-level only)
2. **Unit Tests (Ubuntu Latest)** - Bash 5.x
3. **Unit Tests (Ubuntu - Bash 3.2)** - Compatibility test
4. **Unit Tests (macOS Latest)** - BSD tools, Bash 3.2
5. **Portability Check** - POSIX compliance verification
6. **Integration Tests** - Full workflow testing
7. **File Permissions Test** - Security verification
8. **Init Command Test** - Project initialization
9. **Build Command Test** - Infrastructure generation
10. **Service Generation Test** - Template system
11. **Documentation Check** - Link verification
12. **Security Scan** - Vulnerability detection

**All 12 tests must pass** before merging.

### Platform Testing

Test on multiple platforms if possible:

```bash
# macOS (Bash 3.2, BSD tools)
bash --version  # Should show 3.2.x
nself build && nself start

# Linux (Bash 5.x, GNU tools)
bash --version  # Should show 5.x
nself build && nself start

# WSL (Windows Subsystem for Linux)
# Test in WSL environment if available
```

---

## Submitting Changes

### Branch Naming

- Features: `feature/description`
- Fixes: `fix/description`
- Docs: `docs/description`
- Refactoring: `refactor/description`
- Tests: `test/description`

### Commit Messages

Follow conventional commits:

```bash
# Format
type: Brief description

Longer explanation if needed.

Fixes #issue_number

# Types
feat: New feature
fix: Bug fix
docs: Documentation changes
style: Code style changes (formatting)
refactor: Code refactoring
test: Adding tests
chore: Maintenance tasks
```

### Pull Request Checklist

Before submitting:

- [ ] All CI tests pass
- [ ] Pre-commit compatibility checks pass
- [ ] Code follows existing patterns
- [ ] Documentation updated (if needed)
- [ ] No hallucinated features or commands
- [ ] Tested on macOS or Linux (or both)
- [ ] Branch is up to date with main
- [ ] Commit messages are clear
- [ ] No secrets or sensitive data

### PR Description Template

```markdown
## Description
Brief description of changes.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tested locally
- [ ] CI tests pass
- [ ] Cross-platform tested (specify: macOS/Linux/WSL)
- [ ] Compatibility checks pass

## Checklist
- [ ] Code follows project standards
- [ ] Documentation updated
- [ ] No duplicate code
- [ ] Uses shared utilities
- [ ] POSIX compliant (Bash 3.2+)
```

---

## Getting Help

### Communication Channels

- **GitHub Issues**: [Bug reports and feature requests](https://github.com/acamarata/nself/issues)
- **GitHub Discussions**: [Questions and community discussion](https://github.com/acamarata/nself/discussions)
- **Documentation**: [GitHub Wiki](https://github.com/acamarata/nself/wiki)

### Questions?

- Search existing [Issues](https://github.com/acamarata/nself/issues) and [Discussions](https://github.com/acamarata/nself/discussions)
- Check [FAQ](../guides/FAQ.md) and [Troubleshooting](../guides/TROUBLESHOOTING.md)
- Read [Architecture docs](../architecture/) for system design questions
- Review [DEVELOPMENT.md](DEVELOPMENT.md) for development questions

### Reporting Issues

When reporting issues, include:

1. **Environment**: OS, Bash version, Docker version
2. **Steps to reproduce**: Exact commands run
3. **Expected behavior**: What should happen
4. **Actual behavior**: What actually happens
5. **Logs**: Relevant error messages
6. **Context**: .env configuration (redact secrets!)

```bash
# Gather environment info
uname -a
bash --version
docker --version
docker compose version

# Get nself info
nself version --verbose
nself doctor --verbose
```

---

## Additional Resources

### Documentation

- **[Home](../Home.md)** - Documentation homepage
- **[Commands Reference](../commands/COMMANDS.md)** - Complete CLI reference
- **[Architecture](../architecture/ARCHITECTURE.md)** - System design
- **[Services](../services/SERVICES.md)** - Service documentation
- **[Troubleshooting](../guides/TROUBLESHOOTING.md)** - Common issues

### Development Guides

- **[DEVELOPMENT.md](DEVELOPMENT.md)** - Complete development guide
- **[CROSS-PLATFORM-COMPATIBILITY.md](CROSS-PLATFORM-COMPATIBILITY.md)** - Compatibility rules
- **[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)** - Community guidelines

### Project Info

- **[Changelog](../CHANGELOG.md)** - Version history
- **[Roadmap](../ROADMAP.md)** - Planned features
- **[Releases](../releases/)** - Release notes

---

## Thank You!

Your contributions make nself better for everyone. Whether you're fixing a typo, adding a feature, or improving documentation - every contribution matters.

**Happy coding! 🚀**

---

**Version:** 1.0 | **Last Updated:** October 2025 | **Current Version:** v0.4.0
