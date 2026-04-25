/**
 * Lifecycle orchestrator: defines PluginDefinition, PluginContext, and the
 * definePlugin() factory. Mirrors the plugin.Plugin interface in plugin-sdk-go.
 */

import type { EnvHelper } from './context/env'
import type { HasuraHelper } from './context/hasura'
import type { NginxHelper } from './context/nginx'
import type { DatabaseHelper } from './context/db'
import type { HttpHelper } from './context/http'
import type { Logger } from './context/logger'

/** Identity metadata for a plugin — mirrors plugin-sdk-go plugin.Info. */
export interface PluginInfo {
  name: string
  version: string
  description?: string
  tier?: 'free' | 'pro'
  bundle?: string
  minCli?: string
  minSdk?: string
}

/** Health check result returned by the health hook. */
export interface HealthStatus {
  status: 'ok' | 'degraded' | 'unavailable'
  message?: string
  checks?: Record<string, boolean>
}

/**
 * Runtime context passed to every lifecycle hook.
 * Mirrors the PluginContext struct in the spec and the helpers in plugin-sdk-go.
 */
export interface PluginContext {
  /** TCP port the plugin should bind to (from NSELF_PLUGIN_PORT env var). */
  port: number
  /** Structured JSON logger keyed to the plugin name. */
  logger: Logger
  /** Env var helpers: require(), get(), getInt(), getBool(). */
  env: EnvHelper
  /** Hasura metadata API wrapper. */
  hasura: HasuraHelper
  /** Nginx route injection helper. */
  nginx: NginxHelper
  /**
   * PostgreSQL helper for migration-time DDL only.
   * Not for query-time use — plugins talk to Hasura GraphQL.
   */
  db: DatabaseHelper
  /** Express app factory + health ping client. */
  http: HttpHelper
}

/**
 * The shape a plugin author fills in with definePlugin().
 * All hooks are optional except name + version.
 */
export interface PluginDefinition {
  /** Plugin name — must match the plugin.yaml `name` field. */
  name: string
  /** SemVer string, e.g. "1.0.0". */
  version: string
  /** Short human description. */
  description?: string
  /** Tier reported in /version endpoint. */
  tier?: 'free' | 'pro'
  /** Bundle this plugin belongs to, e.g. "nClaw". */
  bundle?: string

  /**
   * install — runs on `nself plugin install`.
   * Use to apply Hasura metadata, verify env vars, and register Nginx routes.
   */
  install?: (ctx: PluginContext) => Promise<void>

  /**
   * start — runs when `nself start` brings the plugin container up.
   * Should create an HTTP app and call ctx.http.listen(ctx.port).
   */
  start?: (ctx: PluginContext) => Promise<void>

  /**
   * health — polled by `nself doctor`. Return status 'ok' when fully ready.
   * The runner builds a /healthz endpoint automatically; health() is used for
   * deeper readiness checks surfaced via /readyz.
   */
  health?: (ctx: PluginContext) => Promise<HealthStatus>

  /**
   * migrate — runs Hasura metadata + SQL migrations during `nself start` before
   * the plugin begins serving traffic.
   */
  migrate?: (ctx: PluginContext) => Promise<void>

  /**
   * uninstall — runs on `nself plugin remove`.
   * Clean up Hasura metadata and Nginx routes; preserve user data by default.
   */
  uninstall?: (ctx: PluginContext) => Promise<void>
}

/**
 * definePlugin registers a plugin definition and returns it for use by the
 * nSelf CLI runner. Acts as the primary authoring entry point.
 *
 * @example
 * ```ts
 * export default definePlugin({
 *   name: 'notify',
 *   version: '1.0.0',
 *   async start(ctx) {
 *     const app = ctx.http.createApp()
 *     app.post('/notify/send', myHandler)
 *     ctx.http.listen(ctx.port)
 *   },
 * })
 * ```
 */
export function definePlugin(definition: PluginDefinition): PluginDefinition {
  if (!definition.name) throw new Error('plugin SDK: name is required')
  if (!definition.version) throw new Error('plugin SDK: version is required')
  return definition
}
