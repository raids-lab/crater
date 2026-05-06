"""Storage audit pipeline — inspects PVC/storage health and persists audit reports."""

from __future__ import annotations

import logging
from datetime import datetime, timedelta, timezone
from typing import Any

from pipeline.tool_client import PipelineToolClient

logger = logging.getLogger(__name__)


def _as_dict(value: Any) -> dict[str, Any]:
    return value if isinstance(value, dict) else {}


def _as_list(value: Any) -> list[Any]:
    return value if isinstance(value, list) else []


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


def _normalize_pvc_items(payload: Any) -> list[dict[str, Any]]:
    report = _as_dict(payload)
    items = report.get("pvcs")
    if not isinstance(items, list):
        items = report.get("items")
    if not isinstance(items, list):
        items = report.get("list")
    return [_as_dict(item) for item in _as_list(items)]


def _normalize_capacity_summary(payload: Any) -> dict[str, Any]:
    report = _as_dict(payload)
    if isinstance(report.get("summary"), dict):
        return _as_dict(report.get("summary"))
    return report


def _build_storage_anomalies(pvcs: list[dict[str, Any]]) -> list[dict[str, Any]]:
    anomalies: list[dict[str, Any]] = []
    for pvc in pvcs:
        name = str(pvc.get("name") or pvc.get("pvc_name") or "").strip()
        if not name:
            continue
        namespace = str(pvc.get("namespace") or pvc.get("ns") or "default").strip()
        phase = str(pvc.get("phase") or pvc.get("status") or "").strip()
        usage_pct = _to_float(
            pvc.get("usage_pct")
            or pvc.get("usage_percent")
            or pvc.get("usage")
            or pvc.get("capacity_used_pct")
        )

        reasons: list[str] = []
        severity = "info"

        normalized_phase = phase.lower()
        if normalized_phase and normalized_phase not in {"bound", "available"}:
            reasons.append(f"PVC phase={phase}")
            severity = "critical"
        if usage_pct >= 95:
            reasons.append(f"容量使用率 {usage_pct:.1f}%")
            severity = "critical"
        elif usage_pct >= 90:
            reasons.append(f"容量使用率 {usage_pct:.1f}%")
            if severity != "critical":
                severity = "warning"

        last_event = str(pvc.get("last_event") or pvc.get("event") or "").strip()
        if last_event and any(
            token in last_event.lower()
            for token in ("fail", "error", "mount", "attach", "timeout", "denied")
        ):
            reasons.append(f"异常事件: {last_event[:120]}")
            if severity != "critical":
                severity = "warning"

        if not reasons:
            continue

        anomalies.append(
            {
                "pvc_name": name,
                "namespace": namespace,
                "phase": phase,
                "usage_pct": round(usage_pct, 1),
                "severity": severity,
                "reasons": reasons,
            }
        )
    return anomalies


def _build_recommendations(anomalies: list[dict[str, Any]]) -> list[dict[str, Any]]:
    critical = [a for a in anomalies if a.get("severity") == "critical"]
    warning = [a for a in anomalies if a.get("severity") == "warning"]
    recs: list[dict[str, Any]] = []
    if critical:
        recs.append(
            {
                "action": "investigate_critical_pvcs",
                "count": len(critical),
                "reason": "PVC phase异常或容量使用率 >= 95%",
            }
        )
    if warning:
        recs.append(
            {
                "action": "review_warning_pvcs",
                "count": len(warning),
                "reason": "PVC存在容量压力或存储事件告警",
            }
        )
    return recs


def _build_audit_items(anomalies: list[dict[str, Any]]) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    for anomaly in anomalies:
        items.append(
            {
                # save_audit_report expects job_name-like identity; use PVC key here.
                "job_name": f"pvc:{anomaly['namespace']}/{anomaly['pvc_name']}",
                "username": "system",
                "action_type": "storage_review",
                "severity": anomaly.get("severity", "warning"),
                "analysis_detail": {
                    "type": "storage_pvc",
                    "namespace": anomaly.get("namespace"),
                    "pvc_name": anomaly.get("pvc_name"),
                    "phase": anomaly.get("phase"),
                    "usage_pct": anomaly.get("usage_pct"),
                    "reasons": anomaly.get("reasons", []),
                },
            }
        )
    return items


async def run_storage_audit(
    *,
    days: int,
    pvc_limit: int,
    dry_run: bool = False,
) -> dict[str, Any]:
    """Run storage daily audit workflow and persist a storage_audit report."""
    now = datetime.now(timezone.utc)
    period_end = now.isoformat()
    period_start = (now - timedelta(days=days)).isoformat()

    async with PipelineToolClient(
        timeout=90,
        session_source="system",
        session_title="[system] 存储巡检",
    ) as client:
        pvc_result = await client.execute("list_storage_pvcs", {"limit": pvc_limit})
        if pvc_result.get("status") != "success":
            return {
                "status": "failed",
                "summary": {
                    "error": pvc_result.get("message", "Failed to list storage pvcs"),
                },
            }

        capacity_result = await client.execute(
            "get_storage_capacity_overview",
            {"days": days},
        )
        capacity_payload = {}
        if capacity_result.get("status") == "success":
            capacity_payload = _as_dict(capacity_result.get("result"))
        else:
            logger.warning(
                "storage-audit: get_storage_capacity_overview unavailable: %s",
                capacity_result.get("message", "unknown"),
            )

        pvs = _normalize_pvc_items(pvc_result.get("result"))
        anomalies = _build_storage_anomalies(pvs)
        recommendations = _build_recommendations(anomalies)
        capacity_summary = _normalize_capacity_summary(capacity_payload)
        high_pressure_clusters = int(
            _to_float(
                capacity_summary.get("high_pressure_clusters")
                or capacity_summary.get("pressure_clusters")
                or capacity_summary.get("high_pressure_count")
            )
        )

        summary = {
            "total_scanned": len(pvs),
            "idle_detected": len(anomalies),
            "gpu_waste_hours": float(high_pressure_clusters),
            "summary_labels": {
                "total_label": "扫描PVC",
                "middle_label": "异常PVC",
                "right_label": "容量压力集群",
            },
            "recommendations": recommendations,
            "storage_overview": {
                "high_pressure_clusters": high_pressure_clusters,
                "total_clusters": int(
                    _to_float(
                        capacity_summary.get("total_clusters")
                        or capacity_summary.get("cluster_count")
                    )
                ),
            },
            "top_anomalies": anomalies[:10],
        }
        report_json = {
            "report_type": "storage_audit",
            "generated_at": period_end,
            "summary": summary,
            "anomalies": anomalies[:50],
            "capacity": capacity_payload,
        }

        report_id: str | None = None
        if not dry_run:
            save_result = await client.execute(
                "save_audit_report",
                {
                    "report_type": "storage_audit",
                    "trigger_source": "pipeline",
                    "summary": summary,
                    "items": _build_audit_items(anomalies),
                    "report_json": report_json,
                    "period_start": period_start,
                    "period_end": period_end,
                    "job_total": len(pvs),
                    "job_failed": len(anomalies),
                },
            )
            if save_result.get("status") == "success":
                report_id = (
                    str(save_result.get("result", {}).get("report_id") or "").strip()
                    or None
                )
            else:
                logger.warning("storage-audit: save_audit_report failed: %s", save_result)

        return {
            "status": "completed" if not dry_run else "dry_run",
            "report_id": report_id,
            "summary": summary,
            "report": report_json,
        }
