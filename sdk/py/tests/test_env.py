"""Tests for EnvHelper."""

import pytest

from nself_plugin.context.env import EnvHelper


@pytest.fixture()
def env() -> EnvHelper:
    return EnvHelper()


def test_get_returns_value(monkeypatch: pytest.MonkeyPatch, env: EnvHelper) -> None:
    monkeypatch.setenv("TEST_VAR", "hello")
    assert env.get("TEST_VAR") == "hello"


def test_get_returns_default_when_missing(env: EnvHelper) -> None:
    assert env.get("MISSING_VAR_XYZ", "default") == "default"


def test_require_raises_on_missing(env: EnvHelper) -> None:
    with pytest.raises(ValueError, match="REQUIRED_VAR"):
        env.require(["REQUIRED_VAR"])


def test_require_passes_when_all_set(monkeypatch: pytest.MonkeyPatch, env: EnvHelper) -> None:
    monkeypatch.setenv("VAR_A", "a")
    monkeypatch.setenv("VAR_B", "b")
    env.require(["VAR_A", "VAR_B"])  # should not raise


def test_get_int_parses(monkeypatch: pytest.MonkeyPatch, env: EnvHelper) -> None:
    monkeypatch.setenv("MY_PORT", "3850")
    assert env.get_int("MY_PORT", 3000) == 3850


def test_get_int_returns_default_on_non_numeric(
    monkeypatch: pytest.MonkeyPatch, env: EnvHelper
) -> None:
    monkeypatch.setenv("MY_PORT", "abc")
    assert env.get_int("MY_PORT", 3000) == 3000


def test_get_bool_true_variants(monkeypatch: pytest.MonkeyPatch, env: EnvHelper) -> None:
    for v in ("true", "1", "yes", "y", "on"):
        monkeypatch.setenv("BOOL_VAR", v)
        assert env.get_bool("BOOL_VAR", False) is True


def test_get_list_splits_commas(monkeypatch: pytest.MonkeyPatch, env: EnvHelper) -> None:
    monkeypatch.setenv("LIST_VAR", "a, b, c")
    assert env.get_list("LIST_VAR") == ["a", "b", "c"]


def test_get_list_empty(env: EnvHelper) -> None:
    assert env.get_list("UNSET_LIST_VAR") == []
