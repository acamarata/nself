# nself dev

<!-- BEGIN PROSE:summary -->
> Start development environment.
<!-- END PROSE:summary -->

## Synopsis

```
nself dev [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself dev` brings up the project stack in development mode by running `docker compose up -d` from the project root. It is the lighter-weight counterpart to `nself start`: no orchestrated health checks, no automatic database init step, just a fast bring-up suited to inner-loop work.

With `--hot`, the command stays attached, polls the `plugins/` directory every 2 seconds for source changes, and rebuilds only the changed plugin's container with `docker compose up -d --no-deps --build <plugin>`. Polling avoids a filesystem-watch dependency. Hot-reload exits cleanly on Ctrl+C and shuts down the watcher (containers keep running).

Use `nself dev` when you want a minimal start. Use `nself start` (or its alias `nself up`) for the production-grade boot with health checks, port pre-flight, and automatic database initialization.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--hot` | `false` | Enable plugin hot-reload on source changes |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Start the dev stack
nself dev

# Start with plugin hot-reload (rebuilds changed plugins on the fly)
nself dev --hot
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-start]], full boot with health checks and DB init
- [[cmd-restart]], smart restart with config change detection
- [[cmd-build]], regenerate compose and nginx config
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
