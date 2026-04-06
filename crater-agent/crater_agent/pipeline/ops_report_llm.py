"""LLM-assisted analysis for admin ops reports."""

from __future__ import annotations

import json
import logging
from collections import Counter, defaultdict
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
- 数值字段必须与输入数据一致，不要自行估算或改写
- delta字段表示与上一份报告的对比变化，正数表示增加，负数表示减少
- 如果没有上一份报告，delta全部为0
- recommendations中severity为high表示需要立即关注
- 所有分析必须基于提供的数据，不要编造
- 如果某项数据缺失，用合理的默认值（0、空列表等）
"""


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


def _avg(values: list[float]) -> float:
    if not values:
        return 0.0
    return round(sum(values) / len(values), 3)


def _normalize_percentage(value: Any, fallback_total: int = 0, fallback_hit: int = 0) -> float:
    numeric = _to_float(value)
    if numeric <= 1.0 and fallback_total > 0:
        numeric = numeric * 100.0
    if numeric <= 0.0 and fallback_total > 0 and fallback_hit >= 0:
        numeric = (fallback_hit / fallback_total) * 100.0
    return round(numeric, 1)


def _normalize_failure_categories(raw_report: dict[str, Any]) -> list[dict[str, Any]]:
    categories: list[dict[str, Any]] = []
    failed_jobs = _as_list(raw_report.get("failed_jobs"))
    owner_by_job = {
        str(job.get("job_name") or "").strip(): str(job.get("user") or "").strip()
        for job in failed_jobs
        if str(job.get("job_name") or "").strip()
    }

    fail_cats = raw_report.get("failure_categories", [])
    if isinstance(fail_cats, list):
        for entry in sorted(
            _as_list(fail_cats),
            key=lambda item: -_to_int(_as_dict(item).get("count")),
        ):
            item = _as_dict(entry)
            samples = [str(sample).strip() for sample in _as_list(item.get("samples")) if str(sample).strip()]
            top_name = samples[0] if samples else ""
            categories.append(
                {
                    "reason": str(item.get("category") or "unknown").strip() or "unknown",
                    "count": _to_int(item.get("count")),
                    "top_job": {"name": top_name, "owner": owner_by_job.get(top_name, "")},
                }
            )
    elif isinstance(fail_cats, dict):
        categories = [
            {"reason": str(reason), "count": _to_int(count), "top_job": {"name": "", "owner": ""}}
            for reason, count in sorted(fail_cats.items(), key=lambda item: -_to_int(item[1]))
        ]

    return categories


def _build_top_affected_users(raw_report: dict[str, Any]) -> list[str]:
    counter: Counter[str] = Counter()
    for job in _as_list(raw_report.get("failed_jobs")):
        user = str(_as_dict(job).get("user") or _as_dict(job).get("owner") or "").strip()
        if user:
            counter[user] += 1
    return [name for name, _ in counter.most_common(5)]


def _build_failure_patterns(raw_report: dict[str, Any], categories: list[dict[str, Any]]) -> str:
    if not categories:
        return ""

    fragments: list[str] = []
    lead = categories[0]
    fragments.append(f"失败主要集中在{lead['reason']}，共{lead['count']}个作业")

    if len(categories) > 1:
        secondary = categories[1]
        fragments.append(f"其次是{secondary['reason']}，共{secondary['count']}个作业")

    top_users = _build_top_affected_users(raw_report)
    if top_users:
        fragments.append(f"受影响较多的用户包括{', '.join(top_users[:3])}")

    exit_code_counter: Counter[str] = Counter()
    node_counter: Counter[str] = Counter()
    for raw_job in _as_list(raw_report.get("failed_jobs")):
        job = _as_dict(raw_job)
        exit_code = str(job.get("exit_code") or "").strip()
        if exit_code:
            exit_code_counter[exit_code] += 1
        node_name = str(job.get("scheduled_node") or "").strip()
        if node_name:
            node_counter[node_name] += 1

    if exit_code_counter:
        exit_code, count = exit_code_counter.most_common(1)[0]
        fragments.append(f"高频退出码为{exit_code}，出现{count}次")
    if node_counter:
        node_name, count = node_counter.most_common(1)[0]
        fragments.append(f"失败较集中在节点{node_name}，涉及{count}个作业")

    return "；".join(fragments) + "。"


def _build_success_analysis(raw_report: dict[str, Any]) -> dict[str, Any]:
    success_jobs = _as_list(raw_report.get("successful_jobs"))
    duration_by_type: defaultdict[str, list[float]] = defaultdict(list)
    cpu_ratios: list[float] = []
    gpu_ratios: list[float] = []
    memory_ratios: list[float] = []

    for raw_job in success_jobs:
        job = _as_dict(raw_job)
        job_type = str(job.get("job_type") or "unknown").strip() or "unknown"
        running_minutes = _to_int(job.get("running_minutes"))
        if running_minutes > 0:
            duration_by_type[job_type].append(float(running_minutes * 60))

        actual = _as_dict(job.get("actual_usage"))
        cpu_ratios.append(round(_to_float(actual.get("cpu_usage_avg")) / 100.0, 3))
        gpu_ratios.append(round(_to_float(actual.get("gpu_util_avg")) / 100.0, 3))
        memory_ratios.append(round(_to_float(actual.get("mem_usage_avg")) / 100.0, 3))

    return {
        "avg_duration_by_type": {
            job_type: round(sum(durations) / len(durations), 1)
            for job_type, durations in sorted(duration_by_type.items())
            if durations
        },
        "resource_efficiency": {
            "avg_cpu_ratio": _avg(cpu_ratios),
            "avg_gpu_ratio": _avg(gpu_ratios),
            "avg_memory_ratio": _avg(memory_ratios),
        },
    }


def _estimate_over_provisioned_count(raw_report: dict[str, Any]) -> int:
    count = 0
    for raw_job in _as_list(raw_report.get("successful_jobs")):
        job = _as_dict(raw_job)
        requested = _to_int(job.get("gpu_requested"))
        actual = _to_int(job.get("gpu_actual_used"))
        actual_usage = _as_dict(job.get("actual_usage"))
        gpu_util = _to_float(actual_usage.get("gpu_util_avg"))
        if requested > 1 and (requested - actual >= 1 or gpu_util < 40.0):
            count += 1
    return count


def _build_resource_utilization(raw_report: dict[str, Any]) -> dict[str, Any]:
    base = _as_dict(raw_report.get("resource_utilization"))
    idle = _as_dict(raw_report.get("idle_summary"))
    node_hotspots = [
        str(node).strip()
        for node in _as_list(base.get("node_hotspots"))
        if str(node).strip()
    ]

    return {
        "cluster_gpu_avg": round(_to_float(base.get("cluster_gpu_avg")), 1),
        "cluster_cpu_avg": round(_to_float(base.get("cluster_cpu_avg")), 1),
        "cluster_memory_avg": round(_to_float(base.get("cluster_memory_avg")), 1),
        "over_provisioned_count": _estimate_over_provisioned_count(raw_report),
        "idle_gpu_jobs": _to_int(base.get("idle_gpu_jobs") or idle.get("idle_job_count")),
        "node_hotspots": node_hotspots,
    }


def _build_recommendations(raw_report: dict[str, Any], resource_utilization: dict[str, Any]) -> list[dict[str, Any]]:
    overview = _as_dict(raw_report.get("overview"))
    idle = _as_dict(raw_report.get("idle_summary"))

    total = _to_int(overview.get("total_jobs"))
    failed = _to_int(overview.get("failed_jobs"))
    pending = _to_int(overview.get("pending_jobs"))
    failure_pct = round((failed / total) * 100.0, 1) if total > 0 else 0.0
    idle_jobs = _to_int(idle.get("idle_job_count"))
    waste_hours = round(_to_float(idle.get("estimated_gpu_waste_hours")), 1)
    gpu_avg = _to_float(resource_utilization.get("cluster_gpu_avg"))
    over_provisioned = _to_int(resource_utilization.get("over_provisioned_count"))
    hotspots = _as_list(resource_utilization.get("node_hotspots"))

    recommendations: list[dict[str, Any]] = []
    if failure_pct >= 15 or failed >= 10:
        recommendations.append(
            {
                "severity": "high",
                "text": f"失败率已达{failure_pct}%，建议优先排查高频失败类型并回放最近失败作业的退出码与节点分布。",
            }
        )
    elif failed > 0:
        recommendations.append(
            {
                "severity": "medium",
                "text": "存在失败作业，建议优先收敛高频失败类型，补齐镜像、配额和节点侧的共性诊断规则。",
            }
        )

    if idle_jobs > 0:
        recommendations.append(
            {
                "severity": "high" if waste_hours >= 20 else "medium",
                "text": f"检测到{idle_jobs}个低利用率作业，预估浪费{waste_hours} GPU 小时，建议优先通知负责人并执行缩容/释放策略。",
            }
        )

    if over_provisioned > 0:
        recommendations.append(
            {
                "severity": "medium",
                "text": f"至少{over_provisioned}个成功样本存在显著资源过配，建议把历史画像回灌到提交侧做资源推荐与限额提醒。",
            }
        )

    if gpu_avg >= 90 or hotspots:
        hotspot_text = f"，热点节点包括{', '.join(str(node) for node in hotspots[:3])}" if hotspots else ""
        recommendations.append(
            {
                "severity": "high",
                "text": f"当前集群资源压力较高，GPU 平均利用率为{gpu_avg:.1f}%{hotspot_text}，建议优先做容量疏导和异常节点巡检。",
            }
        )

    if pending > 0:
        recommendations.append(
            {
                "severity": "medium",
                "text": f"当前仍有{pending}个等待中作业，建议结合配额、节点热点和空闲作业治理结果评估排队瓶颈。",
            }
        )

    return recommendations[:5]


def build_deterministic_ops_report(
    raw_report: dict[str, Any],
    previous_report_json: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Build a deterministic structured report used as the source of truth."""
    overview = _as_dict(raw_report.get("overview"))
    total = _to_int(overview.get("total_jobs"))
    success = _to_int(overview.get("success_jobs", overview.get("completed_jobs")))
    failed = _to_int(overview.get("failed_jobs"))
    pending = _to_int(overview.get("pending_jobs"))
    success_rate = _normalize_percentage(overview.get("success_rate"), total, success)

    previous_overview = _as_dict(_as_dict(previous_report_json).get("job_overview"))
    delta = {
        "total": total - _to_int(previous_overview.get("total")),
        "failed": failed - _to_int(previous_overview.get("failed")),
        "pending": pending - _to_int(previous_overview.get("pending")),
    }
    if not previous_overview:
        delta = {"total": 0, "failed": 0, "pending": 0}

    categories = _normalize_failure_categories(raw_report)
    top_users = _build_top_affected_users(raw_report)
    patterns = _build_failure_patterns(raw_report, categories)
    success_analysis = _build_success_analysis(raw_report)
    resource_utilization = _build_resource_utilization(raw_report)
    recommendations = _build_recommendations(raw_report, resource_utilization)

    idle = _as_dict(raw_report.get("idle_summary"))
    idle_jobs = _to_int(idle.get("idle_job_count"))
    waste_hours = round(_to_float(idle.get("estimated_gpu_waste_hours")), 1)
    hotspot_nodes = _as_list(resource_utilization.get("node_hotspots"))
    lookback_days = _to_int(raw_report.get("lookback_days"))

    executive_parts = [
        f"近{lookback_days or 1}天共{total}个作业，成功{success}个，失败{failed}个，成功率{success_rate:.1f}%。",
    ]
    if idle_jobs > 0:
        executive_parts.append(f"检测到{idle_jobs}个低利用率作业，预估浪费{waste_hours} GPU 小时。")
    if hotspot_nodes:
        executive_parts.append(f"资源热点集中在{', '.join(str(node) for node in hotspot_nodes[:3])}。")

    return {
        "executive_summary": " ".join(executive_parts),
        "job_overview": {
            "total": total,
            "success": success,
            "failed": failed,
            "pending": pending,
            "success_rate": success_rate,
            "delta": delta,
        },
        "failure_analysis": {
            "categories": categories,
            "top_affected_users": top_users,
            "patterns": patterns,
        },
        "success_analysis": success_analysis,
        "resource_utilization": resource_utilization,
        "recommendations": recommendations,
    }


def _normalize_recommendations(value: Any) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    for raw in _as_list(value):
        item = _as_dict(raw)
        text = str(item.get("text") or "").strip()
        severity = str(item.get("severity") or "low").strip().lower()
        if not text:
            continue
        if severity not in {"high", "medium", "low"}:
            severity = "low"
        items.append({"severity": severity, "text": text})
    return items


def _merge_llm_report(base_report: dict[str, Any], llm_report: dict[str, Any]) -> dict[str, Any]:
    llm_data = _as_dict(llm_report)
    merged = dict(base_report)

    executive_summary = str(llm_data.get("executive_summary") or "").strip()
    if executive_summary:
        merged["executive_summary"] = executive_summary

    merged_failure = dict(_as_dict(base_report.get("failure_analysis")))
    llm_failure = _as_dict(llm_data.get("failure_analysis"))
    patterns = str(llm_failure.get("patterns") or "").strip()
    if patterns:
        merged_failure["patterns"] = patterns

    llm_top_users = [
        str(user).strip()
        for user in _as_list(llm_failure.get("top_affected_users"))
        if str(user).strip()
    ]
    if llm_top_users:
        merged_failure["top_affected_users"] = llm_top_users[:5]
    merged["failure_analysis"] = merged_failure

    llm_recommendations = _normalize_recommendations(llm_data.get("recommendations"))
    if llm_recommendations:
        merged["recommendations"] = llm_recommendations

    return merged


def _build_analysis_prompt(
    raw_report: dict[str, Any],
    previous_report: dict[str, Any] | None = None,
) -> str:
    """Build the user prompt with cluster data for LLM analysis."""
    deterministic = build_deterministic_ops_report(raw_report, previous_report)
    overview = _as_dict(raw_report.get("overview"))
    idle = _as_dict(raw_report.get("idle_summary"))
    node_summary = _as_dict(raw_report.get("node_summary"))
    running = _as_dict(raw_report.get("recent_running_summary"))
    resource = _as_dict(raw_report.get("resource_utilization"))

    parts = [
        "## 当前周期集群数据\n",
        f"- 总作业数: {_to_int(overview.get('total_jobs'))}",
        f"- 成功作业: {_to_int(overview.get('success_jobs', overview.get('completed_jobs')))}",
        f"- 失败作业: {_to_int(overview.get('failed_jobs'))}",
        f"- 等待中: {_to_int(overview.get('pending_jobs'))}",
        f"- 运行中: {_to_int(overview.get('running_jobs'))}",
        f"- 成功率: {deterministic['job_overview']['success_rate']:.1f}%",
        f"- 低利用率作业: {_to_int(idle.get('idle_job_count'))}",
        f"- 预估 GPU 浪费时: {round(_to_float(idle.get('estimated_gpu_waste_hours')), 1)}",
    ]

    failure_categories = deterministic["failure_analysis"]["categories"]
    if failure_categories:
        parts.append("\n## 失败分类")
        for entry in failure_categories[:10]:
            top_job = _as_dict(entry.get("top_job"))
            parts.append(
                f"- {entry.get('reason', 'unknown')}: {entry.get('count', 0)}"
                f" (代表作业: {top_job.get('name', '-')}, 负责人: {top_job.get('owner', '-')})"
            )

    failed_jobs = _as_list(raw_report.get("failed_jobs"))
    if failed_jobs:
        parts.append(f"\n## 失败作业样例 (共{len(failed_jobs)}个)")
        for job in failed_jobs[:10]:
            item = _as_dict(job)
            parts.append(
                f"- {item.get('job_name', 'unknown')}: "
                f"user={item.get('user', 'N/A')}, "
                f"type={item.get('job_type', 'N/A')}, "
                f"exitCode={item.get('exit_code', 'N/A')}, "
                f"category={item.get('failure_category', 'N/A')}, "
                f"gpu={item.get('gpu_requested', 0)}, "
                f"node={item.get('scheduled_node', 'N/A')}"
            )

    success_jobs = _as_list(raw_report.get("successful_jobs"))
    if success_jobs:
        parts.append(f"\n## 成功作业样例 (共{len(success_jobs)}个)")
        for job in success_jobs[:10]:
            item = _as_dict(job)
            actual = _as_dict(item.get("actual_usage"))
            parts.append(
                f"- {item.get('job_name', 'unknown')}: "
                f"user={item.get('user', 'N/A')}, "
                f"type={item.get('job_type', 'N/A')}, "
                f"运行{item.get('running_minutes', 'N/A')}分钟, "
                f"GPU请求={item.get('gpu_requested', 0)}, "
                f"GPU实际估计={item.get('gpu_actual_used', 0)}, "
                f"GPU利用率={actual.get('gpu_util_avg', 'N/A')}%, "
                f"CPU={actual.get('cpu_usage_avg', 'N/A')}%, "
                f"内存={actual.get('mem_usage_avg', 'N/A')}%"
            )

    parts.append("\n## 节点与资源")
    parts.append(f"- 总节点: {_to_int(node_summary.get('total_nodes') or resource.get('total_nodes'))}")
    parts.append(f"- 就绪节点: {_to_int(node_summary.get('ready_nodes') or resource.get('ready_nodes'))}")
    parts.append(f"- GPU总量: {_to_int(node_summary.get('total_gpus') or resource.get('total_gpus'))}")
    parts.append(f"- GPU已分配: {_to_int(node_summary.get('allocated_gpus') or resource.get('allocated_gpus'))}")
    parts.append(f"- GPU平均利用率: {round(_to_float(resource.get('cluster_gpu_avg')), 1)}%")
    parts.append(f"- CPU平均利用率: {round(_to_float(resource.get('cluster_cpu_avg')), 1)}%")
    parts.append(f"- 内存平均利用率: {round(_to_float(resource.get('cluster_memory_avg')), 1)}%")
    node_hotspots = [str(node).strip() for node in _as_list(resource.get("node_hotspots")) if str(node).strip()]
    parts.append(f"- 热点节点: {', '.join(node_hotspots) if node_hotspots else '无'}")

    parts.append(f"\n## 近期运行概况 (最近{_to_int(raw_report.get('lookback_hours')) or 1}小时)")
    parts.append(f"- 运行窗口作业数: {_to_int(running.get('job_count') or running.get('running_count'))}")
    parts.append(f"- 运行窗口申请GPU总数: {_to_int(running.get('total_gpu'))}")

    if previous_report:
        prev_overview = _as_dict(previous_report.get("job_overview"))
        parts.append("\n## 上一份报告数据（用于对比）")
        parts.append(f"- 总作业: {_to_int(prev_overview.get('total'))}")
        parts.append(f"- 失败: {_to_int(prev_overview.get('failed'))}")
        parts.append(f"- 等待: {_to_int(prev_overview.get('pending'))}")
    else:
        parts.append("\n## 上一份报告: 无（首次生成）")

    return "\n".join(parts)


async def analyze_ops_report_with_llm(
    raw_report: dict[str, Any],
    previous_report_json: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Use LLM to add textual analysis on top of deterministic report data."""
    import asyncio

    deterministic_report = build_deterministic_ops_report(raw_report, previous_report_json)

    try:
        factory = ModelClientFactory()
        llm = factory.create("ops_report")

        user_prompt = _build_analysis_prompt(raw_report, previous_report_json)
        messages = [
            {"role": "system", "content": ANALYSIS_SYSTEM_PROMPT},
            {"role": "user", "content": user_prompt},
        ]

        # Hard cap at 45s — if LLM is slow, fallback is better than blocking the pipeline
        response = await asyncio.wait_for(llm.ainvoke(messages), timeout=45)
        content = response.content if hasattr(response, "content") else str(response)

        if "```json" in content:
            content = content.split("```json")[1].split("```")[0].strip()
        elif "```" in content:
            content = content.split("```")[1].split("```")[0].strip()

        llm_report = json.loads(content)
        return _merge_llm_report(deterministic_report, llm_report)

    except asyncio.TimeoutError:
        logger.warning("LLM analysis timed out after 45s, using deterministic report")
        return deterministic_report
    except Exception as exc:
        logger.warning("LLM analysis failed: %s, using deterministic report", exc)
        return deterministic_report
