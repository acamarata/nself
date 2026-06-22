/**
 * NselfClient happy-path tests using a mock fetch.
 *
 * Purpose: Verify that NselfClient correctly maps API responses to typed objects
 *          and handles common error cases.
 * Inputs: Mock globalThis.fetch responses.
 * Outputs: Pass/fail assertions on NselfClient method return values.
 * Constraints: No network I/O. Node 18+ fetch global is mocked.
 * SPORT: F13-CROSS-REPO-DEPS.md — @nself/sdk consumer SDK.
 */

import { NselfClient, type HealthStatus, type Plugin } from './index'

const mockFetch = jest.fn()

beforeEach(() => {
  jest.resetAllMocks()
  // Replace globalThis.fetch with our mock
  ;(globalThis as unknown as { fetch: typeof mockFetch }).fetch = mockFetch
})

function makeResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? 'OK' : 'Error',
    json: () => Promise.resolve(body),
  } as unknown as Response
}

describe('NselfClient', () => {
  const client = new NselfClient({ baseUrl: 'http://localhost:4000' })

  describe('health()', () => {
    it('returns HealthStatus on 200', async () => {
      const payload: HealthStatus = { status: 'ok', version: '1.1.9', uptime: 42 }
      mockFetch.mockResolvedValueOnce(makeResponse(payload))

      const result = await client.health()
      expect(result.status).toBe('ok')
      expect(result.version).toBe('1.1.9')
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:4000/health',
        expect.objectContaining({ headers: expect.any(Object) }),
      )
    })

    it('throws on non-2xx response', async () => {
      mockFetch.mockResolvedValueOnce(makeResponse({}, 503))
      await expect(client.health()).rejects.toThrow('nSelf API error: 503')
    })
  })

  describe('listPlugins()', () => {
    it('returns Plugin array on 200', async () => {
      const payload: Plugin[] = [
        { name: 'ai', version: '1.1.9', status: 'running', tier: 'free' },
        { name: 'claw', version: '1.1.9', status: 'running', tier: 'pro' },
      ]
      mockFetch.mockResolvedValueOnce(makeResponse(payload))

      const plugins = await client.listPlugins()
      expect(plugins).toHaveLength(2)
      expect(plugins[0].name).toBe('ai')
      expect(plugins[1].tier).toBe('pro')
    })
  })

  describe('adminSecret header', () => {
    it('sends x-hasura-admin-secret when provided', async () => {
      const adminClient = new NselfClient({
        baseUrl: 'http://localhost:4000',
        adminSecret: 'secret123',
      })
      mockFetch.mockResolvedValueOnce(makeResponse({ status: 'ok', version: '1.1.9', uptime: 0 }))
      await adminClient.health()

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({ 'x-hasura-admin-secret': 'secret123' }),
        }),
      )
    })

    it('omits x-hasura-admin-secret when not provided', async () => {
      mockFetch.mockResolvedValueOnce(makeResponse({ status: 'ok', version: '1.1.9', uptime: 0 }))
      await client.health()

      const callHeaders = mockFetch.mock.calls[0][1].headers
      expect(callHeaders['x-hasura-admin-secret']).toBeUndefined()
    })
  })

  describe('baseUrl normalization', () => {
    it('strips trailing slash from baseUrl', async () => {
      const slashClient = new NselfClient({ baseUrl: 'http://localhost:4000/' })
      mockFetch.mockResolvedValueOnce(makeResponse({ status: 'ok', version: '1.1.9', uptime: 0 }))
      await slashClient.health()

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:4000/health',
        expect.anything(),
      )
    })
  })
})
