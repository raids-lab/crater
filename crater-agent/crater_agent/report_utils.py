"""Helpers for turning report-like tool results into UI payloads."""

from __future__ import annotations

from typing import Any


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
        return int(str(value).strip())
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


def _report_title(report_type: str) -> str:
    normalized = str(report_type or "").strip().lower()
    if normalized == "gpu_audit":
        return "GPU 利用率审计报告"
    if normalized == "admin_ops_report":
        return "智能运维分析报告"
    return report_type or "分析报告"


def _normalize_report_items(raw_items: Any) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    for raw in _as_list(raw_items):
        item = _as_dict(raw)
        items.append(
            {
                "job_name": str(item.get("job_name") or item.get("jobName") or "").strip(),
                "user": str(item.get("user") or item.get("username") or "").strip(),
                "gpu_util": str(item.get("gpu_util") or item.get("gpuUtil") or "-").strip() or "-",
                "duration": str(item.get("duration") or "").strip(),
                "gpu_requested": _to_int(item.get("gpu_requested") or item.get("gpuRequested")),
                "gpu_actual": _to_int(item.get("gpu_actual") or item.get("gpuActual")),
            }
        )
    return items


def build_pipeline_report_payload_from_admin_ops_report(report: dict[str, Any]) -> dict[str, Any] | None:
    if not isinstance(report, dict):
        return None

    overview = _as_dict(report.get("overview"))
    idle_summary = _as_dict(report.get("idle_summary"))
    categories = []
    for raw in _as_list(report.get("recommended_actions")):
        action = _as_dict(raw)
        categories.append(
            {
                "action": str(action.get("action") or "建议").strip(),
                "severity": str(action.get("severity") or "info").strip() or "info",
                "count": _to_int(action.get("count")),
                "items": _normalize_report_items(action.get("items")),
            }
        )

    return {
        "reportId": str(report.get("report_id") or report.get("id") or "").strip(),
        "reportType": _report_title(str(report.get("report_type") or "admin_ops_report")),
        "completedAt": str(report.get("generated_at") or report.get("completed_at") or "").strip(),
        "summary": {
            "total_scanned": _to_int(overview.get("total_jobs")),
            "idle_detected": _to_int(overview.get("failed_jobs")),
            "gpu_waste_hours": round(_to_float(idle_summary.get("estimated_gpu_waste_hours")), 1),
        },
        "summary_labels": {
            "total_label": "总任务数",
            "middle_label": "失败作业",
            "right_label": "预估浪费 GPU 时",
        },
        "categories": categories,
    }


def build_pipeline_report_payload_from_audit_report(report: dict[str, Any]) -> dict[str, Any] | None:
    if not isinstance(report, dict):
        return None

    summary = _as_dict(report.get("summary"))
    categories = []
    for raw in _as_list(summary.get("recommendations")):
        item = _as_dict(raw)
        action = str(item.get("action") or "建议").strip()
        severity = {"stop": "critical", "notify": "warning", "downscale": "info"}.get(action, "info")
        categories.append(
            {
                "action": action,
                "severity": severity,
                "count": _to_int(item.get("count")),
                "items": [],
            }
        )

    summary_labels = _as_dict(summary.get("summary_labels"))
    return {
        "reportId": str(report.get("id") or report.get("report_id") or "").strip(),
        "reportType": _report_title(str(report.get("report_type") or "gpu_audit")),
        "completedAt": str(report.get("completed_at") or report.get("created_at") or "").strip(),
        "summary": {
            "total_scanned": _to_int(summary.get("total_scanned")),
            "idle_detected": _to_int(summary.get("idle_detected")),
            "gpu_waste_hours": round(_to_float(summary.get("gpu_waste_hours") or summary.get("estimated_gpu_waste_hours")), 1),
        },
        "summary_labels": {
            "total_label": str(summary_labels.get("total_label") or "扫描任务"),
            "middle_label": str(summary_labels.get("middle_label") or "闲置检出"),
            "right_label": str(summary_labels.get("right_label") or "GPU 浪费时"),
        },
        "categories": categories,
    }


def build_pipeline_report_payload(tool_name: str, result: dict[str, Any]) -> dict[str, Any] | None:
    if not isinstance(result, dict) or result.get("status") == "error":
        return None
    payload = result.get("result", result)
    if tool_name == "get_admin_ops_report":
        return build_pipeline_report_payload_from_admin_ops_report(_as_dict(payload))
    if tool_name == "get_latest_audit_report":
        return build_pipeline_report_payload_from_audit_report(_as_dict(payload))
    return None


def build_admin_ops_audit_items(report: dict[str, Any]) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    for raw_action in _as_list(report.get("recommended_actions")):
        action = _as_dict(raw_action)
        action_label = str(action.get("action") or "").strip()
        action_type = {
            "关注失败作业热点": "failure_review",
            "复盘成功作业资源差异": "success_review",
            "处理低利用率作业": "idle_review",
        }.get(action_label, "ops_review")
        severity = str(action.get("severity") or "info").strip() or "info"
        for raw_item in _as_list(action.get("items")):
            item = _as_dict(raw_item)
            items.append(
                {
                    "job_name": str(item.get("job_name") or "").strip(),
                    "username": str(item.get("user") or "").strip(),
                    "action_type": action_type,
                    "severity": severity,
                    "gpu_utilization": _to_float(item.get("gpu_util")),
                    "gpu_requested": _to_int(item.get("gpu_requested")),
                    "gpu_actual_used": _to_int(item.get("gpu_actual")),
                    "analysis_detail": {
                        "source_action": action_label,
                        "duration": str(item.get("duration") or "").strip(),
                    },
                }
            )
    return [item for item in items if item["job_name"]]
