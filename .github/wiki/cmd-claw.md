# nself claw

<!-- BEGIN PROSE:summary -->
> Moved to a plugin (CLI-R11). Install it to keep using `nself claw`.
<!-- END PROSE:summary -->

## This command moved

`claw` was extracted from the core binary into the `claw-cli` plugin as part
of CLI-R11 (the thin-core extraction). The command surface is unchanged — only where the
code lives changed. The plugin's registry name is `claw-cli`, not `claw`
— that name is already taken by the paid ɳClaw backend service plugin this
CLI talks to — but the command itself is still invoked as `nself claw ...`.

```bash
nself install claw-cli
nself claw config set server https://claw.example.com
nself claw config set api-key <key>
nself claw prompt "hello"
nself claw chat
nself claw pair
```

Until it is installed, `nself claw ...` prints an install hint naming
`claw-cli` correctly (verified live 2026-08-31: `cmd/commands/dispatch.go`'s
`relocatedCommand()` and `internal/plugin/router.go`'s
`ProxyCommandWithHint()` resolve the hint through the deprecation registry
rather than deriving it from the command name, so it never suggests the
wrong `nself install claw`). Full documentation (flags, exit codes,
examples) now lives with the plugin:
https://github.com/nself-org/plugins/tree/main/free/claw-cli

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-ai]], manage the nSelf AI plugin and local LLM stack
- [[cmd-install]], plugin/bundle install sugar
- [[cmd-plugin]], plugin lifecycle management
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
