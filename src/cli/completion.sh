#!/usr/bin/env bash
# completion.sh - Shell completion for nself CLI
# Supports bash, zsh, and fish

set -o pipefail

# Source shared utilities
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true

# Fallback logging
if ! declare -f log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi

# ============================================================================
# BASH COMPLETION
# ============================================================================

generate_bash() {
  cat << 'BASH_COMPLETION'
# nself bash completion
# Add to ~/.bashrc: eval "$(nself completion bash)"

_nself_completions() {
    local cur prev words cword
    _init_completion || return

    # Main commands
    local commands="init build start stop restart status logs exec urls doctor help update ssl trust admin clean reset version env deploy prod staging db sync email search functions mlflow metrics monitor"

    # Subcommands by parent
    local db_commands="migrate seed mock backup restore schema types shell query inspect data optimize reset status help"
    local env_commands="create switch list diff validate export import"
    local sync_commands="pull push files config init status history"
    local ssl_commands="generate renew bootstrap check trust"
    local email_commands="test config provider"
    local search_commands="reindex status config"
    local functions_commands="deploy list logs invoke"
    local metrics_commands="profile status export"

    case "${prev}" in
        nself)
            COMPREPLY=($(compgen -W "${commands}" -- "${cur}"))
            return 0
            ;;
        db)
            COMPREPLY=($(compgen -W "${db_commands}" -- "${cur}"))
            return 0
            ;;
        env)
            COMPREPLY=($(compgen -W "${env_commands}" -- "${cur}"))
            return 0
            ;;
        sync)
            COMPREPLY=($(compgen -W "${sync_commands}" -- "${cur}"))
            return 0
            ;;
        ssl)
            COMPREPLY=($(compgen -W "${ssl_commands}" -- "${cur}"))
            return 0
            ;;
        email)
            COMPREPLY=($(compgen -W "${email_commands}" -- "${cur}"))
            return 0
            ;;
        search)
            COMPREPLY=($(compgen -W "${search_commands}" -- "${cur}"))
            return 0
            ;;
        functions)
            COMPREPLY=($(compgen -W "${functions_commands}" -- "${cur}"))
            return 0
            ;;
        metrics)
            COMPREPLY=($(compgen -W "${metrics_commands}" -- "${cur}"))
            return 0
            ;;
        migrate)
            COMPREPLY=($(compgen -W "status up down create fresh repair" -- "${cur}"))
            return 0
            ;;
        seed)
            COMPREPLY=($(compgen -W "run users create status" -- "${cur}"))
            return 0
            ;;
        mock)
            COMPREPLY=($(compgen -W "generate auto preview clear config" -- "${cur}"))
            return 0
            ;;
        backup)
            COMPREPLY=($(compgen -W "create list restore schedule prune" -- "${cur}"))
            return 0
            ;;
        schema)
            COMPREPLY=($(compgen -W "show diff diagram indexes export import scaffold apply" -- "${cur}"))
            return 0
            ;;
        types)
            COMPREPLY=($(compgen -W "typescript go python" -- "${cur}"))
            return 0
            ;;
        inspect)
            COMPREPLY=($(compgen -W "overview size cache index unused-indexes bloat slow locks connections" -- "${cur}"))
            return 0
            ;;
        data)
            COMPREPLY=($(compgen -W "export import anonymize sync" -- "${cur}"))
            return 0
            ;;
        pull|push)
            COMPREPLY=($(compgen -W "staging production" -- "${cur}"))
            return 0
            ;;
        files|config)
            COMPREPLY=($(compgen -W "pull push diff" -- "${cur}"))
            return 0
            ;;
        doctor)
            COMPREPLY=($(compgen -W "--fix --verbose --help" -- "${cur}"))
            return 0
            ;;
        scaffold)
            COMPREPLY=($(compgen -W "basic ecommerce saas blog" -- "${cur}"))
            return 0
            ;;
        logs)
            # Complete with service names from docker-compose
            if [[ -f "docker-compose.yml" ]]; then
                local services=$(grep -E "^  [a-z].*:" docker-compose.yml 2>/dev/null | sed 's/://g' | xargs)
                COMPREPLY=($(compgen -W "${services}" -- "${cur}"))
            fi
            return 0
            ;;
        exec)
            # Complete with service names
            if [[ -f "docker-compose.yml" ]]; then
                local services=$(grep -E "^  [a-z].*:" docker-compose.yml 2>/dev/null | sed 's/://g' | xargs)
                COMPREPLY=($(compgen -W "${services}" -- "${cur}"))
            fi
            return 0
            ;;
    esac

    # Default: show main commands
    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=($(compgen -W "${commands}" -- "${cur}"))
    fi

    return 0
}

complete -F _nself_completions nself
BASH_COMPLETION
}

# ============================================================================
# ZSH COMPLETION
# ============================================================================

generate_zsh() {
  cat << 'ZSH_COMPLETION'
#compdef nself
# nself zsh completion
# Add to ~/.zshrc: eval "$(nself completion zsh)"

_nself() {
    local -a commands db_commands env_commands sync_commands

    commands=(
        'init:Initialize a new nself project'
        'build:Build project configuration and containers'
        'start:Start all services'
        'stop:Stop all services'
        'restart:Restart all services'
        'status:Show service status'
        'logs:View service logs'
        'exec:Execute command in service container'
        'urls:Show all service URLs'
        'doctor:Run system diagnostics'
        'help:Show help information'
        'update:Update nself CLI and admin'
        'ssl:SSL certificate management'
        'trust:Install SSL root CA to system'
        'admin:Open admin dashboard'
        'clean:Clean up containers and volumes'
        'reset:Reset project to initial state'
        'version:Show version information'
        'env:Environment management'
        'deploy:Deploy to remote environment'
        'prod:Production deployment shortcut'
        'staging:Staging deployment shortcut'
        'db:Database management'
        'sync:Environment synchronization'
        'email:Email service configuration'
        'search:Search service management'
        'functions:Serverless functions'
        'mlflow:ML experiment tracking'
        'metrics:Monitoring metrics'
        'monitor:Monitoring dashboard'
        'completion:Generate shell completions'
    )

    db_commands=(
        'migrate:Run database migrations'
        'seed:Seed database with data'
        'mock:Generate mock data'
        'backup:Create database backup'
        'restore:Restore from backup'
        'schema:Schema management tools'
        'types:Generate type definitions'
        'shell:Open database shell'
        'query:Execute SQL query'
        'inspect:Database analysis'
        'data:Data import/export'
        'optimize:Optimize database'
        'reset:Reset database'
        'status:Show database status'
    )

    env_commands=(
        'create:Create new environment'
        'switch:Switch to environment'
        'list:List environments'
        'diff:Compare environments'
        'validate:Validate configuration'
        'export:Export environment config'
        'import:Import environment config'
    )

    sync_commands=(
        'pull:Pull from remote environment'
        'push:Push to remote environment'
        'files:Sync files'
        'config:Sync configuration'
        'init:Initialize sync profiles'
        'status:Show sync status'
        'history:Show sync history'
    )

    _arguments -C \
        '1: :->command' \
        '2: :->subcommand' \
        '*::arg:->args'

    case "$state" in
        command)
            _describe -t commands 'nself command' commands
            ;;
        subcommand)
            case "$words[2]" in
                db)
                    _describe -t db_commands 'db command' db_commands
                    ;;
                env)
                    _describe -t env_commands 'env command' env_commands
                    ;;
                sync)
                    _describe -t sync_commands 'sync command' sync_commands
                    ;;
                logs|exec)
                    # Complete with service names
                    if [[ -f "docker-compose.yml" ]]; then
                        local services
                        services=($(grep -E "^  [a-z].*:" docker-compose.yml 2>/dev/null | sed 's/://g'))
                        _describe -t services 'service' services
                    fi
                    ;;
                doctor)
                    local -a doctor_opts
                    doctor_opts=('--fix:Auto-fix detected issues' '--verbose:Verbose output' '--help:Show help')
                    _describe -t doctor_opts 'doctor option' doctor_opts
                    ;;
            esac
            ;;
    esac
}

_nself "$@"
ZSH_COMPLETION
}

# ============================================================================
# FISH COMPLETION
# ============================================================================

generate_fish() {
  cat << 'FISH_COMPLETION'
# nself fish completion
# Save to ~/.config/fish/completions/nself.fish

# Main commands
complete -c nself -f -n "__fish_use_subcommand" -a "init" -d "Initialize new project"
complete -c nself -f -n "__fish_use_subcommand" -a "build" -d "Build configuration"
complete -c nself -f -n "__fish_use_subcommand" -a "start" -d "Start services"
complete -c nself -f -n "__fish_use_subcommand" -a "stop" -d "Stop services"
complete -c nself -f -n "__fish_use_subcommand" -a "restart" -d "Restart services"
complete -c nself -f -n "__fish_use_subcommand" -a "status" -d "Show status"
complete -c nself -f -n "__fish_use_subcommand" -a "logs" -d "View logs"
complete -c nself -f -n "__fish_use_subcommand" -a "exec" -d "Execute command"
complete -c nself -f -n "__fish_use_subcommand" -a "urls" -d "Show URLs"
complete -c nself -f -n "__fish_use_subcommand" -a "doctor" -d "System diagnostics"
complete -c nself -f -n "__fish_use_subcommand" -a "help" -d "Show help"
complete -c nself -f -n "__fish_use_subcommand" -a "update" -d "Update CLI"
complete -c nself -f -n "__fish_use_subcommand" -a "ssl" -d "SSL management"
complete -c nself -f -n "__fish_use_subcommand" -a "trust" -d "Install root CA"
complete -c nself -f -n "__fish_use_subcommand" -a "admin" -d "Admin dashboard"
complete -c nself -f -n "__fish_use_subcommand" -a "clean" -d "Clean up"
complete -c nself -f -n "__fish_use_subcommand" -a "reset" -d "Reset project"
complete -c nself -f -n "__fish_use_subcommand" -a "version" -d "Show version"
complete -c nself -f -n "__fish_use_subcommand" -a "env" -d "Environment management"
complete -c nself -f -n "__fish_use_subcommand" -a "deploy" -d "Deploy to remote"
complete -c nself -f -n "__fish_use_subcommand" -a "prod" -d "Production deploy"
complete -c nself -f -n "__fish_use_subcommand" -a "staging" -d "Staging deploy"
complete -c nself -f -n "__fish_use_subcommand" -a "db" -d "Database management"
complete -c nself -f -n "__fish_use_subcommand" -a "sync" -d "Environment sync"
complete -c nself -f -n "__fish_use_subcommand" -a "completion" -d "Shell completions"

# db subcommands
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "migrate" -d "Run migrations"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "seed" -d "Seed data"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "mock" -d "Generate mock data"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "backup" -d "Create backup"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "restore" -d "Restore backup"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "schema" -d "Schema tools"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "types" -d "Generate types"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "shell" -d "Database shell"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "query" -d "Execute query"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "inspect" -d "Analyze database"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "data" -d "Import/export data"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "optimize" -d "Optimize database"
complete -c nself -f -n "__fish_seen_subcommand_from db" -a "reset" -d "Reset database"

# sync subcommands
complete -c nself -f -n "__fish_seen_subcommand_from sync" -a "pull" -d "Pull from remote"
complete -c nself -f -n "__fish_seen_subcommand_from sync" -a "push" -d "Push to remote"
complete -c nself -f -n "__fish_seen_subcommand_from sync" -a "files" -d "Sync files"
complete -c nself -f -n "__fish_seen_subcommand_from sync" -a "config" -d "Sync config"
complete -c nself -f -n "__fish_seen_subcommand_from sync" -a "init" -d "Initialize profiles"
complete -c nself -f -n "__fish_seen_subcommand_from sync" -a "status" -d "Connection status"
complete -c nself -f -n "__fish_seen_subcommand_from sync" -a "history" -d "Sync history"

# doctor options
complete -c nself -f -n "__fish_seen_subcommand_from doctor" -l fix -d "Auto-fix issues"
complete -c nself -f -n "__fish_seen_subcommand_from doctor" -l verbose -d "Verbose output"
complete -c nself -f -n "__fish_seen_subcommand_from doctor" -l help -d "Show help"

# Environment completions for sync
complete -c nself -f -n "__fish_seen_subcommand_from pull" -a "staging production" -d "Environment"
complete -c nself -f -n "__fish_seen_subcommand_from push" -a "staging" -d "Environment"
FISH_COMPLETION
}

# ============================================================================
# INSTALL HELPERS
# ============================================================================

install_bash() {
  local target="${1:-$HOME/.bashrc}"

  if [[ -f "$target" ]] && grep -q "nself completion bash" "$target" 2>/dev/null; then
    log_info "Bash completion already installed in $target"
    return 0
  fi

  echo '' >> "$target"
  echo '# nself shell completion' >> "$target"
  echo 'eval "$(nself completion bash)"' >> "$target"

  log_success "Bash completion installed in $target"
  log_info "Restart your shell or run: source $target"
}

install_zsh() {
  local target="${1:-$HOME/.zshrc}"

  if [[ -f "$target" ]] && grep -q "nself completion zsh" "$target" 2>/dev/null; then
    log_info "Zsh completion already installed in $target"
    return 0
  fi

  echo '' >> "$target"
  echo '# nself shell completion' >> "$target"
  echo 'eval "$(nself completion zsh)"' >> "$target"

  log_success "Zsh completion installed in $target"
  log_info "Restart your shell or run: source $target"
}

install_fish() {
  local target="${1:-$HOME/.config/fish/completions/nself.fish}"

  mkdir -p "$(dirname "$target")"
  generate_fish > "$target"

  log_success "Fish completion installed in $target"
  log_info "Completions will be available in new fish sessions"
}

# ============================================================================
# HELP
# ============================================================================

show_help() {
  cat << 'EOF'
nself completion - Generate shell completions

USAGE:
  nself completion <shell> [options]
  nself completion install <shell>

SHELLS:
  bash        Generate bash completion script
  zsh         Generate zsh completion script
  fish        Generate fish completion script

OPTIONS:
  install     Install completion to shell config file

EXAMPLES:
  # Output completion script (manual setup)
  nself completion bash >> ~/.bashrc
  nself completion zsh >> ~/.zshrc
  nself completion fish > ~/.config/fish/completions/nself.fish

  # Or use eval (add to shell config)
  eval "$(nself completion bash)"
  eval "$(nself completion zsh)"

  # Auto-install to config file
  nself completion install bash
  nself completion install zsh
  nself completion install fish

AFTER INSTALLATION:
  Restart your shell or source your config file:
    source ~/.bashrc   # bash
    source ~/.zshrc    # zsh

EOF
}

# ============================================================================
# MAIN
# ============================================================================

main() {
  local command="${1:-help}"
  shift || true

  case "$command" in
    bash)
      generate_bash
      ;;
    zsh)
      generate_zsh
      ;;
    fish)
      generate_fish
      ;;
    install)
      local shell="${1:-bash}"
      case "$shell" in
        bash) install_bash ;;
        zsh) install_zsh ;;
        fish) install_fish ;;
        *) log_error "Unknown shell: $shell"; return 1 ;;
      esac
      ;;
    help|--help|-h)
      show_help
      ;;
    *)
      log_error "Unknown shell: $command"
      echo ""
      show_help
      return 1
      ;;
  esac
}

main "$@"
