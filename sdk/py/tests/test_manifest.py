"""Tests for plugin.yaml manifest parser."""

import pytest

from nself_plugin import parse_manifest_string, ManifestValidationError

VALID_YAML = """
name: test-plugin
version: "1.0.0"
description: A test plugin
tier: free
port: 3850
requiredEnv:
  - DATABASE_URL
"""

PRO_YAML = """
name: claw-plugin
version: "2.1.0"
tier: pro
bundle: nClaw
minCli: "1.0.0"
"""


def test_parses_valid_free_manifest() -> None:
    m = parse_manifest_string(VALID_YAML)
    assert m.name == "test-plugin"
    assert m.version == "1.0.0"
    assert m.tier == "free"
    assert m.port == 3850
    assert m.required_env == ["DATABASE_URL"]


def test_parses_valid_pro_manifest() -> None:
    m = parse_manifest_string(PRO_YAML)
    assert m.name == "claw-plugin"
    assert m.tier == "pro"
    assert m.bundle == "nClaw"


def test_raises_on_missing_name() -> None:
    with pytest.raises(ManifestValidationError, match="name"):
        parse_manifest_string("version: '1.0.0'\ntier: free\n")


def test_raises_on_missing_version() -> None:
    with pytest.raises(ManifestValidationError, match="version"):
        parse_manifest_string("name: x\ntier: free\n")


def test_raises_on_invalid_tier() -> None:
    with pytest.raises(ManifestValidationError, match="tier"):
        parse_manifest_string("name: x\nversion: '1.0.0'\ntier: enterprise\n")


def test_raises_on_invalid_yaml() -> None:
    with pytest.raises(ManifestValidationError):
        parse_manifest_string("name: [unclosed")


def test_raises_on_non_object() -> None:
    with pytest.raises(ManifestValidationError, match="YAML object"):
        parse_manifest_string("- item1\n- item2\n")
