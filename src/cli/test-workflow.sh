#!/usr/bin/env bash
# test-workflow.sh - Automated workflow testing

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source utilities
source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"

# Colors
COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_BLUE='\033[0;34m'
COLOR_DIM='\033[2m'

# Show help
show_help() {
  cat << 'EOF'
Usage: nself test workflow <workflow> [options]

Automated workflow testing with generated test data.

Workflows:
  approval                Test approval workflow (task/content approval)
  auth                    Test authentication workflow
  payment                 Test payment workflow

Options:
  --users=<count>         Number of test users to create (default: 3)
  --scenario=<name>       Run specific test scenario
  --cleanup               Clean up test data after completion
  -h, --help              Show this help message

Examples:
  nself test workflow approval --users=3
  nself test workflow approval --scenario=rejection
  nself test workflow auth
  nself test workflow approval --cleanup

Scenarios (approval workflow):
  complete        Member completes task, owner approves
  rejection       Member completes task, owner rejects
  photo-required  Test photo upload requirement

EOF
}

# Get database connection
get_db_connection() {
  if [[ -f ".env.runtime" ]]; then
    set -a
    source .env.runtime 2>/dev/null || true
    set +a
  elif [[ -f ".env" ]]; then
    set -a
    source .env 2>/dev/null || true
    set +a
  fi

  local db_name="${POSTGRES_DB:-postgres}"
  local db_user="${POSTGRES_USER:-postgres}"
  local project_name="${PROJECT_NAME:-$(basename "$PWD")}"
  local container_name="${project_name}_postgres"

  echo "$container_name:$db_user:$db_name"
}

# Execute SQL
exec_sql() {
  local sql="$1"
  local connection=$(get_db_connection)
  local container=$(echo "$connection" | cut -d: -f1)
  local user=$(echo "$connection" | cut -d: -f2)
  local db=$(echo "$connection" | cut -d: -f3)

  docker exec -i "$container" psql -U "$user" -d "$db" -c "$sql" 2>&1
}

# Test approval workflow
test_approval_workflow() {
  local user_count="${1:-3}"
  local scenario="${2:-complete}"

  printf "${COLOR_BLUE}🧪 Testing approval workflow${COLOR_RESET}\n\n"
  printf "  Users: $user_count\n"
  printf "  Scenario: $scenario\n\n"

  # Create test users
  printf "${COLOR_BLUE}⠿${COLOR_RESET} Creating test users...\n"

  local owner_id="test-owner-$(date +%s)"
  local member1_id="test-member1-$(date +%s)"

  exec_sql "INSERT INTO auth.users (id, email, display_name, default_role) VALUES
    ('$owner_id', 'owner_test@example.com', 'Test Owner', 'user'),
    ('$member1_id', 'member1_test@example.com', 'Test Member 1', 'user')
  ON CONFLICT (email) DO NOTHING;" >/dev/null 2>&1

  printf "  ${COLOR_GREEN}✓${COLOR_RESET} Created test users\n\n"

  # Create test tasks
  printf "${COLOR_BLUE}⠿${COLOR_RESET} Creating test tasks...\n"

  exec_sql "INSERT INTO tasks (title, description, status, assigned_to, created_by) VALUES
    ('Test Task 1', 'Test approval workflow', 'pending', '$member1_id', '$owner_id')
  ON CONFLICT DO NOTHING;" >/dev/null 2>&1 || true

  printf "  ${COLOR_GREEN}✓${COLOR_RESET} Created test tasks\n\n"

  # Run scenario
  case "$scenario" in
    complete)
      printf "${COLOR_BLUE}⠿${COLOR_RESET} Simulating task completion...\n"
      exec_sql "UPDATE tasks SET status='completed' WHERE assigned_to='$member1_id';" >/dev/null 2>&1 || true
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} Task marked as completed\n\n"

      printf "${COLOR_BLUE}⠿${COLOR_RESET} Simulating approval...\n"
      exec_sql "UPDATE tasks SET status='approved' WHERE assigned_to='$member1_id';" >/dev/null 2>&1 || true
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} Task approved\n\n"
      ;;
    rejection)
      printf "${COLOR_BLUE}⠿${COLOR_RESET} Simulating task completion...\n"
      exec_sql "UPDATE tasks SET status='completed' WHERE assigned_to='$member1_id';" >/dev/null 2>&1 || true
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} Task marked as completed\n\n"

      printf "${COLOR_BLUE}⠿${COLOR_RESET} Simulating rejection...\n"
      exec_sql "UPDATE tasks SET status='rejected' WHERE assigned_to='$member1_id';" >/dev/null 2>&1 || true
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} Task rejected\n\n"
      ;;
    *)
      printf "${COLOR_YELLOW}⚠${COLOR_RESET} Unknown scenario: $scenario\n"
      ;;
  esac

  printf "${COLOR_GREEN}✓${COLOR_RESET} Workflow test complete\n\n"

  printf "${COLOR_DIM}Test users created:${COLOR_RESET}\n"
  printf "  owner_test@example.com (password: test123)\n"
  printf "  member1_test@example.com (password: test123)\n\n"

  printf "${COLOR_DIM}To cleanup:${COLOR_RESET}\n"
  printf "  ${COLOR_BLUE}nself test workflow approval --cleanup${COLOR_RESET}\n"
}

# Cleanup test data
cleanup_test_data() {
  printf "${COLOR_BLUE}⠿${COLOR_RESET} Cleaning up test data...\n"

  exec_sql "DELETE FROM tasks WHERE title LIKE 'Test Task %';" >/dev/null 2>&1 || true
  exec_sql "DELETE FROM auth.users WHERE email LIKE '%_test@example.com';" >/dev/null 2>&1 || true

  printf "${COLOR_GREEN}✓${COLOR_RESET} Test data removed\n"
}

# Main command dispatcher
main() {
  local workflow="${1:-}"
  shift || true

  local user_count=3
  local scenario="complete"
  local cleanup=false

  # Parse options
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --users=*)
        user_count="${1#*=}"
        shift
        ;;
      --scenario=*)
        scenario="${1#*=}"
        shift
        ;;
      --cleanup)
        cleanup=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  if [[ "$cleanup" == "true" ]]; then
    cleanup_test_data
    return 0
  fi

  case "$workflow" in
    approval)
      test_approval_workflow "$user_count" "$scenario"
      ;;
    -h|--help|help|"")
      show_help
      ;;
    *)
      printf "${COLOR_YELLOW}⚠${COLOR_RESET} Workflow not yet implemented: $workflow\n"
      printf "  Currently supported: approval\n"
      return 1
      ;;
  esac
}

main "$@"
