"""Pipeline API router — exposes automated audit endpoints."""

from __future__ import annotations

import logging
from typing import Optional

from fastapi import APIRouter, Header, HTTPException
from pydantic import BaseModel, Field

from crater_agent.config import settings
from crater_agent.pipeline.gpu_analyzer import run_gpu_audit
from crater_agent.pipeline.ops_report import run_admin_ops_report

logger = logging.getLogger(__name__)

pipeline_router = APIRouter()


# ---------------------------------------------------------------------------
# Request / Response models
# ---------------------------------------------------------------------------


class GpuAuditRequest(BaseModel):
    """Parameters for a GPU idle-job audit run."""

    gpu_threshold: float = Field(
        default=5.0,
        description="GPU utilization threshold (%). Jobs below this are considered idle.",
    )
    hours: int = Field(
        default=24,
        description="Detection window in hours.",
    )
    auto_notify: bool = Field(
        default=False,
        description="Reserved — automatic notification is not yet implemented.",
    )
    dry_run: bool = Field(
        default=False,
        description="Detect only; do not persist an audit report.",
    )


class GpuAuditResponse(BaseModel):
    """Result of a GPU audit run."""

    report_id: Optional[str] = None
    status: str
    summary: dict


class AdminOpsReportRequest(BaseModel):
    """Parameters for a scheduled admin ops analysis report."""

    days: int = Field(default=1, description="Lookback window in days.")
    lookback_hours: int = Field(default=1, description="Recent running-job window in hours.")
    gpu_threshold: int = Field(default=5, description="Idle-job GPU threshold (%).")
    idle_hours: int = Field(default=1, description="Idle-job time window in hours.")
    running_limit: int = Field(default=20, description="Number of recent running jobs to keep.")
    node_limit: int = Field(default=10, description="Number of node snapshots to keep.")
    dry_run: bool = Field(default=False, description="Analyze only; do not persist a report.")
    use_llm: bool = Field(default=True, description="Use LLM for executive analysis.")


class AdminOpsReportResponse(BaseModel):
    """Result of an admin ops report run."""

    report_id: Optional[str] = None
    status: str
    summary: dict
    report: dict = Field(default_factory=dict)


# ---------------------------------------------------------------------------
# Endpoint
# ---------------------------------------------------------------------------


@pipeline_router.post("/gpu-audit", response_model=GpuAuditResponse)
async def gpu_audit(
    request: GpuAuditRequest,
    x_agent_internal_token: str = Header(..., alias="X-Agent-Internal-Token"),
) -> GpuAuditResponse:
    """Run the GPU utilization audit pipeline.

    Requires a valid ``X-Agent-Internal-Token`` header that matches the
    configured backend internal token.
    """
    if x_agent_internal_token != settings.crater_backend_internal_token:
        raise HTTPException(status_code=403, detail="Invalid internal token")

    result = await run_gpu_audit(
        gpu_threshold=request.gpu_threshold,
        hours=request.hours,
        dry_run=request.dry_run,
    )

    return GpuAuditResponse(
        report_id=result.get("report_id"),
        status=result.get("status", "failed"),
        summary=result.get("summary", {}),
    )


@pipeline_router.post("/admin-ops-report", response_model=AdminOpsReportResponse)
async def admin_ops_report(
    request: AdminOpsReportRequest,
    x_agent_internal_token: str = Header(..., alias="X-Agent-Internal-Token"),
) -> AdminOpsReportResponse:
    """Run the scheduled admin ops report pipeline."""
    if x_agent_internal_token != settings.crater_backend_internal_token:
        raise HTTPException(status_code=403, detail="Invalid internal token")

    try:
        result = await run_admin_ops_report(
            days=request.days,
            lookback_hours=request.lookback_hours,
            gpu_threshold=request.gpu_threshold,
            idle_hours=request.idle_hours,
            running_limit=request.running_limit,
            node_limit=request.node_limit,
            dry_run=request.dry_run,
            use_llm=request.use_llm,
        )
        return AdminOpsReportResponse(
            report_id=result.get("report_id"),
            status=result.get("status", "failed"),
            summary=result.get("summary", {}),
            report=result.get("report", {}),
        )
    except Exception as e:
        logger.exception("admin-ops-report pipeline failed: %s", e)
        return AdminOpsReportResponse(
            report_id=None,
            status="failed",
            summary={"error": str(e)},
            report={},
        )
