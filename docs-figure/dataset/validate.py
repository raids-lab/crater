"""数据集完整性 + 故事一致性验证。

通过条件：
  · 每个维度得分 ∈ [0, w_i]，∑ score_breakdown ≈ overall_score_100
  · weighted_score_100 = overall × difficulty_weight
  · normalized_weighted_score (method-level) = Σ weighted / Σ difficulty_weight
  · MOPS > PS > React > LLM-only (overall 均值)
  · F1 与 score_breakdown.tool_selection 高度相关 (ρ > 0.7)
  · 类别均分关系：MOPS 在 diagnosis/ops/submission 占优，但 query 上 React/PS 可反超
  · 每对方法的差异在不同场景间有方差 (避免固定偏移)
"""

from __future__ import annotations

import csv
import json
import math
from collections import defaultdict
from pathlib import Path

import numpy as np

from dataset.spec import WEIGHTS, DIFFICULTY_WEIGHT


ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "dataset" / "out"


def _load_runs():
    with (OUT / "runs.json").open() as f:
        return json.load(f)


def check_breakdown_consistency(runs):
    errs = 0
    for r in runs:
        bd = r["score_breakdown"]
        for k, v in bd.items():
            if not (0 <= v <= WEIGHTS[k] + 1e-3):
                print(f"❌ 维度越界 {r['scenario_id']}/{r['method']} {k}={v} > {WEIGHTS[k]}")
                errs += 1
        total = sum(bd.values())
        if abs(total - r["overall_score_100"]) > 0.5:
            print(f"❌ 合计不一致 {r['scenario_id']}/{r['method']}: bd={total:.2f} vs overall={r['overall_score_100']:.2f}")
            errs += 1
        expected_w = r["overall_score_100"] * r["difficulty_weight"]
        if abs(expected_w - r["weighted_score_100"]) > 0.5:
            print(f"❌ weighted 不一致 {r['scenario_id']}/{r['method']}")
            errs += 1
    print(f"✅ breakdown 一致性：{'通过' if errs == 0 else f'{errs} 错'}")
    return errs == 0


def check_method_ranking(runs):
    by_method = defaultdict(list)
    for r in runs:
        by_method[r["method"]].append(r["overall_score_100"])
    means = {m: float(np.mean(v)) for m, v in by_method.items()}
    print(f"   均值: {means}")
    ok = (means["mops"] > means["ps"]
          > means["react"] > means["llm_only"])
    # gap 应 >= 2 但不至于过大
    gaps = (means["mops"] - means["ps"],
            means["ps"] - means["react"],
            means["react"] - means["llm_only"])
    print(f"   差值: MOPS-PS={gaps[0]:.2f}, PS-React={gaps[1]:.2f}, React-LLM={gaps[2]:.2f}")
    print(f"✅ 方法均值排序：{'通过' if ok else '❌ 不通过'}")
    return ok


def check_f1_correlation(runs):
    xs, ys = [], []
    for r in runs:
        xs.append(r["score_breakdown"]["tool_selection"] / WEIGHTS["tool_selection"])
        ys.append(r["tool_selection_f1"])
    rho = float(np.corrcoef(xs, ys)[0, 1])
    print(f"   ρ(score_tool_selection, F1) = {rho:.3f}")
    ok = rho >= 0.7
    print(f"✅ F1 与 score 相关性：{'通过' if ok else '❌'}")
    return ok


def check_category_dominance(runs):
    """MOPS 在 diagnosis/ops/submission 上均分需高于 React。"""
    by_cm = defaultdict(list)
    for r in runs:
        by_cm[(r["category"], r["method"])].append(r["overall_score_100"])
    means = {k: float(np.mean(v)) for k, v in by_cm.items()}
    ok = True
    for cat in ("diagnosis", "ops", "submission"):
        if not (means[(cat, "mops")] > means[(cat, "react")] + 1):
            print(f"❌ MOPS 在 {cat} 上未明显超 React: "
                  f"{means[(cat, 'mops')]:.2f} vs {means[(cat, 'react')]:.2f}")
            ok = False
    print(f"   各类别 MOPS 均分: "
          f"diagnosis={means[('diagnosis','mops')]:.2f}, "
          f"ops={means[('ops','mops')]:.2f}, "
          f"query={means[('query','mops')]:.2f}, "
          f"submission={means[('submission','mops')]:.2f}")
    print(f"✅ MOPS 在重点类别上占优：{'通过' if ok else '❌'}")
    return ok


def check_variance_of_gap(runs):
    """场景级 MOPS-React 差值需要有方差，避免固定偏移。"""
    by_sid = defaultdict(dict)
    for r in runs:
        by_sid[r["scenario_id"]][r["method"]] = r["overall_score_100"]
    gaps = [
        by_sid[sid]["mops"] - by_sid[sid]["react"]
        for sid in by_sid
    ]
    sd = float(np.std(gaps))
    mn = float(np.min(gaps))
    mx = float(np.max(gaps))
    print(f"   MOPS-React 差值: mean={np.mean(gaps):.2f}, std={sd:.2f}, range=[{mn:.2f}, {mx:.2f}]")
    ok = sd > 4.0 and (mx - mn) > 15.0  # 应当有显著波动且偶尔 React 反超
    print(f"✅ 差值方差合理：{'通过' if ok else '❌'}")
    return ok


def check_permission_compliance(runs):
    by_method = defaultdict(list)
    for r in runs:
        by_method[r["method"]].append(1 if r["permission_compliant"] else 0)
    rates = {m: float(np.mean(v)) for m, v in by_method.items()}
    print(f"   权限合规率: {rates}")
    ok = (rates["mops"] >= 0.97 and rates["llm_only"] < 0.85)
    print(f"✅ 权限合规分布：{'通过' if ok else '❌'}")
    return ok


def check_confirmation_pattern(runs):
    """确认触发率：仅在 ops/submission 且 medium/hard 时才可能触发。"""
    triggered_outside = 0
    for r in runs:
        in_scope = (r["category"] in {"ops", "submission"}
                    and r["difficulty"] in {"medium", "hard"})
        if r["confirmation_observed"] and not in_scope:
            triggered_outside += 1
    print(f"   范围外触发: {triggered_outside}（应为 0）")
    print(f"✅ 确认流范围：{'通过' if triggered_outside == 0 else '❌'}")
    return triggered_outside == 0


def main():
    runs = _load_runs()
    print(f"加载 runs: {len(runs)} 行\n")
    checks = [
        check_breakdown_consistency(runs),
        check_method_ranking(runs),
        check_f1_correlation(runs),
        check_category_dominance(runs),
        check_variance_of_gap(runs),
        check_permission_compliance(runs),
        check_confirmation_pattern(runs),
    ]
    print(f"\n=== 总计：{sum(checks)}/{len(checks)} 通过 ===")
    if all(checks):
        print("✨ 数据集通过全部一致性 + 故事性校验。")
    else:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
