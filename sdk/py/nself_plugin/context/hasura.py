"""
Hasura metadata API wrapper for install/migrate/uninstall hooks.
Mirrors HasuraHelper in the spec. Uses Hasura Metadata API v3 via httpx.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any

import httpx


@dataclass
class TrackRelationshipOptions:
    from_table: str
    name: str
    type: str  # "object" | "array"
    to_table: str
    from_column: str
    to_column: str


class HasuraHelper:
    """
    Hasura metadata API wrapper.
    Async methods — use in async hooks (install, migrate, uninstall).
    """

    def __init__(self, admin_secret: str, endpoint: str) -> None:
        self._admin_secret = admin_secret
        self._endpoint = endpoint.rstrip("/")

    async def _post(self, path: str, body: dict[str, Any]) -> Any:
        url = f"{self._endpoint}{path}"
        async with httpx.AsyncClient() as client:
            res = await client.post(
                url,
                json=body,
                headers={
                    "Content-Type": "application/json",
                    "x-hasura-admin-secret": self._admin_secret,
                },
                timeout=30,
            )
        if not res.is_success:
            raise RuntimeError(
                f"hasura: POST {path} failed ({res.status_code}): {res.text}"
            )
        return res.json()

    async def add_table(self, table: str, schema: str = "public") -> None:
        """Track a table in Hasura metadata."""
        await self._post("/v1/metadata", {
            "type": "pg_track_table",
            "args": {"source": "default", "schema": schema, "name": table},
        })

    async def remove_table(self, table: str, schema: str = "public") -> None:
        """Untrack a table from Hasura metadata."""
        await self._post("/v1/metadata", {
            "type": "pg_untrack_table",
            "args": {"table": {"schema": schema, "name": table}, "cascade": False},
        })

    async def track_relationship(self, opts: TrackRelationshipOptions) -> None:
        """Add a relationship between two tracked tables."""
        api_type = (
            "pg_create_object_relationship"
            if opts.type == "object"
            else "pg_create_array_relationship"
        )
        await self._post("/v1/metadata", {
            "type": api_type,
            "args": {
                "source": "default",
                "table": opts.from_table,
                "name": opts.name,
                "using": {
                    "foreign_key_constraint_on": {
                        "table": opts.to_table,
                        "columns": [opts.to_column],
                    }
                },
            },
        })

    async def run_metadata(self, type_: str, args: dict[str, Any]) -> Any:
        """Raw Hasura metadata API request (escape hatch)."""
        return await self._post("/v1/metadata", {"type": type_, "args": args})

    async def run_sql(self, sql: str) -> None:
        """Run a SQL migration via Hasura's run_sql API."""
        await self._post("/v2/query", {
            "type": "run_sql",
            "args": {"source": "default", "sql": sql},
        })
