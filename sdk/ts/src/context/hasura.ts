/**
 * Hasura metadata API wrapper for install/migrate/uninstall hooks.
 * Mirrors the HasuraHelper concept from the spec: addTable, trackRelationship,
 * runMigration. Uses the Hasura Metadata API v3.
 */

export interface TrackRelationshipOptions {
  fromTable: string
  name: string
  type: 'object' | 'array'
  toTable: string
  fromColumn: string
  toColumn: string
}

export interface HasuraHelper {
  /** Track a new table in Hasura metadata. */
  addTable(schema: string, table: string): Promise<void>

  /** Untrack a table from Hasura metadata. */
  removeTable(schema: string, table: string): Promise<void>

  /** Add a relationship between two tracked tables. */
  trackRelationship(opts: TrackRelationshipOptions): Promise<void>

  /** Run raw Hasura metadata API request (escape hatch). */
  runMetadata(type: string, args: Record<string, unknown>): Promise<unknown>

  /** Run a SQL migration via Hasura's run_sql API. */
  runSql(sql: string): Promise<void>
}

export function createHasuraHelper(adminSecret: string, endpoint: string): HasuraHelper {
  async function post(path: string, body: unknown): Promise<unknown> {
    const url = `${endpoint}${path}`
    const res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-hasura-admin-secret': adminSecret,
      },
      body: JSON.stringify(body),
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(`hasura: POST ${path} failed (${res.status}): ${text}`)
    }
    return res.json()
  }

  return {
    async addTable(schema: string, table: string): Promise<void> {
      await post('/v1/metadata', {
        type: 'pg_track_table',
        args: {
          source: 'default',
          schema,
          name: table,
        },
      })
    },

    async removeTable(schema: string, table: string): Promise<void> {
      await post('/v1/metadata', {
        type: 'pg_untrack_table',
        args: {
          table: { schema, name: table },
          cascade: false,
        },
      })
    },

    async trackRelationship(opts: TrackRelationshipOptions): Promise<void> {
      const apiType =
        opts.type === 'object' ? 'pg_create_object_relationship' : 'pg_create_array_relationship'
      await post('/v1/metadata', {
        type: apiType,
        args: {
          source: 'default',
          table: opts.fromTable,
          name: opts.name,
          using: {
            foreign_key_constraint_on: {
              table: opts.toTable,
              columns: [opts.toColumn],
            },
          },
        },
      })
    },

    async runMetadata(type: string, args: Record<string, unknown>): Promise<unknown> {
      return post('/v1/metadata', { type, args })
    },

    async runSql(sql: string): Promise<void> {
      await post('/v2/query', {
        type: 'run_sql',
        args: { source: 'default', sql },
      })
    },
  }
}
