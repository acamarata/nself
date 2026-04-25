"""
Env var helpers — mirrors config.Env / config.EnvRequired in plugin-sdk-go.
"""

from __future__ import annotations

import os
from typing import Optional


class EnvHelper:
    """
    Env var helper exposed in PluginContext.
    Mirrors config.Env, config.EnvRequired, etc. in plugin-sdk-go.
    """

    def get(self, key: str, default: Optional[str] = None) -> Optional[str]:
        """Return the env var value or default."""
        v = os.environ.get(key)
        if v is not None and v != "":
            return v
        return default

    def require(self, keys: list[str]) -> None:
        """
        Assert that all named env vars are non-empty.
        Raises ValueError on first missing var — fail-fast at install/start time.
        """
        for key in keys:
            if not os.environ.get(key):
                raise ValueError(f"plugin SDK: required env var {key} is not set")

    def get_int(self, key: str, default: int) -> int:
        """Return the env var parsed as an integer, or default."""
        v = os.environ.get(key)
        if not v:
            return default
        try:
            return int(v)
        except ValueError:
            return default

    def get_bool(self, key: str, default: bool) -> bool:
        """Return the env var parsed as boolean."""
        v = (os.environ.get(key) or "").lower()
        if v in ("true", "1", "yes", "y", "on"):
            return True
        if v in ("false", "0", "no", "n", "off"):
            return False
        return default

    def get_list(self, key: str) -> list[str]:
        """Split the env var on commas and trim whitespace. Empty → []."""
        v = os.environ.get(key, "")
        if not v:
            return []
        return [s.strip() for s in v.split(",") if s.strip()]
