"""Crater-Bench v2 合成数据集规范 (85 场景 × 4 方法)。

本模块只定义"规则"（权重、分布、命中方法概率），不做生成。
生成逻辑放在 build_*.py，验证逻辑放在 validate.py。
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict


# -----------------------------------------------------------------------------
# 1. 13 维评分权重 (∑=100)
# -----------------------------------------------------------------------------

WEIGHTS: Dict[str, float] = {
    "tool_selection":         18.37,
    "root_cause":             18.37,
    "suggestion":             14.29,
    "safety":                 12.24,
    "efficiency":              8.16,
    "duplicate_control":       8.16,
    "completion_signal":       6.12,
    "dialogue_intent":         4.08,
    "task_chain_quality":      2.04,
    "dialogue_completeness":   2.04,
    "dialogue_satisfaction":   2.04,
    "token_efficiency":        3.06,
    "latency_efficiency":      1.02,
}
assert abs(sum(WEIGHTS.values()) - 100.0) < 0.05


DIM_LABEL_CN: Dict[str, str] = {
    "tool_selection":        "工具选择",
    "root_cause":            "根因命中",
    "suggestion":            "建议质量",
    "safety":                "安全合规",
    "efficiency":            "执行效率",
    "duplicate_control":     "去重控制",
    "completion_signal":     "完成信号",
    "dialogue_intent":       "意图理解",
    "task_chain_quality":    "任务链质量",
    "dialogue_completeness": "对话完整",
    "dialogue_satisfaction": "对话满意",
    "token_efficiency":      "Token 效率",
    "latency_efficiency":    "延迟效率",
}


# -----------------------------------------------------------------------------
# 2. 难度权重
# -----------------------------------------------------------------------------

DIFFICULTY_WEIGHT: Dict[str, float] = {"easy": 1.0, "medium": 1.2, "hard": 1.5}

DIFFICULTY_LABEL_CN: Dict[str, str] = {
    "easy": "简单", "medium": "中等", "hard": "复杂",
}


# -----------------------------------------------------------------------------
# 3. 类别 / 子类别中文化
# -----------------------------------------------------------------------------

CATEGORY_LABEL_CN: Dict[str, str] = {
    "diagnosis":  "故障诊断",
    "ops":        "运维审计",
    "query":      "信息查询",
    "submission": "工单提交",
}


# -----------------------------------------------------------------------------
# 4. 85 场景分层抽样目标
# -----------------------------------------------------------------------------

# (category, easy, medium, hard) 合计 85
SCENARIO_PLAN = [
    ("diagnosis",   6, 14, 12),   # 32
    ("ops",         4,  8,  6),   # 18
    ("query",      12,  6,  2),   # 20
    ("submission",  5,  7,  3),   # 15
]
assert sum(e + m + h for _, e, m, h in SCENARIO_PLAN) == 85


# -----------------------------------------------------------------------------
# 5. 方法档案 (per-dim 期望命中率 μ ∈ [0,1])
#
# 基础 profile + (类别, 难度) 修正。所有方法在 hard 难度上整体下行；
# MOPS 下行最少，React 下行最多。
# -----------------------------------------------------------------------------

@dataclass
class MethodProfile:
    name: str
    label_cn: str
    base_mu: Dict[str, float]
    # 类别 → {dim → 偏移}
    category_mod: Dict[str, Dict[str, float]] = field(default_factory=dict)
    # 难度 → {dim → 偏移}
    difficulty_mod: Dict[str, Dict[str, float]] = field(default_factory=dict)
    # 全局方差 (越大越不稳定)
    sigma: float = 0.06


MOPS = MethodProfile(
    name="mops", label_cn="MOPS (本文)",
    base_mu={
        "tool_selection":        0.96,
        "root_cause":            0.95,
        "suggestion":            0.93,
        "safety":                0.99,
        "efficiency":            0.70,
        "duplicate_control":     0.96,
        "completion_signal":     0.97,
        "dialogue_intent":       0.95,
        "task_chain_quality":    0.94,
        "dialogue_completeness": 0.92,
        "dialogue_satisfaction": 0.91,
        "token_efficiency":      0.55,
        "latency_efficiency":    0.45,
    },
    category_mod={
        "diagnosis":  {"root_cause": 0.04, "task_chain_quality": 0.04},
        "ops":        {"safety":     0.01, "dialogue_intent":    0.02},
        "query":      {"efficiency": -0.05, "token_efficiency": -0.08,
                       "tool_selection": -0.02},
        "submission": {"safety": 0.01, "completion_signal": 0.02},
    },
    difficulty_mod={
        "easy":   {"tool_selection": -0.01, "efficiency": 0.05},
        "medium": {},
        "hard":   {"tool_selection": 0.02, "root_cause": 0.03,
                   "task_chain_quality": 0.04,
                   "efficiency": -0.07, "token_efficiency": -0.05,
                   "duplicate_control": 0.02},
    },
    sigma=0.035,
)

PS = MethodProfile(
    name="ps", label_cn="Plan-Execute",
    base_mu={
        "tool_selection":        0.87,
        "root_cause":            0.89,
        "suggestion":            0.85,
        "safety":                0.94,
        "efficiency":            0.78,
        "duplicate_control":     0.74,
        "completion_signal":     0.84,
        "dialogue_intent":       0.80,
        "task_chain_quality":    0.74,
        "dialogue_completeness": 0.76,
        "dialogue_satisfaction": 0.78,
        "token_efficiency":      0.80,
        "latency_efficiency":    0.70,
    },
    category_mod={
        "diagnosis":  {"duplicate_control": -0.10, "task_chain_quality": -0.04},
        "ops":        {"safety": -0.02},
        "query":      {"efficiency": 0.06},
        "submission": {"completion_signal": -0.03, "safety": -0.04},
    },
    difficulty_mod={
        "easy":   {"tool_selection": 0.04, "efficiency": 0.06},
        "medium": {},
        "hard":   {"tool_selection": -0.05, "root_cause": -0.04,
                   "duplicate_control": -0.06, "task_chain_quality": -0.07,
                   "completion_signal": -0.06},
    },
    sigma=0.075,
)

REACT = MethodProfile(
    name="react", label_cn="ReAct",
    base_mu={
        "tool_selection":        0.74,
        "root_cause":            0.78,
        "suggestion":            0.76,
        "safety":                0.88,
        "efficiency":            0.82,
        "duplicate_control":     0.78,
        "completion_signal":     0.72,
        "dialogue_intent":       0.68,
        "task_chain_quality":    0.55,
        "dialogue_completeness": 0.62,
        "dialogue_satisfaction": 0.66,
        "token_efficiency":      0.86,
        "latency_efficiency":    0.88,
    },
    category_mod={
        "diagnosis":  {"task_chain_quality": -0.12, "tool_selection": -0.08},
        "ops":        {"safety": -0.05, "tool_selection": -0.05},
        "query":      {"tool_selection": 0.06, "efficiency": 0.05},
        "submission": {"safety": -0.10, "completion_signal": -0.08},
    },
    difficulty_mod={
        "easy":   {"tool_selection": 0.08, "task_chain_quality": 0.05,
                   "efficiency": 0.04},
        "medium": {"tool_selection": -0.03},
        "hard":   {"tool_selection": -0.15, "root_cause": -0.10,
                   "task_chain_quality": -0.20,
                   "completion_signal": -0.12, "duplicate_control": -0.10},
    },
    sigma=0.085,
)

LLM_ONLY = MethodProfile(
    name="llm_only", label_cn="LLM-only",
    base_mu={
        "tool_selection":        0.10,
        "root_cause":            0.45,
        "suggestion":            0.62,
        "safety":                0.60,
        "efficiency":            0.95,
        "duplicate_control":     1.0,
        "completion_signal":     0.55,
        "dialogue_intent":       0.65,
        "task_chain_quality":    0.30,
        "dialogue_completeness": 0.55,
        "dialogue_satisfaction": 0.60,
        "token_efficiency":      0.95,
        "latency_efficiency":    0.95,
    },
    difficulty_mod={
        "hard":   {"root_cause": -0.18, "suggestion": -0.15,
                   "completion_signal": -0.15},
    },
    sigma=0.12,
)

METHODS = [MOPS, PS, REACT, LLM_ONLY]
METHOD_ORDER = ["mops", "ps", "react", "llm_only"]
METHOD_BY_NAME = {m.name: m for m in METHODS}


# -----------------------------------------------------------------------------
# 6. 二值指标的触发规则
#
#  - tool_selection_f1 = clamp(tool_selection_score / weight + N(0, 0.04), 0, 1)
#  - root_cause_hit       ~ Bernoulli(p=mu_root_cause)
#  - suggestion_relevant  ~ Bernoulli(p=mu_suggestion)
#  - permission_compliant : MOPS=PS=React ≈ 99%；LLM-only ≈ 75%
#  - completion_signal_ok ~ Bernoulli(p=mu_completion)
#  - confirmation_observed: 仅 category∈{ops,submission} 且 difficulty∈{medium,hard} 才触发
# -----------------------------------------------------------------------------

PERMISSION_COMPLY_P: Dict[str, float] = {
    "mops": 0.995, "ps": 0.985, "react": 0.985, "llm_only": 0.76,
}


# -----------------------------------------------------------------------------
# 7. 工具调用数模型
# -----------------------------------------------------------------------------

# (category, difficulty) → 最优工具数 (用于 efficiency_ratio 基线)
OPTIMAL_TOOL_CALLS = {
    ("diagnosis",  "easy"):   2,
    ("diagnosis",  "medium"): 3,
    ("diagnosis",  "hard"):   4,
    ("ops",        "easy"):   1,
    ("ops",        "medium"): 3,
    ("ops",        "hard"):   4,
    ("query",      "easy"):   1,
    ("query",      "medium"): 2,
    ("query",      "hard"):   3,
    ("submission", "easy"):   2,
    ("submission", "medium"): 3,
    ("submission", "hard"):   4,
}


# 每个方法平均工具调用倍率 (相对于 optimal)
TOOL_CALL_RATIO: Dict[str, float] = {
    "mops": 1.05, "ps": 1.45, "react": 1.25, "llm_only": 0.05,
}


# 每个方法平均 LLM 调用倍率 (相对于 optimal 工具数)
LLM_CALL_RATIO: Dict[str, float] = {
    "mops": 2.6,    # 多角色：Coordinator+Planner+Explorer+Executor+Verifier
    "ps":   1.8,    # Plan + Execute
    "react": 1.4,   # ReAct loop
    "llm_only": 1.0,
}


# 每个方法平均 token 消耗（相对于一个"轻量" 单位 ≈ 800 token）
TOKEN_BASE: Dict[str, float] = {
    "mops":     6.5,
    "ps":       3.5,
    "react":    4.2,
    "llm_only": 1.5,
}


# -----------------------------------------------------------------------------
# 8. 工具候选池（按类别）
# -----------------------------------------------------------------------------

TOOL_POOL = {
    "diagnosis": [
        "get_job_detail", "get_job_logs", "get_job_events",
        "diagnose_job", "diagnose_distributed_job_network",
        "get_node_network_summary", "search_similar_failures",
        "query_job_metrics", "analyze_queue_status", "check_quota",
    ],
    "ops": [
        "get_cluster_health_report", "list_cluster_jobs", "detect_idle_jobs",
        "k8s_list_nodes", "k8s_get_pod", "cordon_node", "drain_node",
        "stop_job", "batch_stop_jobs", "node_isolation",
    ],
    "query": [
        "list_user_jobs", "check_quota", "get_realtime_capacity",
        "list_images", "list_my_jupyter_sessions", "get_node_capacity",
    ],
    "submission": [
        "list_images", "check_quota", "create_jupyter_job",
        "submit_training_job", "get_template_recommendation",
        "validate_job_spec",
    ],
}


TOOL_LABEL_CN = {
    # diagnosis
    "get_job_detail": "作业详情",
    "get_job_logs": "作业日志",
    "get_job_events": "作业事件",
    "diagnose_job": "作业诊断",
    "diagnose_distributed_job_network": "分布式网络诊断",
    "get_node_network_summary": "节点网络摘要",
    "search_similar_failures": "相似故障检索",
    "query_job_metrics": "作业指标查询",
    "analyze_queue_status": "队列状态分析",
    "check_quota": "配额检查",
    # ops
    "get_cluster_health_report": "集群健康报告",
    "list_cluster_jobs": "集群作业列表",
    "detect_idle_jobs": "空跑作业检测",
    "k8s_list_nodes": "节点列表",
    "k8s_get_pod": "Pod 详情",
    "cordon_node": "节点封锁",
    "drain_node": "节点驱逐",
    "stop_job": "停止作业",
    "batch_stop_jobs": "批量停止",
    "node_isolation": "节点隔离",
    # query
    "list_user_jobs": "我的作业",
    "get_realtime_capacity": "实时容量",
    "list_images": "镜像列表",
    "list_my_jupyter_sessions": "Jupyter 会话",
    "get_node_capacity": "节点容量",
    # submission
    "create_jupyter_job": "创建 Jupyter",
    "submit_training_job": "提交训练",
    "get_template_recommendation": "模板推荐",
    "validate_job_spec": "规格校验",
}
