"""第 5 章实验图（v2 学术版，14 张）。

设计要点：
  · 仅对比 MOPS / Plan-Execute / ReAct 三个支持多轮和工具调用的方法。
  · 去掉 latency_efficiency / token_efficiency 这两个"时间/token 成本"类
    通常在论文图中会单独写表，不放评分热力的维度（5-4、5-6）。
  · 全部热力图严格方形 (aspect='equal')；颜色统一蓝色单色系。
  · 中文 9pt + 极简 spine + 白色 cell 分隔线。
  · 输出至 docs-figure/output/ch5_v2/。
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import matplotlib.patches as mpatches
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
from matplotlib.lines import Line2D
from scipy import stats

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from dataset.spec import (
    CATEGORY_LABEL_CN,
    DIFFICULTY_LABEL_CN,
    DIM_LABEL_CN,
    TOOL_LABEL_CN,
    WEIGHTS,
)
from styles.academic_style import (
    ACCENT_GOLD, ACCENT_RED, CATEGORY_COLORS, DIFFICULTY_COLORS,
    GRID, HIGHLIGHT, METHOD_COLORS, METHOD_ORDER, MUTED, OK_GREEN,
    SEQUENCE_BLUES, TEXT,
    annotate_bar, apply, cell_grid, diverging_blue_red, heat_blues,
    heat_blues_light, method_color, method_label, save_multi,
    square_heatmap, strip_spines, thin_legend,
)


DATA = ROOT / "dataset" / "out"
OUT = ROOT / "output" / "ch5_v2"
OUT.mkdir(parents=True, exist_ok=True)


# 评分维度：本文版本去掉 latency_efficiency（时间）和 token_efficiency（成本类，
# 主指标已含 Token 量），剩 11 维语义/能力维度
DIMS_PLOT = [
    "tool_selection",
    "root_cause",
    "suggestion",
    "safety",
    "efficiency",
    "duplicate_control",
    "completion_signal",
    "dialogue_intent",
    "task_chain_quality",
    "dialogue_completeness",
    "dialogue_satisfaction",
]


# ----------------------------------------------------------------------------
# 数据加载（默认排除 llm_only）
# ----------------------------------------------------------------------------

def _runs(include_llm_only: bool = False) -> pd.DataFrame:
    df = pd.read_csv(DATA / "runs.csv")
    if not include_llm_only:
        df = df[df["method"] != "llm_only"]
    df["method"] = pd.Categorical(df["method"],
                                  categories=list(METHOD_ORDER),
                                  ordered=True)
    return df


def _summary(include_llm_only: bool = False) -> pd.DataFrame:
    df = pd.read_csv(DATA / "method_summary.csv")
    if not include_llm_only:
        df = df[df["method"] != "llm_only"].reset_index(drop=True)
    return df


def _runs_with_breakdown(include_llm_only: bool = False) -> pd.DataFrame:
    with (DATA / "runs.json").open() as f:
        runs = json.load(f)
    rows = []
    for r in runs:
        if not include_llm_only and r["method"] == "llm_only":
            continue
        row = {k: v for k, v in r.items() if k not in ("score_breakdown", "called_tools")}
        for dim, v in r["score_breakdown"].items():
            row[f"sb_{dim}"] = v
        rows.append(row)
    df = pd.DataFrame(rows)
    df["method"] = pd.Categorical(df["method"],
                                  categories=list(METHOD_ORDER),
                                  ordered=True)
    return df


# ============================================================================
# 图 5.1  6 维主指标雷达图
# 用途：一眼看出 MOPS 在工具选择/根因/建议/权限/完成信号 5 项上接近满分，
#       PS/ReAct 在多个维度上有明显短板。
# ============================================================================

def fig5_1_radar():
    df = _summary().set_index("method")
    metrics = [
        ("normalized_weighted_score_100", "归一化加权得分", 100),
        ("avg_tool_selection_f1",         "工具选择 F1",   1.0),
        ("root_cause_hit_rate",           "根因命中率",    1.0),
        ("suggestion_relevance_rate",     "建议相关率",    1.0),
        ("permission_compliance_rate",    "权限合规率",    1.0),
        ("completion_signal_rate",        "完成信号率",    1.0),
    ]
    labels = [m[1] for m in metrics]
    angles = np.linspace(0, 2 * np.pi, len(metrics), endpoint=False).tolist()
    angles += angles[:1]

    fig, ax = plt.subplots(figsize=(5.0, 4.8), subplot_kw=dict(polar=True))
    ax.set_theta_offset(np.pi / 2)
    ax.set_theta_direction(-1)
    ax.set_rlabel_position(0)

    # ylim 与最外圈网格刻度严格对齐 (避免出现 spine + gridline 两圈套同心圆)
    ax.set_ylim(0.4, 1.0)
    ax.set_yticks([0.5, 0.6, 0.7, 0.8, 0.9, 1.0])
    ax.set_yticklabels(["", "60%", "", "80%", "", "100%"],
                        fontsize=7.5, color=MUTED)
    ax.set_xticks(angles[:-1])
    ax.set_xticklabels(labels, fontsize=9)

    # 完全隐藏极坐标 spine，仅用最外圈网格作视觉边界
    ax.spines["polar"].set_visible(False)
    ax.set_facecolor("#FFFFFF")
    ax.grid(color=GRID, linewidth=0.5, linestyle="-")

    # 先画非本文方法 (浅蓝填充薄)，再画 MOPS (深蓝填充厚) 保证 MOPS 在最上层
    for method in [m for m in METHOD_ORDER if m != "mops"] + ["mops"]:
        vals = [df.loc[method, k] / mx for k, _, mx in metrics]
        vals += vals[:1]
        if method == "mops":
            ax.plot(angles, vals, linewidth=2.2,
                    color=method_color(method), marker="o", markersize=4.5,
                    label=method_label(method, full=True), zorder=4)
            ax.fill(angles, vals, color=method_color(method),
                    alpha=0.22, zorder=3)
        else:
            ax.plot(angles, vals, linewidth=1.4,
                    color=method_color(method), marker="o", markersize=3.4,
                    label=method_label(method, full=True), zorder=2)
            ax.fill(angles, vals, color=method_color(method),
                    alpha=0.07, zorder=1)

    # 自定义 legend 顺序：MOPS 总在最左
    handles, labels_leg = ax.get_legend_handles_labels()
    order = sorted(range(len(labels_leg)),
                    key=lambda i: 0 if "MOPS" in labels_leg[i] else
                                  (1 if "Plan" in labels_leg[i] else 2))
    leg = ax.legend([handles[i] for i in order],
                    [labels_leg[i] for i in order],
                    loc="lower center", bbox_to_anchor=(0.5, -0.13),
                    ncol=3, frameon=True, framealpha=0.92,
                    facecolor="#FFFFFF", edgecolor="none", fontsize=8.5)
    leg.get_frame().set_linewidth(0)

    save_multi(fig, OUT / "fig5-1-radar")


# ============================================================================
# 图 5.2  难度分层综合得分 (含 95% CI)
# 用途：证明 MOPS 在易/中/难三档上都稳定领先，且 hard 难度上下降斜率最缓。
# ============================================================================

def fig5_2_difficulty_score_with_ci():
    df = _runs()
    fig, ax = plt.subplots(figsize=(5.4, 3.2))
    difficulties = ["easy", "medium", "hard"]
    x = np.arange(len(difficulties))
    width = 0.26

    means_by_m = {}
    for i, method in enumerate(METHOD_ORDER):
        means, errs = [], []
        for d in difficulties:
            sub = df[(df["method"] == method) & (df["difficulty"] == d)]["overall_score_100"]
            means.append(float(sub.mean()))
            errs.append(float(sub.std(ddof=1) / np.sqrt(len(sub)) * 1.96))
        means_by_m[method] = means
        bars = ax.bar(x + (i - 1) * width, means, width,
                      yerr=errs, capsize=2.5,
                      color=method_color(method),
                      edgecolor="white", linewidth=0.6,
                      label=method_label(method),
                      error_kw=dict(elinewidth=0.7, ecolor="#374151"))
        # 在 MOPS 柱顶标注（突出本文方法）
        if method == "mops":
            for b, v in zip(bars, means):
                ax.text(b.get_x() + b.get_width() / 2, v + 1.6,
                        f"{v:.1f}", ha="center", va="bottom",
                        fontsize=7.5, color=HIGHLIGHT, fontweight="bold")

    ax.set_xticks(x)
    ax.set_xticklabels([DIFFICULTY_LABEL_CN[d] for d in difficulties])
    ax.set_ylabel("综合得分（百分制）")
    ax.set_ylim(55, 100)
    strip_spines(ax)
    thin_legend(ax, loc="lower left", ncol=3)
    ax.text(1.0, -0.18, "误差棒：95% 置信区间",
            transform=ax.transAxes, ha="right", color=MUTED, fontsize=7.5)
    save_multi(fig, OUT / "fig5-2-difficulty-ci")


# ============================================================================
# 图 5.3  类别 × 方法 综合得分热力图（方形）
# 用途：跨 4 类业务展示 MOPS 全面均衡。
# ============================================================================

def fig5_3_category_method_heatmap():
    df = _runs()
    pivot = df.pivot_table(index="category", columns="method",
                            values="overall_score_100", aggfunc="mean",
                            observed=True)
    pivot = pivot.reindex(index=["diagnosis", "ops", "submission", "query"],
                          columns=list(METHOD_ORDER))

    # 严格方形：每个 cell 1.0×1.0，宽 3×1.0=3，高 4×1.0=4
    fig, ax = plt.subplots(figsize=(3.8, 4.4))
    cmap = heat_blues()
    im = ax.imshow(pivot.values, cmap=cmap, vmin=55, vmax=95)
    square_heatmap(ax)

    ax.set_xticks(np.arange(pivot.shape[1]))
    ax.set_xticklabels([method_label(m) for m in pivot.columns])
    ax.set_yticks(np.arange(pivot.shape[0]))
    ax.set_yticklabels([CATEGORY_LABEL_CN[c] for c in pivot.index])

    cell_grid(ax, pivot.shape[0], pivot.shape[1])

    for i in range(pivot.shape[0]):
        for j in range(pivot.shape[1]):
            v = pivot.values[i, j]
            color = "white" if v > 80 else TEXT
            weight = "bold" if pivot.columns[j] == "mops" else "normal"
            ax.text(j, i, f"{v:.1f}", ha="center", va="center",
                    fontsize=9, color=color, fontweight=weight)

    cbar = fig.colorbar(im, ax=ax, fraction=0.046, pad=0.04, aspect=18)
    cbar.set_label("综合得分", fontsize=8)
    cbar.ax.tick_params(labelsize=7.5)
    cbar.outline.set_linewidth(0)

    save_multi(fig, OUT / "fig5-3-category-heatmap")


# ============================================================================
# 图 5.4  11 维评分 split-violin (MOPS vs ReAct)
# 用途：维度级精细对比，突出 MOPS 在任务链质量/去重/完成信号上的优势。
# ============================================================================

def fig5_4_dim_violin_split():
    df = _runs_with_breakdown()
    fig, ax = plt.subplots(figsize=(7.6, 3.4))
    positions = np.arange(len(DIMS_PLOT))
    for j, method in enumerate(("mops", "react")):
        side = -1 if j == 0 else 1
        for i, dim in enumerate(DIMS_PLOT):
            sub = df[df["method"] == method][f"sb_{dim}"] / WEIGHTS[dim]
            parts = ax.violinplot([sub.values], positions=[i],
                                  widths=0.85, showmeans=False, showmedians=False,
                                  showextrema=False)
            for body in parts["bodies"]:
                m_x = np.mean(body.get_paths()[0].vertices[:, 0])
                verts = body.get_paths()[0].vertices
                if side < 0:
                    verts[:, 0] = np.clip(verts[:, 0], -np.inf, m_x)
                else:
                    verts[:, 0] = np.clip(verts[:, 0], m_x, np.inf)
                body.set_facecolor(method_color(method))
                body.set_edgecolor("white")
                body.set_alpha(0.82)
                body.set_linewidth(0.6)
            mean = sub.mean()
            ax.hlines(mean, i + side * 0.02, i + side * 0.40,
                      color=TEXT, linewidth=0.9)

    ax.set_xticks(positions)
    ax.set_xticklabels([DIM_LABEL_CN[d] for d in DIMS_PLOT],
                        rotation=30, ha="right")
    ax.set_ylim(0, 1.05)
    ax.set_ylabel("达成率（score / 权重）")
    strip_spines(ax)
    legend_h = [
        mpatches.Patch(color=method_color("mops"), label="MOPS（本文，左半）"),
        mpatches.Patch(color=method_color("react"), label="ReAct（右半）"),
    ]
    ax.legend(handles=legend_h, loc="lower left",
              frameon=True, framealpha=0.92,
              facecolor="#FFFFFF", edgecolor="none", fontsize=8.5)
    save_multi(fig, OUT / "fig5-4-dim-violin")


# ============================================================================
# 图 5.5  Token-得分散点 + 拟合
# 用途：性价比定位图，MOPS 处于"高分中等成本"位置。
# ============================================================================

def fig5_5_token_score_scatter():
    df = _runs()
    fig, ax = plt.subplots(figsize=(5.6, 3.4))
    for method in METHOD_ORDER:
        sub = df[df["method"] == method]
        ax.scatter(sub["estimated_total_tokens"], sub["overall_score_100"],
                   color=method_color(method), s=22, alpha=0.62,
                   edgecolor="white", linewidth=0.5,
                   label=method_label(method))
        if len(sub) > 5:
            x = sub["estimated_total_tokens"].values
            y = sub["overall_score_100"].values
            poly = np.polyfit(np.log10(x + 1), y, 1)
            xs = np.geomspace(max(200, x.min()), x.max(), 80)
            ys = np.polyval(poly, np.log10(xs + 1))
            ax.plot(xs, ys, color=method_color(method),
                    linewidth=1.2, alpha=0.65)

    # 在 MOPS 平均位置加一个"Pareto 最优"标签
    mops_x = df[df["method"] == "mops"]["estimated_total_tokens"].mean()
    mops_y = df[df["method"] == "mops"]["overall_score_100"].mean()
    ax.scatter([mops_x], [mops_y], s=120, facecolor="none",
               edgecolor=ACCENT_RED, linewidth=1.4, zorder=5)
    ax.annotate("MOPS 均值", xy=(mops_x, mops_y),
                xytext=(mops_x * 1.4, mops_y - 5),
                fontsize=8, color=ACCENT_RED,
                arrowprops=dict(arrowstyle="->", color=ACCENT_RED,
                                 lw=0.7))

    ax.set_xscale("log")
    ax.set_xlabel("估计 Token 消耗（log 轴）")
    ax.set_ylabel("综合得分")
    ax.set_ylim(45, 100)
    strip_spines(ax)
    thin_legend(ax, loc="lower right")
    save_multi(fig, OUT / "fig5-5-token-score")


# ============================================================================
# 图 5.6  维度×类别×方法 三联热力图（方形 cell）
# 用途：全景图——MOPS 在每类×每维上都更深（高分）。
# ============================================================================

def fig5_6_dim_category_heatmap():
    df = _runs_with_breakdown()
    cats = ["diagnosis", "ops", "submission", "query"]
    methods = ("mops", "ps", "react")

    fig = plt.figure(figsize=(11.0, 3.6))
    gs = fig.add_gridspec(1, 4,
                          width_ratios=[1, 1, 1, 0.06],
                          wspace=0.30)
    axes = [fig.add_subplot(gs[0, i]) for i in range(3)]
    cax = fig.add_subplot(gs[0, 3])
    cmap = heat_blues()
    im = None
    for k, (ax, method) in enumerate(zip(axes, methods)):
        sub = df[df["method"] == method]
        mat = []
        for cat in cats:
            row = []
            for dim in DIMS_PLOT:
                vals = sub[sub["category"] == cat][f"sb_{dim}"] / WEIGHTS[dim]
                row.append(float(vals.mean()) if len(vals) else np.nan)
            mat.append(row)
        mat = np.array(mat)
        im = ax.imshow(mat, cmap=cmap, vmin=0.4, vmax=1.0)
        square_heatmap(ax)
        ax.set_xticks(np.arange(len(DIMS_PLOT)))
        ax.set_xticklabels([DIM_LABEL_CN[d] for d in DIMS_PLOT],
                            rotation=40, ha="right", fontsize=7.5)
        ax.set_yticks(np.arange(len(cats)))
        if k == 0:
            ax.set_yticklabels([CATEGORY_LABEL_CN[c] for c in cats], fontsize=8.5)
        else:
            ax.set_yticklabels([])
        # 子标题（中文 + 高亮本文方法）
        is_self = (method == "mops")
        ax.set_title(method_label(method, full=True),
                      fontsize=9, color=HIGHLIGHT if is_self else TEXT,
                      fontweight="bold" if is_self else "normal", pad=6)
        cell_grid(ax, len(cats), len(DIMS_PLOT))

    cbar = fig.colorbar(im, cax=cax)
    cbar.set_label("达成率", fontsize=8)
    cbar.ax.tick_params(labelsize=7.5)
    cbar.outline.set_linewidth(0)
    save_multi(fig, OUT / "fig5-6-dim-category-heatmap")


# ============================================================================
# 图 5.7  难度 × 类别 小多图 (小提琴)
# 用途：12 个 (类别, 难度) 单元格里 MOPS 都领先。
# ============================================================================

def fig5_7_difficulty_category_small_multiples():
    df = _runs()
    cats = ["diagnosis", "ops", "submission", "query"]
    diffs = ["easy", "medium", "hard"]
    fig, axes = plt.subplots(len(cats), len(diffs), figsize=(7.6, 7.2),
                              sharey=True, sharex=True)
    for ci, cat in enumerate(cats):
        for di, diff in enumerate(diffs):
            ax = axes[ci, di]
            data, colors, labels = [], [], []
            for m in METHOD_ORDER:
                sub = df[(df["category"] == cat)
                          & (df["difficulty"] == diff)
                          & (df["method"] == m)]["overall_score_100"].values
                if len(sub) == 0:
                    continue
                data.append(sub)
                colors.append(method_color(m))
                labels.append(method_label(m))

            if data:
                parts = ax.violinplot(data, positions=range(len(data)),
                                      showmeans=False, showmedians=True,
                                      widths=0.82, showextrema=False)
                for body, c in zip(parts["bodies"], colors):
                    body.set_facecolor(c)
                    body.set_edgecolor("white")
                    body.set_alpha(0.80)
                parts["cmedians"].set_color(TEXT)
                parts["cmedians"].set_linewidth(0.9)
                # 在 MOPS 位置画一个均值点
                if "mops" in labels:
                    idx = labels.index("mops")
                    ax.scatter([idx], [np.mean(data[idx])], s=18,
                               color=ACCENT_RED, zorder=5, edgecolor="white",
                               linewidth=0.6)
                ax.set_xticks(range(len(labels)))
                ax.set_xticklabels(labels, fontsize=7)
            else:
                ax.text(0.5, 0.5, "—", transform=ax.transAxes,
                        ha="center", va="center", fontsize=12, color=MUTED)
                ax.set_xticks([])
            ax.set_ylim(45, 100)
            ax.grid(axis="y", linewidth=0.4, color=GRID)
            ax.grid(axis="x", visible=False)
            if di == 0:
                ax.set_ylabel(CATEGORY_LABEL_CN[cat], fontsize=9)
            if ci == 0:
                ax.set_title(DIFFICULTY_LABEL_CN[diff],
                              fontsize=9.5, color=TEXT, pad=4)
            strip_spines(ax)
    fig.text(0.5, -0.01, "难度（每个子图：MOPS / Plan-Execute / ReAct）",
              ha="center", fontsize=9, color=TEXT)
    fig.text(-0.01, 0.5, "业务类别", va="center", rotation=90,
              fontsize=9, color=TEXT)
    fig.tight_layout()
    save_multi(fig, OUT / "fig5-7-difficulty-category")


# ============================================================================
# 图 5.8  Top-12 工具调用频次（水平柱）
# 用途：工具使用偏好对比。MOPS 更倾向特定语义化工具。
# ============================================================================

def fig5_8_tool_frequency():
    df = _runs()
    tool_rows = []
    for _, r in df.iterrows():
        tools = json.loads(r["called_tools"])
        for t in tools:
            tool_rows.append({"method": r["method"], "tool": t})
    tdf = pd.DataFrame(tool_rows)
    pivot = tdf.pivot_table(index="tool", columns="method",
                             values="tool", aggfunc=len, observed=True).fillna(0)
    pivot = pivot.reindex(columns=list(METHOD_ORDER), fill_value=0)
    pivot["__total"] = pivot.sum(axis=1)
    pivot = pivot.sort_values("__total", ascending=True).tail(12)
    pivot = pivot.drop(columns="__total")

    fig, ax = plt.subplots(figsize=(6.0, 3.8))
    y = np.arange(len(pivot))
    height = 0.27
    for i, m in enumerate(METHOD_ORDER):
        ax.barh(y + (i - 1) * height, pivot[m].values, height,
                color=method_color(m), edgecolor="white", linewidth=0.5,
                label=method_label(m), alpha=0.95)
    ax.set_yticks(y)
    ax.set_yticklabels([TOOL_LABEL_CN.get(t, t) for t in pivot.index],
                        fontsize=8.5)
    ax.set_xlabel("调用次数")
    ax.grid(axis="x", linewidth=0.4, color=GRID)
    ax.grid(axis="y", visible=False)
    strip_spines(ax)
    thin_legend(ax, loc="lower right", ncol=3)
    save_multi(fig, OUT / "fig5-8-tool-frequency")


# ============================================================================
# 图 5.9  LLM 调用次数分布（小提琴 + 散点）
# 用途：透明披露 MOPS 多角色协同代价 ——
#       需要在文中辅以"异构模型 ×0.42 成本"论据补偿。
# ============================================================================

def fig5_9_llm_calls_distribution():
    df = _runs()
    fig, ax = plt.subplots(figsize=(5.0, 3.0))
    positions = []
    for m in METHOD_ORDER:
        vals = df[df["method"] == m]["llm_calls"].values
        positions.append(vals)
    parts = ax.violinplot(positions, positions=range(3), widths=0.78,
                          showmeans=False, showmedians=True, showextrema=False)
    for body, m in zip(parts["bodies"], METHOD_ORDER):
        body.set_facecolor(method_color(m))
        body.set_edgecolor("white")
        body.set_alpha(0.78)
    parts["cmedians"].set_color(TEXT)
    parts["cmedians"].set_linewidth(0.9)
    for i, m in enumerate(METHOD_ORDER):
        vals = df[df["method"] == m]["llm_calls"].values
        jitter = np.random.uniform(-0.08, 0.08, size=len(vals))
        ax.scatter(np.full(len(vals), i) + jitter, vals,
                   color=method_color(m), s=7, alpha=0.45,
                   edgecolor="white", linewidth=0.2)
        # 均值横杠
        ax.hlines(np.mean(vals), i - 0.35, i + 0.35,
                  color=ACCENT_RED, linewidth=1.1, zorder=4)
    ax.set_xticks(range(3))
    ax.set_xticklabels([method_label(m) for m in METHOD_ORDER])
    ax.set_ylabel("LLM 调用次数")
    ax.grid(axis="y", linewidth=0.4, color=GRID)
    ax.grid(axis="x", visible=False)
    strip_spines(ax)
    save_multi(fig, OUT / "fig5-9-llm-calls")


# ============================================================================
# 图 5.10  角色消融柱状图（4 子配置 vs 完整 MOPS）
# 用途：验证 Planner/Verifier/Coordinator 的必要性。
# ============================================================================

def fig5_10_role_ablation():
    df = pd.read_csv(DATA / "ablation_roles_summary.csv")
    df = df.set_index("config_label").reindex(
        ["完整 MOPS", "w/o Planner", "w/o Verifier",
         "w/o Coordinator", "w/o Plan + Verifier"]
    )
    metrics = [
        ("avg_overall_score_100", "综合得分",         (82, 95), "{:.2f}"),
        ("avg_tool_selection_f1", "工具选择 F1",     (0.85, 1.0), "{:.3f}"),
        ("root_cause_hit_rate",   "根因命中率",     (0.80, 1.0), "{:.2f}"),
    ]
    # 5 个配置：完整 MOPS 用深蓝（强调），其余用渐变浅
    palette = ["#08306B", "#2171B5", "#4292C6", "#6BAED6", "#9ECAE1"]
    fig, axes = plt.subplots(1, 3, figsize=(9.2, 3.1))
    for ax, (col, ylabel, ylim, fmt) in zip(axes, metrics):
        bars = ax.bar(range(len(df)), df[col].values,
                      color=palette[:len(df)],
                      edgecolor="white", linewidth=0.6, width=0.62)
        ax.set_xticks(range(len(df)))
        ax.set_xticklabels(df.index, rotation=20, ha="right", fontsize=8)
        ax.set_ylabel(ylabel)
        ax.set_ylim(*ylim)
        ax.grid(axis="y", linewidth=0.4, color=GRID)
        ax.grid(axis="x", visible=False)
        strip_spines(ax)
        for b, v in zip(bars, df[col].values):
            ax.text(b.get_x() + b.get_width() / 2,
                    v + (ylim[1] - ylim[0]) * 0.012,
                    fmt.format(v),
                    ha="center", va="bottom", fontsize=7.5, color=TEXT)
    fig.tight_layout()
    save_multi(fig, OUT / "fig5-10-role-ablation")


# ============================================================================
# 图 5.11  模型配置消融：质量 vs 相对 Token 成本（双坐标轴）
# 用途：对比三种部署配置 + DSV4-Flash 全量，展示异构路由在
#       质量-成本 Pareto 前沿上的优势（以 48% 成本获得接近全量 Pro 的质量）。
# ============================================================================

def fig5_11_model_ablation():
    df = pd.read_csv(DATA / "ablation_models_summary.csv")
    # 使用简短标签用于图表展示
    df["short_label"] = df["config"].map({
        "qwen36plus_baseline": "Qwen3.6-Plus\n全量",
        "ds_pro_all":          "DSV4-Pro\n全量",
        "ds_heterogeneous":    "DSV4-Flash\n+ DSV4-Pro",
        "ds_flash_all":        "DSV4-Flash\n全量",
    })
    df = df.set_index("short_label").reindex([
        "Qwen3.6-Plus\n全量",
        "DSV4-Pro\n全量",
        "DSV4-Flash\n+ DSV4-Pro",
        "DSV4-Flash\n全量",
    ])

    SCORE_COLOR = "#08306B"   # 深蓝：综合得分
    COST_COLOR  = "#4292C6"   # 中蓝：相对成本

    fig, ax = plt.subplots(figsize=(7.2, 4.0))
    x = np.arange(len(df))
    width = 0.32

    # 综合得分柱（左轴）
    bars1 = ax.bar(x - width/2, df["avg_overall_score_100"].values, width,
                   color=SCORE_COLOR,
                   edgecolor="white", linewidth=0.6,
                   label="综合得分", zorder=3)
    ax.set_ylabel("综合得分", color=SCORE_COLOR, fontsize=10)
    ax.tick_params(axis="y", labelcolor=SCORE_COLOR)
    ax.set_ylim(75, 98)
    ax.set_xticks(x)
    ax.set_xticklabels(df.index, fontsize=8.5)
    ax.grid(axis="y", linewidth=0.4, color=GRID)
    ax.grid(axis="x", visible=False)
    ax.set_axisbelow(True)
    strip_spines(ax)

    # 相对成本柱（右轴）
    ax2 = ax.twinx()
    bars2 = ax2.bar(x + width/2, df["relative_cost"].values * 100, width,
                    color=COST_COLOR,
                    edgecolor="white", linewidth=0.6,
                    alpha=0.88, label="相对 Token 成本", zorder=3)
    ax2.set_ylabel("相对 Token 成本 (%)", color=COST_COLOR, fontsize=10)
    ax2.tick_params(axis="y", labelcolor=COST_COLOR)
    ax2.set_ylim(0, 110)
    ax2.grid(False)
    ax2.spines["top"].set_visible(False)

    # 数值标签
    for b, v in zip(bars1, df["avg_overall_score_100"].values):
        ax.text(b.get_x() + b.get_width() / 2, v + 0.35,
                f"{v:.2f}", ha="center", va="bottom",
                fontsize=8.5, color=SCORE_COLOR, fontweight="bold")
    for b, v in zip(bars2, df["relative_cost"].values):
        ax2.text(b.get_x() + b.get_width() / 2, v * 100 + 2.5,
                 f"{v*100:.0f}%", ha="center", va="bottom",
                 fontsize=8.5, color=COST_COLOR, fontweight="bold")

    # 图例（仅两项）
    handles = [mpatches.Patch(color=SCORE_COLOR, label="综合得分"),
               mpatches.Patch(color=COST_COLOR, label="相对 Token 成本")]
    ax.legend(handles=handles, loc="upper right",
              frameon=True, framealpha=0.92,
              facecolor="#FFFFFF", edgecolor="none", fontsize=8.5,
              bbox_to_anchor=(0.98, 0.98))

    # 底部注释
    ax.text(0.98, -0.18,
            "相对 Token 成本 = 配置词元消耗 / Qwen3.6-Plus 基准消耗",
            transform=ax.transAxes, ha="right", fontsize=7, color=MUTED)

    fig.tight_layout()
    save_multi(fig, OUT / "fig5-11-model-ablation")


# ============================================================================
# 图 5.12  线上抽检 5 维 (boxplot + 高/低分率)
# 用途：120 会话人工抽检，证明 MOPS 线上质量在 5 维都稳定。
# ============================================================================

def fig5_12_online_quality():
    sess = pd.read_csv(DATA / "online_sessions.csv")
    summary = pd.read_csv(DATA / "online_summary.csv")
    dims = ["tool_correctness", "diagnosis_accuracy", "helpfulness",
            "safety", "hallucination_avoid"]
    label_map = {
        "tool_correctness": "工具正确性",
        "diagnosis_accuracy": "诊断准确性",
        "helpfulness": "回复有用性",
        "safety": "安全合规性",
        "hallucination_avoid": "幻觉抑制",
    }

    fig, axes = plt.subplots(1, 2, figsize=(8.2, 3.2),
                              gridspec_kw=dict(width_ratios=[1.25, 1]))

    # 左：5 维 boxplot（蓝色阶梯）
    ax = axes[0]
    data = [sess[d].values for d in dims]
    bp = ax.boxplot(
        data, tick_labels=[label_map[d] for d in dims],
        patch_artist=True, widths=0.55,
        medianprops=dict(color="white", linewidth=1.2),
        flierprops=dict(marker="o", markersize=2.5,
                         markerfacecolor=MUTED, markeredgecolor="none",
                         alpha=0.6),
        whiskerprops=dict(color="#374151", linewidth=0.7),
        capprops=dict(color="#374151", linewidth=0.7),
        boxprops=dict(linewidth=0.6),
    )
    palette = ["#08306B", "#2171B5", "#4292C6", "#6BAED6", "#9ECAE1"]
    for patch, c in zip(bp["boxes"], palette):
        patch.set_facecolor(c)
        patch.set_edgecolor(c)
        patch.set_alpha(0.88)
    ax.set_ylim(1, 5.2)
    ax.set_ylabel("评分（1–5 分）")
    plt.setp(ax.get_xticklabels(), rotation=18, ha="right", fontsize=8.5)
    ax.grid(axis="y", linewidth=0.4, color=GRID)
    ax.grid(axis="x", visible=False)
    strip_spines(ax)

    # 右：高分 / 低分双向条形
    ax2 = axes[1]
    summary_dims = summary.set_index("dim_key").reindex(dims)
    high = summary_dims["high_rate"].values * 100
    low = summary_dims["low_rate"].values * 100
    y = np.arange(len(dims))
    ax2.barh(y, high, color=OK_GREEN, height=0.55, alpha=0.88,
             label="≥ 4 分占比")
    ax2.barh(y, -low, color=ACCENT_RED, height=0.55, alpha=0.88,
             label="< 3 分占比")
    ax2.set_yticks(y)
    ax2.set_yticklabels([label_map[d] for d in dims], fontsize=8.5)
    ax2.invert_yaxis()
    ax2.set_xticks([-20, -10, 0, 25, 50, 75, 100])
    ax2.set_xticklabels(["20%", "10%", "0%", "25%", "50%", "75%", "100%"],
                         fontsize=7.5)
    ax2.set_xlim(-25, 110)
    ax2.axvline(0, color="#374151", linewidth=0.6)
    for v, yi in zip(high, y):
        ax2.text(v + 1, yi, f"{v:.0f}%", va="center",
                 fontsize=7.5, color=TEXT)
    for v, yi in zip(low, y):
        ax2.text(-v - 1, yi, f"{v:.0f}%", va="center", ha="right",
                 fontsize=7.5, color=TEXT)
    ax2.grid(axis="x", linewidth=0.4, color=GRID)
    ax2.grid(axis="y", visible=False)
    strip_spines(ax2, keep=("bottom",))
    # legend 放到图外上方，避免遮挡数据
    leg = ax2.legend(loc="lower center", bbox_to_anchor=(0.5, -0.30),
                     ncol=2, frameon=True, framealpha=0.92,
                     facecolor="#FFFFFF", edgecolor="none", fontsize=8)
    leg.get_frame().set_linewidth(0)

    fig.text(0.99, -0.02, f"样本 n = {len(sess)}",
              ha="right", color=MUTED, fontsize=7.5)
    fig.tight_layout()
    save_multi(fig, OUT / "fig5-12-online-quality")


# ============================================================================
# 图 5.13  配对 t-检验 显著性矩阵（方形 3×3）
# 用途：量化 MOPS 优势的统计显著性。
# ============================================================================

def fig5_13_significance():
    df = _runs()
    methods = list(METHOD_ORDER)   # MOPS / Plan-Execute / ReAct
    n = len(methods)
    pmat = np.full((n, n), np.nan)
    diff_mat = np.full((n, n), np.nan)
    # 同场景下逐场景配对单尾 t 检验：H1: 方法 A 优于方法 B
    for i, a in enumerate(methods):
        for j, b in enumerate(methods):
            if i == j:
                continue
            sub_a = df[df["method"] == a].set_index("scenario_id")["overall_score_100"]
            sub_b = df[df["method"] == b].set_index("scenario_id")["overall_score_100"]
            common = sub_a.index.intersection(sub_b.index)
            x = sub_a.loc[common].values
            y = sub_b.loc[common].values
            t = stats.ttest_rel(x, y, alternative="greater")
            pmat[i, j] = t.pvalue
            diff_mat[i, j] = float(np.mean(x - y))

    fig, ax = plt.subplots(figsize=(4.4, 4.4))
    cmap = diverging_blue_red()
    # 对角线 NaN → 完全透明，融入背景
    cmap.set_bad(color="#FFFFFF", alpha=0.0)
    masked = np.ma.masked_invalid(diff_mat)
    vmax = max(1.0, float(np.nanmax(np.abs(diff_mat))))
    im = ax.imshow(masked, cmap=cmap, vmin=-vmax, vmax=vmax)
    square_heatmap(ax)

    ax.set_xticks(range(n))
    ax.set_xticklabels([method_label(m) for m in methods])
    ax.set_yticks(range(n))
    ax.set_yticklabels([method_label(m) for m in methods])
    ax.set_xlabel("被对比方法 B", labelpad=8)
    ax.set_ylabel("主方法 A", labelpad=8)

    # cell 之间用白线分隔；对角线由于透明，分隔线在白底上"隐形"
    cell_grid(ax, n, n)

    for i in range(n):
        for j in range(n):
            if np.isnan(diff_mat[i, j]):
                continue                       # 对角线：不绘制任何文字
            v = diff_mat[i, j]
            p = pmat[i, j]
            color = "white" if abs(v) > vmax * 0.55 else TEXT
            # 主体：粗体大字 Δ 值
            ax.text(j, i - 0.10, f"{v:+.1f}",
                    ha="center", va="center",
                    fontsize=12, color=color, fontweight="bold")
            # 副文本：以"p 值"显式呈现显著性，比 *** 更专业
            if p < 0.001:
                p_text = "p < 0.001"
            elif p < 0.01:
                p_text = f"p = {p:.3f}"
            elif p < 0.05:
                p_text = f"p = {p:.3f}"
            else:
                p_text = "n.s."
            ax.text(j, i + 0.26, p_text,
                    ha="center", va="center",
                    fontsize=7.5, color=color, alpha=0.88)

    cbar = fig.colorbar(im, ax=ax, fraction=0.046, pad=0.04, aspect=18)
    cbar.set_label("A 与 B 平均得分差", fontsize=8)
    cbar.ax.tick_params(labelsize=7.5)
    cbar.outline.set_linewidth(0)

    ax.text(1.0, -0.20,
            "配对单尾 t 检验 (备择假设: A 平均得分 > B)；n.s. 表示 p ≥ 0.05",
            transform=ax.transAxes, ha="right", fontsize=7, color=MUTED)
    save_multi(fig, OUT / "fig5-13-significance")


# ============================================================================
# 图 5.14  多维平行坐标 (3 方法)
# 用途：多指标综合 ranking 视图。
# ============================================================================

def fig5_14_parallel():
    df = _summary().set_index("method")
    dims = [
        ("normalized_weighted_score_100", "综合得分",   60,  95),
        ("avg_tool_selection_f1",         "工具 F1",   0.5, 1.0),
        ("root_cause_hit_rate",           "根因命中", 0.5, 1.0),
        ("suggestion_relevance_rate",     "建议相关", 0.5, 1.0),
        ("permission_compliance_rate",    "权限合规", 0.85, 1.0),
        ("completion_signal_rate",        "完成信号", 0.5, 1.0),
    ]
    fig, ax = plt.subplots(figsize=(7.0, 3.2))
    x = np.arange(len(dims))
    for method in METHOD_ORDER:
        vals = []
        for key, _, lo, hi in dims:
            v = df.loc[method, key]
            vals.append((v - lo) / (hi - lo))
        ax.plot(x, vals, marker="o", linewidth=1.8, markersize=5.5,
                color=method_color(method),
                label=method_label(method, full=True))
    ax.set_xticks(x)
    ax.set_xticklabels([d[1] for d in dims])
    ax.set_ylim(-0.05, 1.1)
    ax.set_ylabel("各维度归一化值")
    for xi in x:
        ax.axvline(xi, color=GRID, linewidth=0.5, zorder=0)
    ax.grid(axis="y", visible=False)
    strip_spines(ax)
    thin_legend(ax, loc="lower left", ncol=3)
    save_multi(fig, OUT / "fig5-14-parallel")


# ============================================================================
# main
# ============================================================================

def main():
    apply()
    funcs = [
        fig5_1_radar,
        fig5_2_difficulty_score_with_ci,
        fig5_3_category_method_heatmap,
        fig5_4_dim_violin_split,
        fig5_5_token_score_scatter,
        fig5_6_dim_category_heatmap,
        fig5_7_difficulty_category_small_multiples,
        fig5_8_tool_frequency,
        fig5_9_llm_calls_distribution,
        fig5_10_role_ablation,
        fig5_11_model_ablation,
        fig5_12_online_quality,
        fig5_13_significance,
        fig5_14_parallel,
    ]
    for f in funcs:
        f()
        print(f"  ✓ {f.__name__}")
    print(f"\nch5 v2 学术版：共 {len(funcs)} 张图，输出到 {OUT}/")


if __name__ == "__main__":
    main()
