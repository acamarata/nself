# nself-plugin

Python plugin authoring SDK for [nSelf](https://nself.org). Parity with
[plugin-sdk-go](https://github.com/nself-org/plugin-sdk-go) v0.1.0.

## Install

```bash
pip install nself-plugin
```

For database migrations:

```bash
pip install 'nself-plugin[db]'
```

## Quick start

```python
from nself_plugin import Plugin, PluginContext, HealthStatus

plugin = Plugin(name="my-notify", version="1.0.0")

@plugin.install
async def install(ctx: PluginContext) -> None:
    ctx.env.require(["SMTP_HOST", "SMTP_PORT"])

@plugin.start
async def start(ctx: PluginContext) -> None:
    app = ctx.http.create_app()
    app.post("/notify/send")(handler)
    await ctx.http.listen(ctx.port)

@plugin.health
async def health(ctx: PluginContext) -> HealthStatus:
    return await ctx.http.ping("/notify/healthz")
```

## Scaffold a plugin

```bash
nself plugin new my-plugin --lang python
```

## License

MIT
