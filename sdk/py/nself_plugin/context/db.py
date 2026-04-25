"""
PostgreSQL helper — migration-time DDL only.
Mirrors db.Pool in plugin-sdk-go: wraps asyncpg, exposes run_sql().
NOT for query-time use — plugins must talk to Hasura GraphQL at runtime.
asyncpg is an optional dependency (extras = ["db"]).
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass
class SqlResult:
    row_count: int
    rows: list[dict[str, Any]]


class DatabaseHelper:
    """
    PostgreSQL helper exposed in PluginContext.
    Use only in install/migrate/uninstall hooks, never at request time.
    """

    def __init__(self, database_url: str) -> None:
        self._database_url = database_url
        self._pool: Any = None  # asyncpg.Pool, typed as Any to avoid hard dep

    async def _get_pool(self) -> Any:
        if self._pool is not None:
            return self._pool
        try:
            import asyncpg  # type: ignore[import-untyped]
        except ImportError as exc:
            raise RuntimeError(
                "plugin SDK: asyncpg is required for database operations. "
                "Install with: pip install 'nself-plugin[db]'"
            ) from exc
        self._pool = await asyncpg.create_pool(self._database_url)
        return self._pool

    async def run_sql(self, sql: str, *params: Any) -> SqlResult:
        """Execute raw SQL. Migration-time only."""
        pool = await self._get_pool()
        async with pool.acquire() as conn:
            result = await conn.fetch(sql, *params)
            rows = [dict(r) for r in result]
        return SqlResult(row_count=len(rows), rows=rows)

    async def close(self) -> None:
        """Close the connection pool. Called by the runner on shutdown."""
        if self._pool is not None:
            await self._pool.close()
            self._pool = None


class DatabaseHelperStub(DatabaseHelper):
    """
    No-op stub for unit tests — no real DB required.
    Usage::

        ctx = PluginContext(..., db=DatabaseHelperStub())
    """

    def __init__(self) -> None:
        super().__init__(database_url="")

    async def run_sql(self, sql: str, *params: Any) -> SqlResult:
        return SqlResult(row_count=0, rows=[])

    async def close(self) -> None:
        pass
