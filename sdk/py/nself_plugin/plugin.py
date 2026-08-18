"""
Core Plugin class and related types.
Mirrors plugin.Plugin interface and plugin.Info struct in plugin-sdk-go.
"""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ._runner import PluginContext


@dataclass
class PluginInfo:
    """Identity metadata for a plugin — mirrors plugin-sdk-go plugin.Info."""
    name: str
    version: str
    description: str = ""
    tier: str = "free"  # "free" | "pro"
    bundle: str = ""
    min_cli: str = ""
    min_sdk: str = ""


@dataclass
class HealthStatus:
    """Health check result returned by the health hook."""
    status: str  # "ok" | "degraded" | "unavailable"
    message: str = ""
    checks: dict[str, bool] = field(default_factory=dict)


# Type aliases for hook callables
HookFn = Callable[["PluginContext"], Awaitable[None]]
HealthFn = Callable[["PluginContext"], Awaitable[HealthStatus]]


class Plugin:
    """
    Plugin is the primary authoring entry point for Python nSelf plugins.
    Mirrors the definePlugin() function in @nself/plugin-sdk (TypeScript) and
    the Plugin interface in plugin-sdk-go.

    Usage::

        plugin = Plugin(name="notify", version="1.0.0")

        @plugin.install
        async def install(ctx: PluginContext) -> None:
            ctx.env.require(["SMTP_HOST"])

        @plugin.start
        async def start(ctx: PluginContext) -> None:
            app = ctx.http.create_app()
            await ctx.http.listen(ctx.port)
    """

    def __init__(
        self,
        name: str,
        version: str,
        description: str = "",
        tier: str = "free",
        bundle: str = "",
    ) -> None:
        if not name:
            raise ValueError("plugin SDK: name is required")
        if not version:
            raise ValueError("plugin SDK: version is required")
        if tier not in ("free", "pro"):
            raise ValueError(f"plugin SDK: tier must be 'free' or 'pro', got {tier!r}")

        self.info = PluginInfo(
            name=name,
            version=version,
            description=description,
            tier=tier,
            bundle=bundle,
        )

        self._install: HookFn | None = None
        self._start: HookFn | None = None
        self._health: HealthFn | None = None
        self._migrate: HookFn | None = None
        self._uninstall: HookFn | None = None

    # Decorator-based hook registration
    def install(self, fn: HookFn) -> HookFn:
        """Register the install hook (runs on `nself plugin install`)."""
        self._install = fn
        return fn

    def start(self, fn: HookFn) -> HookFn:
        """Register the start hook (runs when `nself start` brings the container up)."""
        self._start = fn
        return fn

    def health(self, fn: HealthFn) -> HealthFn:
        """Register the health hook (polled by `nself doctor`)."""
        self._health = fn
        return fn

    def migrate(self, fn: HookFn) -> HookFn:
        """Register the migrate hook (runs Hasura + SQL migrations on start)."""
        self._migrate = fn
        return fn

    def uninstall(self, fn: HookFn) -> HookFn:
        """Register the uninstall hook (runs on `nself plugin remove`)."""
        self._uninstall = fn
        return fn
