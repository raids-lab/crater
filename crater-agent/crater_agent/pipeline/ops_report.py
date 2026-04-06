"""Admin ops report pipeline — collects cluster data and uses LLM for analysis."""

from __future__ import annotations

import logging
from datetime import datetime, timezone, timedelta
from typing import Any

from crater_agent.pipeline.ops_report_llm import (
    analyze_ops_report_with_llm,
    build_deterministic_ops_report,
)
from crater_agent.report_utils import (
    build_admin_ops_audit_items,
    build_pipeline_report_payload_from_admin_ops_report,
)
from crater_agent.pipeline.tool_client import PipelineToolClient

logger = logging.getLogger(__name__)


async def run_admin_ops_report(
    *,
    days: int,
    lookback_hours: int,
    gpu_threshold: int,
    idle_hours: int,
    running_limit: int,
    node_limit: int,
    dry_run: bool = False,
    use_llm: bool = True,
) -> dict:
    now = datetime.now(timezone.utc)
    period_end = now.isoformat()
    period_start = (now - timedelta(days=days)).isoformat()

    async with PipelineToolClient(timeout=120) as client:
        # Step 1: Collect raw data via Go backend tools
        report_result = await client.execute(
            "get_admin_ops_report",
            {
                "days": days,
                "success_limit": 5,
                "failure_limit": 5,
                "lookback_hours": lookback_hours,
                "gpu_threshold": gpu_threshold,
                "idle_hours": idle_hours,
                "running_limit": running_limit,
                "node_limit": node_limit,
            },
        )
        if report_result.get("status") != "success":
            return {
                "status": "failed",
                "summary": {
                    "error": report_result.get("message", "Failed to build admin ops report"),
                },
            }

        raw_report = report_result.get("result", {})

        # Step 2: Fetch previous report for trend comparison
        previous_report_json = None
        prev_result = await client.execute(
            "get_latest_audit_report",
            {"report_type": "admin_ops_report"},
        )
        if prev_result.get("status") == "success":
            prev_data = prev_result.get("result", {})
            if isinstance(prev_data, dict) and prev_data.get("id"):
                prev_report_json = prev_data.get("report_json")
                if isinstance(prev_report_json, dict):
                    previous_report_json = prev_report_json
                elif isinstance(prev_report_json, str):
                    import json as _json

                    try:
                        parsed_prev = _json.loads(prev_report_json)
                    except Exception:
                        parsed_prev = None
                    if isinstance(parsed_prev, dict):
                        previous_report_json = parsed_prev
                if previous_report_json is None:
                    prev_summary = prev_data.get("summary")
                    if isinstance(prev_summary, dict):
                        previous_report_json = prev_summary

        # Step 3: LLM analysis (or fallback)
        report_json = None
        if use_llm:
            report_json = await analyze_ops_report_with_llm(raw_report, previous_report_json)
        else:
            report_json = build_deterministic_ops_report(raw_report, previous_report_json)

        # Step 4: Build pipeline payload (for backward compat with existing report card)
        pipeline_payload = build_pipeline_report_payload_from_admin_ops_report(raw_report) or {}
        summary = dict(pipeline_payload.get("summary") or {})
        summary["summary_labels"] = pipeline_payload.get("summary_labels") or {}
        summary["lookback_days"] = raw_report.get("lookback_days", days)
        summary["lookback_hours"] = raw_report.get("lookback_hours", lookback_hours)
        summary["recent_running_summary"] = raw_report.get("recent_running_summary") or {}
        summary["recent_running_jobs"] = raw_report.get("recent_running_jobs") or []
        summary["node_summary"] = raw_report.get("node_summary") or {}
        summary["resource_utilization"] = raw_report.get("resource_utilization") or {}

        # Merge LLM executive summary into summary if available
        if report_json and report_json.get("executive_summary"):
            summary["executive_summary"] = report_json["executive_summary"]

        # Step 5: Build extended audit items
        base_items = build_admin_ops_audit_items(raw_report)
        _enrich_audit_items(base_items, raw_report)

        # Step 6: Persist
        report_id: str | None = None
        overview = raw_report.get("overview", {})
        if not dry_run:
            save_args: dict[str, Any] = {
                "report_type": "admin_ops_report",
                "trigger_source": "pipeline",
                "summary": summary,
                "items": base_items,
                "period_start": period_start,
                "period_end": period_end,
                "job_total": int(overview.get("total_jobs", 0)),
                "job_success": int(overview.get("success_jobs", overview.get("completed_jobs", 0))),
                "job_failed": int(overview.get("failed_jobs", 0)),
                "job_pending": int(overview.get("pending_jobs", 0)),
            }
            if report_json:
                import json as _json
                # Send as dict, not string — Go's json.RawMessage needs raw JSON object
                save_args["report_json"] = report_json
            save_result = await client.execute("save_audit_report", save_args)
            if save_result.get("status") == "success":
                report_id = str(save_result.get("result", {}).get("report_id") or "").strip() or None
            else:
                logger.warning("save_audit_report failed: %s", save_result)

        if report_id:
            pipeline_payload["reportId"] = report_id
            raw_report["report_id"] = report_id

        return {
            "status": "completed" if not dry_run else "dry_run",
            "report_id": report_id,
            "summary": summary,
            "report": pipeline_payload,
            "report_json": report_json,
            "raw_report": raw_report,
        }


def _enrich_audit_items(items: list[dict[str, Any]], raw_report: dict[str, Any]) -> None:
    """Add category and failure_reason to audit items based on raw report data."""
    # Go returns snake_case keys: job_name, user, failure_category, exit_code, etc.
    failed_jobs = {
        j.get("job_name", j.get("jobName", "")): j
        for j in raw_report.get("failed_jobs", [])
        if j.get("job_name") or j.get("jobName")
    }
    success_jobs = {
        j.get("job_name", j.get("jobName", "")): j
        for j in raw_report.get("successful_jobs", [])
        if j.get("job_name") or j.get("jobName")
    }
    for item in items:
        job_name = item.get("job_name", "")
        if job_name in failed_jobs:
            fj = failed_jobs[job_name]
            item["category"] = "failure"
            item["failure_reason"] = str(
                fj.get("failure_category", fj.get("failureReason", ""))
            ).strip() or None
            item["exit_code"] = fj.get("exit_code", fj.get("exitCode"))
            item["job_type"] = fj.get("job_type", fj.get("jobType", ""))
            item["owner"] = fj.get("user", fj.get("owner", "")) or item.get("username", "")
            item["namespace"] = fj.get("namespace", "")
        elif job_name in success_jobs:
            sj = success_jobs[job_name]
            item["category"] = "success"
            item["job_type"] = sj.get("job_type", sj.get("jobType", ""))
            item["owner"] = sj.get("user", sj.get("owner", "")) or item.get("username", "")
            item["namespace"] = sj.get("namespace", "")
        else:
            action_type = item.get("action_type", "")
            if action_type in ("idle_review", "stop"):
                item["category"] = "idle"
            else:
                item["category"] = "other"
