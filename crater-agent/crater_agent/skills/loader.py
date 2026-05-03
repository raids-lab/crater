"""Loader for YAML-based diagnostic skills (knowledge injection, not RAG)."""

from __future__ import annotations

import os
import re
from pathlib import Path

import yaml


SKILLS_DIR = Path(__file__).parent


def load_skill(path: str | Path) -> dict:
    """Load a single skill YAML file."""
    with open(path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def format_skill_for_prompt(skill: dict) -> str:
    """Format a skill into text suitable for system prompt injection."""
    lines = []
    lines.append(f"### {skill.get('name', 'Unknown Skill')}")
    lines.append(f"**描述**: {skill.get('description', '')}")

    # Trigger signals
    triggers = skill.get("trigger_signals", {})
    if triggers:
        trigger_parts = []
        if triggers.get("exit_codes"):
            trigger_parts.append(f"退出码: {triggers['exit_codes']}")
        if triggers.get("event_reasons"):
            trigger_parts.append(f"事件: {triggers['event_reasons']}")
        if triggers.get("log_keywords"):
            trigger_parts.append(f"日志关键词: {triggers['log_keywords']}")
        if triggers.get("job_status"):
            trigger_parts.append(f"作业状态: {triggers['job_status']}")
        if trigger_parts:
            lines.append(f"**触发信号**: {'; '.join(trigger_parts)}")

    # Diagnosis knowledge
    knowledge = skill.get("diagnosis_knowledge", "")
    if knowledge:
        lines.append(f"**诊断知识**: {knowledge.strip()}")

    # Common solutions
    solutions = skill.get("common_solutions", [])
    if solutions:
        lines.append("**常见解决方案**:")
        for sol in solutions:
            condition = sol.get("condition", "")
            suggestion = sol.get("suggestion", sol.get("name", ""))
            if condition:
                lines.append(f"  - 当 {condition}: {suggestion}")
            else:
                lines.append(f"  - {suggestion}")

    return "\n".join(lines)


def _skill_match_score(skill: dict, query: str) -> int:
    normalized = str(query or "").strip().lower()
    if not normalized:
        return 0

    haystacks: list[tuple[str, int]] = [
        (str(skill.get("name") or ""), 3),
        (str(skill.get("description") or ""), 2),
        (str(skill.get("diagnosis_knowledge") or ""), 1),
    ]
    triggers = skill.get("trigger_signals") if isinstance(skill.get("trigger_signals"), dict) else {}
    for key in ("exit_codes", "event_reasons", "log_keywords", "job_status"):
        values = triggers.get(key) or []
        if isinstance(values, list):
            haystacks.extend((str(value), 4) for value in values)

    score = 0
    compact_query = re.sub(r"\s+", "", normalized)
    for text, weight in haystacks:
        text_normalized = str(text or "").strip().lower()
        if not text_normalized:
            continue
        if text_normalized in normalized or re.sub(r"\s+", "", text_normalized) in compact_query:
            score += weight
            continue
        for token in re.findall(r"[a-z0-9_./-]{3,}|[\u4e00-\u9fff]{2,}", text_normalized):
            if token and token in normalized:
                score += weight
                break
    return score


def load_relevant_skills(
    user_message: str,
    skills_dir: str | Path | None = None,
    *,
    max_skills: int = 2,
) -> str:
    """Load only skills that are relevant to the current user message."""
    if skills_dir is None:
        skills_dir = SKILLS_DIR

    skills_dir = Path(skills_dir)
    if not skills_dir.exists():
        return ""

    ranked: list[tuple[int, str]] = []
    for yaml_file in sorted(skills_dir.glob("*.yaml")):
        try:
            skill = load_skill(yaml_file)
            if not skill:
                continue
            score = _skill_match_score(skill, user_message)
            if score > 0:
                ranked.append((score, format_skill_for_prompt(skill)))
        except Exception:
            continue

    if not ranked:
        return ""

    ranked.sort(key=lambda item: item[0], reverse=True)
    selected = [text for _, text in ranked[: max(1, int(max_skills))]]
    return "## 诊断参考知识\n\n" + "\n\n".join(selected)


def load_all_skills(skills_dir: str | Path | None = None) -> str:
    """Load all skill YAML files and format them for system prompt injection.

    Returns:
        Formatted string containing all skills knowledge, ready for prompt injection.
    """
    if skills_dir is None:
        skills_dir = SKILLS_DIR

    skills_dir = Path(skills_dir)
    if not skills_dir.exists():
        return ""

    skill_texts = []
    for yaml_file in sorted(skills_dir.glob("*.yaml")):
        try:
            skill = load_skill(yaml_file)
            if skill:
                skill_texts.append(format_skill_for_prompt(skill))
        except Exception:
            continue

    if not skill_texts:
        return ""

    return "## 诊断参考知识\n\n" + "\n\n".join(skill_texts)
