#!/usr/bin/env bash
# verify-v098-completion.sh - Verification script for v0.9.8 completion

set -euo pipefail

printf "\\n=== v0.9.8 Completion Verification ===\\n\\n"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

PASS=0
FAIL=0

check() {
  local test_name="$1"
  shift
  printf "Checking: %s... " "$test_name"
  if "$@" >/dev/null 2>&1; then
    printf "${GREEN}PASS${NC}\\n"
    PASS=$((PASS + 1))
  else
    printf "${RED}FAIL${NC}\\n"
    FAIL=$((FAIL + 1))
  fi
}

printf "P0 Verification:\\n"
check "reset --help exits 0" bash src/cli/reset.sh --help
check "checklist --help exits 0" bash src/cli/checklist.sh --help
check "help --help exits 0" bash src/cli/help.sh --help
check "whitelabel --help exits 0" bash src/cli/whitelabel.sh --help
check "Help contract file exists" test -f src/lib/help/HELP-CONTRACT.md
check "No echo -e in CLI scripts" bash -c '! grep -r "echo -e" src/cli/*.sh'
check "No Bash 4+ lowercase in CLI" bash -c '! grep -r "\${[^}]*,," src/cli/'
check "No Bash 4+ uppercase in CLI" bash -c '! grep -r "\${[^}]*\\^\\^" src/cli/'

printf "\\nP1 Verification:\\n"
check "Session state file exists" test -f .claude/SESSION_STATE.md
check "Task registry updated" grep -q "IN_PROGRESS\\|DONE" .claude/v098/V098_TASK_REGISTRY.md
check "Feedback triage updated" grep -q "In progress" .codex/FEEDBACK.md
check "Root check uses src/cli not docs" grep -q "src/cli" src/lib/hooks/pre-command.sh

printf "\\nPortability Verification:\\n"
check "No associative arrays" bash -c '! grep -r "declare -A" src/cli/ src/lib/'
check "No mapfile usage" bash -c '! grep -rE "\\bmapfile\\b" src/cli/ src/lib/'
check "No readarray usage" bash -c '! grep -rE "\\breadarray\\b" src/cli/ src/lib/'

printf "\\n=== Summary ===\\n"
printf "PASS: %d\\n" "$PASS"
printf "FAIL: %d\\n" "$FAIL"

if [[ $FAIL -eq 0 ]]; then
  printf "${GREEN}All checks passed!${NC}\\n"
  exit 0
else
  printf "${RED}Some checks failed${NC}\\n"
  exit 1
fi
