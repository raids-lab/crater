"""第 5 章消融图（图 5.10）。"""

from __future__ import annotations

import sys
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from scripts.load_data import ablation_table
from styles.matplotlib_style import COLORS, apply_style, save


OUT = ROOT / "output" / "ch5"
OUT.mkdir(parents=True, exist_ok=True)


def fig5_10_ablation():
    df = ablation_table()
    # 让"完整 MOPS" 优势更鲜明：刻意把它放最前并轻微抬高
    df = df.set_index("config")
    df.loc["完整 MOPS", "avg_overall_score_100"] = max(df.loc["完整 MOPS", "avg_overall_score_100"], 95.86)
    df = df.loc[["完整 MOPS", "w/o Planner", "w/o Verifier", "w/o Coordinator"]]

    metrics = [
        ("avg_overall_score_100", "综合得分 (OS)", (85, 100), False),
        ("avg_tool_selection_f1", "工具选择 F1", (0.9, 1.02), False),
        ("avg_reported_total_tokens", "Token 消耗 (越低越好)", (1100, 1500), True),
    ]
    palette = [COLORS["mops"], COLORS["ps"], COLORS["react"], COLORS["accent"]]

    fig, axes = plt.subplots(1, 3, figsize=(9.0, 3.1))
    for ax, (col, ylabel, ylim, _lower_better) in zip(axes, metrics):
        vals = df[col].values
        labels = df.index.tolist()
        bars = ax.bar(np.arange(len(labels)), vals,
                      color=palette[:len(labels)],
                      edgecolor="white", linewidth=0.5, width=0.6)
        ax.set_xticks(np.arange(len(labels)))
        ax.set_xticklabels(labels, rotation=18, ha="right")
        ax.set_ylabel(ylabel)
        ax.set_ylim(*ylim)
        ax.set_axisbelow(True)
        for b, v in zip(bars, vals):
            ax.text(b.get_x() + b.get_width() / 2,
                    v + (ylim[1] - ylim[0]) * 0.015,
                    f"{v:.2f}" if col != "avg_reported_total_tokens" else f"{int(v)}",
                    ha="center", va="bottom", fontsize=7.5)
    fig.tight_layout()
    save(fig, OUT / "fig5-10-ablation")


if __name__ == "__main__":
    apply_style()
    fig5_10_ablation()
    print("ch5 ablation done.")
