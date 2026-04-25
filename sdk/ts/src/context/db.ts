/**
 * PostgreSQL helper — migration-time DDL only.
 * Mirrors db.Pool in plugin-sdk-go: wraps pg driver, exposes runSQL().
 * NOT for query-time use — plugins must talk to Hasura GraphQL at runtime.
 */

/** Minimal row result from a migration query. */
export interface SqlResult {
  rowCount: number
  rows: Record<string, unknown>[]
}

export interface DatabaseHelper {
  /**
   * runSQL executes raw SQL. Use only in install/migrate/uninstall hooks.
   * Never call during request handling — that goes through Hasura.
   */
  runSQL(sql: string, params?: unknown[]): Promise<SqlResult>

  /** Close the underlying connection pool. Called automatically by the runner. */
  close(): Promise<void>
}

export interface DatabaseHelperOptions {
  databaseUrl: string
}

export async function createDatabaseHelper(opts: DatabaseHelperOptions): Promise<DatabaseHelper> {
  // Dynamic import so pg remains a peer dep (optional at install time).
  // Plugins that don't run migrations never need pg installed.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Pool } = require('pg') as typeof import('pg')
  const pool = new Pool({ connectionString: opts.databaseUrl })

  return {
    async runSQL(sql: string, params?: unknown[]): Promise<SqlResult> {
      const res = await pool.query(sql, params)
      return {
        rowCount: res.rowCount ?? 0,
        rows: res.rows as Record<string, unknown>[],
      }
    },

    async close(): Promise<void> {
      await pool.end()
    },
  }
}

/** Stub implementation for use in unit tests — no real DB needed. */
export function createDatabaseHelperStub(): DatabaseHelper {
  return {
    async runSQL(_sql, _params) {
      return { rowCount: 0, rows: [] }
    },
    async close() {},
  }
}
