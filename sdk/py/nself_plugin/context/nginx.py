"""
Nginx route injection helper.
Mirrors NginxHelper in the spec. Routes are declared in plugin.yaml;
this helper is used in install hooks to validate / extend them.
"""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class RouteOptions:
    location: str   # e.g. "/notify/"
    upstream: str   # e.g. "127.0.0.1:3850"
    extra_directives: list[str] = field(default_factory=list)


class NginxHelper:
    """Nginx route injection helper exposed in PluginContext."""

    def __init__(self) -> None:
        self._routes: list[RouteOptions] = []

    def add_route(self, opts: RouteOptions) -> None:
        """Register a proxy_pass route to the plugin's upstream."""
        if not opts.location.startswith("/"):
            raise ValueError(
                f"nginx: location must start with '/', got {opts.location!r}"
            )
        self._routes.append(opts)

    def remove_route(self, location: str) -> None:
        """Remove a previously registered route (used in uninstall)."""
        self._routes = [r for r in self._routes if r.location != location]

    def get_routes(self) -> list[RouteOptions]:
        """Returns all routes registered in this session."""
        return list(self._routes)
