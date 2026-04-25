/**
 * @nself/plugin-sdk — TypeScript plugin authoring SDK for nSelf.
 *
 * Mirror of plugin-sdk-go v0.1.0: lifecycle hooks (install, start, health,
 * migrate, uninstall), PluginContext with env/hasura/nginx/db/http/logger
 * helpers, and a plugin.yaml manifest parser.
 *
 * @example
 * ```ts
 * import { definePlugin, type PluginContext } from '@nself/plugin-sdk'
 *
 * export default definePlugin({
 *   name: 'my-notify',
 *   version: '1.0.0',
 *
 *   async install(ctx: PluginContext) {
 *     await ctx.env.require(['SMTP_HOST', 'SMTP_PORT'])
 *   },
 *
 *   async start(ctx: PluginContext) {
 *     const app = ctx.http.createApp()
 *     app.get('/notify/healthz', (_req, res) => res.json({ status: 'ok' }))
 *     ctx.http.listen(ctx.port)
 *   },
 *
 *   async health(ctx: PluginContext) {
 *     return ctx.http.ping('/notify/healthz')
 *   },
 * })
 * ```
 */

export { definePlugin } from './runner'
export type {
  PluginDefinition,
  PluginContext,
  HealthStatus,
  PluginInfo,
} from './runner'
export type { EnvHelper } from './context/env'
export type { HasuraHelper } from './context/hasura'
export type { NginxHelper } from './context/nginx'
export type { DatabaseHelper } from './context/db'
export type { HttpHelper } from './context/http'
export type { Logger } from './context/logger'
export { parseManifest, type PluginManifest } from './manifest/parser'
