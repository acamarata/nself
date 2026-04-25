"""
nself-plugin — Python plugin authoring SDK for nSelf.

Mirror of plugin-sdk-go v0.1.0: lifecycle hooks (install, start, health,
migrate, uninstall), PluginContext with env/hasura/nginx/db/http/logger
helpers, and a plugin.yaml manifest parser.

Example::

    from nself_plugin import Plugin, PluginContext, HealthStatus

    plugin = Plugin(name="my-notify", version="1.0.0")

    @plugin.install
    async def install(ctx: PluginContext) -> None:
        ctx.env.require(["SMTP_HOST", "SMTP_PORT"])

    @plugin.start
    async def start(ctx: PluginContext) -> None:
        app = ctx.http.create_app()
        app.get("/notify/healthz")(lambda: {"status": "ok"})
        await ctx.http.listen(ctx.port)

    @plugin.health
    async def health(ctx: PluginContext) -> HealthStatus:
        return await ctx.http.ping("/notify/healthz")
"""

from .plugin import Plugin, PluginInfo, HealthStatus
from .context.env import EnvHelper
from .context.hasura import HasuraHelper
from .context.nginx import NginxHelper
from .context.db import DatabaseHelper
from .context.http import HttpHelper
from ._manifest import PluginManifest, parse_manifest, parse_manifest_string, ManifestValidationError
from ._runner import PluginContext

__all__ = [
    "Plugin",
    "PluginInfo",
    "HealthStatus",
    "PluginContext",
    "EnvHelper",
    "HasuraHelper",
    "NginxHelper",
    "DatabaseHelper",
    "HttpHelper",
    "PluginManifest",
    "parse_manifest",
    "parse_manifest_string",
    "ManifestValidationError",
]

__version__ = "0.1.0"
