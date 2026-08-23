# nself ollama

<!-- BEGIN PROSE:summary -->
> **Deprecated.** Use [`nself model --provider ollama`](cmd-ai-models.md) instead.
<!-- END PROSE:summary -->

## Synopsis

```
nself ollama <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself ollama` manages the local Ollama container and its model library. This command is
preserved for compatibility but the `model` command is the current interface for provider
management.

See [cmd-ai-models.md](cmd-ai-models.md) for full documentation.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `models` | Manage Ollama models |
| `status` | Show Ollama service status and default model |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself ollama models list
  nself ollama models pull llama3.2:3b
  nself ollama models remove gemma-3-4b
  nself ollama status
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
