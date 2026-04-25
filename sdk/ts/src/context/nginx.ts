/**
 * Nginx route injection helper.
 * nSelf generates nginx/conf.d/<plugin>.conf at build time; this helper
 * provides the typed interface for plugins to declare their routes.
 * At runtime the CLI reads plugin.yaml for routes — this helper is used
 * during install hooks to validate / extend the manifest-declared routes.
 */

export interface RouteOptions {
  /** Location path, e.g. "/notify/" */
  location: string
  /** Upstream host:port, e.g. "127.0.0.1:3830" */
  upstream: string
  /** Extra nginx directives injected inside the location block. */
  extraDirectives?: string[]
}

export interface NginxHelper {
  /** Register a proxy_pass route to the plugin's upstream. */
  addRoute(opts: RouteOptions): void

  /** Remove a previously registered route (used in uninstall). */
  removeRoute(location: string): void

  /** Returns all routes registered in this session. */
  getRoutes(): RouteOptions[]
}

export function createNginxHelper(): NginxHelper {
  const routes: RouteOptions[] = []

  return {
    addRoute(opts: RouteOptions): void {
      if (!opts.location.startsWith('/')) {
        throw new Error(`nginx: location must start with "/", got "${opts.location}"`)
      }
      routes.push(opts)
    },

    removeRoute(location: string): void {
      const idx = routes.findIndex((r) => r.location === location)
      if (idx !== -1) routes.splice(idx, 1)
    },

    getRoutes(): RouteOptions[] {
      return [...routes]
    },
  }
}
