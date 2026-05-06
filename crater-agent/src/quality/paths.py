"""Output path management for quality eval artifacts."""
from __future__ import annotations

from datetime import datetime
from pathlib import Path

from config import settings


def _ensure_writable_dir(path: Path) -> Path | None:
    try:
        path.mkdir(parents=True, exist_ok=True)
    except OSError:
        return None
    return path


def get_eval_output_dir() -> Path:
    configured = settings.resolve_quality_eval_output_dir()
    resolved = _ensure_writable_dir(configured)
    if resolved is not None:
        return resolved
    raise PermissionError(
        f"quality eval artifact directory is not writable: {configured}"
    )


def feedback_artifact_path(
    session_id: str,
    turn_id: str | None,
    ts: datetime | None = None,
) -> Path:
    ts = ts or datetime.now()
    date_str = ts.strftime("%Y-%m-%d")
    suffix = f"{session_id}_{turn_id or 'session'}_{ts.strftime('%H%M%S')}"
    return get_eval_output_dir() / "quality" / "feedback" / date_str / f"{suffix}.md"


def offline_artifact_path(batch_id: str, ts: datetime | None = None) -> tuple[Path, Path]:
    """Returns (md_path, csv_path)."""
    ts = ts or datetime.now()
    date_str = ts.strftime("%Y-%m-%d")
    base = get_eval_output_dir() / "quality" / "offline" / date_str
    return base / f"{batch_id}.md", base / f"{batch_id}.csv"


def manual_artifact_path(session_id: str, ts: datetime | None = None) -> Path:
    ts = ts or datetime.now()
    date_str = ts.strftime("%Y-%m-%d")
    suffix = f"{session_id}_{ts.strftime('%H%M%S')}"
    return get_eval_output_dir() / "quality" / "manual" / date_str / f"{suffix}.md"
