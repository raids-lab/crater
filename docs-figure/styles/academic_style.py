"""中文计算机学术论文风格 (蓝色系 + 极简)。

设计参考：
  · 计算机学报、软件学报、CCF 推荐期刊常用风格：
    - 主色单一色系（蓝），强调主方法（深蓝），其他用浅蓝/灰
    - 仅保留左、下两条 spine，宽 0.6pt
    - 9pt 五号字，CJK 用 PingFang SC / 宋体回退
    - 热力图严格方形（aspect='equal'）
    - 颜色条 (colorbar) 横向、紧凑，labelsize=7.5
  · 配色为 ColorBrewer Blues + 自定义警示红/优秀绿
  · 全图不写标题（论文 caption 单独写）
"""

from __future__ import annotations

from pathlib import Path

import matplotlib as mpl
import matplotlib.pyplot as plt
from matplotlib import font_manager
from matplotlib.colors import LinearSegmentedColormap


# -----------------------------------------------------------------------------
# 主色板：3 个方法，单一蓝色系，浓度阶梯
# -----------------------------------------------------------------------------

METHOD_COLORS = {
    "mops":  "#08306B",   # 深海军蓝（本文方法，最深）
    "ps":    "#4292C6",   # 中钴蓝
    "react": "#9ECAE1",   # 浅天蓝（对比基线，最浅）
}

# 兼容：偶尔仍需要 llm_only 时（一般不画）
METHOD_COLOR_FALLBACK = "#C7C7C7"

METHOD_LABEL = {
    "mops":  "MOPS",
    "ps":    "Plan-Execute",
    "react": "ReAct",
}

METHOD_LABEL_FULL = {
    "mops":  "MOPS (本文)",
    "ps":    "Plan-Execute",
    "react": "ReAct",
}

METHOD_ORDER = ("mops", "ps", "react")


# 维度配色（消融图/分组对比图用，从浅到深的蓝色阶）
SEQUENCE_BLUES = (
    "#08306B", "#08519C", "#2171B5", "#4292C6", "#6BAED6",
    "#9ECAE1", "#C6DBEF", "#DEEBF7",
)

# 难度三档（浅 → 深）
DIFFICULTY_COLORS = {
    "easy":   "#C6DBEF",
    "medium": "#6BAED6",
    "hard":   "#2171B5",
}

# 业务类别（次色板：青蓝/深蓝/浅蓝/灰蓝，与主蓝区分）
CATEGORY_COLORS = {
    "diagnosis":  "#08519C",
    "ops":        "#2171B5",
    "query":      "#6BAED6",
    "submission": "#C6DBEF",
}

# 强调色（仅用于注释、警示、最佳值标记）
ACCENT_RED  = "#B2182B"   # 警示/落后
ACCENT_GOLD = "#D4A017"   # 中间
OK_GREEN    = "#1A6B3A"   # 最佳/通过
HIGHLIGHT   = "#08306B"

# 中性色
TEXT  = "#1F2937"
MUTED = "#6B7280"
GRID  = "#E5E7EB"
BG    = "#FFFFFF"


# -----------------------------------------------------------------------------
# 字体
# -----------------------------------------------------------------------------

_FONT_CANDIDATES = (
    "PingFang SC",
    "Hiragino Sans GB",
    "Source Han Sans SC",
    "Noto Sans CJK SC",
    "Songti SC",
    "STSong",
    "SimSun",
    "Microsoft YaHei",
    "SimHei",
)


def _pick_font() -> str:
    # 重建字体缓存（解决 macOS 更新后中文乱码）
    font_manager._load_fontmanager(try_read_cache=False)
    avail = {f.name for f in font_manager.fontManager.ttflist}
    for c in _FONT_CANDIDATES:
        if c in avail:
            return c
    # 回退：尝试 Arial Unicode MS（macOS 内置，广泛支持 CJK）
    if "Arial Unicode MS" in avail:
        return "Arial Unicode MS"
    return "DejaVu Sans"


def apply():
    """启用中文学术论文风格。"""
    font = _pick_font()
    plt.rcdefaults()
    mpl.rcParams.update({
        "font.family":          ["sans-serif"],
        "font.sans-serif":      [font, "Helvetica", "Arial", "DejaVu Sans"],
        "font.size":            9,
        "axes.titlesize":       9,
        "axes.labelsize":       9,
        "xtick.labelsize":      8,
        "ytick.labelsize":      8,
        "legend.fontsize":      8,
        "axes.unicode_minus":   False,
        # spine
        "axes.spines.top":      False,
        "axes.spines.right":    False,
        "axes.spines.left":     True,
        "axes.spines.bottom":   True,
        "axes.edgecolor":       "#374151",
        "axes.labelcolor":      TEXT,
        "axes.titlecolor":      TEXT,
        "axes.linewidth":       0.7,
        # grid
        "axes.grid":            True,
        "axes.axisbelow":       True,
        "grid.color":           GRID,
        "grid.linewidth":       0.45,
        "grid.linestyle":       "-",
        "grid.alpha":           1.0,
        # tick
        "xtick.color":          "#374151",
        "ytick.color":          "#374151",
        "xtick.major.width":    0.6,
        "ytick.major.width":    0.6,
        "xtick.major.size":     2.8,
        "ytick.major.size":     2.8,
        "xtick.direction":      "out",
        "ytick.direction":      "out",
        # legend
        "legend.frameon":       True,
        "legend.framealpha":    0.92,
        "legend.facecolor":     "#FFFFFF",
        "legend.edgecolor":     "none",
        "legend.borderpad":     0.4,
        "legend.labelspacing":  0.3,
        "legend.handlelength":  1.5,
        # save
        "figure.dpi":           150,
        "savefig.dpi":          240,
        "savefig.bbox":         "tight",
        "savefig.pad_inches":   0.08,
        "pdf.fonttype":         42,
        "svg.fonttype":         "none",
    })
    return font


# -----------------------------------------------------------------------------
# colormap：单色蓝渐变（专用于方形热力图）
# -----------------------------------------------------------------------------

def heat_blues() -> LinearSegmentedColormap:
    """ColorBrewer Blues 9 阶（去掉两端最极端的）— 论文风格的标准蓝色渐变。"""
    return LinearSegmentedColormap.from_list(
        "academic_blues",
        ["#F7FBFF", "#DEEBF7", "#C6DBEF", "#9ECAE1",
         "#6BAED6", "#4292C6", "#2171B5", "#08519C", "#08306B"],
        N=256,
    )


def heat_blues_light() -> LinearSegmentedColormap:
    """淡蓝→中蓝（用于差异不大、需要拉伸的数据）。"""
    return LinearSegmentedColormap.from_list(
        "academic_blues_light",
        ["#F0F6FB", "#C6DBEF", "#6BAED6", "#2171B5"],
        N=256,
    )


def diverging_blue_red() -> LinearSegmentedColormap:
    """蓝-白-红（用于差值图，显著性矩阵等）。"""
    return LinearSegmentedColormap.from_list(
        "academic_brbg",
        ["#B2182B", "#EF8A62", "#FDDBC7", "#F7F7F7",
         "#D1E5F0", "#67A9CF", "#2166AC", "#08306B"],
        N=256,
    )


# -----------------------------------------------------------------------------
# 辅助函数
# -----------------------------------------------------------------------------

def method_color(method: str) -> str:
    return METHOD_COLORS.get(method, METHOD_COLOR_FALLBACK)


def method_label(method: str, full: bool = False) -> str:
    return (METHOD_LABEL_FULL if full else METHOD_LABEL).get(method, method)


def strip_spines(ax, keep=("left", "bottom")):
    for s in ("top", "right", "left", "bottom"):
        ax.spines[s].set_visible(s in keep)
    if "left" in keep:
        ax.spines["left"].set_color("#374151")
        ax.spines["left"].set_linewidth(0.7)
    if "bottom" in keep:
        ax.spines["bottom"].set_color("#374151")
        ax.spines["bottom"].set_linewidth(0.7)


def square_heatmap(ax):
    """强制 1:1 方形 cell。"""
    ax.set_aspect("equal", adjustable="box")


def grid_off(ax):
    ax.grid(False)


def cell_grid(ax, n_rows: int, n_cols: int, color: str = "white", lw: float = 0.9):
    """在热力图上用白线画 cell 分割线（避免相邻 cell 视觉粘连）。"""
    ax.set_xticks(np.arange(n_cols + 1) - 0.5, minor=True)
    ax.set_yticks(np.arange(n_rows + 1) - 0.5, minor=True)
    ax.tick_params(which="minor", length=0)
    ax.grid(which="minor", color=color, linewidth=lw)
    ax.grid(which="major", visible=False)


def annotate_bar(ax, bars, fmt="{:.2f}", *, dy: float = 0.0,
                 color: str = TEXT, fontsize: float = 7.5):
    for b in bars:
        h = b.get_height()
        ax.text(b.get_x() + b.get_width() / 2, h + dy,
                fmt.format(h), ha="center", va="bottom",
                color=color, fontsize=fontsize)


def thin_legend(ax, *, loc="best", ncol=1, fontsize: float = 8.0):
    leg = ax.legend(loc=loc, ncol=ncol, frameon=True, framealpha=0.92,
                    facecolor="#FFFFFF", edgecolor="none",
                    fontsize=fontsize)
    if leg:
        leg.get_frame().set_linewidth(0)
        for t in leg.get_texts():
            t.set_color(TEXT)
    return leg


def save_multi(fig, out_path: str | Path, formats=("png", "pdf", "svg")):
    out_path = Path(out_path)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    written = []
    for fmt in formats:
        target = out_path.with_suffix(f".{fmt}")
        fig.savefig(target, format=fmt)
        written.append(target)
    plt.close(fig)
    return written


# 防止上面 cell_grid 引用 np 失败
import numpy as np  # noqa: E402
