# Plugin Architecture

ɳSelf's plugin system extends the base stack with additional services, communication tools, AI engines, media pipelines, commerce systems, and more. Free plugins are MIT licensed and install without any key. Pro plugins require a valid license key, but otherwise follow exactly the same architecture. The install pipeline, schema isolation, compose overlay, and Nginx route injection work identically for both tiers; the only difference is a license check gate before the download proceeds.

---

## Plugin Manifest

Every plugin ships with a `plugin.json` manifest at its root. This manifest is the single source of truth for what the plugin needs: ports, database tables, environment variables, dependencies, and health check information. The CLI reads this manifest at install time, at build time, and whenever it generates or validates configuration.

| Field | Type | Purpose |
|-------|------|---------|
| `name` | string | Unique plugin ID (e.g., `"chat"`) |
| `version` | semver | e.g., `"1.0.0"` |
| `description` | string | What the plugin does |
| `category` | string | `communication`, `media`, `commerce`, etc. |
| `license` | string | `"MIT"` (free) or `"Source-Available"` (pro) |
| `port` | int | Network port the plugin service runs on |
| `language` | string | `go`, `rust`, `typescript`, `python` |
| `tables` | string[] | Postgres tables the plugin owns (prefixed `np_{name}_`) |
| `envVars` | object[] | Required and optional environment variables |
| `dependencies` | string[] | Other plugins this plugin requires |
| `health_endpoint` | string | Health check path (e.g., `"/health"`) |

A minimal manifest looks like this:

```json
{
  "name": "chat",
  "version": "1.0.0",
  "description": "Real-time messaging service backed by Postgres and WebSockets",
  "category": "communication",
  "license": "MIT",
  "port": 3401,
  "language": "go",
  "tables": ["rooms", "messages", "members"],
  "envVars": [
    { "key": "CHAT_MAX_MESSAGE_SIZE", "default": "4096", "required": false }
  ],
  "dependencies": [],
  "health_endpoint": "/health"
}
```

---

## Schema Isolation

Every plugin gets its own Postgres schema. This ensures plugins never collide with each other or with the base ɳSelf schema, and makes it trivial to identify which objects belong to which plugin.

The naming conventions are strict and enforced by the CLI:

- **Schema:** `np_{plugin_name}` (e.g., `np_chat`)
- **Role:** `np_{plugin_name}_role` (e.g., `np_chat_role`)
- **Tables:** `np_{plugin_name}_{table}` (e.g., `np_chat_messages`, `np_chat_rooms`)
- **Version tracking:** all plugin schema versions are recorded in `np_common.schema_versions`

Schema creation is idempotent, the CLI uses `CREATE SCHEMA IF NOT EXISTS` and migration guards throughout, so it is always safe to re-run. This means `nself plugin install` can be repeated without corrupting existing data, and upgrades apply only the missing migration steps.

Hasura automatically tracks tables in `np_*` schemas so plugin data is immediately queryable through the GraphQL API without manual configuration.

---

## Compose Overlay Injection

When a plugin is installed, it ships a `docker-compose.plugin.yml` overlay file alongside its other assets. During `nself build`, the CLI merges all installed plugin overlays into the generated `docker-compose.yml` using a deep-merge strategy.

Plugins can contribute:

- **New service definitions**, the primary plugin container and any sidecar processes it needs
- **New named volumes**, persistent storage scoped to the plugin
- **New network connections**, attaching the plugin service to the shared `nself` bridge network

Plugins **cannot** remove, rename, or override existing base services. The merge is additive only. If a plugin overlay attempts to redefine a service that already exists in the base configuration, the build step rejects it with a validation error.

**Example:** Running `nself plugin install chat` adds a `chat` service running on port 3401 to the generated compose file:

```yaml
# docker-compose.plugin.yml (chat plugin)
services:
  chat:
    image: nself/plugin-chat:1.0.0
    restart: unless-stopped
    ports:
      - "127.0.0.1:3401:3401"
    environment:
      - CHAT_MAX_MESSAGE_SIZE=${CHAT_MAX_MESSAGE_SIZE}
    networks:
      - nself
    depends_on:
      - postgres
```

---

## Nginx Route Injection

Plugins declare their Nginx routes in the manifest. During `nself build`, the CLI writes a dedicated configuration file for each installed plugin at `nginx/routes/plugin-{name}.conf`. Routes from different plugins never share a file, so removing a plugin cleanly removes its routes without touching anything else.

Plugins can declare three kinds of routes:

- **Subdomain routes**, map `chat.{BASE_DOMAIN}` to the plugin service running on its declared port (e.g., the chat plugin proxies `chat.example.com` → `127.0.0.1:3401`)
- **Webhook endpoints**, available at `webhooks.{BASE_DOMAIN}/{plugin-name}` for inbound HTTP callbacks from third-party services
- **Custom domain routes**, if `PLUGIN_{NAME}_WEBHOOK_DOMAIN` is set in the environment, the plugin can serve traffic on that custom domain instead of the default subdomain pattern

All plugin routes are managed entirely through `nself build`. Never hand-edit files under `nginx/routes/`, they are regenerated on every build and manual changes will be overwritten.

---

## Config Templating

Plugins declare their required and optional environment variables in the manifest `envVars` array. During `nself plugin install`, the CLI processes each declared variable:

1. **Required vars with no default**, the CLI prompts the user to enter a value interactively. The install will not proceed until all required vars are satisfied.
2. **Optional vars with defaults**, written to `.env.dev` automatically without prompting.
3. **All plugin env vars**, written to a plugin-scoped env file at `~/.nself/plugins/{name}/.env` with permissions `600`. This file is mounted into the plugin container at runtime.

This means plugin configuration is always traceable: every value either came from user input at install time or from a manifest default. There are no hidden side-effects.

---

## License Validation Flow

For Pro plugins, the CLI runs a license check before any download occurs. The flow is:

```
nself plugin install ai
  ├── 1. Check plugin list — "ai" is in the pro list
  ├── 2. Read license key from NSELF_PLUGIN_LICENSE_KEY or ~/.nself/license/key
  ├── 3. Validate format: must start with nself_pro_, nself_max_, nself_ent_, or nself_owner_
  ├── 4. Check local cache (~/.nself/license/cache, 24h TTL)
  ├── 5. If cache miss: POST https://ping.nself.org/license/validate
  │       Body: {"license_key": "...", "product": "plugins-pro"}
  │       200 → valid, cache result
  │       401/403/404 → invalid, deny
  │       Network error → fail open with warning
  └── 6. If valid: proceed with download + install
```

The cache has a 24-hour TTL so repeated installs in a session do not hit the network every time. On a network error, the CLI fails open with a warning rather than blocking the install, this is intentional so that air-gapped or offline environments are not broken by transient connectivity issues.

License keys are stored at `~/.nself/license/key` with `chmod 600`. Set your key with:

```bash
nself license set nself_pro_xxxxxxxx...
```

The key format encodes the tier in its prefix. Tier enforcement is server-side, the `ping.nself.org/license/validate` endpoint determines which plugins a given key is permitted to install.

---

## Installation Flow

The complete install sequence for any plugin:

```
nself plugin install <name>
  1. Fetch registry (with cache + GitHub fallback)
  2. License check (pro only — see above)
  3. Download tarball → verify SHA256 checksum
  4. Extract to ~/.nself/plugins/{name}/
  5. Install system dependencies (apt/brew/yum)
  6. Create Postgres schema (np_{name})
  7. Resolve plugin dependencies (recursive)
  8. Generate ~/.nself/plugins/{name}/.env
```

Dependency resolution in step 7 is recursive and cycle-safe. If plugin A depends on plugin B, and plugin B is not yet installed, the CLI installs B first (running its own full install sequence), then resumes installing A. Circular dependencies are detected and rejected at validation time before any download begins.

After installation, run `nself build` to regenerate the compose file and Nginx configuration with the new plugin included. Then `nself restart` to apply the changes.

---

## Uninstall

Removing a plugin comes in two forms depending on whether you want to keep the data:

```bash
nself plugin remove <name>              # Remove plugin + drop schema (destroys data)
nself plugin remove <name> --keep-data  # Remove plugin, keep schema and all data intact
```

The `--keep-data` flag is the safer default for production use, it pulls the plugin service out of compose and Nginx while preserving everything in Postgres. You can reinstall the plugin later and it will resume with the existing data.

After removing a plugin, run `nself build` and `nself restart` to apply the changes.

---

**See also:** [[Architecture]] | [[Compose-Generation]] | [[Nginx-Generation]] | [[Home]]
