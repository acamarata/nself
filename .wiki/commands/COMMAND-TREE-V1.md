# nself Command Tree v1

Authoritative list of all top-level commands. Maximum 31 top-level commands (see VISION.md non-goals).

## Core Lifecycle (7)

| Command | Description |
|---------|-------------|
| `nself init` | Initialize a new nself project |
| `nself build` | Generate docker-compose.yml, nginx configs, SSL from .env |
| `nself start` | Start all services (alias: `nself up`) |
| `nself stop` | Stop all services (alias: `nself down`) |
| `nself restart` | Restart all services |
| `nself destroy` | Remove all containers, volumes, and generated files |
| `nself status` | Show service status and health |

## Configuration (4)

| Command | Description |
|---------|-------------|
| `nself config` | View/edit project configuration |
| `nself env` | Manage environment files and variables |
| `nself secrets` | Manage secret values (.env.secrets) |
| `nself validate` | Validate project configuration |

## Database (2)

| Command | Description |
|---------|-------------|
| `nself db` | Database operations (backup, restore, migrate, shell) |
| `nself migrate` | Run database migrations |

## Services (5)

| Command | Description |
|---------|-------------|
| `nself service` | Manage individual services |
| `nself nginx` | Nginx proxy management |
| `nself ssl` | SSL certificate management |
| `nself auth` | Authentication service management |
| `nself hasura` | Hasura GraphQL console and metadata |

## Plugins (2)

| Command | Description |
|---------|-------------|
| `nself plugin` | Install, remove, update, list plugins |
| `nself license` | Manage plugin license key and tier |

## Deployment (3)

| Command | Description |
|---------|-------------|
| `nself deploy` | Deploy to staging or production |
| `nself infra` | Infrastructure provisioning and management |
| `nself cloud` | Cloud hosting operations |

## Multi-Tenancy (1)

| Command | Description |
|---------|-------------|
| `nself tenant` | Tenant CRUD, isolation, billing, branding |

## Development (3)

| Command | Description |
|---------|-------------|
| `nself dev` | Development mode with hot-reload |
| `nself logs` | View service logs |
| `nself exec` | Execute command in a service container |

## Operations (3)

| Command | Description |
|---------|-------------|
| `nself doctor` | Health check and diagnostics |
| `nself backup` | Backup databases, volumes, configs |
| `nself monitor` | Monitoring stack management |

## Utility (1)

| Command | Description |
|---------|-------------|
| `nself version` | Show CLI version |

## Total: 31 top-level commands

Note: Several command files in `src/cli/` are subcommand implementations (e.g., `plugin_install.sh`, `plugin_config.sh`) or aliases (e.g., `up.sh` -> `start.sh`, `down.sh` -> `stop.sh`). These are NOT separate top-level commands.

## Plugin-Provided Commands

These commands are available when the corresponding plugin is installed:

| Command | Plugin | Description |
|---------|--------|-------------|
| `nself ai` | nself-ai | AI provider management and routing |
| `nself claw` | nself-claw | nClaw AI assistant management |
| `nself mux` | nself-mux | Email/message multiplexer |
| `nself voice` | nself-voice | Voice service management |
| `nself browser` | nself-browser | Browser automation service |
| `nself notify` | nself-notify | Push notification management |
| `nself cron` | nself-cron | Scheduled task management |
| `nself search` | nself-search | Full-text search (MeiliSearch) |
| `nself email` | nself-mail | Email service management |
