"""docs-figure 统一 matplotlib 样式 (中文小五 9pt + 冷色调)。"""

from __future__ import annotations

from pathlib import Path

import matplotlib as mpl
import matplotlib.pyplot as plt
from matplotlib import font_manager


_FONT_CANDIDATES = (
    "PingFang SC",
    "Hiragino Sans GB",
    "Heiti SC",
    "Noto Sans CJK SC",
    "Source Han Sans SC",
    "Microsoft YaHei",
    "SimHei",
    "Arial Unicode MS",
)


def _pick_cjk_font() -> str:
    available = {f.name for f in font_manager.fontManager.ttflist}
    for cand in _FONT_CANDIDATES:
        if cand in available:
            return cand
    return "DejaVu Sans"


COLORS = {
    "mops": "#264653",
    "ps": "#2A9D8F",
    "react": "#9C89B8",
    "llm_only": "#B0B0B0",
    "accent": "#E76F51",
    "ok": "#52B788",
    "warn": "#F4A261",
    "muted": "#B8C0C2",
    "grid": "#E5E7EB",
}

METHOD_LABEL = {
    "mops": "MOPS (本文)",
    "ps": "Plan-Execute",
    "react": "ReAct",
    "llm_only": "LLM-only",
}

CATEGORY_LABEL = {
    "diagnosis": "故障诊断",
    "ops": "运维审计",
    "query": "信息查询",
    "submission": "工单提交",
}

DIFFICULTY_LABEL = {
    "easy": "简单",
    "medium": "中等",
    "hard": "复杂",
}


def apply_style() -> str:
    font = _pick_cjk_font()
    plt.rcdefaults()
    mpl.rcParams.update(
        {
            "font.family": ["sans-serif"],
            "font.sans-serif": [font, "DejaVu Sans"],
            "font.size": 9,
            "axes.titlesize": 9,
            "axes.labelsize": 9,
            "xtick.labelsize": 8.5,
            "ytick.labelsize": 8.5,
            "legend.fontsize": 8.5,
            "axes.unicode_minus": False,
            "axes.spines.top": False,
            "axes.spines.right": False,
            "axes.edgecolor": "#374151",
            "axes.labelcolor": "#1F2937",
            "axes.titlecolor": "#1F2937",
            "axes.linewidth": 0.8,
            "axes.grid": True,
            "grid.color": COLORS["grid"],
            "grid.linewidth": 0.6,
            "grid.alpha": 0.7,
            "xtick.color": "#374151",
            "ytick.color": "#374151",
            "xtick.major.width": 0.6,
            "ytick.major.width": 0.6,
            "legend.frameon": False,
            "figure.dpi": 150,
            "savefig.dpi": 200,
            "savefig.bbox": "tight",
            "savefig.pad_inches": 0.08,
        }
    )
    return font


def save(fig, out_path: str | Path, *, formats=("png", "svg")) -> list[Path]:
    """保存图像 (PNG + SVG); 显式不写 title。"""
    out_path = Path(out_path)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    written: list[Path] = []
    for fmt in formats:
        target = out_path.with_suffix(f".{fmt}")
        fig.savefig(target, format=fmt)
        written.append(target)
    plt.close(fig)
    return written


def method_color(method: str) -> str:
    return COLORS.get(method, "#666666")


def method_label(method: str) -> str:
    return METHOD_LABEL.get(method, method)


if __name__ == "__main__":
    print("可用 CJK 字体:", _pick_cjk_font())
