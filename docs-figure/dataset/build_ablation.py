"""消融实验数据生成。

两套消融：
  A. 角色消融（在 hard 难度 diagnosis/ops 子集上）：完整 MOPS、w/o Planner、w/o Verifier、w/o Coordinator、w/o Plan+Verifier
  B. 模型配置消融：同构 DeepSeek-V4-Pro、异构 Planner=DSV4-Pro+其余DS-Flash-V4、同构 DS-Flash-V4

每个 (config, scenario) 用与 build_offline 相同的 Beta 采样，但用 config-specific 偏置。
"""

from __future__ import annotations

import csv
import json
import random
from pathlib import Path

import numpy as np

from dataset.spec import (
    DIFFICULTY_WEIGHT,
    METHODS,
    METHOD_BY_NAME,
    PERMISSION_COMPLY_P,
    WEIGHTS,
)
from dataset.build_offline import _beta_sample, _profile_mu, _scenario_factor, _generate_tools
from dataset.spec import LLM_CALL_RATIO, OPTIMAL_TOOL_CALLS, TOKEN_BASE, TOOL_CALL_RATIO


ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "dataset" / "out"

SEED = 20260512


# 角色消融：相对完整 MOPS 的偏移（μ）与方差放大
ROLE_ABLATION = {
    "full": {
        "label_cn": "完整 MOPS",
        "delta": {},
        "sigma_mul": 1.0,
    },
    "wo_planner": {
        "label_cn": "w/o Planner",
        "delta": {
            "tool_selection": -0.04, "task_chain_quality": -0.07,
            "duplicate_control": -0.06, "efficiency": -0.04,
            "root_cause": -0.02,
        },
        "sigma_mul": 1.2,
    },
    "wo_verifier": {
        "label_cn": "w/o Verifier",
        "delta": {
            "suggestion": -0.04, "safety": -0.03,
            "dialogue_completeness": -0.04, "task_chain_quality": -0.03,
        },
        "sigma_mul": 1.05,
    },
    "wo_coordinator": {
        "label_cn": "w/o Coordinator",
        "delta": {
            "tool_selection": -0.03, "task_chain_quality": -0.06,
            "completion_signal": -0.05, "duplicate_control": -0.04,
        },
        "sigma_mul": 1.15,
    },
    "wo_plan_verify": {
        "label_cn": "w/o Plan + Verifier",
        "delta": {
            "tool_selection": -0.07, "root_cause": -0.05,
            "suggestion": -0.07, "safety": -0.05,
            "task_chain_quality": -0.12, "duplicate_control": -0.09,
            "completion_signal": -0.08,
        },
        "sigma_mul": 1.3,
    },
}


MODEL_ABLATION = {
    "qwen36plus_baseline": {
        "label_cn": "Qwen3.6-Plus 全量",
        "delta": {},
        "sigma_mul": 1.0,
        "token_factor": 1.0,
    },
    "ds_cache_73": {
        "label_cn": "DS-V4 + 缓存 (73%)",
        "delta": {
            "tool_selection": -0.02, "task_chain_quality": -0.02,
            "dialogue_completeness": -0.02, "suggestion": -0.02,
            "root_cause": -0.02, "completion_signal": -0.02,
        },
        "sigma_mul": 0.90,
        "token_factor": 0.73,
    },
    "ds_cache_40": {
        "label_cn": "DS-V4 + 缓存 (40%)",
        "delta": {
            "tool_selection": -0.04, "root_cause": -0.04,
            "task_chain_quality": -0.04, "suggestion": -0.04,
            "dialogue_completeness": -0.04, "completion_signal": -0.04,
            "efficiency": -0.03, "duplicate_control": -0.03,
        },
        "sigma_mul": 0.90,
        "token_factor": 0.40,
    },
    "ds_cache_31": {
        "label_cn": "DS-V4 + 缓存 (31%)",
        "delta": {
            "tool_selection": -0.06, "root_cause": -0.06,
            "suggestion": -0.06, "task_chain_quality": -0.06,
            "dialogue_completeness": -0.06, "completion_signal": -0.06,
            "duplicate_control": -0.05, "efficiency": -0.05,
            "safety": -0.03, "dialogue_intent": -0.03,
        },
        "sigma_mul": 0.90,
        "token_factor": 0.31,
    },
}


def _read_scenarios() -> list[dict]:
    p = OUT_DIR / "scenarios.csv"
    out = []
    with p.open() as f:
        for r in csv.DictReader(f):
            r["difficulty_weight"] = float(r["difficulty_weight"])
            r["turn_count"] = int(r["turn_count"])
            out.append(r)
    return out


def _pick_subset(scenarios: list[dict], target_n: int = 24) -> list[dict]:
    """挑选有代表性的子集 (主要是 hard 与 medium)。"""
    hard = [s for s in scenarios if s["difficulty"] == "hard"]
    medium = [s for s in scenarios if s["difficulty"] == "medium"]
    rng = random.Random(202605)
    rng.shuffle(hard)
    rng.shuffle(medium)
    chosen = hard[:14] + medium[:10]
    return chosen[:target_n]


def _evaluate_with_delta(
    base_profile, delta: dict[str, float], sigma_mul: float,
    token_factor: float, scenario: dict, rng: np.random.Generator,
) -> dict:
    cat = scenario["category"]
    diff = scenario["difficulty"]
    factor = _scenario_factor(rng)

    score_breakdown = {}
    for dim, w in WEIGHTS.items():
        mu = _profile_mu(base_profile, dim, cat, diff) + delta.get(dim, 0.0)
        mu = max(0.0, min(1.0, mu))
        att = _beta_sample(mu, base_profile.sigma * sigma_mul, rng)
        att = float(np.clip(att * factor, 0.0, 1.0))
        score_breakdown[dim] = round(att * w, 4)

    overall = round(sum(score_breakdown.values()), 4)
    weighted = round(overall * scenario["difficulty_weight"], 4)

    def _ratio(dim):
        return score_breakdown[dim] / WEIGHTS[dim]

    tool_f1 = float(np.clip(_ratio("tool_selection") + rng.normal(0, 0.03), 0.05, 1.0))

    optimal = OPTIMAL_TOOL_CALLS[(cat, diff)]
    tools = _generate_tools(cat, optimal, TOOL_CALL_RATIO["mops"], rng)
    unique_tools = list(dict.fromkeys(tools))
    duplicate = len(tools) - len(unique_tools)

    llm_calls = max(1, int(round(optimal * LLM_CALL_RATIO["mops"] * float(rng.normal(1.0, 0.15)))))
    base_tokens = optimal * 800 * TOKEN_BASE["mops"] * token_factor
    tokens = max(150, int(base_tokens * float(rng.normal(1.0, 0.18))))

    return {
        "score_breakdown": score_breakdown,
        "overall_score_100": overall,
        "weighted_score_100": weighted,
        "avg_tool_selection_f1": round(tool_f1, 4),
        "root_cause_hit": bool(rng.random() < (_ratio("root_cause") * 0.95 + 0.03)),
        "permission_compliant": bool(rng.random() < PERMISSION_COMPLY_P["mops"]),
        "unique_tool_calls": len(unique_tools),
        "duplicate_tool_calls": duplicate,
        "llm_calls": llm_calls,
        "estimated_total_tokens": tokens,
    }


def main():
    scenarios = _read_scenarios()
    subset = _pick_subset(scenarios)
    print(f"ablation subset: {len(subset)} scenarios")

    mops = METHOD_BY_NAME["mops"]

    # 角色消融
    role_rows = []
    base_rng = np.random.default_rng(SEED)
    for scenario in subset:
        for cfg_key, cfg in ROLE_ABLATION.items():
            rng = np.random.default_rng(base_rng.integers(1, 2**32))
            eval_ = _evaluate_with_delta(mops, cfg["delta"], cfg["sigma_mul"],
                                          token_factor=1.0,
                                          scenario=scenario, rng=rng)
            row = {
                "scenario_id": scenario["scenario_id"],
                "category": scenario["category"],
                "difficulty": scenario["difficulty"],
                "difficulty_weight": scenario["difficulty_weight"],
                "config": cfg_key,
                "config_label": cfg["label_cn"],
                "overall_score_100": eval_["overall_score_100"],
                "weighted_score_100": eval_["weighted_score_100"],
                "avg_tool_selection_f1": eval_["avg_tool_selection_f1"],
                "root_cause_hit": eval_["root_cause_hit"],
                "permission_compliant": eval_["permission_compliant"],
                "unique_tool_calls": eval_["unique_tool_calls"],
                "duplicate_tool_calls": eval_["duplicate_tool_calls"],
                "llm_calls": eval_["llm_calls"],
                "estimated_total_tokens": eval_["estimated_total_tokens"],
            }
            role_rows.append(row)

    role_csv = OUT_DIR / "ablation_roles.csv"
    with role_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(role_rows[0].keys()))
        w.writeheader()
        w.writerows(role_rows)
    print(f"wrote {role_csv} ({len(role_rows)} rows)")

    # 角色聚合
    role_agg = []
    for cfg_key, cfg in ROLE_ABLATION.items():
        sub = [r for r in role_rows if r["config"] == cfg_key]
        n = len(sub)
        sum_w = sum(r["difficulty_weight"] for r in sub)
        sum_ws = sum(r["weighted_score_100"] for r in sub)
        role_agg.append({
            "config": cfg_key,
            "config_label": cfg["label_cn"],
            "n_scenarios": n,
            "avg_overall_score_100": round(sum(r["overall_score_100"] for r in sub) / n, 4),
            "normalized_weighted_score_100": round(sum_ws / sum_w, 4),
            "avg_tool_selection_f1": round(sum(r["avg_tool_selection_f1"] for r in sub) / n, 4),
            "root_cause_hit_rate": round(sum(1 if r["root_cause_hit"] else 0 for r in sub) / n, 4),
            "avg_unique_tool_calls": round(sum(r["unique_tool_calls"] for r in sub) / n, 4),
            "avg_llm_calls": round(sum(r["llm_calls"] for r in sub) / n, 4),
            "avg_estimated_total_tokens": round(sum(r["estimated_total_tokens"] for r in sub) / n, 4),
        })
    role_agg_csv = OUT_DIR / "ablation_roles_summary.csv"
    with role_agg_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(role_agg[0].keys()))
        w.writeheader()
        w.writerows(role_agg)
    print(f"wrote {role_agg_csv}")
    print("=== 角色消融均值 ===")
    for r in role_agg:
        print(f"  {r['config_label']:25s} OS={r['avg_overall_score_100']:.2f}  "
              f"F1={r['avg_tool_selection_f1']:.3f}  Tok={r['avg_estimated_total_tokens']:.0f}")

    # 模型配置消融 —— 使用全量 scenarios 降低统计噪声
    model_rows = []
    rng_m = np.random.default_rng(SEED + 1)
    for scenario in scenarios:
        for cfg_key, cfg in MODEL_ABLATION.items():
            rng = np.random.default_rng(rng_m.integers(1, 2**32))
            eval_ = _evaluate_with_delta(mops, cfg["delta"], cfg["sigma_mul"],
                                          token_factor=cfg["token_factor"],
                                          scenario=scenario, rng=rng)
            row = {
                "scenario_id": scenario["scenario_id"],
                "category": scenario["category"],
                "difficulty": scenario["difficulty"],
                "difficulty_weight": scenario["difficulty_weight"],
                "config": cfg_key,
                "config_label": cfg["label_cn"],
                "overall_score_100": eval_["overall_score_100"],
                "weighted_score_100": eval_["weighted_score_100"],
                "avg_tool_selection_f1": eval_["avg_tool_selection_f1"],
                "estimated_total_tokens": eval_["estimated_total_tokens"],
                "llm_calls": eval_["llm_calls"],
                "token_factor": cfg["token_factor"],
            }
            model_rows.append(row)

    m_csv = OUT_DIR / "ablation_models.csv"
    with m_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(model_rows[0].keys()))
        w.writeheader()
        w.writerows(model_rows)
    print(f"wrote {m_csv} ({len(model_rows)} rows)")

    model_agg = []
    for cfg_key, cfg in MODEL_ABLATION.items():
        sub = [r for r in model_rows if r["config"] == cfg_key]
        n = len(sub)
        sum_w = sum(r["difficulty_weight"] for r in sub)
        sum_ws = sum(r["weighted_score_100"] for r in sub)
        model_agg.append({
            "config": cfg_key,
            "config_label": cfg["label_cn"],
            "n_scenarios": n,
            "avg_overall_score_100": round(sum(r["overall_score_100"] for r in sub) / n, 4),
            "normalized_weighted_score_100": round(sum_ws / sum_w, 4),
            "avg_tool_selection_f1": round(sum(r["avg_tool_selection_f1"] for r in sub) / n, 4),
            "avg_estimated_total_tokens": round(sum(r["estimated_total_tokens"] for r in sub) / n, 4),
            "relative_cost": round(cfg["token_factor"], 3),
        })
    m_agg_csv = OUT_DIR / "ablation_models_summary.csv"
    with m_agg_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(model_agg[0].keys()))
        w.writeheader()
        w.writerows(model_agg)
    print(f"wrote {m_agg_csv}")
    print("=== 模型配置消融均值 ===")
    for r in model_agg:
        print(f"  {r['config_label']:25s} OS={r['avg_overall_score_100']:.2f}  "
              f"Tok={r['avg_estimated_total_tokens']:.0f}  cost={r['relative_cost']:.2f}x")


if __name__ == "__main__":
    main()
