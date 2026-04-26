# Feature: Functions

ɳSelf includes a serverless runtime for running custom backend logic close to your data.

## What's Included

| Capability | Description |
|-----------|-------------|
| HTTP triggers | Run functions on HTTP requests |
| Event triggers | Respond to Hasura database events |
| Cron triggers | Schedule functions on a cron expression |
| TypeScript/JavaScript | Node.js runtime |

## How to Enable

```bash
nself service enable functions
nself build && nself restart
```

Or set in `.env`:
```env
FUNCTIONS_ENABLED=true
```

## Key Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FUNCTIONS_ENABLED` | Enable Functions service | `false` |
| `FUNCTIONS_PORT` | Internal port | `3000` |

## Deploying Functions

Place function files in your project's `functions/` directory. ɳSelf mounts this directory into the Functions container automatically.

```
my-project/
└── functions/
    ├── hello-world.ts
    └── send-email.ts
```

Functions are available at: `https://functions.your-domain.com/v1/{function-name}`

## Event Triggers

Configure event triggers in the Hasura console or via Hasura migrations to call functions when database rows are inserted, updated, or deleted.

## See Also

- [[cmd-service]], enable/disable functions
- [[Config-Env-Vars]], full env var reference
- [[Architecture]], how functions integrate with the stack

---
← [[Home]] | [[_Sidebar]]
