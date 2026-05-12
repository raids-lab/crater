"""图 2.3 — 运维任务类型 × 操作风险等级 热力图。"""

from __future__ import annotations

import sys
from pathlib import Path

import matplotlib.pyplot as plt
import numpy as np

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from scripts.load_data import risk_category_matrix
from styles.matplotlib_style import COLORS, apply_style, save


OUT = ROOT / "output" / "ch2"
OUT.mkdir(parents=True, exist_ok=True)


def fig2_3_risk_matrix():
    df = risk_category_matrix()
    fig, ax = plt.subplots(figsize=(6.4, 3.6))
    cmap = plt.get_cmap("Blues")
    data = df.values.astype(float)
    im = ax.imshow(data, aspect="auto", cmap=cmap, vmin=0, vmax=data.max() * 1.05)
    ax.set_xticks(np.arange(df.shape[1]))
    ax.set_xticklabels(df.columns, rotation=12, ha="right")
    ax.set_yticks(np.arange(df.shape[0]))
    ax.set_yticklabels(df.index)
    ax.grid(False)
    for i in range(df.shape[0]):
        for j in range(df.shape[1]):
            v = int(data[i, j])
            if v == 0:
                continue
            color = "white" if v / data.max() > 0.55 else "#1F2937"
            ax.text(j, i, str(v), ha="center", va="center", fontsize=9.5,
                    color=color, fontweight="bold")
    ax.set_xticks(np.arange(df.shape[1] + 1) - 0.5, minor=True)
    ax.set_yticks(np.arange(df.shape[0] + 1) - 0.5, minor=True)
    ax.tick_params(which="minor", length=0)
    ax.grid(which="minor", color="white", linewidth=0.8)
    cbar = fig.colorbar(im, ax=ax, fraction=0.04, pad=0.02)
    cbar.set_label("场景数量", fontsize=8)
    cbar.ax.tick_params(labelsize=7.5)
    save(fig, OUT / "fig2-3-risk-matrix")


if __name__ == "__main__":
    apply_style()
    fig2_3_risk_matrix()
    print("ch2 done.")
