#!/bin/bash

# env-utils.sh - Safe environment variable handling utilities

# Function to safely load environment files without executing JSON values
load_env_safe() {
    local env_file="${1:-.env.local}"
    
    if [ ! -f "$env_file" ]; then
        return 1
    fi
    
    # Read the file line by line and export variables
    while IFS= read -r line || [ -n "$line" ]; do
        # Skip empty lines and comments
        if [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*# ]]; then
            continue
        fi
        
        # Check if line contains a variable assignment
        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
            var_name="${BASH_REMATCH[1]}"
            var_value="${BASH_REMATCH[2]}"
            
            # Remove inline comments (everything after # that's not inside quotes)
            # First check if value starts with a quote
            if [[ "$var_value" =~ ^[\"\'] ]]; then
                # For quoted values, keep as is (comments inside quotes are part of the value)
                true
            else
                # For unquoted values, strip inline comments
                var_value="${var_value%%#*}"
                # Trim trailing whitespace
                var_value="${var_value%"${var_value##*[![:space:]]}"}"
            fi
            
            # Remove leading/trailing quotes if present (handling both single and double quotes)
            if [[ "$var_value" =~ ^\"(.*)\"$ ]] || [[ "$var_value" =~ ^\'(.*)\'$ ]]; then
                var_value="${BASH_REMATCH[1]}"
            fi
            
            # Export the variable safely
            export "$var_name=$var_value"
        fi
    done < "$env_file"
}

# Function to get a variable value safely (without expanding JSON)
get_env_var() {
    local var_name="$1"
    local env_file="${2:-.env.local}"
    
    if [ ! -f "$env_file" ]; then
        return 1
    fi
    
    # Use grep to find the variable and extract its value
    local line=$(grep "^${var_name}=" "$env_file" | head -1)
    
    if [[ -n "$line" ]]; then
        # Extract value after the = sign
        local value="${line#*=}"
        
        # Remove leading/trailing quotes if present
        if [[ "$value" =~ ^[\"\'](.*)[\"\']$ ]]; then
            value="${BASH_REMATCH[1]}"
        fi
        
        echo "$value"
    fi
}

# Function to escape special characters for safe use in config files
escape_for_config() {
    local value="$1"
    # Escape single quotes for YAML/config files
    echo "$value" | sed "s/'/\\\\'/g"
}

# Function to check if a value is JSON
is_json() {
    local value="$1"
    if [[ "$value" =~ ^\{.*\}$ ]] || [[ "$value" =~ ^\[.*\]$ ]]; then
        return 0
    fi
    return 1
}

# Function to safely expand variables (skipping JSON values)
expand_vars_safe() {
    local template="$1"
    local result="$template"
    
    # Find all variable references like ${VAR_NAME}
    while [[ "$result" =~ \$\{([A-Za-z_][A-Za-z0-9_]*)\} ]]; do
        local var_name="${BASH_REMATCH[1]}"
        local var_value="${!var_name}"
        
        # Check if the value is JSON and skip expansion if it is
        if is_json "$var_value"; then
            # Replace with a placeholder to avoid further expansion
            result="${result//\${${var_name}\}/JSON_PLACEHOLDER_${var_name}}"
        else
            # Safely replace the variable
            result="${result//\${${var_name}\}/${var_value}}"
        fi
    done
    
    # Restore JSON placeholders
    while [[ "$result" =~ JSON_PLACEHOLDER_([A-Za-z_][A-Za-z0-9_]*) ]]; do
        local var_name="${BASH_REMATCH[1]}"
        local var_value="${!var_name}"
        result="${result//JSON_PLACEHOLDER_${var_name}/${var_value}}"
    done
    
    echo "$result"
}

# Export functions for use in other scripts
export -f load_env_safe
export -f get_env_var
export -f escape_for_config
export -f is_json
export -f expand_vars_safe