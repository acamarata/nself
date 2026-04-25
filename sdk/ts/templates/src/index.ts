import { definePlugin, type PluginContext, type HealthStatus } from '@nself/plugin-sdk'

export default definePlugin({
  name: 'my-plugin',
  version: '1.0.0',

  async install(ctx: PluginContext): Promise<void> {
    ctx.logger.info('installing my-plugin')
    // Verify required env vars
    ctx.env.require([])
    // Register Nginx route
    ctx.nginx.addRoute({
      location: '/my-plugin/',
      upstream: `127.0.0.1:${ctx.port}`,
    })
  },

  async start(ctx: PluginContext): Promise<void> {
    ctx.logger.info('starting my-plugin', { port: ctx.port })
    const app = ctx.http.createApp()

    // Add your routes here
    app.get('/my-plugin/hello', (_req, res) => {
      res.json({ message: 'Hello from my-plugin!' })
    })

    ctx.http.listen(ctx.port)
  },

  async health(ctx: PluginContext): Promise<HealthStatus> {
    return ctx.http.ping('/my-plugin/healthz')
  },

  async migrate(_ctx: PluginContext): Promise<void> {
    // Run Hasura metadata or SQL migrations here
  },

  async uninstall(ctx: PluginContext): Promise<void> {
    ctx.logger.info('uninstalling my-plugin')
    ctx.nginx.removeRoute('/my-plugin/')
  },
})
