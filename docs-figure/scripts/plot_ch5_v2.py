"""第 5 章实验图 V2（Nature 风格、新数据集 v2、共 14 张）。"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import matplotlib.patches as mpatches
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
from matplotlib.lines import Line2D
from matplotlib.patches import FancyBboxPatch
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
from styles.nature_style import (
    ACCENT, CATEGORY_COLORS, DIFFICULTY_COLORS, GRID, METHOD_COLORS,
    MUTED, OK_GREEN, TEXT,
    apply, divergent_cmap, heat_cmap, method_color, method_label,
    nice_legend, save_multi, strip_spines, annotate_bar,
)


DATA = ROOT / "dataset" / "out"
OUT = ROOT / "output" / "ch5"
OUT.mkdir(parents=True, exist_ok=True)


# -----------------------------------------------------------------------------
# 数据加载
# -----------------------------------------------------------------------------

def _runs() -> pd.DataFrame:
    df = pd.read_csv(DATA / "runs.csv")
    df["method"] = pd.Categorical(df["method"],
                                  categories=["mops", "ps", "react", "llm_only"],
                                  ordered=True)
    return df


def _summary() -> pd.DataFrame:
    return pd.read_csv(DATA / "method_summary.csv")


def _runs_with_breakdown() -> pd.DataFrame:
    with (DATA / "runs.json").open() as f:
        runs = json.load(f)
    rows = []
    for r in runs:
        row = {k: v for k, v in r.items() if k not in ("score_breakdown", "called_tools")}
        for dim, v in r["score_breakdown"].items():
            row[f"sb_{dim}"] = v
        rows.append(row)
    df = pd.DataFrame(rows)
    df["method"] = pd.Categorical(df["method"],
                                  categories=["mops", "ps", "react", "llm_only"],
                                  ordered=True)
    return df


# =============================================================================
# 图 5.1 三方法主指标雷达 (4 方法)
# =============================================================================

def fig5_1_radar():
    df = _summary().set_index("method")
    metrics = [
        ("normalized_weighted_score_100", "归一化加权得分", 100),
        ("avg_tool_selection_f1",         "工具选择 F1", 1.0),
        ("root_cause_hit_rate",           "根因命中率", 1.0),
        ("suggestion_relevance_rate",     "建议相关率", 1.0),
        ("permission_compliance_rate",    "权限合规率", 1.0),
        ("completion_signal_rate",        "完成信号率", 1.0),
    ]
    labels = [m[1] for m in metrics]
    angles = np.linspace(0, 2 * np.pi, len(metrics), endpoint=False).tolist()
    angles += angles[:1]

    fig, ax = plt.subplots(figsize=(5.4, 5.0), subplot_kw=dict(polar=True))
    ax.set_theta_offset(np.pi / 2)
    ax.set_theta_direction(-1)
    ax.set_rlabel_position(45)
    ax.set_yticks([0.25, 0.5, 0.75, 1.0])
    ax.set_yticklabels(["25%", "50%", "75%", "100%"], fontsize=7.5, color=MUTED)
    ax.set_xticks(angles[:-1])
    ax.set_xticklabels(labels)
    ax.set_ylim(0, 1.05)

    for method in ("mops", "ps", "react", "llm_only"):
        vals = [df.loc[method, k] / mx for k, _, mx in metrics]
        vals += vals[:1]
        ax.plot(angles, vals, linewidth=1.7, color=method_color(method),
                label=method_label(method, full=True))
        ax.fill(angles, vals, color=method_color(method), alpha=0.10)

    # 极简网格
    ax.grid(color=GRID, linewidth=0.5, linestyle="-")
    ax.spines["polar"].set_color(GRID)
    ax.spines["polar"].set_linewidth(0.7)

    leg = ax.legend(loc="lower center", bbox_to_anchor=(0.5, -0.16),
                    ncol=4, frameon=True, framealpha=0.85,
                    facecolor="#FFFFFF", edgecolor="none",
                    fontsize=8.5)
    leg.get_frame().set_linewidth(0.0)

    save_multi(fig, OUT / "fig5-1-radar")


# =============================================================================
# 图 5.2 难度 × 方法 综合得分 (含 95% CI)
# =============================================================================

def fig5_2_difficulty_score_with_ci():
    df = _runs()
    fig, ax = plt.subplots(figsize=(6.4, 3.5))
    difficulties = ["easy", "medium", "hard"]
    x = np.arange(len(difficulties))
    width = 0.20

    for i, method in enumerate(("mops", "ps", "react", "llm_only")):
        means, errs = [], []
        for d in difficulties:
            sub = df[(df["method"] == method) & (df["difficulty"] == d)]["overall_score_100"]
            means.append(float(sub.mean()))
            errs.append(float(sub.std(ddof=1) / np.sqrt(len(sub)) * 1.96))
        bars = ax.bar(x + (i - 1.5) * width, means, width,
                      yerr=errs, capsize=2.5,
                      color=method_color(method),
                      edgecolor="white", linewidth=0.5,
                      label=method_label(method),
                      error_kw=dict(elinewidth=0.8, ecolor="#374151"))

    ax.set_xticks(x)
    ax.set_xticklabels([DIFFICULTY_LABEL_CN[d] for d in difficulties])
    ax.set_ylabel("综合得分（百分制）")
    ax.set_ylim(40, 100)
    strip_spines(ax)
    nice_legend(ax, loc="lower left", ncol=4)
    # 注脚
    ax.text(1.0, 0.02, "误差棒：95% 置信区间",
            transform=ax.transAxes, ha="right", color=MUTED, fontsize=7.5)
    save_multi(fig, OUT / "fig5-2-difficulty-ci")


# =============================================================================
# 图 5.3 类别 × 方法 综合得分热力 + 显著性符号
# =============================================================================

def fig5_3_category_method_heatmap():
    df = _runs()
    pivot = df.pivot_table(index="category", columns="method",
                            values="overall_score_100", aggfunc="mean",
                            observed=True)
    pivot = pivot.reindex(index=["diagnosis", "ops", "submission", "query"],
                          columns=["mops", "ps", "react", "llm_only"])

    fig, ax = plt.subplots(figsize=(5.4, 3.0))
    cmap = heat_cmap()
    im = ax.imshow(pivot.values, aspect="auto", cmap=cmap,
                   vmin=40, vmax=100)

    ax.set_xticks(np.arange(pivot.shape[1]))
    ax.set_xticklabels([method_label(m) for m in pivot.columns])
    ax.set_yticks(np.arange(pivot.shape[0]))
    ax.set_yticklabels([CATEGORY_LABEL_CN[c] for c in pivot.index])
    ax.grid(False)
    ax.set_xticks(np.arange(pivot.shape[1] + 1) - 0.5, minor=True)
    ax.set_yticks(np.arange(pivot.shape[0] + 1) - 0.5, minor=True)
    ax.tick_params(which="minor", length=0)
    ax.grid(which="minor", color="white", linewidth=1.0)

    # 标注得分（仅在 cell > 60 时显示，避免空白噪声）
    for i in range(pivot.shape[0]):
        for j in range(pivot.shape[1]):
            v = pivot.values[i, j]
            color = "white" if v > 75 else TEXT
            ax.text(j, i, f"{v:.1f}", ha="center", va="center",
                    fontsize=8.5, color=color)

    cbar = fig.colorbar(im, ax=ax, fraction=0.04, pad=0.02)
    cbar.set_label("综合得分", fontsize=8)
    cbar.ax.tick_params(labelsize=7.5)

    save_multi(fig, OUT / "fig5-3-category-heatmap")


# =============================================================================
# 图 5.4 评分 13 维 split-violin (MOPS vs ReAct)
# =============================================================================

def fig5_4_dim_violin_split():
    df = _runs_with_breakdown()
    dims = list(WEIGHTS.keys())
    # 只对比 MOPS 与 React (论点最鲜明)
    fig, ax = plt.subplots(figsize=(7.5, 3.6))
    positions = np.arange(len(dims))
    for j, method in enumerate(("mops", "react")):
        side = -1 if j == 0 else 1
        for i, dim in enumerate(dims):
            sub = df[df["method"] == method][f"sb_{dim}"] / WEIGHTS[dim]
            parts = ax.violinplot([sub.values], positions=[i],
                                  widths=0.85, showmeans=False, showmedians=False,
                                  showextrema=False)
            for body in parts["bodies"]:
                m = np.mean(body.get_paths()[0].vertices[:, 0])
                # 砍半（只保留一侧）
                verts = body.get_paths()[0].vertices
                if side < 0:
                    verts[:, 0] = np.clip(verts[:, 0], -np.inf, m)
                else:
                    verts[:, 0] = np.clip(verts[:, 0], m, np.inf)
                body.set_facecolor(method_color(method))
                body.set_edgecolor("white")
                body.set_alpha(0.78)
                body.set_linewidth(0.6)
            # 均值横杠
            mean = sub.mean()
            ax.hlines(mean, i + (side * 0.02), i + side * 0.40,
                      color=TEXT, linewidth=0.9)

    ax.set_xticks(positions)
    ax.set_xticklabels([DIM_LABEL_CN[d] for d in dims], rotation=35, ha="right")
    ax.set_ylim(0, 1.05)
    ax.set_ylabel("达成率（score / 权重）")
    strip_spines(ax)
    legend_h = [mpatches.Patch(color=method_color("mops"), label="MOPS (本文，左)"),
                mpatches.Patch(color=method_color("react"), label="ReAct（右）")]
    ax.legend(handles=legend_h, loc="lower left", frameon=True,
              framealpha=0.85, facecolor="#FFFFFF", edgecolor="none")
    save_multi(fig, OUT / "fig5-4-dim-violin")


# =============================================================================
# 图 5.5 Token 消耗 vs 综合得分散点 + 各方法回归线
# =============================================================================

def fig5_5_token_score_scatter():
    df = _runs()
    fig, ax = plt.subplots(figsize=(5.8, 3.6))
    for method in ("mops", "ps", "react", "llm_only"):
        sub = df[df["method"] == method]
        ax.scatter(sub["estimated_total_tokens"], sub["overall_score_100"],
                   color=method_color(method), s=26, alpha=0.62,
                   edgecolor="white", linewidth=0.5,
                   label=method_label(method))
        # 拟合
        if len(sub) > 5:
            x = sub["estimated_total_tokens"].values
            y = sub["overall_score_100"].values
            order = np.argsort(x)
            poly = np.polyfit(np.log10(x + 1), y, 1)
            xs = np.geomspace(max(150, x.min()), x.max(), 80)
            ys = np.polyval(poly, np.log10(xs + 1))
            ax.plot(xs, ys, color=method_color(method),
                    linewidth=1.1, alpha=0.6)

    ax.set_xscale("log")
    ax.set_xlabel("估计 Token 消耗（log10）")
    ax.set_ylabel("综合得分")
    ax.set_ylim(20, 105)
    strip_spines(ax)
    nice_legend(ax, loc="lower right")
    save_multi(fig, OUT / "fig5-5-token-score")


# =============================================================================
# 图 5.6 评分维度热力图 (3×categories × dimensions, MOPS vs PS vs React)
# =============================================================================

def fig5_6_dim_category_heatmap():
    df = _runs_with_breakdown()
    dims = list(WEIGHTS.keys())
    cats = ["diagnosis", "ops", "query", "submission"]
    methods = ("mops", "ps", "react")

    fig, axes = plt.subplots(1, 3, figsize=(11.5, 3.4),
                              gridspec_kw=dict(wspace=0.42, width_ratios=[1, 1, 1]))
    cmap = heat_cmap()
    im = None
    for ax, method in zip(axes, methods):
        sub = df[df["method"] == method]
        mat = []
        for cat in cats:
            row = []
            for dim in dims:
                vals = sub[sub["category"] == cat][f"sb_{dim}"] / WEIGHTS[dim]
                row.append(float(vals.mean()) if len(vals) else np.nan)
            mat.append(row)
        mat = np.array(mat)
        im = ax.imshow(mat, aspect="auto", cmap=cmap, vmin=0.3, vmax=1.0)
        ax.set_xticks(np.arange(len(dims)))
        ax.set_xticklabels([DIM_LABEL_CN[d] for d in dims],
                            rotation=35, ha="right", fontsize=7.5)
        ax.set_yticks(np.arange(len(cats)))
        ax.set_yticklabels([CATEGORY_LABEL_CN[c] for c in cats], fontsize=8.5)
        ax.set_xlabel(method_label(method, full=True), labelpad=4)
        ax.grid(False)
        ax.set_xticks(np.arange(len(dims) + 1) - 0.5, minor=True)
        ax.set_yticks(np.arange(len(cats) + 1) - 0.5, minor=True)
        ax.tick_params(which="minor", length=0)
        ax.grid(which="minor", color="white", linewidth=0.8)
        # 只在最高/最低 1-2 个 cell 标注，其他不写数字
        flat = mat.flatten()
        top_idx = np.argsort(flat)[-3:]
        low_idx = np.argsort(flat)[:2]
        for idx in list(top_idx) + list(low_idx):
            i, j = divmod(idx, len(dims))
            v = mat[i, j]
            color = "white" if v > 0.7 else TEXT
            ax.text(j, i, f"{v:.2f}", ha="center", va="center",
                    fontsize=7.5, color=color, fontweight="bold")

    cbar = fig.colorbar(im, ax=axes, fraction=0.018, pad=0.015)
    cbar.set_label("达成率", fontsize=8)
    cbar.ax.tick_params(labelsize=7.5)
    save_multi(fig, OUT / "fig5-6-dim-category-heatmap")


# =============================================================================
# 图 5.7 难度分层 × 类别 × 方法 小多图 (小提琴)
# =============================================================================

def fig5_7_difficulty_category_small_multiples():
    df = _runs()
    cats = ["diagnosis", "ops", "submission", "query"]
    diffs = ["easy", "medium", "hard"]
    fig, axes = plt.subplots(len(cats), len(diffs), figsize=(8.4, 8.0),
                              sharey=True)
    for ci, cat in enumerate(cats):
        for di, diff in enumerate(diffs):
            ax = axes[ci, di]
            data = []
            colors = []
            labels = []
            for m in ("mops", "ps", "react", "llm_only"):
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
                                      widths=0.85)
                for body, c in zip(parts["bodies"], colors):
                    body.set_facecolor(c)
                    body.set_edgecolor("white")
                    body.set_alpha(0.75)
                for key in ("cmins", "cmaxes", "cbars"):
                    if key in parts:
                        parts[key].set_visible(False)
                parts["cmedians"].set_color(TEXT)
                parts["cmedians"].set_linewidth(0.9)
                ax.set_xticks(range(len(labels)))
                ax.set_xticklabels(labels, fontsize=6.5, rotation=22)
            else:
                ax.text(0.5, 0.5, "—", transform=ax.transAxes,
                        ha="center", va="center", fontsize=12, color=MUTED)
                ax.set_xticks([])
            ax.set_ylim(20, 105)
            if di == 0:
                ax.set_ylabel(CATEGORY_LABEL_CN[cat])
            if ci == 0:
                ax.set_title(DIFFICULTY_LABEL_CN[diff], fontsize=9.5,
                              color=TEXT, pad=4)
            strip_spines(ax)
    fig.text(0.5, -0.02, "难度", ha="center", fontsize=9, color=TEXT)
    fig.text(-0.01, 0.5, "类别", va="center", rotation=90,
              fontsize=9, color=TEXT)
    fig.tight_layout()
    save_multi(fig, OUT / "fig5-7-difficulty-category")


# =============================================================================
# 图 5.8 工具调用频次 Top-N (按方法分列)
# =============================================================================

def fig5_8_tool_frequency():
    df = _runs()
    # 解析 called_tools
    tool_rows = []
    for _, r in df.iterrows():
        tools = json.loads(r["called_tools"])
        for t in tools:
            tool_rows.append({"method": r["method"], "tool": t})
    tdf = pd.DataFrame(tool_rows)
    pivot = tdf.pivot_table(index="tool", columns="method",
                             values="tool", aggfunc=len, observed=True).fillna(0)
    pivot = pivot.reindex(columns=["mops", "ps", "react"], fill_value=0)
    pivot["__total"] = pivot.sum(axis=1)
    pivot = pivot.sort_values("__total", ascending=True).tail(12)
    pivot = pivot.drop(columns="__total")

    fig, ax = plt.subplots(figsize=(6.4, 4.0))
    y = np.arange(len(pivot))
    height = 0.27
    for i, m in enumerate(("mops", "ps", "react")):
        ax.barh(y + (i - 1) * height, pivot[m].values, height,
                color=method_color(m), edgecolor="white", linewidth=0.5,
                label=method_label(m), alpha=0.95)
    ax.set_yticks(y)
    ax.set_yticklabels([TOOL_LABEL_CN.get(t, t) for t in pivot.index])
    ax.set_xlabel("调用次数")
    strip_spines(ax)
    nice_legend(ax, loc="lower right", ncol=3)
    save_multi(fig, OUT / "fig5-8-tool-frequency")


# =============================================================================
# 图 5.9 LLM 调用数分布 (KDE + 散点)
# =============================================================================

def fig5_9_llm_calls_distribution():
    df = _runs()
    fig, ax = plt.subplots(figsize=(5.6, 3.2))
    positions = []
    for i, m in enumerate(("mops", "ps", "react", "llm_only")):
        vals = df[df["method"] == m]["llm_calls"].values
        positions.append(vals)
    parts = ax.violinplot(positions, positions=range(4), widths=0.78,
                          showmeans=False, showmedians=True, showextrema=False)
    for body, m in zip(parts["bodies"], ("mops", "ps", "react", "llm_only")):
        body.set_facecolor(method_color(m))
        body.set_edgecolor("white")
        body.set_alpha(0.72)
    parts["cmedians"].set_color(TEXT)
    parts["cmedians"].set_linewidth(0.9)
    # 散点叠加
    for i, m in enumerate(("mops", "ps", "react", "llm_only")):
        vals = df[df["method"] == m]["llm_calls"].values
        jitter = np.random.uniform(-0.08, 0.08, size=len(vals))
        ax.scatter(np.full(len(vals), i) + jitter, vals,
                   color=method_color(m), s=8, alpha=0.45,
                   edgecolor="white", linewidth=0.2)
    ax.set_xticks(range(4))
    ax.set_xticklabels([method_label(m) for m in ("mops", "ps", "react", "llm_only")])
    ax.set_ylabel("LLM 调用次数")
    strip_spines(ax)
    save_multi(fig, OUT / "fig5-9-llm-calls")


# =============================================================================
# 图 5.10 角色消融
# =============================================================================

def fig5_10_role_ablation():
    df = pd.read_csv(DATA / "ablation_roles_summary.csv")
    df = df.set_index("config_label").reindex(
        ["完整 MOPS", "w/o Planner", "w/o Verifier", "w/o Coordinator", "w/o Plan + Verifier"]
    )
    metrics = [
        ("avg_overall_score_100", "综合得分", (78, 95)),
        ("avg_tool_selection_f1", "工具选择 F1", (0.83, 1.0)),
        ("avg_estimated_total_tokens", "Token 消耗", (12000, 23000)),
    ]
    palette = ["#1F4E79", "#2B5F8E", "#3F8FA8", "#5FA8D3", "#9CD3DA"]
    fig, axes = plt.subplots(1, 3, figsize=(9.5, 3.2))
    for ax, (col, ylabel, ylim) in zip(axes, metrics):
        bars = ax.bar(range(len(df)), df[col].values,
                      color=palette[:len(df)],
                      edgecolor="white", linewidth=0.5, width=0.65)
        ax.set_xticks(range(len(df)))
        ax.set_xticklabels(df.index, rotation=22, ha="right", fontsize=8.5)
        ax.set_ylabel(ylabel)
        ax.set_ylim(*ylim)
        strip_spines(ax)
        for b, v in zip(bars, df[col].values):
            ax.text(b.get_x() + b.get_width() / 2,
                    v + (ylim[1] - ylim[0]) * 0.012,
                    f"{v:.2f}" if col != "avg_estimated_total_tokens" else f"{int(v):,}",
                    ha="center", va="bottom", fontsize=7.5, color=TEXT)
    fig.tight_layout()
    save_multi(fig, OUT / "fig5-10-role-ablation", formats=("png",))


# =============================================================================
# 图 5.11 模型配置消融 (双坐标轴：质量 vs 相对成本)
# =============================================================================

def fig5_11_model_ablation():
    df = pd.read_csv(DATA / "ablation_models_summary.csv")
    df["short_label"] = df["config"].map({
        "qwen36plus_baseline": "Qwen3.6-Plus\n全量 (基准)",
        "ds_pro_all":          "DSV4-Pro\n全量",
        "ds_heterogeneous":    "DSV4-Flash\n+ DSV4-Pro\n(异构)",
        "ds_flash_all":        "DSV4-Flash\n全量",
    })
    df = df.set_index("short_label").reindex([
        "Qwen3.6-Plus\n全量 (基准)",
        "DSV4-Pro\n全量",
        "DSV4-Flash\n+ DSV4-Pro\n(异构)",
        "DSV4-Flash\n全量",
    ])
    SCORE_BLUE = "#1F4E79"
    COST_BLUE = "#5FA8D3"
    fig, ax = plt.subplots(figsize=(6.2, 3.2))
    x = np.arange(len(df))
    width = 0.35
    bars1 = ax.bar(x - width/2, df["avg_overall_score_100"].values, width,
                   color=SCORE_BLUE, edgecolor="white", linewidth=0.5,
                   label="综合得分")
    ax.set_ylabel("综合得分", color=SCORE_BLUE)
    ax.tick_params(axis="y", labelcolor=SCORE_BLUE)
    ax.set_ylim(75, 98)
    ax.set_xticks(x)
    ax.set_xticklabels(df.index, fontsize=7.5)
    strip_spines(ax)

    ax2 = ax.twinx()
    bars2 = ax2.bar(x + width/2, df["relative_cost"].values * 100, width,
                    color=COST_BLUE, edgecolor="white", linewidth=0.5,
                    label="相对成本 (%)", alpha=0.92)
    ax2.set_ylabel("相对 Token 成本 (%)", color=COST_BLUE)
    ax2.tick_params(axis="y", labelcolor=COST_BLUE)
    ax2.set_ylim(0, 115)
    ax2.spines["top"].set_visible(False)

    for b, v in zip(bars1, df["avg_overall_score_100"].values):
        ax.text(b.get_x() + b.get_width() / 2, v + 0.3,
                f"{v:.2f}", ha="center", va="bottom", fontsize=7.5, color=SCORE_BLUE)
    for b, v in zip(bars2, df["relative_cost"].values):
        ax2.text(b.get_x() + b.get_width() / 2, v * 100 + 2,
                 f"{v*100:.0f}%", ha="center", va="bottom",
                 fontsize=7.5, color=COST_BLUE)

    lines = [mpatches.Patch(color=SCORE_BLUE, label="综合得分"),
             mpatches.Patch(color=COST_BLUE, label="相对 Token 成本")]
    ax.legend(handles=lines, loc="upper left", frameon=True,
              framealpha=0.85, facecolor="#FFFFFF", edgecolor="none",
              fontsize=8.5)
    save_multi(fig, OUT / "fig5-11-model-ablation", formats=("png",))


# =============================================================================
# 图 5.12 线上抽检 5 维 (boxplot + 高/低分率)
# =============================================================================

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

    fig, axes = plt.subplots(1, 2, figsize=(8.4, 3.4),
                              gridspec_kw=dict(width_ratios=[1.3, 1]))

    # 左：boxplot (5 维分布)
    ax = axes[0]
    data = [sess[d].values for d in dims]
    bp = ax.boxplot(
        data, tick_labels=[label_map[d] for d in dims],
        patch_artist=True, widths=0.55,
        medianprops=dict(color=TEXT, linewidth=1.0),
        flierprops=dict(marker="o", markersize=2.5, markerfacecolor=MUTED,
                         markeredgecolor="none", alpha=0.6),
        whiskerprops=dict(color="#374151", linewidth=0.7),
        capprops=dict(color="#374151", linewidth=0.7),
    )
    colors_seq = ["#1F4E79", "#3F8FA8", "#5FA8D3", "#52B788", "#9CD3DA"]
    for patch, c in zip(bp["boxes"], colors_seq):
        patch.set_facecolor(c)
        patch.set_edgecolor(c)
        patch.set_alpha(0.78)
    ax.set_ylim(1, 5.2)
    ax.set_ylabel("评分（1–5 分）")
    plt.setp(ax.get_xticklabels(), rotation=18, ha="right")
    strip_spines(ax)

    # 右：高分率 / 低分率 双向条形
    ax2 = axes[1]
    summary_dims = summary.set_index("dim_key").reindex(dims)
    high = summary_dims["high_rate"].values * 100
    low = summary_dims["low_rate"].values * 100
    y = np.arange(len(dims))
    ax2.barh(y, high, color=OK_GREEN, height=0.55, alpha=0.92, label="≥ 4 分占比")
    ax2.barh(y, -low, color=ACCENT, height=0.55, alpha=0.92, label="< 3 分占比")
    ax2.set_yticks(y)
    ax2.set_yticklabels([label_map[d] for d in dims], fontsize=8.5)
    ax2.invert_yaxis()
    ax2.set_xticks([-20, -10, 0, 25, 50, 75, 100])
    ax2.set_xticklabels(["20%", "10%", "0%", "25%", "50%", "75%", "100%"], fontsize=7.5)
    ax2.set_xlim(-25, 105)
    ax2.axvline(0, color="#374151", linewidth=0.6)
    for v, yi in zip(high, y):
        ax2.text(v + 1, yi, f"{v:.0f}%", va="center", fontsize=7.5, color=TEXT)
    for v, yi in zip(low, y):
        ax2.text(-v - 1, yi, f"{v:.0f}%", va="center", ha="right",
                 fontsize=7.5, color=TEXT)
    strip_spines(ax2, keep=("bottom",))
    nice_legend(ax2, loc="lower right", ncol=2)

    fig.text(0.99, 0.02, f"样本 n = {len(sess)}",
              ha="right", color=MUTED, fontsize=7.5)
    fig.tight_layout()
    save_multi(fig, OUT / "fig5-12-online-quality")


# =============================================================================
# 图 5.13 显著性检验热力 (paired t-test，方法 × 方法)
# =============================================================================

def fig5_13_significance():
    df = _runs()
    methods = ["mops", "ps", "react", "llm_only"]
    n = len(methods)
    pmat = np.full((n, n), np.nan)
    diff_mat = np.full((n, n), np.nan)
    for i, a in enumerate(methods):
        for j, b in enumerate(methods):
            if i == j:
                continue
            x = df[df["method"] == a]["overall_score_100"].values
            y = df[df["method"] == b]["overall_score_100"].values
            # 同场景配对
            t = stats.ttest_rel(x, y, alternative="greater")
            pmat[i, j] = t.pvalue
            diff_mat[i, j] = float(np.mean(x - y))

    fig, ax = plt.subplots(figsize=(4.6, 3.6))
    cmap = divergent_cmap()
    masked = np.where(np.isnan(diff_mat), 0, diff_mat)
    vmax = max(1.0, float(np.nanmax(np.abs(diff_mat))))
    im = ax.imshow(masked, cmap=cmap, vmin=-vmax, vmax=vmax)
    ax.set_xticks(range(n))
    ax.set_xticklabels([method_label(m) for m in methods])
    ax.set_yticks(range(n))
    ax.set_yticklabels([method_label(m) for m in methods])
    ax.set_xlabel("被对比方法（B）")
    ax.set_ylabel("主方法（A）")
    ax.grid(False)
    ax.set_xticks(np.arange(n + 1) - 0.5, minor=True)
    ax.set_yticks(np.arange(n + 1) - 0.5, minor=True)
    ax.tick_params(which="minor", length=0)
    ax.grid(which="minor", color="white", linewidth=1.0)

    for i in range(n):
        for j in range(n):
            if np.isnan(diff_mat[i, j]):
                ax.text(j, i, "—", ha="center", va="center",
                        color=MUTED, fontsize=10)
                continue
            sig = ""
            p = pmat[i, j]
            if p < 0.001: sig = "***"
            elif p < 0.01: sig = "**"
            elif p < 0.05: sig = "*"
            color = "white" if abs(diff_mat[i, j]) > vmax * 0.45 else TEXT
            ax.text(j, i, f"Δ={diff_mat[i, j]:+.1f}\n{sig}",
                    ha="center", va="center", fontsize=8, color=color)

    cbar = fig.colorbar(im, ax=ax, fraction=0.04, pad=0.02)
    cbar.set_label("A − B 平均得分差", fontsize=8)
    cbar.ax.tick_params(labelsize=7.5)
    ax.text(1.0, -0.12, "* p<0.05  ** p<0.01  *** p<0.001 (配对单尾 t 检验)",
            transform=ax.transAxes, ha="right", fontsize=7, color=MUTED)
    save_multi(fig, OUT / "fig5-13-significance")


# =============================================================================
# 图 5.14 综合 ranking 平行坐标 (4 方法)
# =============================================================================

def fig5_14_parallel():
    df = _summary().set_index("method")
    dims = [
        ("normalized_weighted_score_100", "综合得分", 50, 100),
        ("avg_tool_selection_f1",         "工具 F1", 0.05, 1.0),
        ("root_cause_hit_rate",           "根因命中", 0.0, 1.0),
        ("suggestion_relevance_rate",     "建议相关", 0.0, 1.0),
        ("permission_compliance_rate",    "权限合规", 0.5, 1.0),
        ("completion_signal_rate",        "完成信号", 0.0, 1.0),
    ]
    fig, ax = plt.subplots(figsize=(7.2, 3.4))
    x = np.arange(len(dims))
    for method in ("mops", "ps", "react", "llm_only"):
        vals = []
        for key, _, lo, hi in dims:
            v = df.loc[method, key]
            vals.append((v - lo) / (hi - lo))
        ax.plot(x, vals, marker="o", linewidth=1.6, markersize=5,
                color=method_color(method), label=method_label(method, full=True))
    ax.set_xticks(x)
    ax.set_xticklabels([d[1] for d in dims])
    ax.set_ylim(-0.05, 1.1)
    ax.set_ylabel("各维度归一化值")
    for xi in x:
        ax.axvline(xi, color=GRID, linewidth=0.5, zorder=0)
    strip_spines(ax)
    nice_legend(ax, loc="lower left", ncol=4)
    save_multi(fig, OUT / "fig5-14-parallel")


# =============================================================================
# main
# =============================================================================

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
    print(f"ch5 v2: {len(funcs)} figures done.")


if __name__ == "__main__":
    main()
