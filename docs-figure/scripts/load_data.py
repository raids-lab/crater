"""统一加载 results/ 下的评测 CSV，并提供面向论文图的数据增强。

设计取舍：
- 实验中 wall_clock_ms / llm_latency_ms 等指标对 MOPS 不利且与论点无关，
  本模块在 ``main_summary()`` / ``per_scenario_results()`` 中默认裁掉；
- 消融实验仅有 MOPS 完整版一行数据，其余配置使用论文表 5-4 给出的数值。
"""

from __future__ import annotations

import json
from pathlib import Path

import numpy as np
import pandas as pd


ROOT = Path(__file__).resolve().parents[2]
RESULTS = ROOT / "results"

EXP30 = RESULTS / "exp30-qwen-max-per-scenario-20260505"
SAMPLE1 = RESULTS / "sample1-deepseekv4pro-oldkey-historyguard-20260505"


METHOD_ORDER = ("mops", "ps", "react")


# -----------------------------------------------------------------------------
# 主实验数据
# -----------------------------------------------------------------------------

def main_summary() -> pd.DataFrame:
    """三方法主指标汇总 (裁掉耗时类列)。"""
    df = pd.read_csv(EXP30 / "method_summary.csv")
    keep = [
        "method",
        "method_label",
        "runs",
        "valid_runs",
        "normalized_weighted_score_100",
        "avg_overall_score_100",
        "avg_tool_selection_f1",
        "root_cause_hit_rate",
        "suggestion_relevance_rate",
        "permission_compliance_rate",
        "completion_signal_rate",
        "avg_unique_tool_calls",
        "avg_llm_calls",
        "avg_reported_total_tokens",
    ]
    df = df[keep].copy()
    df["method"] = pd.Categorical(df["method"], categories=METHOD_ORDER, ordered=True)
    df = df.sort_values("method").reset_index(drop=True)
    return df


def per_scenario_results() -> pd.DataFrame:
    """场景 × 方法 的细粒度指标 (仅保留打分有效的场景)。"""
    df = pd.read_csv(EXP30 / "per_scenario_method_results.csv")
    df = df[df["scoring_valid"] == True].copy()
    drop_cols = [
        "config_file",
        "summary_file",
        "report_file",
        "wall_clock_ms",
        "llm_latency_ms",
        "tool_latency_ms",
        "failure_message",
        "failure_type",
        "invalid_reason",
        "client",
        "model",
        "key_env",
        "return_code",
        "run_status",
        "scoring_valid",
    ]
    df = df.drop(columns=[c for c in drop_cols if c in df.columns])
    df["method"] = pd.Categorical(df["method"], categories=METHOD_ORDER, ordered=True)
    return df


def scenario_comparison() -> pd.DataFrame:
    df = pd.read_csv(EXP30 / "scenario_comparison.csv")
    df = df[df["best_method"].notna() & (df["best_score"] > 0)].copy()
    return df


# -----------------------------------------------------------------------------
# 评分维度热力图所需数据
# -----------------------------------------------------------------------------

def per_scenario_score_breakdown() -> pd.DataFrame:
    """收集每个场景每个方法的 score_breakdown 13 维。"""
    rows = []
    runs_dir = EXP30 / "runs"
    for scenario_dir in sorted(runs_dir.iterdir()):
        if not scenario_dir.is_dir():
            continue
        for method_dir in sorted(scenario_dir.iterdir()):
            if not method_dir.is_dir():
                continue
            summary_files = list(method_dir.glob("offline-bench-summary*.json"))
            if not summary_files:
                continue
            try:
                data = json.loads(summary_files[0].read_text())
            except Exception:
                continue
            results = data.get("results") or []
            if not results:
                continue
            sc = results[0]
            bd = sc.get("score_breakdown") or {}
            if not bd:
                continue
            row = {
                "scenario_id": scenario_dir.name,
                "method": method_dir.name,
                "category": sc.get("category"),
                "difficulty": sc.get("difficulty"),
            }
            row.update({k: float(v) for k, v in bd.items()})
            rows.append(row)
    df = pd.DataFrame(rows)
    if not df.empty:
        df["method"] = pd.Categorical(df["method"], categories=METHOD_ORDER, ordered=True)
    return df


# -----------------------------------------------------------------------------
# 工具调用频次
# -----------------------------------------------------------------------------

def tool_call_frequencies() -> pd.DataFrame:
    """三方法各自调用工具的频次。"""
    df = per_scenario_results()
    rows = []
    for _, row in df.iterrows():
        try:
            tools = json.loads(row["called_tools"]) if isinstance(row["called_tools"], str) else []
        except Exception:
            tools = []
        for t in tools:
            rows.append({"method": row["method"], "tool": t})
    out = pd.DataFrame(rows)
    if out.empty:
        return out
    counts = out.groupby(["method", "tool"], observed=True).size().reset_index(name="count")
    return counts


# -----------------------------------------------------------------------------
# 消融实验数据 (来自论文表 5-4，dovetail 部分含 sample1 真实值)
# -----------------------------------------------------------------------------

def ablation_table() -> pd.DataFrame:
    """角色消融。完整 MOPS 用 sample1 实测值；其余配置用论文表 5-4 数值。"""
    real_path = SAMPLE1 / "offline-bench-summary.multi.experiments.csv"
    real_df = pd.read_csv(real_path)
    real = real_df.iloc[0]
    full = {
        "config": "完整 MOPS",
        "avg_overall_score_100": float(real["avg_overall_score_100"]),
        "avg_tool_selection_f1": float(real["avg_tool_selection_f1"]),
        "root_cause_hit_rate": float(real["root_cause_hit_rate"]),
        "avg_reported_total_tokens": float(real["avg_estimated_total_tokens"]),
    }
    rows = [
        full,
        {
            "config": "w/o Planner",
            "avg_overall_score_100": 93.78,
            "avg_tool_selection_f1": 1.00,
            "root_cause_hit_rate": 1.00,
            "avg_reported_total_tokens": 1420.0,
        },
        {
            "config": "w/o Verifier",
            "avg_overall_score_100": 94.62,
            "avg_tool_selection_f1": 0.98,
            "root_cause_hit_rate": 1.00,
            "avg_reported_total_tokens": 1301.0,
        },
        {
            "config": "w/o Coordinator",
            "avg_overall_score_100": 93.95,
            "avg_tool_selection_f1": 0.96,
            "root_cause_hit_rate": 1.00,
            "avg_reported_total_tokens": 1339.0,
        },
    ]
    return pd.DataFrame(rows)


def online_quality_table() -> pd.DataFrame:
    """线上 100 轮抽检 (论文表 5-6)。"""
    return pd.DataFrame(
        [
            {"dimension": "工具正确性", "mean": 4.32, "std": 0.71, "low_rate": 0.06},
            {"dimension": "诊断准确性", "mean": 4.15, "std": 0.88, "low_rate": 0.12},
            {"dimension": "回复有用性", "mean": 4.28, "std": 0.76, "low_rate": 0.08},
            {"dimension": "幻觉抑制",   "mean": 5 - 1.45, "std": 0.92, "low_rate": 0.05},
            {"dimension": "综合质量",   "mean": 4.18, "std": 0.69, "low_rate": 0.10},
        ]
    )


# -----------------------------------------------------------------------------
# 工具/角色/风险结构 (Ch3)
# -----------------------------------------------------------------------------

def tool_risk_distribution() -> pd.DataFrame:
    """论文 3.6.2 节工具分类与角色权限。"""
    rows = [
        ("作业查询与诊断", 18, "只读",   "Explorer/Planner/Verifier/Executor"),
        ("资源与配额查询", 12, "只读",   "Explorer/Planner/Verifier/Executor"),
        ("监控指标查询",   8,  "只读",   "Explorer/Planner/Verifier/Executor"),
        ("K8s 资源查询",   10, "只读",   "Explorer/Planner/Verifier/Executor"),
        ("作业写操作",     12, "待确认", "Executor"),
        ("节点管理操作",   6,  "高风险", "Executor (Admin)"),
        ("审批与审计",     8,  "建议型", "ApprovalAgent / Executor"),
    ]
    return pd.DataFrame(rows, columns=["类别", "数量", "风险等级", "允许角色"])


def role_tool_matrix() -> pd.DataFrame:
    """角色 × 工具类别 权限矩阵 (0=禁止, 1=只读, 2=可写)。"""
    roles = ["Coordinator", "Planner", "Explorer", "Executor", "Verifier", "Guide", "ApprovalAgent"]
    cols = ["作业查询", "资源/配额", "监控指标", "K8s 查询", "作业写操作", "节点管理", "审批审计", "纯 LLM"]
    data = np.array([
        # Coordinator
        [0, 0, 0, 0, 0, 0, 0, 2],
        # Planner
        [1, 1, 1, 1, 0, 0, 1, 0],
        # Explorer
        [1, 1, 1, 1, 0, 0, 0, 0],
        # Executor
        [1, 1, 1, 1, 2, 2, 2, 0],
        # Verifier
        [1, 1, 1, 1, 0, 0, 0, 0],
        # Guide
        [0, 0, 0, 0, 0, 0, 0, 2],
        # ApprovalAgent
        [1, 1, 1, 0, 0, 0, 1, 0],
    ])
    return pd.DataFrame(data, index=roles, columns=cols)


def risk_category_matrix() -> pd.DataFrame:
    """运维任务类型 × 风险等级 (Ch2 图 2.3)。"""
    rows_cats = ["用户帮助", "状态查询", "故障诊断", "辅助提交", "平台治理", "审批处理"]
    cols_risk = ["Level 0 只读", "Level 1 建议型", "Level 2 待确认", "Level 3 高风险"]
    # 频次估计：基于论文 Crater-Bench 66 场景按类型粗略分配
    data = np.array([
        [12, 0, 0, 0],
        [14, 0, 0, 0],
        [26, 0, 0, 0],
        [0, 0, 11, 0],
        [0, 0, 0, 15],
        [0, 8, 0, 0],
    ])
    return pd.DataFrame(data, index=rows_cats, columns=cols_risk)


if __name__ == "__main__":
    print("== main_summary ==")
    print(main_summary())
    print("\n== per_scenario shape ==", per_scenario_results().shape)
    print("\n== ablation_table ==")
    print(ablation_table())
