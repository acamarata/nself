# nself ai

**This command moved to a plugin.**

`nself ai` is no longer part of the CLI core. Talking to language models is not
part of the self-hosted backend lifecycle the core covers.

## Install

```bash
nself install ai-cli
```

The plugin is called `ai-cli` rather than `ai`, because the paid `ai` service
plugin already owns that name. The command is still `nself ai` — the CLI proxies
it to the plugin.

## Commands

```bash
nself ai chat
nself ai local health
nself ai local models
nself ai pool status
nself ai pool rotate
```

## Related

`nself doctor` keeps its AI checks and will suggest the relevant `nself ai`
commands once the plugin is installed.

---

← [[Commands]] · [[Plugin-Overview]] · [[Home]]
