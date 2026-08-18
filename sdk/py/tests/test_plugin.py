"""Tests for Plugin class — lifecycle hook registration and validation."""

import pytest

from nself_plugin import HealthStatus, Plugin, PluginContext
from nself_plugin.context.db import DatabaseHelperStub
from nself_plugin.context.env import EnvHelper
from nself_plugin.context.http import HttpHelper
from nself_plugin.context.nginx import NginxHelper


def make_ctx(port: int = 3850) -> PluginContext:
    """Build a minimal PluginContext for testing."""
    import logging
    return PluginContext(
        port=port,
        logger=logging.getLogger("test"),
        env=EnvHelper(),
        hasura=None,  # type: ignore[arg-type]
        nginx=NginxHelper(),
        db=DatabaseHelperStub(),
        http=HttpHelper("test-plugin", "1.0.0", port),
    )


def test_plugin_stores_info() -> None:
    p = Plugin(name="test", version="1.0.0", tier="free")
    assert p.info.name == "test"
    assert p.info.version == "1.0.0"
    assert p.info.tier == "free"


def test_plugin_requires_name() -> None:
    with pytest.raises(ValueError, match="name is required"):
        Plugin(name="", version="1.0.0")


def test_plugin_requires_version() -> None:
    with pytest.raises(ValueError, match="version is required"):
        Plugin(name="test", version="")


def test_plugin_rejects_invalid_tier() -> None:
    with pytest.raises(ValueError, match="tier"):
        Plugin(name="test", version="1.0.0", tier="enterprise")


async def test_install_hook_called() -> None:
    plugin = Plugin(name="notify", version="1.0.0")
    called: list[bool] = []

    @plugin.install
    async def install(ctx: PluginContext) -> None:
        called.append(True)

    ctx = make_ctx()
    assert plugin._install is not None
    await plugin._install(ctx)
    assert called == [True]


async def test_health_hook_returns_status() -> None:
    plugin = Plugin(name="notify", version="1.0.0")

    @plugin.health
    async def health(ctx: PluginContext) -> HealthStatus:
        return HealthStatus(status="ok")

    ctx = make_ctx()
    assert plugin._health is not None
    result = await plugin._health(ctx)
    assert result.status == "ok"


async def test_all_hooks_can_be_registered() -> None:
    plugin = Plugin(name="full", version="2.0.0", tier="pro", bundle="nClaw")

    @plugin.install
    async def install(ctx: PluginContext) -> None: pass

    @plugin.start
    async def start(ctx: PluginContext) -> None: pass

    @plugin.health
    async def health(ctx: PluginContext) -> HealthStatus:
        return HealthStatus(status="ok")

    @plugin.migrate
    async def migrate(ctx: PluginContext) -> None: pass

    @plugin.uninstall
    async def uninstall(ctx: PluginContext) -> None: pass

    assert plugin._install is not None
    assert plugin._start is not None
    assert plugin._health is not None
    assert plugin._migrate is not None
    assert plugin._uninstall is not None
    assert plugin.info.bundle == "nClaw"
