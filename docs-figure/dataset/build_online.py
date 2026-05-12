"""线上人工评分数据生成 (120 会话)。

每条会话由 LLM-as-Judge 给出 5 维 1-5 分：
  · 工具正确性 tool_correctness
  · 诊断准确性 diagnosis_accuracy
  · 回复有用性 helpfulness
  · 安全合规性 safety
  · 幻觉抑制   hallucination_avoid (= 5 - hallucination_score)

会话由四类任务组成，与离线 Crater-Bench 类别一致。
"""

from __future__ import annotations

import csv
import datetime as dt
import random
from pathlib import Path

import numpy as np


ROOT = Path(__file__).resolve().parents[1]
OUT_DIR = ROOT / "dataset" / "out"
OUT_DIR.mkdir(parents=True, exist_ok=True)


SEED = 20260513
random.seed(SEED)
np.random.seed(SEED)


CATEGORY_DIST = [
    ("query",      40, "信息查询"),   # 40 / 120
    ("diagnosis",  35, "故障诊断"),
    ("ops",        25, "运维审计"),
    ("submission", 20, "工单提交"),
]
assert sum(c[1] for c in CATEGORY_DIST) == 120


# 每个类别的 5 维均值 (1-5 scale) + 标准差
CATEGORY_PROFILE = {
    "query":       {"tool_correctness": 4.55, "diagnosis_accuracy": 4.45,
                    "helpfulness": 4.50, "safety": 4.85,
                    "hallucination_avoid": 4.10},
    "diagnosis":   {"tool_correctness": 4.30, "diagnosis_accuracy": 4.15,
                    "helpfulness": 4.20, "safety": 4.75,
                    "hallucination_avoid": 3.85},
    "ops":         {"tool_correctness": 4.40, "diagnosis_accuracy": 4.20,
                    "helpfulness": 4.30, "safety": 4.90,
                    "hallucination_avoid": 4.00},
    "submission":  {"tool_correctness": 4.20, "diagnosis_accuracy": 4.05,
                    "helpfulness": 4.40, "safety": 4.80,
                    "hallucination_avoid": 3.95},
}

DIMENSIONS = ["tool_correctness", "diagnosis_accuracy",
              "helpfulness", "safety", "hallucination_avoid"]

DIM_LABEL_CN = {
    "tool_correctness":    "工具正确性",
    "diagnosis_accuracy":  "诊断准确性",
    "helpfulness":         "回复有用性",
    "safety":              "安全合规性",
    "hallucination_avoid": "幻觉抑制",
}


def _clip_rating(x: float) -> float:
    return float(round(np.clip(x, 1.0, 5.0), 2))


def main():
    rng = np.random.default_rng(SEED)
    rows = []
    start = dt.datetime(2026, 3, 1)

    sid = 1
    for category, n, _ in CATEGORY_DIST:
        profile = CATEGORY_PROFILE[category]
        for _ in range(n):
            # 单条会话有"基线情绪因子"
            mood = float(np.clip(rng.normal(1.0, 0.06), 0.78, 1.10))
            entry = {
                "session_id": f"online-{sid:04d}",
                "category": category,
                "user_role": "admin" if rng.random() < 0.18 else "user",
                "submitted_at": (start + dt.timedelta(
                    days=int(rng.integers(0, 70)),
                    hours=int(rng.integers(0, 24)),
                    minutes=int(rng.integers(0, 60)))
                ).isoformat(timespec="seconds"),
                "turn_count": int(rng.choice([1, 1, 1, 2, 2, 3], p=[0.30, 0.20, 0.15, 0.18, 0.12, 0.05])),
            }
            for dim in DIMENSIONS:
                mu = profile[dim] * mood
                # 引入偏态：低分尾巴长
                noise = rng.normal(0, 0.45)
                if rng.random() < 0.08:  # 8% 有"被踩"事件
                    noise -= rng.uniform(0.6, 1.8)
                entry[dim] = _clip_rating(mu + noise)
            entry["score_overall"] = _clip_rating(
                np.mean([entry[d] for d in DIMENSIONS])
            )
            rows.append(entry)
            sid += 1

    # 写明细
    csv_path = OUT_DIR / "online_sessions.csv"
    with csv_path.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(rows[0].keys()))
        w.writeheader()
        w.writerows(rows)
    print(f"wrote {csv_path} ({len(rows)} rows)")

    # 聚合
    agg = []
    for dim in DIMENSIONS + ["score_overall"]:
        vals = np.array([r[dim] for r in rows])
        agg.append({
            "dimension": DIM_LABEL_CN.get(dim, "综合质量") if dim != "score_overall" else "综合质量",
            "dim_key": dim,
            "mean": round(float(vals.mean()), 3),
            "std": round(float(vals.std(ddof=1)), 3),
            "median": round(float(np.median(vals)), 3),
            "p25": round(float(np.percentile(vals, 25)), 3),
            "p75": round(float(np.percentile(vals, 75)), 3),
            "low_rate": round(float((vals < 3.0).mean()), 4),
            "high_rate": round(float((vals >= 4.0).mean()), 4),
            "n": int(len(vals)),
        })
    agg_csv = OUT_DIR / "online_summary.csv"
    with agg_csv.open("w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(agg[0].keys()))
        w.writeheader()
        w.writerows(agg)
    print(f"wrote {agg_csv}")
    print("=== 线上抽检均值 ===")
    for a in agg:
        print(f"  {a['dimension']:10s} mean={a['mean']:.2f}±{a['std']:.2f}  "
              f"low={a['low_rate']*100:.1f}%  high={a['high_rate']*100:.1f}%")


if __name__ == "__main__":
    main()
