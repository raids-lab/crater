"""Generate quality eval reports in markdown and CSV formats."""
from __future__ import annotations

import csv
from datetime import datetime
from pathlib import Path


def write_md_report(
    path: Path,
    session_id: str,
    turn_id: str | None,
    chat_scores: dict,
    chain_scores: dict,
    trigger_source: str,
    rating: int | None = None,
) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    ts = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    lines = [
        f"# Quality Eval Report",
        f"",
        f"- **Session**: `{session_id}`",
        f"- **Turn**: `{turn_id or 'session-level'}`",
        f"- **Trigger**: {trigger_source}",
        f"- **User Rating**: {'👍' if rating == 1 else '👎' if rating == -1 else 'N/A'}",
        f"- **Generated**: {ts}",
        f"",
        f"## Chat Quality (xiaomi)",
        f"",
    ]
    for k, v in chat_scores.items():
        if k != "reasoning":
            lines.append(f"- **{k}**: {v}/5")
    if "reasoning" in chat_scores:
        lines += ["", f"> {chat_scores['reasoning']}"]
    lines += ["", "## Reasoning Chain Quality (qwen)", ""]
    for k, v in chain_scores.items():
        if k != "reasoning":
            lines.append(f"- **{k}**: {v}/5")
    if "reasoning" in chain_scores:
        lines += ["", f"> {chain_scores['reasoning']}"]

    path.write_text("\n".join(lines), encoding="utf-8")


def append_csv_row(csv_path: Path, row: dict) -> None:
    csv_path.parent.mkdir(parents=True, exist_ok=True)
    write_header = not csv_path.exists()
    with open(csv_path, "a", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=list(row.keys()))
        if write_header:
            writer.writeheader()
        writer.writerow(row)
