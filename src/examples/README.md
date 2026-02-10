# nself Examples

This directory contains practical examples and demonstrations of nself features.

## Available Examples

### CLI Output Demo

**File:** `cli-output-demo.sh`

Comprehensive demonstration of the CLI output library, showing all available functions and their visual output.

```bash
# Run the full demo
bash src/examples/cli-output-demo.sh

# Run without colors
NO_COLOR=1 bash src/examples/cli-output-demo.sh
```

**What it demonstrates:**
- Basic message types (success, error, warning, info)
- Sections and headers
- Boxes (simple and detailed)
- Lists (bullet, numbered, checklists)
- Tables with auto-sizing columns
- Progress bars and spinners
- Summaries and banners
- Utility functions (centering, indentation, separators)
- Practical usage examples
- Error handling patterns

### Testing Examples

See `src/tests/unit/` for unit tests that also serve as usage examples:

- `test-cli-output-quick.sh` - Quick validation test
- `test-cli-output.sh` - Comprehensive test suite

## Running Examples

All example scripts are self-contained and can be run directly:

```bash
# Make executable (first time only)
chmod +x src/examples/*.sh

# Run any example
./src/examples/cli-output-demo.sh
```

## Creating Your Own

To use the CLI output library in your own scripts:

```bash
#!/usr/bin/env bash

# Source the library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../src/lib/utils/cli-output.sh"

# Use the functions
cli_header "My Script"
cli_info "Starting operation..."
cli_success "Operation complete!"
```

## Documentation

Full API documentation: [/.wiki/contributing/CLI-OUTPUT-LIBRARY.md](/.wiki/contributing/CLI-OUTPUT-LIBRARY.md)

## Contributing

When adding new examples:

1. Place them in this directory
2. Make them executable (`chmod +x`)
3. Add descriptive comments
4. Update this README
5. Test on macOS and Linux
