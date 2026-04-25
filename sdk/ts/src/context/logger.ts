/**
 * Structured JSON logger interface and default pino implementation.
 * Mirrors logger.New() in plugin-sdk-go: every record carries plugin + version.
 */

/** Logger interface exposed in PluginContext — mirrors slog.Logger API shape. */
export interface Logger {
  info(msg: string, attrs?: Record<string, unknown>): void
  warn(msg: string, attrs?: Record<string, unknown>): void
  error(msg: string, attrs?: Record<string, unknown>): void
  debug(msg: string, attrs?: Record<string, unknown>): void
}

/** Factory — creates a structured JSON logger keyed to a plugin name. */
export function createLogger(plugin: string, version: string): Logger {
  const base: Record<string, unknown> = { plugin, version }

  function log(level: string, msg: string, attrs?: Record<string, unknown>): void {
    const entry = {
      time: new Date().toISOString(),
      level,
      msg,
      ...base,
      ...(attrs ?? {}),
    }
    process.stderr.write(JSON.stringify(entry) + '\n')
  }

  return {
    info: (msg, attrs) => log('INFO', msg, attrs),
    warn: (msg, attrs) => log('WARN', msg, attrs),
    error: (msg, attrs) => log('ERROR', msg, attrs),
    debug: (msg, attrs) => log('DEBUG', msg, attrs),
  }
}
