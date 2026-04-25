/**
 * Env var helpers — mirrors config.Env / config.EnvRequired in plugin-sdk-go.
 */

/** Env helper interface exposed in PluginContext. */
export interface EnvHelper {
  /**
   * get returns the env var value or the default.
   * Missing + no default → undefined.
   */
  get(key: string, defaultValue?: string): string | undefined

  /**
   * require asserts that all named env vars are non-empty.
   * Throws on first missing var — fail-fast at install/start time.
   */
  require(keys: string[]): void

  /** getInt returns the env var parsed as an integer, or the default. */
  getInt(key: string, defaultValue: number): number

  /** getBool returns the env var parsed as boolean ("1"/"true"/"yes" → true). */
  getBool(key: string, defaultValue: boolean): boolean

  /** getList splits the env var on commas and trims whitespace. */
  getList(key: string): string[]
}

export function createEnvHelper(): EnvHelper {
  return {
    get(key: string, defaultValue?: string): string | undefined {
      const v = process.env[key]
      if (v !== undefined && v !== '') return v
      return defaultValue
    },

    require(keys: string[]): void {
      for (const key of keys) {
        if (!process.env[key] || process.env[key] === '') {
          throw new Error(`plugin SDK: required env var ${key} is not set`)
        }
      }
    },

    getInt(key: string, defaultValue: number): number {
      const v = process.env[key]
      if (!v) return defaultValue
      const n = parseInt(v, 10)
      return isNaN(n) ? defaultValue : n
    },

    getBool(key: string, defaultValue: boolean): boolean {
      const v = (process.env[key] ?? '').toLowerCase()
      if (['true', '1', 'yes', 'y', 'on'].includes(v)) return true
      if (['false', '0', 'no', 'n', 'off'].includes(v)) return false
      return defaultValue
    },

    getList(key: string): string[] {
      const v = process.env[key]
      if (!v) return []
      return v
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    },
  }
}
