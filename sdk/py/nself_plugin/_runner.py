"""
Lifecycle event orchestrator and PluginContext dataclass.
Mirrors the runner.go concept in plugin-sdk-go.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass

from .context.db import DatabaseHelper
from .context.env import EnvHelper
from .context.hasura import HasuraHelper
from .context.http import HttpHelper
from .context.nginx import NginxHelper


@dataclass
class PluginContext:
    """
    Runtime context passed to every lifecycle hook.
    Mirrors PluginContext in the spec and the helpers in plugin-sdk-go.
    """

    # TCP port the plugin should bind to (from NSELF_PLUGIN_PORT env var).
    port: int

    # Structured logger keyed to the plugin name.
    logger: logging.Logger

    # Env var helpers: require(), get(), get_int(), get_bool().
    env: EnvHelper

    # Hasura metadata API wrapper.
    hasura: HasuraHelper

    # Nginx route injection helper.
    nginx: NginxHelper

    # PostgreSQL helper for migration-time DDL only.
    # Not for query-time use — plugins talk to Hasura GraphQL.
    db: DatabaseHelper

    # FastAPI app factory + httpx health client.
    http: HttpHelper
