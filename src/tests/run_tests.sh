#!/usr/bin/env bash

# Test runner for nself test suite

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          nself Test Suite Runner             ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
echo ""

# Check if bats is installed
if ! command -v bats >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Warning: bats not installed${NC}"
    echo -e "${BLUE}Installing bats would enable full test suite${NC}"
    echo -e "${BLUE}Visit: https://github.com/bats-core/bats-core${NC}"
    echo ""
    echo -e "${BLUE}Running basic tests without bats...${NC}"
    echo ""
    
    # Run basic tests without bats
    TESTS_PASSED=0
    TESTS_FAILED=0
    
    # Test 1: Check if install.sh exists
    echo -n "Testing install.sh existence... "
    if [ -f "../install.sh" ]; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    # Test 2: Check if nself.sh exists
    echo -n "Testing nself.sh existence... "
    if [ -f "../bin/nself.sh" ]; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    # Test 3: Check VERSION file
    echo -n "Testing VERSION file... "
    if [ -f "../bin/VERSION" ]; then
        VERSION=$(cat ../bin/VERSION)
        echo -e "${GREEN}✓${NC} (v$VERSION)"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    # Test 4: Check shell script syntax
    echo -n "Testing install.sh syntax... "
    if bash -n ../install.sh 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    echo -n "Testing nself.sh syntax... "
    if bash -n ../bin/nself.sh 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    # Test 5: Check for required functions in install.sh
    echo -n "Testing install.sh functions... "
    if grep -q "check_existing_installation" ../install.sh && \
       grep -q "check_requirements" ../install.sh && \
       grep -q "show_spinner" ../install.sh; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    # Test 6: Check for required functions in nself.sh
    echo -n "Testing nself.sh functions... "
    if grep -q "cmd_update" ../bin/nself.sh && \
       grep -q "cmd_init" ../bin/nself.sh && \
       grep -q "show_spinner" ../bin/nself.sh; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC} | ${RED}Failed: $TESTS_FAILED${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}✅ All basic tests passed!${NC}"
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ bats is installed${NC}"
    echo ""
    
    # Run bats tests
    echo -e "${BLUE}Running test suites...${NC}"
    echo ""
    
    # Run each test file
    for test_file in *.bats; do
        if [ -f "$test_file" ]; then
            echo -e "${BLUE}Running $test_file...${NC}"
            bats "$test_file"
            echo ""
        fi
    done
    
    echo -e "${GREEN}✅ All tests completed!${NC}"
fi

echo ""