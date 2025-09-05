#!/usr/bin/env bats

setup() {
    # Create temp test directory
    TEST_DIR=$(mktemp -d)
    cd "$TEST_DIR"
    
    # Copy nself to test location
    NSELF_PATH="/Users/admin/Sites/nself"
    export PATH="$NSELF_PATH/bin:$PATH"
}

teardown() {
    # Clean up test directory
    cd /
    rm -rf "$TEST_DIR"
}

@test "build command completes without hanging" {
    # Initialize project
    nself init
    
    # Build should complete within 60 seconds (was hanging before)
    timeout 60 nself build
    
    # Check that key files were created
    [ -f docker-compose.yml ]
    [ -d nginx ]
    [ -d postgres ]
}

@test "build command handles missing generate_password function" {
    # Initialize project
    nself init
    
    # Add custom service that might trigger password generation
    echo "NSELF_ADMIN_ENABLED=true" >> .env
    
    # Build should not fail with "generate_password: command not found"
    timeout 60 nself build
    
    # Should create docker-compose without errors
    [ -f docker-compose.yml ]
    grep -q "nself-admin" docker-compose.yml
}

@test "compose generation works with proper environment loading" {
    # Initialize project
    nself init
    
    # Generate compose file should work without hanging
    timeout 30 nself build
    
    # Check compose file was generated properly
    [ -f docker-compose.yml ]
    grep -q "postgres" docker-compose.yml
    grep -q "hasura" docker-compose.yml
}

@test "service generation has timeout protection" {
    # Initialize project
    nself init
    
    # Enable services that might cause hanging
    echo "SERVICES_ENABLED=true" >> .env
    echo "FUNCTIONS_ENABLED=true" >> .env
    
    # Should complete even with service generation enabled
    timeout 60 nself build
    
    [ -f docker-compose.yml ]
}