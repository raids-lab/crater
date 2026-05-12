"""Nature 类期刊风格 matplotlib 样式 + 复用组件。

设计原则：
  · 极简 spine（仅左/下），中文衬线/黑体 9pt；
  · 半透明 legend 浅灰底，无边框；
  · 主色仅 4 个（方法）+ 2 个强调，其他维度退入灰阶；
  · 不画 title；图例置图内或图下，避免遮挡；
  · 默认 DPI 200，保存 PNG+PDF。
"""

from __future__ import annotations

from pathlib import Path

import matplotlib as mpl
import matplotlib.pyplot as plt
from matplotlib import font_manager
from matplotlib.colors import LinearSegmentedColormap


# -----------------------------------------------------------------------------
# 配色（与 drawio 一致）
# -----------------------------------------------------------------------------

METHOD_COLORS = {
    "mops":     "#1F4E79",   # 深蓝
    "ps":       "#2A9D8F",   # 青绿
    "react":    "#9C6BAE",   # 紫
    "llm_only": "#B3B3B3",   # 中性灰
}

DIFFICULTY_COLORS = {
    "easy":   "#9CD3DA",
    "medium": "#3F8FA8",
    "hard":   "#1F4E79",
}

CATEGORY_COLORS = {
    "diagnosis":  "#264653",
    "ops":        "#2A9D8F",
    "query":      "#E9C46A",
    "submission": "#F4A261",
}

ACCENT = "#E76F51"
OK_GREEN = "#52B788"
GRID = "#E7E9EC"
TEXT = "#1F2937"
MUTED = "#7C8693"


METHOD_LABEL = {
    "mops":     "MOPS",
    "ps":       "Plan-Execute",
    "react":    "ReAct",
    "llm_only": "LLM-only",
}

# 用于雷达图等：MOPS 标"(本文)"
METHOD_LABEL_FULL = {
    "mops":     "MOPS (本文)",
    "ps":       "Plan-Execute",
    "react":    "ReAct",
    "llm_only": "LLM-only",
}


# -----------------------------------------------------------------------------
# 字体
# -----------------------------------------------------------------------------

_FONT_CANDIDATES = (
    "PingFang SC",
    "Hiragino Sans GB",
    "Source Han Sans SC",
    "Noto Sans CJK SC",
    "Microsoft YaHei",
    "SimHei",
)


def _pick_font() -> str:
    avail = {f.name for f in font_manager.fontManager.ttflist}
    for c in _FONT_CANDIDATES:
        if c in avail:
            return c
    return "DejaVu Sans"


def apply():
    """启用 Nature 风格。"""
    font = _pick_font()
    plt.rcdefaults()
    mpl.rcParams.update({
        "font.family": ["sans-serif"],
        "font.sans-serif": [font, "Helvetica", "Arial", "DejaVu Sans"],
        "font.size": 9,
        "axes.titlesize": 9.5,
        "axes.labelsize": 9.5,
        "xtick.labelsize": 8.5,
        "ytick.labelsize": 8.5,
        "legend.fontsize": 8.5,
        "axes.unicode_minus": False,
        "axes.spines.top": False,
        "axes.spines.right": False,
        "axes.edgecolor": "#4B5563",
        "axes.labelcolor": TEXT,
        "axes.titlecolor": TEXT,
        "axes.linewidth": 0.8,
        "axes.grid": True,
        "axes.axisbelow": True,
        "grid.color": GRID,
        "grid.linewidth": 0.55,
        "grid.alpha": 1.0,
        "grid.linestyle": "-",
        "xtick.color": "#4B5563",
        "ytick.color": "#4B5563",
        "xtick.major.width": 0.6,
        "ytick.major.width": 0.6,
        "xtick.major.size": 3.0,
        "ytick.major.size": 3.0,
        "legend.frameon": True,
        "legend.framealpha": 0.92,
        "legend.facecolor": "#FFFFFF",
        "legend.edgecolor": "none",
        "legend.borderpad": 0.4,
        "legend.labelspacing": 0.3,
        "legend.handlelength": 1.6,
        "figure.dpi": 150,
        "savefig.dpi": 220,
        "savefig.bbox": "tight",
        "savefig.pad_inches": 0.08,
        "pdf.fonttype": 42,
        "svg.fonttype": "none",
    })
    return font


# -----------------------------------------------------------------------------
# 颜色辅助
# -----------------------------------------------------------------------------

def method_color(method: str) -> str:
    return METHOD_COLORS.get(method, "#666666")


def method_label(method: str, full: bool = False) -> str:
    return (METHOD_LABEL_FULL if full else METHOD_LABEL).get(method, method)


def heat_cmap() -> LinearSegmentedColormap:
    return LinearSegmentedColormap.from_list(
        "heat", ["#F1F5F9", "#CFE3EA", "#5FA8D3", "#1F4E79"],
    )


def divergent_cmap() -> LinearSegmentedColormap:
    return LinearSegmentedColormap.from_list(
        "diverg", ["#C44536", "#F4A261", "#F1F5F9", "#5FA8D3", "#1F4E79"],
    )


# -----------------------------------------------------------------------------
# 视觉组件
# -----------------------------------------------------------------------------

def nice_legend(ax, *, loc="best", ncol=1, frame_alpha: float = 0.78):
    leg = ax.legend(loc=loc, ncol=ncol, frameon=True, framealpha=frame_alpha,
                    facecolor="#FFFFFF", edgecolor="none",
                    fancybox=True)
    if leg:
        leg.get_frame().set_linewidth(0.0)
        for txt in leg.get_texts():
            txt.set_color(TEXT)
    return leg


def strip_spines(ax, keep=("left", "bottom")):
    for sp in ("top", "right", "left", "bottom"):
        ax.spines[sp].set_visible(sp in keep)


def annotate_bar(ax, bars, fmt="{:.2f}", *, dy: float = 0.0,
                 color: str = TEXT, fontsize: float = 8):
    for b in bars:
        h = b.get_height()
        ax.text(b.get_x() + b.get_width() / 2, h + dy,
                fmt.format(h), ha="center", va="bottom",
                color=color, fontsize=fontsize)


def save_multi(fig, out_path: str | Path, formats=("png", "svg", "pdf")):
    out_path = Path(out_path)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    written = []
    for fmt in formats:
        target = out_path.with_suffix(f".{fmt}")
        fig.savefig(target, format=fmt)
        written.append(target)
    plt.close(fig)
    return written
