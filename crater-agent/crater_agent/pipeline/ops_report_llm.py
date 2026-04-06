"""LLM-based analysis for admin ops reports."""

from __future__ import annotations

import json
import logging
from typing import Any

from crater_agent.llm.client import ModelClientFactory

logger = logging.getLogger(__name__)

ANALYSIS_SYSTEM_PROMPT = """你是一位集群运维分析师。根据提供的集群任务数据，生成结构化的运维分析报告。

你必须输出一个严格的JSON对象，包含以下字段：
{
  "executive_summary": "2-3句话的总体概述",
  "job_overview": {
    "total": <int>,
    "success": <int>,
    "failed": <int>,
    "pending": <int>,
    "success_rate": <float>,
    "delta": {"total": <int>, "failed": <int>, "pending": <int>}
  },
  "failure_analysis": {
    "categories": [
      {"reason": "<失败类型>", "count": <int>, "top_job": {"name": "<作业名>", "owner": "<用户>"}}
    ],
    "top_affected_users": ["<用户名>"],
    "patterns": "<失败模式分析文本>"
  },
  "success_analysis": {
    "avg_duration_by_type": {"<类型>": <秒数>},
    "resource_efficiency": {"avg_cpu_ratio": <float>, "avg_gpu_ratio": <float>, "avg_memory_ratio": <float>}
  },
  "resource_utilization": {
    "cluster_gpu_avg": <float>,
    "cluster_cpu_avg": <float>,
    "cluster_memory_avg": <float>,
    "over_provisioned_count": <int>,
    "idle_gpu_jobs": <int>,
    "node_hotspots": ["<节点名>"]
  },
  "recommendations": [
    {"severity": "high|medium|low", "text": "<建议文本>"}
  ]
}

注意事项：
- delta字段表示与上一份报告的对比变化，正数表示增加，负数表示减少
- 如果没有上一份报告，delta全部为0
- recommendations中severity为high表示需要立即关注
- 所有分析必须基于提供的数据，不要编造
- 如果某项数据缺失，用合理的默认值（0、空列表等）
"""


def _build_analysis_prompt(
    raw_report: dict[str, Any],
    previous_report: dict[str, Any] | None = None,
) -> str:
    """Build the user prompt with cluster data for LLM analysis."""
    overview = raw_report.get("overview", {})
    parts = [
        "## 当前周期集群数据\n",
        f"- 总作业数: {overview.get('total_jobs', 0)}",
        f"- 成功作业: {overview.get('success_jobs', overview.get('completed_jobs', 0))}",
        f"- 失败作业: {overview.get('failed_jobs', 0)}",
        f"- 等待中: {overview.get('pending_jobs', 0)}",
        f"- 运行中: {overview.get('running_jobs', 0)}",
        f"- 成功率: {overview.get('success_rate', 'N/A')}",
        f"- 失败率: {overview.get('failure_rate', 'N/A')}",
    ]

    # failure_categories is a list of {category, count, samples}, NOT a dict
    fail_cats = raw_report.get("failure_categories", [])
    if fail_cats and isinstance(fail_cats, list):
        parts.append("\n## 失败分类")
        for entry in fail_cats:
            if isinstance(entry, dict):
                parts.append(f"- {entry.get('category', 'unknown')}: {entry.get('count', 0)} (样例: {', '.join(entry.get('samples', [])[:3])})")
    elif isinstance(fail_cats, dict):
        parts.append("\n## 失败分类")
        for cat, count in fail_cats.items():
            parts.append(f"- {cat}: {count}")

    # failed_jobs fields: job_name, name, user, failure_category, exit_code, exit_reason, etc.
    failed_jobs = raw_report.get("failed_jobs", [])
    if failed_jobs:
        parts.append(f"\n## 失败作业样例 (共{len(failed_jobs)}个)")
        for job in failed_jobs[:10]:
            parts.append(
                f"- {job.get('job_name', job.get('jobName', 'unknown'))}: "
                f"user={job.get('user', job.get('owner', 'N/A'))}, "
                f"exitCode={job.get('exit_code', job.get('exitCode', 'N/A'))}, "
                f"category={job.get('failure_category', job.get('failureReason', 'N/A'))}, "
                f"gpu={job.get('gpu_requested', job.get('gpuCount', 0))}, "
                f"node={job.get('scheduled_node', 'N/A')}"
            )

    # successful_jobs fields: job_name, name, user, actual_usage, running_minutes, etc.
    success_jobs = raw_report.get("successful_jobs", [])
    if success_jobs:
        parts.append(f"\n## 成功作业样例 (共{len(success_jobs)}个)")
        for job in success_jobs[:10]:
            actual = job.get("actual_usage", {}) if isinstance(job.get("actual_usage"), dict) else {}
            gpu_util = actual.get("gpu_util_avg", "N/A")
            cpu_usage = actual.get("cpu_usage_avg", "N/A")
            mem_usage = actual.get("mem_usage_avg", "N/A")
            parts.append(
                f"- {job.get('job_name', job.get('jobName', 'unknown'))}: "
                f"user={job.get('user', 'N/A')}, "
                f"运行{job.get('running_minutes', 'N/A')}分钟, "
                f"GPU请求={job.get('gpu_requested', 0)}, "
                f"GPU利用率={gpu_util}%, CPU={cpu_usage}%, 内存={mem_usage}%"
            )

    idle = raw_report.get("idle_summary", {})
    if idle:
        parts.append("\n## 空闲资源情况")
        parts.append(f"- 空闲作业数: {idle.get('idle_job_count', 0)}")
        parts.append(f"- 预估GPU浪费时: {idle.get('estimated_gpu_waste_hours', 0)}")

    node_summary = raw_report.get("node_summary", {})
    if node_summary:
        parts.append("\n## 节点概况")
        parts.append(f"- 总节点: {node_summary.get('total_nodes', 0)}")
        parts.append(f"- 就绪节点: {node_summary.get('ready_nodes', 0)}")
        parts.append(f"- GPU总量: {node_summary.get('total_gpus', 0)}")
        parts.append(f"- GPU已分配: {node_summary.get('allocated_gpus', 0)}")

    running = raw_report.get("recent_running_summary", {})
    if running:
        parts.append(f"\n## 近期运行概况 (最近{raw_report.get('lookback_hours', 1)}小时)")
        parts.append(f"- 运行中作业: {running.get('running_count', 0)}")
        parts.append(f"- 总GPU使用: {running.get('total_gpu', 0)}")

    if previous_report:
        prev_overview = previous_report.get("job_overview", {})
        parts.append("\n## 上一份报告数据（用于对比）")
        parts.append(f"- 总作业: {prev_overview.get('total', 0)}")
        parts.append(f"- 失败: {prev_overview.get('failed', 0)}")
        parts.append(f"- 等待: {prev_overview.get('pending', 0)}")
    else:
        parts.append("\n## 上一份报告: 无（首次生成）")

    return "\n".join(parts)


async def analyze_ops_report_with_llm(
    raw_report: dict[str, Any],
    previous_report_json: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Use LLM to analyze raw report data and produce structured report_json.

    Falls back to a deterministic report if LLM is unavailable or slow.
    """
    import asyncio

    try:
        factory = ModelClientFactory()
        llm = factory.create(purpose="ops_report", orchestration_mode="single_agent")

        user_prompt = _build_analysis_prompt(raw_report, previous_report_json)

        messages = [
            {"role": "system", "content": ANALYSIS_SYSTEM_PROMPT},
            {"role": "user", "content": user_prompt},
        ]

        # Hard cap at 45s — if LLM is slow, fallback is better than blocking the pipeline
        response = await asyncio.wait_for(llm.ainvoke(messages), timeout=45)
        content = response.content if hasattr(response, "content") else str(response)

        # Parse JSON from LLM response
        if "```json" in content:
            content = content.split("```json")[1].split("```")[0].strip()
        elif "```" in content:
            content = content.split("```")[1].split("```")[0].strip()

        report_json = json.loads(content)
        return report_json

    except asyncio.TimeoutError:
        logger.warning("LLM analysis timed out after 45s, using fallback report")
        return _build_fallback_report(raw_report)
    except Exception as e:
        logger.warning("LLM analysis failed: %s, using fallback report", e)
        return _build_fallback_report(raw_report)


def _build_fallback_report(raw_report: dict[str, Any]) -> dict[str, Any]:
    """Generate a basic structured report when LLM is unavailable."""
    overview = raw_report.get("overview", {})
    total = int(overview.get("total_jobs", 0))
    success = int(overview.get("success_jobs", overview.get("completed_jobs", 0)))
    failed = int(overview.get("failed_jobs", 0))
    pending = int(overview.get("pending_jobs", 0))
    rate = round(success / total * 100, 1) if total > 0 else 0.0

    # failure_categories is a list of {category, count, samples}
    fail_cats = raw_report.get("failure_categories", [])
    categories = []
    if isinstance(fail_cats, list):
        for entry in sorted(fail_cats, key=lambda x: -(x.get("count", 0) if isinstance(x, dict) else 0)):
            if isinstance(entry, dict):
                samples = entry.get("samples", [])
                top_name = samples[0] if samples else ""
                categories.append({
                    "reason": entry.get("category", "unknown"),
                    "count": entry.get("count", 0),
                    "top_job": {"name": top_name, "owner": ""},
                })
    elif isinstance(fail_cats, dict):
        categories = [
            {"reason": reason, "count": count, "top_job": {"name": "", "owner": ""}}
            for reason, count in sorted(fail_cats.items(), key=lambda x: -x[1])
        ]

    idle = raw_report.get("idle_summary", {})

    return {
        "executive_summary": f"本周期共{total}个作业，成功{success}个，失败{failed}个，成功率{rate}%。",
        "job_overview": {
            "total": total, "success": success, "failed": failed, "pending": pending,
            "success_rate": rate, "delta": {"total": 0, "failed": 0, "pending": 0},
        },
        "failure_analysis": {"categories": categories, "top_affected_users": [], "patterns": ""},
        "success_analysis": {
            "avg_duration_by_type": {},
            "resource_efficiency": {"avg_cpu_ratio": 0, "avg_gpu_ratio": 0, "avg_memory_ratio": 0},
        },
        "resource_utilization": {
            "cluster_gpu_avg": 0, "cluster_cpu_avg": 0, "cluster_memory_avg": 0,
            "over_provisioned_count": 0, "idle_gpu_jobs": int(idle.get("idle_job_count", 0)),
            "node_hotspots": [],
        },
        "recommendations": [],
    }
