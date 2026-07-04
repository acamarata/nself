/**
 * @nself/sdk — TypeScript consumer SDK for a running nSelf instance.
 *
 * Purpose: Thin typed HTTP client over a local nSelf backend's admin API
 *          (health checks, plugin status). Not a GraphQL client — for
 *          Hasura GraphQL access, use @nself/graphql-client instead.
 * Inputs: NselfClientOptions (baseUrl + optional adminSecret).
 * Outputs: Typed HealthStatus / Plugin results; throws on non-2xx responses.
 * Constraints: Requires a global `fetch` (Node 18+ or browser).
 * SPORT: F13-CROSS-REPO-DEPS.md — @nself/sdk consumer SDK.
 */

export interface NselfClientOptions {
  baseUrl: string
  adminSecret?: string
}

export interface HealthStatus {
  status: string
  version: string
  uptime: number
}

export interface Plugin {
  name: string
  version: string
  status: string
  tier: string
}

export class NselfClient {
  private readonly baseUrl: string
  private readonly adminSecret?: string

  constructor(options: NselfClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, '')
    this.adminSecret = options.adminSecret
  }

  private buildHeaders(): Record<string, string> {
    const headers: Record<string, string> = { 'content-type': 'application/json' }
    if (this.adminSecret) {
      headers['x-hasura-admin-secret'] = this.adminSecret
    }
    return headers
  }

  private async request<T>(path: string): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, { headers: this.buildHeaders() })
    if (!response.ok) {
      throw new Error(`nSelf API error: ${response.status}`)
    }
    return response.json() as Promise<T>
  }

  async health(): Promise<HealthStatus> {
    return this.request<HealthStatus>('/health')
  }

  async listPlugins(): Promise<Plugin[]> {
    return this.request<Plugin[]>('/plugins')
  }
}
