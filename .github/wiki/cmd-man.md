# nself man

> Generate man pages for all nself commands.

## Synopsis

```
nself man [flags]
```

## Description

`nself man` generates groff/troff man pages (section 1) for every command in the nSelf CLI tree and writes them to a local directory. By default the output directory is `./man/` relative to the current working directory. Pass `--output` to write to a different path.

The command walks the full Cobra command tree, including subcommands, and produces one `.1` file per command. Hidden commands are skipped. Each page follows the standard man page format: `NAME`, `SYNOPSIS`, `DESCRIPTION`, `OPTIONS`, and `SEE ALSO`. The `SEE ALSO` section links to the parent command's man page; top-level commands link to `nself(1)`.

File names use the hyphen-joined command path: `nself-plugin-install.1`, `nself-ai-studio-bridge.1`, and so on. The output directory is created if it does not exist.

After generation, copy the pages to your system's man path and rebuild the man database:

```bash
sudo cp man/*.1 /usr/local/share/man/man1/
mandb
```

Man pages are generated from live command metadata, so they reflect the exact flags, descriptions, and usage lines compiled into the current binary.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `man` | Directory to write man pages into |

## Examples

```bash
# Generate man pages into ./man/
nself man
```

```bash
# Write to a custom directory
nself man --output /tmp/nself-man
```

```bash
# Write directly to the system man path (requires write permission)
nself man --output /usr/local/share/man/man1
```

```bash
# Generate and install in one step
nself man && sudo cp man/*.1 /usr/local/share/man/man1/ && mandb
```

```bash
# Verify a specific page was written
nself man --output ./out && ls ./out/ | grep nself-build
```

```bash
# Use a short flag for output directory
nself man -o /usr/share/man/man1
```

## See Also

- [[cmd-build]] — command whose man page is generated as `nself-build.1`
- [[cmd-plugin]] — command whose man page is generated as `nself-plugin.1`
- [[cmd-doctor]] — command whose man page is generated as `nself-doctor.1`
- [[cmd-completion]] — generate shell completions (complements man pages)

← [[Commands]] | [[Home]] →
