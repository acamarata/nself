import { parseManifestFromString, ManifestValidationError } from '../src/manifest/parser'

const VALID_YAML = `
name: test-plugin
version: 1.0.0
description: A test plugin
tier: free
port: 3850
requiredEnv:
  - DATABASE_URL
`

const PRO_YAML = `
name: claw-plugin
version: 2.1.0
tier: pro
bundle: nClaw
minCli: "1.0.0"
`

describe('parseManifestFromString', () => {
  it('parses a valid free plugin manifest', () => {
    const m = parseManifestFromString(VALID_YAML)
    expect(m.name).toBe('test-plugin')
    expect(m.version).toBe('1.0.0')
    expect(m.tier).toBe('free')
    expect(m.port).toBe(3850)
    expect(m.requiredEnv).toEqual(['DATABASE_URL'])
  })

  it('parses a valid pro plugin manifest', () => {
    const m = parseManifestFromString(PRO_YAML)
    expect(m.name).toBe('claw-plugin')
    expect(m.tier).toBe('pro')
    expect(m.bundle).toBe('nClaw')
  })

  it('throws on missing name', () => {
    expect(() =>
      parseManifestFromString('version: 1.0.0\ntier: free\n'),
    ).toThrow(ManifestValidationError)
  })

  it('throws on missing version', () => {
    expect(() =>
      parseManifestFromString('name: x\ntier: free\n'),
    ).toThrow(ManifestValidationError)
  })

  it('throws on invalid tier', () => {
    expect(() =>
      parseManifestFromString('name: x\nversion: 1.0.0\ntier: enterprise\n'),
    ).toThrow(ManifestValidationError)
  })

  it('throws on invalid YAML', () => {
    expect(() => parseManifestFromString('name: [unclosed')).toThrow(ManifestValidationError)
  })
})
