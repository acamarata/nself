import { definePlugin } from '../src/runner'

describe('definePlugin', () => {
  it('returns the definition unchanged when valid', () => {
    const def = definePlugin({ name: 'my-plugin', version: '1.0.0' })
    expect(def.name).toBe('my-plugin')
    expect(def.version).toBe('1.0.0')
  })

  it('throws when name is missing', () => {
    expect(() => definePlugin({ name: '', version: '1.0.0' })).toThrow('name is required')
  })

  it('throws when version is missing', () => {
    expect(() => definePlugin({ name: 'x', version: '' })).toThrow('version is required')
  })

  it('accepts all optional hooks without error', () => {
    const def = definePlugin({
      name: 'full',
      version: '2.0.0',
      description: 'Full plugin',
      tier: 'pro',
      bundle: 'nClaw',
      async install() {},
      async start() {},
      async health() { return { status: 'ok' } },
      async migrate() {},
      async uninstall() {},
    })
    expect(def.tier).toBe('pro')
    expect(def.bundle).toBe('nClaw')
  })
})
