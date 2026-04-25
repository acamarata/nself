import { createEnvHelper } from '../src/context/env'

describe('EnvHelper', () => {
  const originalEnv = process.env

  beforeEach(() => {
    process.env = { ...originalEnv }
  })

  afterEach(() => {
    process.env = originalEnv
  })

  it('get returns env var value', () => {
    process.env['TEST_VAR'] = 'hello'
    const env = createEnvHelper()
    expect(env.get('TEST_VAR')).toBe('hello')
  })

  it('get returns default when var is missing', () => {
    const env = createEnvHelper()
    expect(env.get('MISSING_VAR', 'default')).toBe('default')
  })

  it('require throws on missing var', () => {
    const env = createEnvHelper()
    delete process.env['REQUIRED_VAR']
    expect(() => env.require(['REQUIRED_VAR'])).toThrow('REQUIRED_VAR')
  })

  it('require passes when all vars are set', () => {
    process.env['VAR_A'] = 'a'
    process.env['VAR_B'] = 'b'
    const env = createEnvHelper()
    expect(() => env.require(['VAR_A', 'VAR_B'])).not.toThrow()
  })

  it('getInt parses integers', () => {
    process.env['MY_PORT'] = '3850'
    const env = createEnvHelper()
    expect(env.getInt('MY_PORT', 3000)).toBe(3850)
  })

  it('getInt returns default on non-numeric', () => {
    process.env['MY_PORT'] = 'abc'
    const env = createEnvHelper()
    expect(env.getInt('MY_PORT', 3000)).toBe(3000)
  })

  it('getBool parses true variants', () => {
    const env = createEnvHelper()
    for (const v of ['true', '1', 'yes', 'y', 'on']) {
      process.env['BOOL_VAR'] = v
      expect(env.getBool('BOOL_VAR', false)).toBe(true)
    }
  })

  it('getList splits on commas', () => {
    process.env['LIST_VAR'] = 'a, b, c'
    const env = createEnvHelper()
    expect(env.getList('LIST_VAR')).toEqual(['a', 'b', 'c'])
  })
})
