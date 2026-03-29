# Commands

nSelf CLI v2 provides 24 top-level commands.

| Command | Description |
|---------|-------------|
| [[cmd-init]] | Initialize a new nSelf project — generates `.env` configuration via interactive wizard |
| [[cmd-build]] | Compose infrastructure from `.env` — generates `docker-compose.yml`, Nginx config, and SSL certificates |
| [[cmd-start]] | Boot the nSelf stack with health checks and automatic database initialization |
| [[cmd-stop]] | Gracefully shut down all services or specific named services |
| [[cmd-restart]] | Smart restart with config change detection — only restarts affected services |
| [[cmd-status]] | Show health status of all services with exit code signaling |
| [[cmd-logs]] | View and filter service logs with color, search, and tail support |
| [[cmd-urls]] | Display all service URLs grouped by type with route conflict detection |
| [[cmd-config]] | Manage project configuration — show, get, set, list, validate, export, import |
| [[cmd-service]] | Enable, disable, and list optional services (Redis, MinIO, email, search, monitoring) |
| [[cmd-ssl]] | Manage SSL certificates — status and force renewal |
| [[cmd-db]] | Database operations — migrate, seed, backup, restore, shell, reset, Hasura metadata |
| [[cmd-plugin]] | Manage nSelf plugins — install, remove, update, list, start, stop, status |
| [[cmd-doctor]] | Run comprehensive system diagnostics — infrastructure, Docker, config, network |
| [[cmd-health]] | Health check management — check, watch, history, single service or endpoint |
| [[cmd-exec]] | Execute a command inside a running service container |
| [[cmd-clean]] | Remove Docker resources associated with the current project |
| [[cmd-reset]] | Stop containers and remove generated files (preserves `.env` and data) |
| [[cmd-update]] | Update the nSelf CLI binary and admin Docker image |
| [[cmd-version]] | Show version and system information |
| [[cmd-migrate]] | Detect and migrate v1 (Bash CLI) artifacts to v2 |
| [[cmd-admin]] | Open the nSelf Admin dashboard in browser at `localhost:3021` |
| [[cmd-completion]] | Generate shell completion scripts for bash, zsh, or fish |
| [[cmd-license]] | Manage Pro membership license key — set, show, validate, clear, upgrade |
