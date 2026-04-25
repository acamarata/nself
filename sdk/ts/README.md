# @nself/plugin-sdk

TypeScript/Node.js plugin authoring SDK for [nSelf](https://nself.org). Parity with
[plugin-sdk-go](https://github.com/nself-org/plugin-sdk-go) v0.1.0.

## Install

```bash
pnpm add @nself/plugin-sdk
```

## Quick start

```ts
import { definePlugin, type PluginContext } from '@nself/plugin-sdk'

export default definePlugin({
  name: 'my-notify',
  version: '1.0.0',

  async install(ctx: PluginContext) {
    await ctx.env.require(['SMTP_HOST', 'SMTP_PORT'])
  },

  async start(ctx: PluginContext) {
    const app = ctx.http.createApp()
    app.post('/notify/send', myHandler)
    ctx.http.listen(ctx.port)
  },

  async health(ctx: PluginContext) {
    return ctx.http.ping('/notify/healthz')
  },
})
```

## Scaffold a plugin

```bash
nself plugin new my-plugin --lang typescript
```

## License

MIT
