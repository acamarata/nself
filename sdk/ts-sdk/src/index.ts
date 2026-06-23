/**
 * @nself/sdk — TypeScript consumer SDK for nSelf.
 *
 * Provides typed helpers for querying the nSelf GraphQL API, managing plugins,
 * and performing admin operations from any TypeScript or JavaScript application.
 *
 * @example
 * ```ts
 * import { NselfClient } from '@nself/sdk'
 *
 * const client = new NselfClient({ baseUrl: 'http://localhost:4000' })
 * const plugins = await client.plugins.list()
 * ```
 *
 * @module
 */

// Purpose: Consumer-facing SDK surface for nSelf API integration.
// Inputs: NselfClient options (baseUrl, adminSecret, timeout).
// Outputs: Typed responses from nSelf API endpoints.
// Constraints: Node 18+; no DOM dependency; pure fetch-based.
// SPORT: F13-CROSS-REPO-DEPS.md — SDK Module Consumption row for @nself/sdk.

/** Options for constructing an NselfClient. */
export interface NselfClientOptions {
  /** Base URL of the nSelf instance, e.g. "http://localhost:4000" */
  baseUrl: string
  /** Hasura admin secret for admin operations. Optional for public queries. */
  adminSecret?: string
  /** Request timeout in milliseconds. Default: 30000. */
  timeoutMs?: number
}

/** A plugin entry returned by the nSelf plugin list endpoint. */
export interface Plugin {
  name: string
  version: string
  status: 'running' | 'stopped' | 'error'
  tier: 'free' | 'pro'
}

/** Health status returned by the nSelf health endpoint. */
export interface HealthStatus {
  status: 'ok' | 'degraded'
  version: string
  uptime: number
}

/**
 * NselfClient is the top-level entry point for the @nself/sdk.
 * All operations are thin typed wrappers around the nSelf REST/GraphQL API.
 */
export class NselfClient {
  private readonly baseUrl: string
  private readonly adminSecret?: string
  private readonly timeoutMs: number

  constructor(options: NselfClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/$/, '')
    this.adminSecret = options.adminSecret
    this.timeoutMs = options.timeoutMs ?? 30_000
  }

  /** Checks the health of the connected nSelf instance. */
  async health(): Promise<HealthStatus> {
    const res = await this.fetch('/health')
    return res.json() as Promise<HealthStatus>
  }

  /** Lists all installed plugins. */
  async listPlugins(): Promise<Plugin[]> {
    const res = await this.fetch('/api/v1/plugins')
    return res.json() as Promise<Plugin[]>
  }

  private async fetch(path: string): Promise<Response> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }
    if (this.adminSecret) {
      headers['x-hasura-admin-secret'] = this.adminSecret
    }
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), this.timeoutMs)
    try {
      const res = await globalThis.fetch(`${this.baseUrl}${path}`, {
        headers,
        signal: controller.signal,
      })
      if (!res.ok) {
        throw new Error(`nSelf API error: ${res.status} ${res.statusText}`)
      }
      return res
    } finally {
      clearTimeout(timer)
    }
  }
}
