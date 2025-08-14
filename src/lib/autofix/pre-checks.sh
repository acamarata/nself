#!/bin/bash

# Pre-flight checks before starting Docker
# Fixes common issues before they cause Docker failures

run_pre_checks() {
    local silent="${1:-false}"
    local issues_fixed=0
    
    # Check 1: Ensure config-server files exist
    if [[ ! -f "config-server/index.js" ]]; then
        [[ "$silent" != "true" ]] && printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating config-server files..."
        
        mkdir -p config-server
        if [[ -f "/Users/admin/Sites/nself/src/lib/auto-fix/dockerfile-generator.sh" ]]; then
            source "/Users/admin/Sites/nself/src/lib/auto-fix/dockerfile-generator.sh"
            generate_config_server "config-server" >/dev/null 2>&1
            ((issues_fixed++))
            [[ "$silent" != "true" ]] && printf "\r${COLOR_GREEN}✓${COLOR_RESET} Generated config-server files        \n"
        fi
    fi
    
    # Check 2: Verify Postgres port configuration
    local expected_port="${POSTGRES_PORT:-5432}"
    if grep -q "POSTGRES_PORT=5433" .env.local 2>/dev/null && [[ "$expected_port" == "5432" ]]; then
        [[ "$silent" != "true" ]] && printf "${COLOR_BLUE}⠋${COLOR_RESET} Fixing Postgres port configuration..."
        sed -i '' 's/POSTGRES_PORT=5433/POSTGRES_PORT=5432/' .env.local
        ((issues_fixed++))
        [[ "$silent" != "true" ]] && printf "\r${COLOR_GREEN}✓${COLOR_RESET} Fixed Postgres port configuration    \n"
    fi
    
    # Check 3: Ensure Docker network exists
    if ! docker network inspect unity_default >/dev/null 2>&1; then
        [[ "$silent" != "true" ]] && printf "${COLOR_BLUE}⠋${COLOR_RESET} Creating Docker network..."
        docker network create unity_default >/dev/null 2>&1
        ((issues_fixed++))
        [[ "$silent" != "true" ]] && printf "\r${COLOR_GREEN}✓${COLOR_RESET} Created Docker network               \n"
    fi
    
    # Check 4: Clean up zombie containers
    local zombies=$(docker ps -aq -f status=exited -f name=unity_ 2>/dev/null | wc -l)
    if [[ $zombies -gt 0 ]]; then
        [[ "$silent" != "true" ]] && printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning up stopped containers..."
        docker ps -aq -f status=exited -f name=unity_ | xargs docker rm >/dev/null 2>&1
        ((issues_fixed++))
        [[ "$silent" != "true" ]] && printf "\r${COLOR_GREEN}✓${COLOR_RESET} Cleaned up stopped containers        \n"
    fi
    
    # Check 5: Ensure critical services have resources
    local project_name="${PROJECT_NAME:-unity}"
    if docker ps -q -f name="${project_name}_postgres" >/dev/null 2>&1; then
        # Postgres is running, check if it's actually ready
        if ! docker exec "${project_name}_postgres" pg_isready -U postgres >/dev/null 2>&1; then
            [[ "$silent" != "true" ]] && printf "${COLOR_BLUE}⠋${COLOR_RESET} Restarting Postgres..."
            docker restart "${project_name}_postgres" >/dev/null 2>&1
            sleep 3
            ((issues_fixed++))
            [[ "$silent" != "true" ]] && printf "\r${COLOR_GREEN}✓${COLOR_RESET} Restarted Postgres                   \n"
        else
            # Postgres is ready, ensure schemas exist
            [[ "$silent" != "true" ]] && printf "${COLOR_BLUE}⠋${COLOR_RESET} Ensuring database schemas exist..."
            
            # Create necessary schemas
            docker exec "${project_name}_postgres" psql -U postgres -c "CREATE SCHEMA IF NOT EXISTS auth;" >/dev/null 2>&1
            docker exec "${project_name}_postgres" psql -U postgres -c "CREATE SCHEMA IF NOT EXISTS storage;" >/dev/null 2>&1
            docker exec "${project_name}_postgres" psql -U postgres -c "CREATE SCHEMA IF NOT EXISTS public;" >/dev/null 2>&1
            
            # Grant permissions
            docker exec "${project_name}_postgres" psql -U postgres -c "
                GRANT ALL ON SCHEMA auth TO postgres;
                GRANT ALL ON SCHEMA storage TO postgres;
                GRANT ALL ON SCHEMA public TO postgres;
            " >/dev/null 2>&1
            
            [[ "$silent" != "true" ]] && printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database schemas ready                     \n"
            ((issues_fixed++))
        fi
    fi
    
    # Check 6: Ensure bullmq workers have package.json
    for worker_dir in services/bullmq/* workers/* services/node/bull*; do
        if [[ -d "$worker_dir" ]] && [[ ! -f "$worker_dir/package.json" ]]; then
            local worker_name=$(basename "$worker_dir")
            [[ "$silent" != "true" ]] && printf "${COLOR_BLUE}⠋${COLOR_RESET} Creating package.json for $worker_name..."
            
            cat > "$worker_dir/package.json" << 'EOF'
{
  "name": "worker",
  "version": "1.0.0",
  "main": "src/index.js",
  "scripts": {
    "start": "node src/index.js"
  },
  "dependencies": {
    "bullmq": "^5.0.0",
    "ioredis": "^5.3.2"
  }
}
EOF
            ((issues_fixed++))
            [[ "$silent" != "true" ]] && printf "\r${COLOR_GREEN}✓${COLOR_RESET} Created package.json for $worker_name   \n"
        fi
    done
    
    return $issues_fixed
}