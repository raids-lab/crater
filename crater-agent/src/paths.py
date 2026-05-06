"""Filesystem paths for the flat crater-agent source layout."""

from __future__ import annotations

from pathlib import Path


def agent_root() -> Path:
    """Return the crater-agent project directory."""
    return Path(__file__).resolve().parents[1]


def repo_root() -> Path:
    """Return the repository root that contains crater-agent and backend."""
    return agent_root().parent
