"""
plugin.yaml manifest parser and validator.
Mirrors the manifest parser in plugin-sdk-go (config/config.go + plugin.Info).
"""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

import yaml


class ManifestValidationError(ValueError):
    """Raised when the plugin.yaml manifest is missing required fields or malformed."""

    def __init__(self, message: str, field_name: str | None = None) -> None:
        self.field_name = field_name
        super().__init__(f"plugin manifest: {message}")


@dataclass
class PluginManifest:
    """Full plugin manifest shape as declared in plugin.yaml."""
    name: str
    version: str
    tier: str  # "free" | "pro"
    description: str = ""
    bundle: str = ""
    min_cli: str = ""
    min_sdk: str = ""
    port: int = 0
    routes: list[dict[str, str]] = field(default_factory=list)
    required_env: list[str] = field(default_factory=list)
    metadata: dict[str, str] = field(default_factory=dict)


def parse_manifest(file_path: str = "plugin.yaml") -> PluginManifest:
    """
    Read and validate a plugin.yaml file.
    Raises ManifestValidationError on missing required fields.
    """
    path = Path(file_path).resolve()
    try:
        raw = path.read_text(encoding="utf-8")
    except OSError as exc:
        raise ManifestValidationError(f"cannot read file: {path}: {exc}") from exc
    return parse_manifest_string(raw)


def parse_manifest_string(content: str) -> PluginManifest:
    """
    Parse plugin.yaml content from a string. Useful in tests.
    """
    try:
        data = yaml.safe_load(content)
    except yaml.YAMLError as exc:
        raise ManifestValidationError(f"YAML parse error: {exc}") from exc

    return _validate(data)


def _validate(data: object) -> PluginManifest:
    if not isinstance(data, dict):
        raise ManifestValidationError("manifest must be a YAML object")

    name = data.get("name")
    if not name or not isinstance(name, str):
        raise ManifestValidationError("name is required and must be a string", "name")

    version = data.get("version")
    if not version or not isinstance(version, str):
        # pyyaml parses bare 1.0.0 as a float; coerce to string
        if isinstance(version, (int, float)):
            version = str(version)
        else:
            raise ManifestValidationError("version is required and must be a string", "version")

    tier = data.get("tier")
    if tier not in ("free", "pro"):
        raise ManifestValidationError('tier must be "free" or "pro"', "tier")

    return PluginManifest(
        name=name,
        version=version,
        tier=tier,
        description=data.get("description", ""),
        bundle=data.get("bundle", ""),
        min_cli=data.get("minCli", ""),
        min_sdk=data.get("minSdk", ""),
        port=int(data.get("port", 0)),
        routes=data.get("routes") or [],
        required_env=data.get("requiredEnv") or [],
        metadata=data.get("metadata") or {},
    )
