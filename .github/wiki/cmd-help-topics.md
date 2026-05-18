# nself help-topics

> Browse built-in help topics for common nSelf tasks.

## Synopsis

```bash
nself help-topics [topic]
```

## Description

`nself help-topics` provides quick access to built-in guides covering common nSelf workflows. Running the command without a topic argument prints the full topic index. Pass a topic name to display the article for that topic.

This command is a local reference that works without a network connection. For the full documentation site, visit [docs.nself.org](https://docs.nself.org).

## Available Topics

| Topic | Summary |
|-------|---------|
| `quickstart` | Get a backend running in under 5 minutes |
| `plugins` | Install, manage, and build plugins |
| `license` | Manage license keys for paid plugin bundles |
| `envs` | Managing multiple environments (dev, staging, production) |
| `doctor` | Diagnose and fix common issues |
| `errors` | Error code reference (E001–E399) |

## Flags

This command has no flags.

## Examples

```bash
# Print the full topic index
nself help-topics
```

```bash
# Read the quickstart guide
nself help-topics quickstart
```

```bash
# Read the plugin system guide
nself help-topics plugins
```

```bash
# Read the license and bundles guide
nself help-topics license
```

```bash
# Read the environments guide
nself help-topics envs
```

```bash
# Read the diagnostics guide
nself help-topics doctor
```

```bash
# Read the error code reference
nself help-topics errors
```

## See Also

- [[cmd-doctor]] — run the nSelf diagnostics suite
- [[cmd-license]] — manage license keys
- [[cmd-plugin]] — manage the plugin catalog
- [[cmd-init]] — initialise a new nSelf project

← [[Commands]] | [[Home]] →
