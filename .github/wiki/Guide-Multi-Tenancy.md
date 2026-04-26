# Guide: Multi-Tenancy

Run multiple independent ɳSelf projects on a single server. Each project is fully isolated with its own containers, network, and port range.

## How It Works

Each `nself init` call creates a named project. Projects share the host OS but are separated by:

- **Docker network isolation**, each project gets its own bridge network (`{project}_default`)
- **Port separation**, each project is assigned a unique port range for internal services
- **Nginx isolation**, separate virtual host configs under `nginx/sites/`
- **Data isolation**, separate Postgres instance and volume per project

## Setting Up Multiple Projects

```bash
# Project 1
cd /srv/project1
nself init project1 --env prod
# Edit .env.prod: set BASE_DOMAIN=project1.example.com
nself build && nself start

# Project 2
cd /srv/project2
nself init project2 --env prod
# Edit .env.prod: set BASE_DOMAIN=project2.example.com
nself build && nself start
```

## Port Allocation

ɳSelf assigns unique ports per project to avoid conflicts. Check assigned ports:

```bash
nself urls        # shows all service URLs for the current project
nself status      # shows running services and their ports
```

If you initialise projects in separate directories, ɳSelf handles port selection automatically.

## DNS Configuration

Each project should have its own subdomain:

```
A  project1.example.com  → server IP
A  project2.example.com  → server IP
```

Or use path-based routing if your use case supports it.

## Resource Considerations

Each project runs its own Postgres, Hasura, and Auth containers. On a 4 GB server, 2 projects is the practical limit. For 3+ projects, use an 8 GB+ server.

## See Also

- [[Guide-Production-Deployment]], single-project setup
- [[Guide-Security-Hardening]], firewall rules per project
- [[Config-System]], project configuration reference

---
← [[Home]] | [[_Sidebar]]
