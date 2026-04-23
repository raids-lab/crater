"""Output path management for quality eval artifacts."""
from __future__ import annotations

import os
from datetime import datetime
from pathlib import Path


def get_eval_output_dir() -> Path:
    base = os.environ.get("CRATER_EVAL_OUTPUT_DIR", "/var/log/crater-agent/eval")
    return Path(base)


def feedback_artifact_path(session_id: str, turn_id: str | None, ts: datetime | None = None) -> Path:
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
