# nself install

<!-- BEGIN PROSE:summary -->
> Install a plugin or bundle.
<!-- END PROSE:summary -->

## Synopsis

```
nself install <name> [name...] [flags]
```

**Alias:** `nself add`

## Description

<!-- BEGIN PROSE:description -->
Install a plugin or bundle by name.

This is the short form of `nself plugin install`, with bundle awareness: if
the name is a bundle (`nchat`, `nclaw`, `ntv`, `nfamily`, `clawde`, `nsentry`)
the whole bundle is installed, otherwise the name is resolved as a plugin.

Third-party plugins install by URL rather than by name:

  nself install https://example.com/my-plugin.tar.gz

Once installed, the plugin's commands are available directly:

  nself install waf
  nself waf status
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Reinstall even when the plugin is already present |
| `--yes` | `false` | Skip confirmation prompts (required for third-party URL installs in CI) |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself install waf                  # one plugin
  nself install waf cdn analytics    # several at once
  nself install nchat                # a whole bundle
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
