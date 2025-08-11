# nself Documentation

> **Comprehensive documentation for nself - Self-Hosted Infrastructure Manager**

This directory contains complete documentation for developers, contributors, and automated tools working with nself.

## 📚 Documentation Index

### Getting Started
- **[EXAMPLES.md](EXAMPLES.md)** - 🎯 Complete command examples with actual outputs
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - 🔧 Diagnose and fix common issues
- **[API.md](API.md)** - 📖 Complete command reference and usage
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - 🤝 Guidelines for contributing to the project

### Architecture & Design
- **[ARCHITECTURE.MD](ARCHITECTURE.MD)** - 🏗️ System design, principles, and patterns
- **[DIRECTORY_STRUCTURE.MD](DIRECTORY_STRUCTURE.MD)** - 📁 Complete file organization map
- **[CODE_STYLE.MD](CODE_STYLE.MD)** - ✨ Coding standards and best practices
- **[DECISIONS.md](DECISIONS.md)** - 🎯 Architectural decision records
- **[OUTPUT_FORMATTING.MD](OUTPUT_FORMATTING.MD)** - 🖥️ User interface and output standards

### Development & Testing
- **[TESTING_STRATEGY.MD](TESTING_STRATEGY.MD)** - 🧪 Testing approaches and guidelines
- **[REFACTORING_ROADMAP.MD](REFACTORING_ROADMAP.MD)** - 🚀 Future improvements and technical debt
- **[CHANGELOG.md](CHANGELOG.md)** - 📝 Version history and release notes

## 🎯 Quick Navigation

### By User Type

#### For New Users
Start here to get nself running:
1. [EXAMPLES.md](EXAMPLES.md) - See what nself can do
2. [API.md](API.md) - Learn the commands
3. [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Fix any issues

#### For Contributors
Want to contribute? Read these:
1. [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
2. [ARCHITECTURE.MD](ARCHITECTURE.MD) - Understand the system
3. [CODE_STYLE.MD](CODE_STYLE.MD) - Follow coding standards
4. [TESTING_STRATEGY.MD](TESTING_STRATEGY.MD) - Test your changes

#### For Automated Tools
Documentation designed for programmatic access:
1. [ARCHITECTURE.MD](ARCHITECTURE.MD) - System design patterns
2. [DIRECTORY_STRUCTURE.MD](DIRECTORY_STRUCTURE.MD) - File locations
3. [API.md](API.md) - Command specifications
4. [OUTPUT_FORMATTING.MD](OUTPUT_FORMATTING.MD) - Output parsing

## 📖 Documentation Highlights

### Command Examples
Every single nself command documented with real output:
```bash
$ nself init
╔════════════════════════════════════════════════════════════════╗
║                NSELF PROJECT INITIALIZATION                    ║
╚════════════════════════════════════════════════════════════════╝
[INFO] Initializing project: myproject
[SUCCESS] Created .env.local
```
See [EXAMPLES.md](EXAMPLES.md) for all commands.

### Troubleshooting Guide
Comprehensive solutions for common issues:
- Docker problems
- Network issues
- Service failures
- Configuration errors
- Performance problems

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for solutions.

### Architecture Overview
```
┌─────────────────────────────────────────────────────────────┐
│                         NGINX Proxy                         │
└─────────────┬───────────┬───────────┬───────────┬──────────┘
              │           │           │           │
     ┌────────▼────┐ ┌────▼────┐ ┌───▼────┐ ┌───▼────┐
     │   Hasura    │ │  Auth   │ │ MinIO  │ │Custom  │
     │  GraphQL    │ │Service  │ │Storage │ │Services│
     └────────┬────┘ └────┬────┘ └───┬────┘ └───┬────┘
              │           │           │           │
     ┌────────▼───────────▼───────────▼───────────▼────┐
     │              PostgreSQL Database                 │
     └──────────────────────────────────────────────────┘
```
See [ARCHITECTURE.MD](ARCHITECTURE.MD) for details.

## 🔍 Key Principles for Development

### 1. Never Run in nself Repository
```bash
# WRONG - Don't do this
cd /path/to/nself && nself init

# RIGHT - Use a project directory
cd ~/myproject && nself init
```

### 2. Always Use the Compose Wrapper
```bash
# WRONG
docker compose up

# RIGHT
compose up
```

### 3. Follow the Hooks Pattern
```bash
pre_command "commandname" || exit $?
# command logic
post_command "commandname" $?
```

### 4. Use Standardized Logging
```bash
log_info "Message"    # Not echo
log_error "Error"     # Goes to stderr
log_success "Done"    # Green success
log_warning "Warning" # Yellow warning
```

## 📊 Documentation Standards

### File Naming
- Use UPPERCASE with underscores: `ARCHITECTURE.MD`
- Exception: `README.md`, `CHANGELOG.md` (conventions)

### Content Structure
1. **Title and description**
2. **Table of contents** (for long docs)
3. **Main content** with examples
4. **References** to related docs

### Code Examples
- Show actual command output
- Include error cases
- Provide solutions

### Cross-References
Use relative links between documents:
```markdown
See [TROUBLESHOOTING.md](TROUBLESHOOTING.md#docker-issues)
```

## 🚀 Quick Reference

### Essential Commands
```bash
nself init          # Initialize project
nself build         # Build infrastructure
nself up            # Start services
nself status        # Check status
nself logs          # View logs
nself down          # Stop services
nself doctor        # Run diagnostics
```

### Common Workflows
```bash
# Development
nself init && nself build && nself up
nself hot-reload
nself logs -f

# Production
nself prod --domain api.example.com
nself validate-env
ENV=production nself up -d

# Maintenance
nself doctor
nself db backup
nself update
```

## 🔄 Keeping Documentation Updated

### For Contributors
1. **Update docs with code changes** - Documentation is part of the PR
2. **Test examples** - Ensure command outputs are accurate
3. **Add to CHANGELOG.md** - Document your changes
4. **Update TROUBLESHOOTING.md** - Add solutions for new issues

### For Maintainers
1. **Review documentation** in PRs
2. **Keep examples current** with new releases
3. **Update version numbers** in documentation
4. **Sync wiki** after merges

## 🌐 GitHub Wiki

This documentation is automatically synchronized to the GitHub Wiki:
- **Wiki URL**: https://github.com/acamarata/nself/wiki
- **Sync Script**: `.github/scripts/init-wiki.sh`
- **GitHub Action**: `.github/workflows/sync-wiki.yml`

To manually sync to wiki:
```bash
.github/scripts/init-wiki.sh
```

## 📝 Documentation TODO

- [ ] Add video tutorials
- [ ] Create interactive examples
- [ ] Add performance benchmarks
- [ ] Document cloud deployments
- [ ] Add security audit guide

## 💡 Tips for Reading Documentation

1. **Start with EXAMPLES.md** - See nself in action
2. **Use TROUBLESHOOTING.md** - When things go wrong
3. **Reference API.md** - For command details
4. **Study ARCHITECTURE.MD** - To understand the system
5. **Follow CONTRIBUTING.md** - To contribute back

## 🆘 Getting Help

- **GitHub Issues**: [Report bugs](https://github.com/acamarata/nself/issues)
- **GitHub Discussions**: [Ask questions](https://github.com/acamarata/nself/discussions)
- **Documentation**: You're here!
- **Wiki**: [GitHub Wiki](https://github.com/acamarata/nself/wiki)

## 📄 License

Documentation is part of the nself project and covered under the same [MIT License](../LICENSE).

## 🙏 Support Development

If you find nself and its documentation useful:
- ⭐ [Star on GitHub](https://github.com/acamarata/nself)
- 💰 [Support on Patreon](https://patreon.com/acamarata)
- 🐛 [Report Issues](https://github.com/acamarata/nself/issues)
- 📝 [Contribute Documentation](CONTRIBUTING.md)

---

*This documentation is the source of truth for nself. The GitHub Wiki is automatically generated from these files.*