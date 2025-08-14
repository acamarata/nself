# nself v0.3.1 Release Notes

## Enhanced Configuration Validation & Auto-Fix System

This release significantly improves nself's robustness by addressing 22+ configuration validation issues identified through comprehensive testing. Users will now experience automatic fixes for common configuration mistakes and professional-quality output formatting.

### 🎯 Key Improvements

#### **Enhanced Configuration Validation**
- **Empty file handling** - Automatically creates minimal configuration when `.env.local` is empty
- **Port validation** - Comprehensive checking for valid port ranges (1-65535), detects negative/non-numeric values
- **Password security** - Enhanced validation with special character escaping and strength checking
- **Whitespace handling** - Automatic trimming of leading/trailing spaces and tab replacement
- **Quote validation** - Detects and fixes mismatched single/double quotes and mixed quote styles
- **Duplicate detection** - Identifies and removes duplicate variable definitions
- **Inline comment cleanup** - Properly removes inline comments that could cause parsing issues

#### **Smart Auto-Fix System**
- **25+ auto-fix strategies** covering all common configuration mistakes
- **Safe backup creation** before applying any fixes
- **Comprehensive fix reporting** with detailed progress indicators
- **Port conflict resolution** with alternative port suggestions
- **Docker naming validation** ensuring compatibility with Docker requirements
- **Memory format validation** for proper resource specification

#### **Professional Output Formatting**
- **Perfect alignment** of all boxed messages and banners
- **Consistent spacing** with properly aligned right-side borders (`║`)
- **Beautiful Unicode boxes** using proper box-drawing characters
- **Color-coded indicators** for errors, warnings, and success messages
- **Professional welcome screens** with centered text alignment

#### **Environment File Priority System**
- **Strict priority enforcement**: `.env` > `.env.local` > `.env.dev`
- **No file merging** - only one environment file is loaded
- **Complete isolation** - higher priority files completely ignore lower priority ones
- **Smart defaults** applied after loading the chosen file

### 🔧 Technical Enhancements

#### **New Validation Capabilities**
- **IP address validation** - Validates IPv4 format and hostnames
- **Memory format validation** - Checks memory specifications (512M, 2G, etc.)
- **Timezone validation** - Validates timezone format strings
- **SSL configuration validation** - Ensures SSL cert/key paths exist
- **File path validation** - Verifies referenced files exist
- **Service list validation** - Comprehensive service name checking

#### **Updated Components**
- **Enhanced config-validator-v2.sh** - Complete rewrite with 30+ validation functions
- **Comprehensive auto-fixer-v2.sh** - Handles all identified edge cases
- **Updated templates** - All `.env` templates now have perfect formatting
- **Improved test coverage** - 30+ test scenarios covering all validation cases

#### **System Improvements**
- **GitHub release-only updates** - Update mechanism now pulls from releases, not development branch
- **Better error messages** - More descriptive validation errors with actionable solutions
- **Platform compatibility** - Enhanced cross-platform support (macOS, Linux, WSL)

### 📦 What's Changed

#### **Files Added**
- `src/lib/auto-fix/config-validator-v2.sh` - Enhanced validation engine
- `src/lib/auto-fix/auto-fixer-v2.sh` - Comprehensive auto-fix system
- `src/lib/utils/header.sh` - Professional header formatting
- `src/lib/utils/preflight.sh` - Pre-command validation checks

#### **Files Enhanced**
- `src/lib/utils/error-templates.sh` - Perfect text alignment
- `src/templates/.env.example` - Updated with proper spacing
- `src/templates/.env.local` - Professional formatting
- `src/cli/init.sh` - Uses enhanced validators
- `src/cli/build.sh` - Enhanced validation integration
- `src/cli/up.sh` - Improved error handling
- `src/cli/update.sh` - GitHub release-only updates

### 🐛 Fixes Resolved

1. **Empty `.env.local` files** causing build failures
2. **Port numbers >65535** or negative values crashing services
3. **Special characters in passwords** causing shell escaping issues
4. **Leading/trailing whitespace** in configuration values
5. **Quote mismatches** breaking configuration parsing
6. **Duplicate variables** causing undefined behavior
7. **Inline comments** interfering with value parsing
8. **Invalid IP addresses** in host configurations
9. **Malformed memory specifications** causing container issues
10. **Docker naming violations** preventing container creation
11. **Missing file references** breaking SSL and other features
12. **Mixed case booleans** causing validation failures
13. **Service list formatting** issues with commas and spaces
14. **Project names with spaces** breaking Docker networks
15. **JWT keys too short** for security requirements
16. **Weak passwords** compromising security
17. **Port conflicts** between services
18. **Boolean value inconsistencies** (yes/no vs true/false)
19. **Service name hyphen issues** (Docker incompatibility)
20. **Timezone format problems** causing scheduling issues
21. **SSL configuration errors** when paths don't exist
22. **Environment priority confusion** when multiple files exist

### 🎨 Visual Improvements

- **Perfect box alignment** - All headers and banners now have properly aligned borders
- **Consistent spacing** - 3-space indentation after `║` characters throughout
- **Professional appearance** - Suitable for public presentation and screenshots
- **Color consistency** - Unified color scheme across all output types
- **Unicode box characters** - Proper box-drawing characters for clean lines

### 🔄 Backward Compatibility

- All existing `.env.local` files continue to work
- Previous configuration formats are automatically upgraded
- No breaking changes to command interface
- Smooth migration path for existing projects

### 📋 Testing Coverage

This release includes comprehensive testing covering:
- 30+ validation scenarios
- All auto-fix strategies
- Environment priority loading
- Cross-platform compatibility
- Edge case handling
- Integration testing

### 🚀 Getting v0.3.1

**New installations:**
```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

**Existing installations:**
```bash
nself update
```

The update command now pulls only from GitHub releases, ensuring stability.

### 📖 Documentation

- Updated environment configuration priority documentation
- Enhanced troubleshooting guides
- New validation error reference
- Auto-fix strategy documentation

### 🙏 Acknowledgments

This release addresses comprehensive feedback from community testing, resulting in a significantly more robust and user-friendly experience.

---

**Full Changelog**: https://github.com/acamarata/nself/compare/v0.3.0...v0.3.1