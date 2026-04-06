"""GPU audit pipeline — detects idle GPU jobs and produces audit reports.

Communicates with the Crater Go backend via tool execution HTTP calls
to gather metrics and persist results.
"""

from __future__ import annotations

import logging

from crater_agent.pipeline.tool_client import PipelineToolClient

logger = logging.getLogger(__name__)


async def run_gpu_audit(
    gpu_threshold: float,
    hours: int,
    dry_run: bool = False,
) -> dict:
    """Run GPU utilization audit pipeline.

    Steps:
      1. Call ``detect_idle_jobs`` tool via Go backend.
      2. For each idle job, call ``query_job_metrics`` for detailed GPU metrics.
      3. Classify into stop / notify / downscale categories.
      4. If not *dry_run*, write report via Go backend.
    """
    async with PipelineToolClient(timeout=60) as client:
        # Step 1: Detect idle jobs
        idle_result = await client.execute(
            "detect_idle_jobs",
            {"gpu_threshold": gpu_threshold, "hours": hours},
        )

        if idle_result.get("status") != "success":
            return {
                "status": "failed",
                "summary": {
                    "error": idle_result.get("message", "Failed to detect idle jobs"),
                },
            }

        idle_jobs: list[dict] = idle_result.get("result", {}).get("idle_jobs", [])

        if not idle_jobs:
            summary = {
                "total_scanned": idle_result.get("result", {}).get("total_scanned", 0),
                "idle_detected": 0,
                "estimated_gpu_waste_hours": 0,
                "recommendations": [],
            }
            return {
                "status": "completed" if not dry_run else "dry_run",
                "report_id": None,
                "summary": summary,
            }

        # Step 2: Deep analysis for each idle job
        audit_items: list[dict] = []
        for job in idle_jobs:
            job_name = job.get("job_name", "")
            metrics_result = await _execute_tool(
                client,
                "query_job_metrics",
                {
                    "job_name": job_name,
                    "metrics": ["gpu_utilization", "gpu_memory"],
                    "time_range": f"last_{hours}h",
                },
            )

            gpu_util: float = job.get("gpu_utilization", 0)
            gpu_requested: int = job.get("gpu_count", 1)

            item: dict = {
                "job_name": job_name,
                "user_id": job.get("user_id"),
                "account_id": job.get("account_id"),
                "username": job.get("username", ""),
                "gpu_utilization": gpu_util,
                "gpu_requested": gpu_requested,
                "gpu_actual_used": _estimate_actual_gpu(gpu_util, gpu_requested),
                "analysis_detail": {
                    "metrics_summary": _summarize_metrics(metrics_result),
                    "idle_duration_hours": job.get("idle_hours", hours),
                },
            }

            # Classify based on utilization / duration
            idle_hours = job.get("idle_hours", 0)
            if gpu_util < 1.0 and idle_hours >= 24:
                item["action_type"] = "stop"
                item["severity"] = "critical"
            elif gpu_util < gpu_threshold and idle_hours >= 12:
                item["action_type"] = "notify"
                item["severity"] = "warning"
            elif (
                gpu_requested > 1
                and _estimate_actual_gpu(gpu_util, gpu_requested) < gpu_requested
            ):
                item["action_type"] = "downscale"
                item["severity"] = "info"
            else:
                item["action_type"] = "notify"
                item["severity"] = "info"

            audit_items.append(item)

        # Build summary
        total_scanned = idle_result.get("result", {}).get("summary", {}).get(
            "total_running_jobs", len(idle_jobs)
        )
        waste_hours = sum(
            i.get("gpu_requested", 1)
            * i.get("analysis_detail", {}).get("idle_duration_hours", 0)
            for i in audit_items
        )

        recommendations: list[dict] = []
        reasons = {
            "stop": "GPU利用率 < 1% 超过 24h",
            "notify": f"GPU利用率 < {gpu_threshold}% 超过 12h",
            "downscale": "申请多卡实际用少卡",
        }
        for action in ("stop", "notify", "downscale"):
            items_for_action = [i for i in audit_items if i["action_type"] == action]
            if items_for_action:
                recommendations.append({
                    "action": action,
                    "count": len(items_for_action),
                    "reason": reasons[action],
                })

        summary = {
            "total_scanned": total_scanned,
            "idle_detected": len(audit_items),
            "estimated_gpu_waste_hours": round(waste_hours, 1),
            "recommendations": recommendations,
        }

        report_id: str | None = None
        if not dry_run:
            report_result = await _execute_tool(
                client,
                "save_audit_report",
                {
                    "report_type": "gpu_audit",
                    "trigger_source": "pipeline",
                    "summary": summary,
                    "items": audit_items,
                },
            )
            report_id = report_result.get("result", {}).get("report_id")

        return {
            "status": "completed" if not dry_run else "dry_run",
            "report_id": report_id,
            "summary": summary,
        }


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------


async def _execute_tool(
    client: PipelineToolClient,
    tool_name: str,
    tool_args: dict,
) -> dict:
    """Execute a tool via the shared internal pipeline client."""
    return await client.execute(tool_name, tool_args)


def _estimate_actual_gpu(gpu_util: float, gpu_requested: int) -> int:
    """Estimate actual GPU cards being used based on utilization."""
    if gpu_util < 1.0:
        return 0
    if gpu_requested <= 1:
        return 1
    # Rough estimate: if util is 25% with 4 GPUs, likely using 1
    estimated = max(1, round(gpu_util / 100.0 * gpu_requested))
    return min(estimated, gpu_requested)


def _summarize_metrics(metrics_result: dict) -> dict:
    """Summarize a metrics result for inclusion in an audit item."""
    if metrics_result.get("status") != "success":
        return {"error": "metrics_unavailable"}
    result = metrics_result.get("result", {})
    return {
        "gpu_utilization_avg": result.get("gpu_utilization_avg"),
        "gpu_utilization_max": result.get("gpu_utilization_max"),
        "gpu_memory_avg": result.get("gpu_memory_avg"),
    }
