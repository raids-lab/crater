"""Admin ops report pipeline — collects cluster data and uses LLM for analysis."""

from __future__ import annotations

import logging
from datetime import datetime, timedelta, timezone
from typing import Any

from pipeline.ops_report_llm import (
    analyze_ops_report_with_llm,
    build_deterministic_ops_report,
)
from pipeline.tool_client import PipelineToolClient
from report_utils import (
    build_admin_ops_audit_items,
    build_pipeline_report_payload_from_admin_ops_report,
)

logger = logging.getLogger(__name__)


def _as_dict(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


def _as_list(value: Any) -> list[Any]:
    return value if isinstance(value, list) else []


def _to_int(value: Any) -> int:
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, (int, float)):
        return int(value)
    try:
        return int(str(value or "").strip())
    except (TypeError, ValueError):
        return 0


def _to_float(value: Any) -> float:
    if isinstance(value, bool):
        return float(value)
    if isinstance(value, (int, float)):
        return float(value)
    text = str(value or "").strip().rstrip("%")
    try:
        return float(text)
    except ValueError:
        return 0.0


async def _load_previous_report_json(
    client: PipelineToolClient,
    *,
    report_type: str,
) -> dict[str, Any] | None:
    prev_result = await client.execute("get_latest_audit_report", {"report_type": report_type})
    if prev_result.get("status") != "success":
        return None
    prev_data = _as_dict(prev_result.get("result"))
    if not prev_data.get("id"):
        return None

    prev_report_json = prev_data.get("report_json")
    if isinstance(prev_report_json, dict):
        return prev_report_json
    if isinstance(prev_report_json, str):
        import json as _json

        try:
            parsed_prev = _json.loads(prev_report_json)
        except Exception:
            parsed_prev = None
        if isinstance(parsed_prev, dict):
            return parsed_prev
    prev_summary = prev_data.get("summary")
    return prev_summary if isinstance(prev_summary, dict) else None


async def _collect_compute_domain(
    client: PipelineToolClient,
    *,
    days: int,
    lookback_hours: int,
    gpu_threshold: int,
    idle_hours: int,
    running_limit: int,
    node_limit: int,
) -> tuple[dict[str, Any], dict[str, Any]]:
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
        return {}, {
            "status": "error",
            "error": report_result.get("message", "get_admin_ops_report failed"),
            "source_tool": "get_admin_ops_report",
        }
    return _as_dict(report_result.get("result")), {
        "status": "success",
        "source_tool": "get_admin_ops_report",
    }


async def _collect_storage_domain(
    client: PipelineToolClient,
    *,
    days: int,
) -> dict[str, Any]:
    errors: list[str] = []
    pvc_items: list[dict[str, Any]] = []
    capacity_payload: dict[str, Any] = {}

    pvc_result = await client.execute("list_storage_pvcs", {"limit": 200})
    if pvc_result.get("status") == "success":
        pvc_payload = _as_dict(pvc_result.get("result"))
        raw_items = pvc_payload.get("pvcs")
        if not isinstance(raw_items, list):
            raw_items = pvc_payload.get("items")
        pvc_items = [_as_dict(item) for item in _as_list(raw_items)]
    else:
        errors.append(str(pvc_result.get("message") or "list_storage_pvcs failed"))

    capacity_result = await client.execute("get_storage_capacity_overview", {"days": days})
    if capacity_result.get("status") == "success":
        capacity_payload = _as_dict(capacity_result.get("result"))
    else:
        errors.append(
            str(capacity_result.get("message") or "get_storage_capacity_overview failed")
        )

    anomaly_count = 0
    for pvc in pvc_items:
        phase = str(pvc.get("phase") or pvc.get("status") or "").strip().lower()
        usage_pct = _to_float(
            pvc.get("usage_pct")
            or pvc.get("usage_percent")
            or pvc.get("usage")
            or pvc.get("capacity_used_pct")
        )
        if (phase and phase not in {"bound", "available"}) or usage_pct >= 90:
            anomaly_count += 1

    capacity_summary = _as_dict(capacity_payload.get("summary")) if capacity_payload else {}
    high_pressure_clusters = _to_int(
        capacity_summary.get("high_pressure_clusters")
        or capacity_summary.get("pressure_clusters")
        or capacity_summary.get("high_pressure_count")
    )
    total_clusters = _to_int(
        capacity_summary.get("total_clusters") or capacity_summary.get("cluster_count")
    )

    status = "success"
    if errors and (pvc_items or capacity_payload):
        status = "partial"
    elif errors and not pvc_items and not capacity_payload:
        status = "unavailable"

    return {
        "status": status,
        "total_pvcs": len(pvc_items),
        "anomaly_pvcs": anomaly_count,
        "high_pressure_clusters": high_pressure_clusters,
        "total_clusters": total_clusters,
        "errors": errors,
    }


async def _collect_network_domain(
    client: PipelineToolClient,
    *,
    lookback_hours: int,
    running_limit: int,
    node_limit: int,
) -> dict[str, Any]:
    errors: list[str] = []
    degraded_nodes = 0
    network_alerts = 0
    diagnosed_jobs = 0
    high_risk_jobs = 0

    node_result = await client.execute("get_node_network_summary", {"limit": node_limit})
    node_payload = {}
    if node_result.get("status") == "success":
        node_payload = _as_dict(node_result.get("result"))
    else:
        errors.append(str(node_result.get("message") or "get_node_network_summary failed"))

    nodes = _as_list(_as_dict(node_payload).get("nodes"))
    degraded_nodes = _to_int(node_payload.get("degraded_nodes"))
    if degraded_nodes <= 0 and nodes:
        degraded_nodes = sum(
            1
            for raw in nodes
            if str(_as_dict(raw).get("status") or "").strip().lower()
            in {"degraded", "warning", "critical", "notready"}
        )
    for raw in nodes:
        alerts = _as_dict(raw).get("alerts")
        if isinstance(alerts, list):
            network_alerts += len(alerts)

    diagnosis_result = await client.execute(
        "diagnose_distributed_job_network",
        {"lookback_hours": lookback_hours, "limit": running_limit},
    )
    diagnosis_payload = {}
    if diagnosis_result.get("status") == "success":
        diagnosis_payload = _as_dict(diagnosis_result.get("result"))
    else:
        errors.append(
            str(diagnosis_result.get("message") or "diagnose_distributed_job_network failed")
        )

    raw_jobs = diagnosis_payload.get("jobs")
    if isinstance(raw_jobs, list):
        diagnosed_jobs = len(raw_jobs)
        high_risk_jobs = sum(
            1
            for raw in raw_jobs
            if str(_as_dict(raw).get("severity") or "").strip().lower() in {"high", "critical"}
        )
    else:
        diagnosed_jobs = _to_int(
            diagnosis_payload.get("diagnosed_jobs") or diagnosis_payload.get("count")
        )
        high_risk_jobs = _to_int(
            diagnosis_payload.get("high_risk_jobs") or diagnosis_payload.get("high_risk_count")
        )

    status = "success"
    if errors and (nodes or diagnosis_payload):
        status = "partial"
    elif errors and not nodes and not diagnosis_payload:
        status = "unavailable"

    return {
        "status": status,
        "degraded_nodes": degraded_nodes,
        "network_alerts": network_alerts,
        "diagnosed_jobs": diagnosed_jobs,
        "high_risk_jobs": high_risk_jobs,
        "errors": errors,
    }


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

    async with PipelineToolClient(
        timeout=120,
        session_source="system",
        session_title="[system] 智能巡检日报",
    ) as client:
        # Step 1: Collect domain data via backend tools.
        compute_report, compute_collector = await _collect_compute_domain(
            client,
            days=days,
            lookback_hours=lookback_hours,
            gpu_threshold=gpu_threshold,
            idle_hours=idle_hours,
            running_limit=running_limit,
            node_limit=node_limit,
        )
        if compute_collector.get("status") != "success":
            return {
                "status": "failed",
                "summary": {
                    "error": compute_collector.get(
                        "error",
                        "Failed to build admin ops report",
                    ),
                },
            }
        storage_summary = await _collect_storage_domain(client, days=days)
        network_summary = await _collect_network_domain(
            client,
            lookback_hours=lookback_hours,
            running_limit=running_limit,
            node_limit=node_limit,
        )

        raw_report = dict(compute_report)
        raw_report["storage_summary"] = storage_summary
        raw_report["network_summary"] = network_summary
        raw_report["domain_collectors"] = {
            "compute": compute_collector,
            "storage": {"status": storage_summary.get("status", "unavailable")},
            "network": {"status": network_summary.get("status", "unavailable")},
        }

        # Step 2: Fetch previous report for trend comparison
        previous_report_json = await _load_previous_report_json(
            client,
            report_type="admin_ops_report",
        )

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
        summary["storage_summary"] = storage_summary
        summary["network_summary"] = network_summary
        summary["domain_collectors"] = raw_report.get("domain_collectors") or {}

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
                # Send as dict, not string — Go's json.RawMessage needs raw JSON object
                save_args["report_json"] = report_json
            save_result = await client.execute("save_audit_report", save_args)
            if save_result.get("status") == "success":
                report_id = (
                    str(save_result.get("result", {}).get("report_id") or "").strip()
                    or None
                )
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
