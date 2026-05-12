"""第 5 章线上抽检图（图 5.11）。"""

from __future__ import annotations

import sys
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np
from matplotlib.patches import Patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from scripts.load_data import online_quality_table
from styles.matplotlib_style import COLORS, apply_style, save


OUT = ROOT / "output" / "ch5"
OUT.mkdir(parents=True, exist_ok=True)


def fig5_11_online_quality():
    df = online_quality_table()
    # 排除"综合质量"作为顶轴单独展示
    main = df[df["dimension"] != "综合质量"].reset_index(drop=True)
    overall = df[df["dimension"] == "综合质量"].iloc[0]

    fig, axes = plt.subplots(1, 2, figsize=(8.5, 3.6),
                             gridspec_kw=dict(width_ratios=[1.0, 1.05]))

    # 左：4 维雷达
    ax = axes[0]
    ax.remove()
    ax = fig.add_subplot(1, 2, 1, polar=True)
    labels = main["dimension"].tolist()
    vals = main["mean"].tolist()
    angles = np.linspace(0, 2 * np.pi, len(labels), endpoint=False).tolist()
    vals_closed = vals + vals[:1]
    angles_closed = angles + angles[:1]

    ax.set_theta_offset(np.pi / 2)
    ax.set_theta_direction(-1)
    ax.set_yticks([1, 2, 3, 4, 5])
    ax.set_yticklabels(["1", "2", "3", "4", "5"], fontsize=7, color="#6B7280")
    ax.set_xticks(angles)
    ax.set_xticklabels(labels, fontsize=9)
    ax.tick_params(axis="x", pad=8)
    ax.set_ylim(0, 5.4)
    ax.plot(angles_closed, vals_closed, color=COLORS["mops"], linewidth=1.8)
    ax.fill(angles_closed, vals_closed, color=COLORS["mops"], alpha=0.18)
    ax.grid(color=COLORS["grid"], linewidth=0.6)
    for ang, v in zip(angles, vals):
        ax.text(ang, v - 0.55, f"{v:.2f}", ha="center", va="center",
                fontsize=9, color="#1F2937", fontweight="bold")

    # 右：低分率条形 + 综合分参考
    ax2 = axes[1]
    bars = ax2.barh(
        main["dimension"], main["low_rate"] * 100,
        color=COLORS["accent"], edgecolor="white", linewidth=0.5, height=0.55,
    )
    ax2.set_xlabel("低分率 (%)")
    ax2.set_xlim(0, 18)
    ax2.invert_yaxis()
    for b, v in zip(bars, main["low_rate"]):
        ax2.text(v * 100 + 0.3, b.get_y() + b.get_height() / 2,
                 f"{v * 100:.0f}%", ha="left", va="center", fontsize=8)
    # 标注综合分
    ax2.axvline(overall["low_rate"] * 100, color=COLORS["mops"],
                linestyle="--", linewidth=1.0)
    ax2.text(overall["low_rate"] * 100 + 0.3, -0.4,
             f"综合低分率 {overall['low_rate']*100:.0f}%",
             color=COLORS["mops"], fontsize=8)
    ax2.set_axisbelow(True)

    # 左下角注脚（图例）
    legend_elems = [
        Patch(facecolor=COLORS["mops"], alpha=0.35, edgecolor=COLORS["mops"],
              label="平均分（满分 5）"),
        Patch(facecolor=COLORS["accent"], label="低分率 (<3 分)"),
    ]
    fig.legend(handles=legend_elems, loc="lower center", ncol=2,
               bbox_to_anchor=(0.5, -0.03), frameon=False)
    fig.tight_layout(rect=[0, 0.04, 1, 1])
    save(fig, OUT / "fig5-11-online-quality")


if __name__ == "__main__":
    apply_style()
    fig5_11_online_quality()
    print("ch5 online done.")
