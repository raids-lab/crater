"""图 3.8 / 3.9 — Ch3 工具风险分布 与 角色 × 工具类别 权限矩阵。"""

from __future__ import annotations

import sys
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np
from matplotlib.patches import Patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from scripts.load_data import role_tool_matrix, tool_risk_distribution
from styles.matplotlib_style import COLORS, apply_style, save


OUT = ROOT / "output" / "ch3"
OUT.mkdir(parents=True, exist_ok=True)


# -----------------------------------------------------------------------------
# 图 3.8 工具数量 × 风险等级 (水平堆叠 + 总和环)
# -----------------------------------------------------------------------------

def fig3_8_tool_risk_distribution():
    df = tool_risk_distribution()

    risk_color = {
        "只读": COLORS["mops"],
        "建议型": COLORS["ps"],
        "待确认": COLORS["accent"],
        "高风险": "#C44536",
    }
    fig, (ax_left, ax_right) = plt.subplots(
        1, 2, figsize=(8.4, 3.4), gridspec_kw=dict(width_ratios=[1.6, 1])
    )

    # 左：按类别水平条形 (按数量降序)
    df_sorted = df.sort_values("数量", ascending=True)
    bars = ax_left.barh(
        df_sorted["类别"], df_sorted["数量"],
        color=[risk_color[r] for r in df_sorted["风险等级"]],
        edgecolor="white", linewidth=0.6, height=0.62,
    )
    for b, v, role in zip(bars, df_sorted["数量"], df_sorted["允许角色"]):
        ax_left.text(v + 0.3, b.get_y() + b.get_height() / 2,
                     f"{v}", ha="left", va="center", fontsize=9)
    ax_left.set_xlabel("工具数量")
    ax_left.set_xlim(0, df["数量"].max() * 1.18)
    ax_left.set_axisbelow(True)
    legend_handles = [Patch(facecolor=c, label=k) for k, c in risk_color.items()]
    ax_left.legend(handles=legend_handles, loc="lower right", ncol=2, fontsize=8.5)

    # 右：按风险等级聚合饼
    agg = df.groupby("风险等级", sort=False)["数量"].sum()
    order = ["只读", "建议型", "待确认", "高风险"]
    agg = agg.reindex(order).fillna(0)
    colors = [risk_color[r] for r in agg.index]
    wedges, texts, autotexts = ax_right.pie(
        agg.values, labels=agg.index, colors=colors,
        autopct=lambda p: f"{p:.0f}%\n({int(round(p * agg.sum() / 100))})",
        startangle=90, wedgeprops=dict(linewidth=1.2, edgecolor="white"),
        textprops=dict(fontsize=8.5),
    )
    for t in autotexts:
        t.set_color("white")
        t.set_fontsize(8)
        t.set_fontweight("bold")
    ax_right.set_aspect("equal")

    save(fig, OUT / "fig3-8-tool-risk-dist")


# -----------------------------------------------------------------------------
# 图 3.9 角色 × 工具类别 权限矩阵
# -----------------------------------------------------------------------------

def fig3_9_role_tool_matrix():
    df = role_tool_matrix()
    # 0=禁止, 1=只读, 2=可写
    fig, ax = plt.subplots(figsize=(7.6, 3.4))
    cmap_colors = ["#F3F4F6", COLORS["mops"], COLORS["accent"]]
    from matplotlib.colors import ListedColormap, BoundaryNorm
    cmap = ListedColormap(cmap_colors)
    norm = BoundaryNorm([-0.5, 0.5, 1.5, 2.5], cmap.N)
    im = ax.imshow(df.values, cmap=cmap, norm=norm, aspect="auto")
    ax.set_xticks(np.arange(df.shape[1]))
    ax.set_xticklabels(df.columns, rotation=18, ha="right")
    ax.set_yticks(np.arange(df.shape[0]))
    ax.set_yticklabels(df.index)
    ax.grid(False)

    symbols = {0: "", 1: "○", 2: "●"}
    for i in range(df.shape[0]):
        for j in range(df.shape[1]):
            v = df.values[i, j]
            color = "white" if v >= 1 else "#9CA3AF"
            ax.text(j, i, symbols[v], ha="center", va="center",
                    fontsize=14 if v == 2 else 12, color=color,
                    fontweight="bold")
    ax.set_xticks(np.arange(df.shape[1] + 1) - 0.5, minor=True)
    ax.set_yticks(np.arange(df.shape[0] + 1) - 0.5, minor=True)
    ax.tick_params(which="minor", length=0)
    ax.grid(which="minor", color="white", linewidth=1.0)

    legend_handles = [
        Patch(facecolor=cmap_colors[1], label="○ 只读访问"),
        Patch(facecolor=cmap_colors[2], label="● 可写访问"),
        Patch(facecolor=cmap_colors[0], edgecolor="#9CA3AF", label="  禁止"),
    ]
    ax.legend(handles=legend_handles, loc="lower center", ncol=3,
              bbox_to_anchor=(0.5, -0.32), frameon=False, fontsize=8.5)
    save(fig, OUT / "fig3-9-role-tool-matrix")


if __name__ == "__main__":
    apply_style()
    fig3_8_tool_risk_distribution()
    fig3_9_role_tool_matrix()
    print("ch3 done.")
