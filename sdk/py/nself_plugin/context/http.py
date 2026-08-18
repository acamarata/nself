"""
HTTP helper: FastAPI app factory + httpx health client.
Mirrors server.New() in plugin-sdk-go: auto-mounts /healthz, /readyz,
/version on every plugin server.
"""

from __future__ import annotations

import logging
from collections.abc import Callable
from typing import Any, cast

import httpx
from fastapi import FastAPI

from ..plugin import HealthStatus


class HttpHelper:
    """HTTP helper exposed in PluginContext."""

    def __init__(
        self,
        plugin_name: str,
        plugin_version: str,
        port: int,
        health_check: Callable[[], Any] | None = None,
    ) -> None:
        self._plugin_name = plugin_name
        self._plugin_version = plugin_version
        self._port = port
        self._health_check = health_check
        self._app: FastAPI | None = None
        self._logger = logging.getLogger(f"nself.{plugin_name}.http")

    def create_app(self) -> FastAPI:
        """
        Return a FastAPI application pre-configured with:
        - GET /healthz   → liveness (always 200)
        - GET /readyz    → delegates to health_check
        - GET /version   → plugin name + version JSON
        """
        if self._app is not None:
            return self._app

        app = FastAPI(title=self._plugin_name, version=self._plugin_version)

        plugin_name = self._plugin_name
        plugin_version = self._plugin_version
        health_check = self._health_check

        @app.get("/healthz")
        async def healthz() -> dict[str, str]:
            return {"status": "ok", "plugin": plugin_name, "version": plugin_version}

        @app.get("/readyz")
        async def readyz() -> dict[str, Any]:
            if health_check is not None:
                result = await health_check()
                if isinstance(result, HealthStatus):
                    return {"status": result.status, "message": result.message}
                return cast(dict[str, Any], result)
            return {"status": "ready"}

        @app.get("/version")
        async def version() -> dict[str, str]:
            return {
                "plugin": plugin_name,
                "version": plugin_version,
                "sdk": "nself-plugin",
            }

        self._app = app
        return app

    async def listen(self, port: int) -> None:
        """Start the uvicorn server on the given port. Blocks until shutdown."""
        import uvicorn

        app = self.create_app()
        config = uvicorn.Config(app, host="127.0.0.1", port=port, log_level="info")
        server = uvicorn.Server(config)
        self._logger.info("plugin listening", extra={"port": port})
        await server.serve()

    async def ping(self, path: str) -> HealthStatus:
        """
        Send GET to path on localhost and return a HealthStatus.
        Used in the health hook to check this plugin's own /healthz endpoint.
        """
        url = f"http://127.0.0.1:{self._port}{path}"
        try:
            async with httpx.AsyncClient() as client:
                res = await client.get(url, timeout=5.0)
            if res.is_success:
                return HealthStatus(status="ok")
            return HealthStatus(
                status="degraded",
                message=f"GET {path} returned HTTP {res.status_code}",
            )
        except Exception as exc:  # noqa: BLE001 — any failure to reach our own
            # endpoint means the plugin is unavailable; the specific exception
            # type is reported to the caller rather than acted on.
            return HealthStatus(status="unavailable", message=str(exc))
