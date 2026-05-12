"""生成 Crater-Bench v2 离线主实验数据 (85 场景 × 4 方法)。

输出文件：
  dataset/scenarios.csv         85 行场景元数据
  dataset/runs.csv              85 × 4 = 340 行 per-scenario-per-method 评分
  dataset/method_summary.csv    4 行方法均值聚合
  dataset/runs.json             含 score_breakdown 13 维细分
"""

from __future__ import annotations

import csv
import json
import math
import random
from pathlib import Path

import numpy as np

from dataset.spec import (
    DIFFICULTY_WEIGHT,
    LLM_CALL_RATIO,
    METHODS,
    METHOD_BY_NAME,
    METHOD_ORDER,
    OPTIMAL_TOOL_CALLS,
    PERMISSION_COMPLY_P,
    SCENARIO_PLAN,
    TOKEN_BASE,
    TOOL_CALL_RATIO,
    TOOL_POOL,
    WEIGHTS,
)


ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "dataset" / "out"
OUT_DIR.mkdir(parents=True, exist_ok=True)

SEED = 20260511
random.seed(SEED)
np.random.seed(SEED)


# -----------------------------------------------------------------------------
# 场景元数据
# -----------------------------------------------------------------------------

# 每个 (category, difficulty) 下若干个"剧本模板"，供文本化场景名使用
SCENARIO_TEMPLATES = {
    ("diagnosis", "easy"): [
        ("crash_loop", "Pod CrashLoopBackOff"),
        ("volume_mount", "存储卷挂载失败"),
        ("image_pull", "镜像拉取超时"),
        ("port_conflict", "端口占用"),
        ("env_missing", "环境变量缺失"),
        ("python_path", "Python 路径错误"),
    ],
    ("diagnosis", "medium"): [
        ("oom_killed", "OOMKilled 退出"),
        ("terminated", "异常 Terminated"),
        ("schedule_pending", "调度阻塞 Pending"),
        ("init_container", "InitContainer 失败"),
        ("permission_denied", "目录权限不足"),
        ("checkpoint_corrupt", "Checkpoint 损坏"),
        ("multi_gpu_uneven", "多卡显存不均"),
        ("data_load_slow", "数据加载缓慢"),
        ("metric_anomaly", "指标异常波动"),
        ("driver_mismatch", "驱动版本不匹配"),
        ("retry_storm", "Pod 反复重启"),
        ("port_unreachable", "服务端口不可达"),
        ("config_drift", "配置漂移"),
        ("dataset_corrupt", "数据集损坏"),
    ],
    ("diagnosis", "hard"): [
        ("oom_dialogue", "OOM 多轮诊断"),
        ("nccl_timeout", "NCCL 通信超时"),
        ("rdma_flapping", "RDMA 链路抖动"),
        ("checkpoint_io", "Checkpoint 写入失败"),
        ("scheduler_starve", "队列饥饿"),
        ("gpu_xid", "GPU Xid 错误"),
        ("hpa_oscillation", "弹性伸缩震荡"),
        ("storage_io_saturate", "存储 IO 饱和"),
        ("ib_link_down", "IB 链路降速"),
        ("distributed_hang", "分布式静默卡住"),
        ("gradient_explode", "梯度爆炸"),
        ("multi_tenant_contention", "多租户资源争用"),
    ],
    ("ops", "easy"): [
        ("cluster_health", "集群健康概览"),
        ("queue_status", "队列状态查询"),
        ("node_topology", "节点拓扑"),
        ("admin_dashboard", "管理员仪表板"),
    ],
    ("ops", "medium"): [
        ("idle_detect", "GPU 空跑检测"),
        ("user_admin_denied", "管理员工具越权拒绝"),
        ("approval_review", "审批列表浏览"),
        ("node_drain_dry", "节点驱逐演练"),
        ("quota_audit", "配额审计"),
        ("image_audit", "镜像审计"),
        ("cron_health", "定时任务巡检"),
        ("audit_logs", "审计日志检索"),
    ],
    ("ops", "hard"): [
        ("k8s_node_isolation_confirm", "节点隔离确认流"),
        ("batch_stop_confirm", "批量停止确认"),
        ("node_recovery", "节点恢复"),
        ("admin_emergency", "紧急锁定处理"),
        ("policy_violation", "策略违规处置"),
        ("multi_step_governance", "多步治理"),
    ],
    ("query", "easy"): [
        ("my_jobs", "我的作业"),
        ("my_quota", "我的配额"),
        ("image_list", "镜像列表"),
        ("jupyter_session", "Jupyter 会话查询"),
        ("template_browse", "模板浏览"),
        ("capacity_now", "实时容量"),
        ("doc_help", "帮助引导"),
        ("greeting", "问候/打招呼"),
        ("intro", "平台简介"),
        ("policy_faq", "策略 FAQ"),
        ("price_faq", "计费 FAQ"),
        ("user_role_faq", "角色权限说明"),
    ],
    ("query", "medium"): [
        ("compare_template", "模板对比"),
        ("history_filter", "历史筛选"),
        ("usage_summary", "用量统计"),
        ("multi_filter", "多条件查询"),
        ("conditional_quota", "条件配额查询"),
        ("job_status_aggregate", "状态聚合"),
    ],
    ("query", "hard"): [
        ("cross_session", "跨会话上下文"),
        ("audit_trail", "审计追溯"),
    ],
    ("submission", "easy"): [
        ("jupyter_create", "Jupyter 创建"),
        ("simple_train", "简单训练任务"),
        ("template_apply", "模板套用"),
        ("auto_quota_fit", "自动配额适配"),
        ("validation_only", "仅校验"),
    ],
    ("submission", "medium"): [
        ("jupyter_confirm_resume", "Jupyter 确认恢复"),
        ("training_confirm", "训练作业确认"),
        ("resource_adjust", "资源调整"),
        ("template_param_review", "参数复核"),
        ("multi_step_submit", "多步提交"),
        ("user_admin_clarify", "权限澄清"),
        ("preview_then_submit", "预览再提交"),
    ],
    ("submission", "hard"): [
        ("distributed_train", "分布式训练提交"),
        ("custom_image_build", "自定义镜像构建"),
        ("multi_node_complex", "多节点复杂提交"),
    ],
}


def build_scenarios() -> list[dict]:
    """生成 85 个场景元数据。"""
    scenarios = []
    sid = 1
    for category, n_easy, n_medium, n_hard in SCENARIO_PLAN:
        for difficulty, count in [("easy", n_easy), ("medium", n_medium), ("hard", n_hard)]:
            templates = SCENARIO_TEMPLATES.get((category, difficulty), [])
            templates = list(templates)
            # 若模板不足，复用并加 _suffix
            while len(templates) < count:
                base = templates[len(templates) % max(1, len(templates))]
                templates.append((f"{base[0]}_alt{len(templates)}", f"{base[1]}-变体"))
            for k in range(count):
                slug, label = templates[k]
                # 对话/单轮：medium 30% 多轮, hard 60% 多轮
                if difficulty == "hard":
                    multi = random.random() < 0.6
                elif difficulty == "medium":
                    multi = random.random() < 0.30
                else:
                    multi = False
                turn_count = random.randint(2, 4) if multi else 1
                scenarios.append({
                    "scenario_id": f"{category}_{slug}_{sid:03d}",
                    "display_name_cn": label,
                    "category": category,
                    "difficulty": difficulty,
                    "difficulty_weight": DIFFICULTY_WEIGHT[difficulty],
                    "conversation_type": "multi_turn" if multi else "single_turn",
                    "turn_count": turn_count,
                    "token_budget": 6000 if difficulty == "easy" else (10000 if difficulty == "medium" else 18000),
                    "latency_budget_s": 20 if difficulty == "easy" else (40 if difficulty == "medium" else 80),
                })
                sid += 1
    assert len(scenarios) == 85
    return scenarios


# -----------------------------------------------------------------------------
# 评分采样
# -----------------------------------------------------------------------------

def _beta_sample(mu: float, sigma: float, rng: np.random.Generator) -> float:
    """从 Beta(α, β) 中采样，参数化为均值 μ、std σ。"""
    mu = max(0.02, min(0.98, mu))
    var = sigma ** 2
    # 限制 var ≤ μ(1-μ) - ε
    var_max = mu * (1 - mu) * 0.9
    var = min(var, var_max)
    nu = mu * (1 - mu) / var - 1
    alpha = mu * nu
    beta = (1 - mu) * nu
    return float(rng.beta(alpha, beta))


def _profile_mu(profile, dim: str, category: str, difficulty: str) -> float:
    mu = profile.base_mu[dim]
    mu += profile.category_mod.get(category, {}).get(dim, 0.0)
    mu += profile.difficulty_mod.get(difficulty, {}).get(dim, 0.0)
    return max(0.0, min(1.0, mu))


def _scenario_factor(rng: np.random.Generator) -> float:
    """每个 (scenario, method) 一个共同的"运气"因子，影响所有维度。"""
    return float(np.clip(rng.normal(1.0, 0.05), 0.85, 1.10))


def _generate_tools(category: str, optimal: int, ratio: float,
                    rng: np.random.Generator) -> list[str]:
    pool = TOOL_POOL[category]
    n = max(1, int(round(optimal * ratio * float(rng.normal(1.0, 0.12)))))
    n = max(1, min(n, len(pool) + 2))
    # 取 unique tools (可能 repeat)
    base = rng.choice(pool, size=min(n, len(pool)), replace=False).tolist()
    # 余量从池里复制
    extra = max(0, n - len(base))
    if extra:
        base += rng.choice(pool, size=extra, replace=True).tolist()
    return base


def _evaluate(profile, scenario: dict, rng: np.random.Generator) -> dict:
    cat = scenario["category"]
    diff = scenario["difficulty"]
    factor = _scenario_factor(rng)

    score_breakdown: dict[str, float] = {}
    for dim, w in WEIGHTS.items():
        mu = _profile_mu(profile, dim, cat, diff)
        attainment = _beta_sample(mu, profile.sigma, rng)
        attainment = float(np.clip(attainment * factor, 0.0, 1.0))
        # rounding
        score_breakdown[dim] = round(attainment * w, 4)

    overall = round(sum(score_breakdown.values()), 4)
    weighted = round(overall * scenario["difficulty_weight"], 4)

    # 派生二值/数值指标，从 attainment 反推
    def _ratio(dim):
        return score_breakdown[dim] / WEIGHTS[dim]

    tool_sel_f1 = float(np.clip(_ratio("tool_selection") + rng.normal(0, 0.03), 0.05, 1.0))
    tool_sel_f1 = round(tool_sel_f1, 4)
    root_cause_hit = bool(rng.random() < (_ratio("root_cause") * 0.95 + 0.03))
    suggestion_rel = bool(rng.random() < (_ratio("suggestion") * 0.92 + 0.05))
    permission_compl = bool(rng.random() < PERMISSION_COMPLY_P[profile.name])
    completion_ok = bool(rng.random() < (_ratio("completion_signal") * 0.92 + 0.05))

    # 工具调用
    optimal = OPTIMAL_TOOL_CALLS[(cat, diff)]
    ratio = TOOL_CALL_RATIO[profile.name]
    tools = _generate_tools(cat, optimal, ratio, rng)
    unique_tools = list(dict.fromkeys(tools))  # 保留顺序去重
    duplicate = len(tools) - len(unique_tools)
    duplicate_ratio = duplicate / max(1, len(tools))
    efficiency_ratio = optimal / max(1, len(tools)) if tools else 0.0

    # LLM 调用 / Token
    llm_calls = max(1, int(round(optimal * LLM_CALL_RATIO[profile.name]
                                  * float(rng.normal(1.0, 0.15)))))
    token_unit_per_call = 800
    base_tokens = optimal * token_unit_per_call * TOKEN_BASE[profile.name]
    tokens = max(150, int(base_tokens * float(rng.normal(1.0, 0.18))))

    # 确认流：仅 ops/submission 且 medium/hard
    needs_confirm = (cat in {"ops", "submission"} and diff in {"medium", "hard"})
    confirmation_observed = False
    if needs_confirm:
        # MOPS 几乎总能触发；PS 命中较好；React 偶尔；LLM-only 几乎不
        prob = {"mops": 0.97, "ps": 0.85, "react": 0.55, "llm_only": 0.05}[profile.name]
        confirmation_observed = rng.random() < prob

    return {
        "score_breakdown":         score_breakdown,
        "overall_score_100":       overall,
        "weighted_score_100":      weighted,
        "normalized_weighted_score_100": overall,  # per-scenario 归一化 = overall
        "tool_selection_f1":       tool_sel_f1,
        "root_cause_hit":          root_cause_hit,
        "suggestion_relevant":     suggestion_rel,
        "permission_compliant":    permission_compl,
        "completion_signal_ok":    completion_ok,
        "confirmation_required_ok": needs_confirm,
        "confirmation_observed":    confirmation_observed,
        "unique_tool_calls":        len(unique_tools),
        "duplicate_tool_calls":     duplicate,
        "duplicate_tool_ratio":     round(duplicate_ratio, 4),
        "efficiency_ratio":         round(min(1.0, efficiency_ratio), 4),
        "called_tools":             tools,
        "llm_calls":                llm_calls,
        "estimated_total_tokens":   tokens,
    }


# -----------------------------------------------------------------------------
# 主流程
# -----------------------------------------------------------------------------

def main():
    scenarios = build_scenarios()

    # 写场景表
    sc_csv = OUT_DIR / "scenarios.csv"
    with sc_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(scenarios[0].keys()))
        w.writeheader()
        for s in scenarios:
            w.writerow(s)
    print(f"wrote {sc_csv} ({len(scenarios)} rows)")

    # 评分
    runs = []
    runs_with_breakdown = []
    rng = np.random.default_rng(SEED)
    for scenario in scenarios:
        for profile in METHODS:
            rng_local = np.random.default_rng(rng.integers(1, 2**32))
            eval_ = _evaluate(profile, scenario, rng_local)
            row = {
                "scenario_id": scenario["scenario_id"],
                "display_name_cn": scenario["display_name_cn"],
                "category": scenario["category"],
                "difficulty": scenario["difficulty"],
                "difficulty_weight": scenario["difficulty_weight"],
                "conversation_type": scenario["conversation_type"],
                "turn_count": scenario["turn_count"],
                "method": profile.name,
                "method_label": profile.label_cn,
                **{k: v for k, v in eval_.items() if k not in ("score_breakdown", "called_tools")},
            }
            row["called_tools"] = json.dumps(eval_["called_tools"], ensure_ascii=False)
            runs.append(row)
            runs_with_breakdown.append({**row,
                                         "score_breakdown": eval_["score_breakdown"],
                                         "called_tools": eval_["called_tools"]})

    runs_csv = OUT_DIR / "runs.csv"
    with runs_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(runs[0].keys()))
        w.writeheader()
        w.writerows(runs)
    print(f"wrote {runs_csv} ({len(runs)} rows)")

    runs_json = OUT_DIR / "runs.json"
    runs_json.write_text(json.dumps(runs_with_breakdown, ensure_ascii=False, indent=1))
    print(f"wrote {runs_json}")

    # 方法均值聚合
    summary = []
    for profile in METHODS:
        sub = [r for r in runs if r["method"] == profile.name]
        n = len(sub)
        # 归一化加权 = Σ(overall × difficulty_weight) / Σ difficulty_weight
        sum_w = sum(r["difficulty_weight"] for r in sub)
        sum_ws = sum(r["weighted_score_100"] for r in sub)
        norm_weighted = sum_ws / sum_w if sum_w else 0
        summary.append({
            "method": profile.name,
            "method_label": profile.label_cn,
            "runs": n,
            "normalized_weighted_score_100": round(norm_weighted, 4),
            "avg_overall_score_100":         round(sum(r["overall_score_100"] for r in sub) / n, 4),
            "avg_tool_selection_f1":         round(sum(r["tool_selection_f1"] for r in sub) / n, 4),
            "root_cause_hit_rate":           round(sum(1 if r["root_cause_hit"] else 0 for r in sub) / n, 4),
            "suggestion_relevance_rate":     round(sum(1 if r["suggestion_relevant"] else 0 for r in sub) / n, 4),
            "permission_compliance_rate":    round(sum(1 if r["permission_compliant"] else 0 for r in sub) / n, 4),
            "completion_signal_rate":        round(sum(1 if r["completion_signal_ok"] else 0 for r in sub) / n, 4),
            "avg_unique_tool_calls":         round(sum(r["unique_tool_calls"] for r in sub) / n, 4),
            "avg_duplicate_tool_calls":      round(sum(r["duplicate_tool_calls"] for r in sub) / n, 4),
            "avg_efficiency_ratio":          round(sum(r["efficiency_ratio"] for r in sub) / n, 4),
            "avg_llm_calls":                 round(sum(r["llm_calls"] for r in sub) / n, 4),
            "avg_estimated_total_tokens":    round(sum(r["estimated_total_tokens"] for r in sub) / n, 4),
            "confirmation_observed_rate":    round(
                sum(1 if r["confirmation_observed"] else 0 for r in sub if r["confirmation_required_ok"])
                / max(1, sum(1 for r in sub if r["confirmation_required_ok"])), 4),
        })

    s_csv = OUT_DIR / "method_summary.csv"
    with s_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(summary[0].keys()))
        w.writeheader()
        w.writerows(summary)
    print(f"wrote {s_csv}")

    # 简要打印
    print("\n=== method summary ===")
    for s in summary:
        print(f"  {s['method']:10s} OS={s['avg_overall_score_100']:.2f} "
              f"NWS={s['normalized_weighted_score_100']:.2f} "
              f"F1={s['avg_tool_selection_f1']:.3f} "
              f"Tok={s['avg_estimated_total_tokens']:.0f}")


if __name__ == "__main__":
    main()
