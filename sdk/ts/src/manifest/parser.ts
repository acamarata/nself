/**
 * plugin.yaml manifest parser and validator.
 * The manifest is the schema the nSelf CLI reads to register a plugin.
 * Mirrors the manifest parser in plugin-sdk-go (config/config.go + plugin.Info).
 */

import { parse } from 'yaml'
import { readFileSync } from 'fs'
import { resolve } from 'path'

/** Full plugin manifest shape as declared in plugin.yaml. */
export interface PluginManifest {
  name: string
  version: string
  description?: string
  tier: 'free' | 'pro'
  bundle?: string
  minCli?: string
  minSdk?: string
  port?: number

  /** Hasura GraphQL routes provided by this plugin. */
  routes?: Array<{
    path: string
    upstream: string
  }>

  /** Env vars required at install time. */
  requiredEnv?: string[]

  /** Free-form metadata extension slot. */
  metadata?: Record<string, string>
}

/** Validation error returned when the manifest is malformed. */
export class ManifestValidationError extends Error {
  constructor(
    message: string,
    public readonly field?: string,
  ) {
    super(`plugin manifest: ${message}`)
    this.name = 'ManifestValidationError'
  }
}

/**
 * parseManifest reads and validates a plugin.yaml file.
 * Throws ManifestValidationError on missing required fields.
 *
 * @param filePath - path to plugin.yaml (defaults to ./plugin.yaml)
 */
export function parseManifest(filePath?: string): PluginManifest {
  const resolvedPath = resolve(filePath ?? 'plugin.yaml')
  let raw: string
  try {
    raw = readFileSync(resolvedPath, 'utf-8')
  } catch {
    throw new ManifestValidationError(`cannot read file: ${resolvedPath}`)
  }

  let parsed: unknown
  try {
    parsed = parse(raw)
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err)
    throw new ManifestValidationError(`YAML parse error: ${msg}`)
  }

  return validateManifest(parsed)
}

/**
 * parseManifestFromString parses plugin.yaml content from a string.
 * Useful in tests.
 */
export function parseManifestFromString(content: string): PluginManifest {
  let parsed: unknown
  try {
    parsed = parse(content)
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err)
    throw new ManifestValidationError(`YAML parse error: ${msg}`)
  }
  return validateManifest(parsed)
}

function validateManifest(raw: unknown): PluginManifest {
  if (typeof raw !== 'object' || raw === null) {
    throw new ManifestValidationError('manifest must be a YAML object')
  }

  const m = raw as Record<string, unknown>

  if (!m['name'] || typeof m['name'] !== 'string') {
    throw new ManifestValidationError('name is required and must be a string', 'name')
  }
  if (!m['version'] || typeof m['version'] !== 'string') {
    throw new ManifestValidationError('version is required and must be a string', 'version')
  }
  if (m['tier'] !== 'free' && m['tier'] !== 'pro') {
    throw new ManifestValidationError('tier must be "free" or "pro"', 'tier')
  }

  return m as unknown as PluginManifest
}
