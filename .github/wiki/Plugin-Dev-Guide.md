# Plugin Development Guide

Build a new nSelf plugin. Plugins extend the nSelf stack with additional Docker services, Nginx routes, and CLI commands.

## Plugin Structure

A plugin is a directory (or Git repository) containing:

```
my-plugin/
├── manifest.json       # plugin metadata and declarations
├── compose.yml         # Docker Compose overlay
├── nginx.conf          # Nginx location blocks (optional)
└── README.md
```

## Manifest Format

```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "Short description of what the plugin does.",
  "tier": "free",
  "compose": "compose.yml",
  "nginx": "nginx.conf",
  "env_vars": [
    {
      "key": "MY_PLUGIN_PORT",
      "description": "Port the plugin service listens on",
      "default": "3100",
      "required": true
    },
    {
      "key": "MY_PLUGIN_SECRET",
      "description": "Shared secret for internal auth",
      "secret": true
    }
  ]
}
```

**Tier values:** `free` | `basic` | `pro` | `elite` | `business` | `enterprise`

## Compose Overlay

The `compose.yml` defines Docker services to add to the stack. nSelf merges this into the generated `docker-compose.yml` at build time:

```yaml
services:
  my-plugin:
    image: myorg/my-plugin:1.0.0
    restart: unless-stopped
    environment:
      MY_PLUGIN_SECRET: "${MY_PLUGIN_SECRET}"
    ports:
      - "127.0.0.1:${MY_PLUGIN_PORT}:3100"
    networks:
      - default
```

All services must:
- Bind to `127.0.0.1` on the host (not `0.0.0.0`)
- Use `restart: unless-stopped`
- Connect to the `default` network

## Nginx Injection

Add Nginx location blocks to route external traffic to your plugin:

```nginx
location /my-plugin/ {
    proxy_pass http://my-plugin:3100/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

nSelf appends this block to the generated Nginx server config for your domain.

## Env Var Declaration

All env vars your plugin needs must be declared in the manifest. nSelf:
1. Validates required vars are present before build
2. Auto-generates secret vars (if `"secret": true` and not already set)
3. Injects declared vars into the compose overlay

## Testing Locally

```bash
# Install from a local directory
nself plugin install --local ./my-plugin/

# Rebuild and start
nself build && nself restart

# Check it's running
nself status
nself urls
```

## Publishing Checklist

Before submitting a PR to the plugins repository:

- [ ] `manifest.json` validates (run `nself plugin validate ./my-plugin/`)
- [ ] Plugin installs cleanly on a fresh `nself init`
- [ ] All declared env vars are documented
- [ ] Compose overlay uses `127.0.0.1` port binding
- [ ] Nginx config tested (no syntax errors)
- [ ] `README.md` covers: what it does, env vars, usage

**Free plugins:** submit a PR to [nself-org/plugins](https://github.com/nself-org/plugins).
**Pro plugins:** contact the nSelf team.

## See Also

- [[Plugin-Architecture]] — how plugins integrate with the core
- [[Plugin-Overview]] — existing plugin catalogue
- [[Contributing]] — general contribution guidelines

---
← [[Home]] | [[_Sidebar]]
