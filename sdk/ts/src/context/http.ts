/**
 * HTTP helper: express app factory + health ping client.
 * Mirrors server.New() in plugin-sdk-go: auto-mounts /healthz, /readyz,
 * /version, /metrics on every plugin server.
 */

import express, { type Express, type Request, type Response } from 'express'
import type { HealthStatus } from '../runner'

export interface HttpHelper {
  /**
   * createApp returns an Express application pre-configured with:
   * - GET /healthz  → liveness (always 200)
   * - GET /readyz   → delegates to the health hook
   * - GET /version  → plugin name + version JSON
   */
  createApp(): Express

  /**
   * listen starts the HTTP server on the given port. Returns the net.Server
   * so callers can shut it down gracefully in the uninstall hook.
   */
  listen(port: number): ReturnType<Express['listen']>

  /**
   * ping sends GET to path on localhost:port and returns a HealthStatus.
   * Used in the health hook to check this plugin's own /healthz endpoint.
   */
  ping(path: string): Promise<HealthStatus>
}

export interface HttpHelperOptions {
  pluginName: string
  pluginVersion: string
  port: number
  healthCheck?: () => Promise<HealthStatus>
}

export function createHttpHelper(opts: HttpHelperOptions): HttpHelper {
  let app: Express | null = null

  function getOrCreateApp(): Express {
    if (app) return app

    app = express()
    app.use(express.json())

    // Liveness
    app.get('/healthz', (_req: Request, res: Response) => {
      res.json({ status: 'ok', plugin: opts.pluginName, version: opts.pluginVersion })
    })

    // Readiness — delegates to health hook if provided
    app.get('/readyz', async (_req: Request, res: Response) => {
      if (opts.healthCheck) {
        try {
          const result = await opts.healthCheck()
          const code = result.status === 'ok' ? 200 : 503
          res.status(code).json(result)
        } catch (err: unknown) {
          const message = err instanceof Error ? err.message : String(err)
          res.status(503).json({ status: 'unavailable', message })
        }
      } else {
        res.json({ status: 'ready' })
      }
    })

    // Version
    app.get('/version', (_req: Request, res: Response) => {
      res.json({
        plugin: opts.pluginName,
        version: opts.pluginVersion,
        sdk: '@nself/plugin-sdk',
      })
    })

    return app
  }

  return {
    createApp(): Express {
      return getOrCreateApp()
    },

    listen(port: number) {
      const application = getOrCreateApp()
      return application.listen(port)
    },

    async ping(path: string): Promise<HealthStatus> {
      const url = `http://127.0.0.1:${opts.port}${path}`
      try {
        const res = await fetch(url, { signal: AbortSignal.timeout(5000) })
        if (res.ok) {
          return { status: 'ok' }
        }
        return {
          status: 'degraded',
          message: `GET ${path} returned HTTP ${res.status}`,
        }
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err)
        return { status: 'unavailable', message }
      }
    },
  }
}
