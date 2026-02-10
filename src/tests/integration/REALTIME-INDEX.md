# Real-Time Integration Tests - File Index

## Overview

Complete integration test suite for nself v0.8.0 Sprint 16: Real-Time Collaboration

**Created**: 2026-01-29
**Test Count**: 45 tests across 7 suites
**Code Size**: 1,185 lines of Bash
**Coverage**: 100% of database layer

## File Structure

```
src/tests/integration/
├── test-realtime.sh                 # Main test suite (45 tests)
├── websocket-test-helpers.sh        # WebSocket utilities (future)
├── run-all-realtime-tests.sh        # Test runner
├── REALTIME-TEST-README.md          # Comprehensive documentation
├── REALTIME-TESTS-SUMMARY.md        # Delivery summary
├── REALTIME-QUICKSTART.md           # Quick start guide
└── REALTIME-INDEX.md                # This file
```

## File Details

### 1. test-realtime.sh (592 lines, 24 KB)

**Purpose**: Main integration test suite for real-time collaboration system

**Test Suites** (8 functions, 45 tests):
- `test_connection_lifecycle()` - 7 tests - WebSocket connection management
- `test_channel_management()` - 7 tests - Channel create/join/leave
- `test_messaging()` - 6 tests - Message send/receive/persist
- `test_presence()` - 7 tests - User presence and status
- `test_broadcasts()` - 6 tests - Ephemeral events
- `test_security()` - 7 tests - Authorization and permissions
- `test_views_and_monitoring()` - 5 tests - Database views
- `main()` - Test orchestration and reporting

**Key Features**:
- ✓ POSIX-compliant (Bash 3.2+)
- ✓ Automatic database detection (Docker or direct)
- ✓ Automatic cleanup via trap
- ✓ Comprehensive test utilities
- ✓ Colored output with `printf`
- ✓ No external dependencies (except PostgreSQL)

**Usage**:
```bash
./src/tests/integration/test-realtime.sh
```

### 2. websocket-test-helpers.sh (392 lines, 10 KB)

**Purpose**: Helper functions for WebSocket testing (future use)

**Functions**:
- `ws_check_wscat()` - Check if wscat is available
- `ws_connect()` - Establish WebSocket connection
- `ws_send()` - Send message over WebSocket
- `ws_read()` - Read messages from WebSocket
- `ws_disconnect()` - Close WebSocket connection
- `ws_test_upgrade()` - Test WebSocket upgrade with curl
- `pg_listen()` - Listen for PostgreSQL notifications
- `pg_notify()` - Send PostgreSQL notification
- `ws_integration_test()` - Full flow test
- `ws_load_test()` - Load testing helper

**Key Features**:
- ✓ Multiple WebSocket testing strategies (wscat, curl)
- ✓ PostgreSQL NOTIFY/LISTEN integration
- ✓ Load testing capabilities
- ✓ Example usage included
- ✓ Ready for future WebSocket server implementation

**Usage**:
```bash
source src/tests/integration/websocket-test-helpers.sh
ws_connect "ws://localhost:8080/realtime"
```

### 3. run-all-realtime-tests.sh (201 lines, 5.4 KB)

**Purpose**: Test runner with prerequisite checking and reporting

**Features**:
- ✓ Prerequisite validation (PostgreSQL, migration)
- ✓ Test suite orchestration
- ✓ Verbose and quick modes
- ✓ Colored output and progress reporting
- ✓ Overall summary statistics

**Options**:
- `--verbose, -v` - Show detailed output
- `--quick, -q` - Skip WebSocket tests
- `--help, -h` - Show help

**Usage**:
```bash
./src/tests/integration/run-all-realtime-tests.sh
./src/tests/integration/run-all-realtime-tests.sh --verbose
./src/tests/integration/run-all-realtime-tests.sh --quick
```

### 4. REALTIME-TEST-README.md (8.9 KB)

**Purpose**: Comprehensive documentation and troubleshooting guide

**Contents**:
- Overview of all 45 tests
- Prerequisites and setup
- Running tests (multiple methods)
- Test architecture explanation
- Database functions tested
- Security policies verified
- WebSocket testing (future)
- Troubleshooting guide
- CI/CD integration examples
- Contributing guidelines

**Use Case**: Primary reference for understanding and maintaining tests

### 5. REALTIME-TESTS-SUMMARY.md (9.9 KB)

**Purpose**: Delivery summary and sprint completion documentation

**Contents**:
- Files created
- Test coverage breakdown (45 tests)
- Database functions tested (8 functions)
- Views tested (2 views)
- Security features verified
- Test architecture details
- Running instructions
- Future enhancements
- Performance characteristics
- CI/CD integration
- Sprint completion checklist

**Use Case**: Sprint delivery verification and team communication

### 6. REALTIME-QUICKSTART.md (Previous file)

**Purpose**: Quick reference for running tests

**Contents**:
- 5-second quick run
- What gets tested
- File manifest
- Sample output
- Prerequisites
- Troubleshooting
- Test architecture summary
- Future WebSocket testing
- CI/CD integration
- Coverage summary

**Use Case**: Developer quick reference

### 7. REALTIME-INDEX.md (This file)

**Purpose**: File organization and navigation

**Contents**:
- File structure overview
- Detailed file descriptions
- Usage examples
- Quick reference table

**Use Case**: Understanding the complete test suite organization

## Quick Reference Table

| File | Type | Lines | Size | Purpose | Executable |
|------|------|-------|------|---------|------------|
| `test-realtime.sh` | Test Suite | 592 | 24 KB | Main integration tests | ✓ Yes |
| `websocket-test-helpers.sh` | Utilities | 392 | 10 KB | WebSocket test helpers | ✓ Yes |
| `run-all-realtime-tests.sh` | Runner | 201 | 5.4 KB | Test orchestration | ✓ Yes |
| `REALTIME-TEST-README.md` | Docs | - | 8.9 KB | Comprehensive guide | No |
| `REALTIME-TESTS-SUMMARY.md` | Docs | - | 9.9 KB | Delivery summary | No |
| `REALTIME-QUICKSTART.md` | Docs | - | ~3 KB | Quick start guide | No |
| `REALTIME-INDEX.md` | Docs | - | ~2 KB | This file | No |

**Total**: 7 files, 1,185 lines of code, ~58 KB

## Test Execution Flow

```
run-all-realtime-tests.sh
│
├─► Check prerequisites
│   ├─► PostgreSQL running?
│   ├─► Migration applied?
│   └─► Test files executable?
│
├─► Run test-realtime.sh
│   ├─► Connection tests (7)
│   ├─► Channel tests (7)
│   ├─► Messaging tests (6)
│   ├─► Presence tests (7)
│   ├─► Broadcast tests (6)
│   ├─► Security tests (7)
│   └─► Monitoring tests (5)
│
├─► WebSocket checks (optional)
│   └─► Check wscat availability
│
└─► Generate summary report
    ├─► Total tests: 45
    ├─► Pass/fail counts
    └─► Exit code
```

## Database Coverage

### Tables Tested (7/7 = 100%)
- [x] `realtime.connections`
- [x] `realtime.channels`
- [x] `realtime.channel_members`
- [x] `realtime.messages`
- [x] `realtime.presence`
- [x] `realtime.broadcasts`
- [x] `realtime.subscriptions`

### Functions Tested (8/8 = 100%)
- [x] `realtime.connect()`
- [x] `realtime.disconnect()`
- [x] `realtime.update_presence()`
- [x] `realtime.send_message()`
- [x] `realtime.get_online_users()`
- [x] `realtime.broadcast()`
- [x] `realtime.cleanup_expired_broadcasts()`
- [x] `realtime.cleanup_stale_connections()`

### Views Tested (2/2 = 100%)
- [x] `realtime.active_connections`
- [x] `realtime.channel_activity`

### Triggers Tested (1/1 = 100%)
- [x] `update_last_seen_on_message`

## Getting Started

### 1. Quick Test
```bash
cd /Users/admin/Sites/nself
./src/tests/integration/run-all-realtime-tests.sh
```

### 2. Read Documentation
Start with `REALTIME-QUICKSTART.md`, then `REALTIME-TEST-README.md` for details.

### 3. Understand Tests
Review `test-realtime.sh` to see test implementation.

### 4. Extend Tests
Use `websocket-test-helpers.sh` for WebSocket testing when ready.

## Documentation Hierarchy

```
REALTIME-QUICKSTART.md          ← Start here (5-second quick run)
    ↓
REALTIME-INDEX.md               ← File organization (this file)
    ↓
REALTIME-TEST-README.md         ← Comprehensive guide
    ↓
REALTIME-TESTS-SUMMARY.md       ← Delivery details
    ↓
test-realtime.sh                ← Source code
websocket-test-helpers.sh       ← Future utilities
```

## Compliance and Standards

### Cross-Platform Compatibility
- ✓ Bash 3.2+ (macOS default)
- ✓ POSIX-compliant patterns
- ✓ No `echo -e` (uses `printf`)
- ✓ No Bash 4+ features
- ✓ Platform-agnostic database connection

### Code Quality
- ✓ shellcheck clean (error-level)
- ✓ Consistent coding style
- ✓ Comprehensive comments
- ✓ Error handling
- ✓ Automatic cleanup

### Testing Best Practices
- ✓ Test isolation (unique IDs)
- ✓ Automatic cleanup (trap)
- ✓ Comprehensive assertions
- ✓ Clear test names
- ✓ Progress reporting

## Related Files

### Migration
- `/Users/admin/Sites/nself/postgres/migrations/012_create_realtime_system.sql`

### Platform Utilities
- `/Users/admin/Sites/nself/src/lib/utils/platform-compat.sh`

### Other Integration Tests
- `/Users/admin/Sites/nself/src/tests/integration/test-roles.sh`
- `/Users/admin/Sites/nself/src/tests/integration/test-redis.sh`
- `/Users/admin/Sites/nself/src/tests/integration/test-backup.sh`

## Sprint Completion

**Sprint**: 16 - Real-Time Collaboration
**Points**: 70
**Deliverables**:
- ✓ Migration 012 (real-time schema)
- ✓ 45 integration tests
- ✓ 3 test utilities
- ✓ 4 documentation files
- ✓ 100% coverage of database layer

**Status**: Complete

---

**Last Updated**: 2026-01-29
**Version**: nself v0.8.0
**Files**: 7
**Lines**: 1,185
**Tests**: 45
