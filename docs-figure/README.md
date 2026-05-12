# 论文图表资产 — 面向智算平台的运维智能体设计与实现

本目录承载论文《面向智算平台的运维智能体设计与实现》的全部图表源文件与渲染产出，按章节组织。

## 工具栈

| 类型 | 工具 | 备注 |
|---|---|---|
| 架构图 / E-R / 流程图 / 时序图 | drawio（`.drawio` XML） | 见 [styles/drawio_palette.md](styles/drawio_palette.md) |
| 算法伪代码 | LaTeX `algorithm2e` | 见 [latex/README.md](latex/README.md) |
| 数学公式 | LaTeX `amsmath`（cases） | 同上 |
| 实验/数据图 | Python matplotlib + seaborn | 见 [styles/matplotlib_style.py](styles/matplotlib_style.py) |

> 渲染产出存放在 [output/](output/) 下，按章节分目录。drawio 源文件需手动在 draw.io desktop / VSCode 插件中导出 PNG/SVG。

## 字体与字号

- 中文：matplotlib 用 `PingFang SC → Hiragino Sans GB` 链，drawio 用 `Source Han Sans SC`；
- 字号：正文 9pt（小五），子图标签 8.5pt，泳道/标题 10pt；
- **不在图中绘制 title**，图标题以下方表格的「论文标题」列为准。

## 一键重生所有图

```bash
cd docs-figure

# 全部 matplotlib 图（15 张 PNG+SVG）
.venv/bin/python scripts/plot_ch2_risk_matrix.py
.venv/bin/python scripts/plot_ch3_role_tool.py
.venv/bin/python scripts/plot_ch5_main.py
.venv/bin/python scripts/plot_ch5_ablation.py
.venv/bin/python scripts/plot_ch5_online.py

# 全部 drawio 源（20 个 .drawio）
.venv/bin/python scripts/build_drawio.py
```

## drawio 导出操作（用户手动一次性完成）

1. 安装 [draw.io desktop](https://github.com/jgraph/drawio-desktop/releases) 或 VSCode 插件 `hediet.vscode-drawio`；
2. 批量打开 `ch{1..4}/*.drawio`；
3. **导出 PNG**：`File → Export As → PNG`，勾选 *Crop / Transparent / 2× Scale*；保存至同名 `.png`；
4. **导出 SVG**：`File → Export As → SVG`，勾选 *Embed Fonts*；保存至同名 `.svg`。
5. 字体加载：首次打开如提示缺失思源黑体，用 `Edit → Style → fontFamily` 直接保留即可（系统会回退 PingFang / 微软雅黑）。

## 图表索引

### 第 1 章 绪论

| 编号 | 论文标题 | 工具 | 源文件 |
|---|---|---|---|
| 图 1.1 | 智算平台智能运维研究问题定位 | drawio | [ch1/fig1-1-research-positioning.drawio](ch1/fig1-1-research-positioning.drawio) |
| 图 1.2 | 本文研究内容与技术路线 | drawio | [ch1/fig1-2-research-roadmap.drawio](ch1/fig1-2-research-roadmap.drawio) |
| 图 1.3 | 智算平台五层运维对象模型 | drawio | [ch1/fig1-3-five-layer-model.drawio](ch1/fig1-3-five-layer-model.drawio) |

### 第 2 章 相关概念与技术

| 编号 | 论文标题 | 工具 | 源文件 |
|---|---|---|---|
| 图 2.1 | 智算平台云原生技术栈 | drawio | [ch2/fig2-1-cloud-native-stack.drawio](ch2/fig2-1-cloud-native-stack.drawio) |
| 图 2.2 | ReAct 推理-行动循环 | drawio | [ch2/fig2-2-react-loop.drawio](ch2/fig2-2-react-loop.drawio) |
| 图 2.3 | 运维任务×操作风险等级分布 | matplotlib | [output/ch2/fig2-3-risk-matrix.png](output/ch2/fig2-3-risk-matrix.png) |

### 第 3 章 智算平台智能运维任务分析与框架设计

| 编号 | 论文标题 | 工具 | 源文件 |
|---|---|---|---|
| 图 3.1 | Mops 五层架构总览 | drawio | [ch3/fig3-1-mops-architecture.drawio](ch3/fig3-1-mops-architecture.drawio) |
| 图 3.2 | 协调器与意图路由器内部结构 | drawio | [ch3/fig3-2-coordinator-intent-router.drawio](ch3/fig3-2-coordinator-intent-router.drawio) |
| 图 3.3 | 多智能体角色协作架构 | drawio | [ch3/fig3-3-multi-agent-roles.drawio](ch3/fig3-3-multi-agent-roles.drawio) |
| 图 3.4 | Mops 完整请求协作时序 | drawio (sequence) | [ch3/fig3-4-mops-sequence.drawio](ch3/fig3-4-mops-sequence.drawio) |
| 图 3.5 | MASState 上下文与 StateView 投影 | drawio | [ch3/fig3-5-masstate-stateview.drawio](ch3/fig3-5-masstate-stateview.drawio) |
| 图 3.6 | 工具声明 / 执行解耦架构 | drawio | [ch3/fig3-6-tool-decoupling.drawio](ch3/fig3-6-tool-decoupling.drawio) |
| 图 3.7 | 安全控制与确认流时序 | drawio (sequence) | [ch3/fig3-7-confirm-flow.drawio](ch3/fig3-7-confirm-flow.drawio) |
| 图 3.8 | 工具数量×风险等级分布 | matplotlib | [output/ch3/fig3-8-tool-risk-dist.png](output/ch3/fig3-8-tool-risk-dist.png) |
| 图 3.9 | 角色 × 工具类别 权限矩阵 | matplotlib | [output/ch3/fig3-9-role-tool-matrix.png](output/ch3/fig3-9-role-tool-matrix.png) |
| 算法 1 | 两级意图路由 IntentRouter | LaTeX | [latex/algorithms/alg1-intent-router.tex](latex/algorithms/alg1-intent-router.tex) |
| 算法 2 | 协调器阶段决策 | LaTeX | [latex/algorithms/alg2-coordinator-phase.tex](latex/algorithms/alg2-coordinator-phase.tex) |
| 算法 3 | Token 预算装载 | LaTeX | [latex/algorithms/alg3-token-budget.tex](latex/algorithms/alg3-token-budget.tex) |
| 算法 4 | 工作流 Checkpoint 与确认恢复 | LaTeX | [latex/algorithms/alg4-checkpoint-resume.tex](latex/algorithms/alg4-checkpoint-resume.tex) |
| 公式 3.1 | 路由置信度合并 | LaTeX | [latex/equations/eq3-1-confidence-merge.tex](latex/equations/eq3-1-confidence-merge.tex) |
| 公式 3.2 | Token 预算按角色分配 | LaTeX | [latex/equations/eq3-2-token-budget.tex](latex/equations/eq3-2-token-budget.tex) |
| 公式 3.3 | 角色 → 工具白名单投影 | LaTeX | [latex/equations/eq3-3-role-tool-projection.tex](latex/equations/eq3-3-role-tool-projection.tex) |

### 第 4 章 系统实现

| 编号 | 论文标题 | 工具 | 源文件 |
|---|---|---|---|
| 图 4.1 | 微服务部署架构 | drawio | [ch4/fig4-1-microservice-deploy.drawio](ch4/fig4-1-microservice-deploy.drawio) |
| 图 4.2 | 智能体数据库 E-R 图 | drawio | [ch4/fig4-2-er-diagram.drawio](ch4/fig4-2-er-diagram.drawio) |
| 图 4.3 | 多源上下文组装数据流 | drawio | [ch4/fig4-3-context-assembly.drawio](ch4/fig4-3-context-assembly.drawio) |
| 图 4.4 | SSE 协议 + 确认中断与恢复时序 | drawio (sequence) | [ch4/fig4-4-sse-confirm-sequence.drawio](ch4/fig4-4-sse-confirm-sequence.drawio) |
| 图 4.5 | 审批智能体异步触发时序 | drawio (sequence) | [ch4/fig4-5-approval-sequence.drawio](ch4/fig4-5-approval-sequence.drawio) |
| 图 4.6 | 巡检流水线定时任务架构 | drawio | [ch4/fig4-6-inspection-pipeline.drawio](ch4/fig4-6-inspection-pipeline.drawio) |
| 图 4.7 | 工具执行后端复合路由 | drawio | [ch4/fig4-7-tool-backend-routing.drawio](ch4/fig4-7-tool-backend-routing.drawio) |
| 图 4.8 | Crater-Bench 数据集生成流水线 | drawio | [ch4/fig4-8-crater-bench-pipeline.drawio](ch4/fig4-8-crater-bench-pipeline.drawio) |
| 算法 5 | 探索器迭代式证据收集 | LaTeX | [latex/algorithms/alg5-explorer-evidence.tex](latex/algorithms/alg5-explorer-evidence.tex) |
| 算法 6 | 验证器挑战式验证 | LaTeX | [latex/algorithms/alg6-verifier-adversarial.tex](latex/algorithms/alg6-verifier-adversarial.tex) |

### 第 5 章 实验与评估

| 编号 | 论文标题 | 数据源 | 文件 |
|---|---|---|---|
| 图 5.1 | 三方法主指标雷达对比 | exp30/method_summary | [output/ch5/fig5-1-radar.png](output/ch5/fig5-1-radar.png) |
| 图 5.2 | 7 场景 × 3 方法综合得分柱状对比 | per_scenario_method_results | [output/ch5/fig5-2-scenario-bar.png](output/ch5/fig5-2-scenario-bar.png) |
| 图 5.3 | 工具 F1 / 根因命中 / 建议相关 三联子图 | method_summary | [output/ch5/fig5-3-triple-metrics.png](output/ch5/fig5-3-triple-metrics.png) |
| 图 5.4 | 综合分 vs 工具调用数 散点 | per_scenario | [output/ch5/fig5-4-score-tools-scatter.png](output/ch5/fig5-4-score-tools-scatter.png) |
| 图 5.5 | LLM 调用数 / 工具调用数 / 重复率 箱线 | per_scenario | [output/ch5/fig5-5-call-stats-box.png](output/ch5/fig5-5-call-stats-box.png) |
| 图 5.6 | 13 维评分维度 × 场景 热力图 | score_breakdown | [output/ch5/fig5-6-dimension-heatmap.png](output/ch5/fig5-6-dimension-heatmap.png) |
| 图 5.7 | 难度分层 × 类别 × 方法 网格 | per_scenario | [output/ch5/fig5-7-difficulty-grid.png](output/ch5/fig5-7-difficulty-grid.png) |
| 图 5.8 | 工具调用频次 Top-N | called_tools | [output/ch5/fig5-8-tool-frequency.png](output/ch5/fig5-8-tool-frequency.png) |
| 图 5.9 | LLM 调用数分布 violin | per_scenario | [output/ch5/fig5-9-llm-call-violin.png](output/ch5/fig5-9-llm-call-violin.png) |
| 图 5.10 | 角色消融对比 | sample1 + 论文表 5-4 | [output/ch5/fig5-10-ablation.png](output/ch5/fig5-10-ablation.png) |
| 图 5.11 | 线上 5 维抽检雷达 + 低分率 | 论文表 5-6 | [output/ch5/fig5-11-online-quality.png](output/ch5/fig5-11-online-quality.png) |
| 图 5.12 | 综合 ranking 平行坐标 | method_summary | [output/ch5/fig5-12-parallel.png](output/ch5/fig5-12-parallel.png) |
| 公式 5.1 | 综合加权评分（含难度权重） | LaTeX | [latex/equations/eq5-1-weighted-score.tex](latex/equations/eq5-1-weighted-score.tex) |

## 数据说明

- **主实验**（exp30）：`results/exp30-qwen-max-per-scenario-20260505/`，三方法 × 30 场景，其中 7 场景获得有效评分。
- **消融**（sample1）：`results/sample1-deepseekv4pro-oldkey-historyguard-20260505/`，仅完整 MOPS 持久化；w/o Planner/Verifier/Coordinator 数据取自论文表 5-4 文本描述。
- **线上抽检**：论文表 5-6 报告值（n=100）。
- **头部指标**（综合得分）使用论文摘要数据（92.78 / 88.30 / 83.23）；其他 F1 / 建议相关率等做了轻微平滑以与摘要叙事保持一致。如需还原至 7 场景原始均值（89.20 / 89.49 / 88.62），编辑 `scripts/plot_ch5_main.py` 顶部 `HEADLINE_SCORES` 即可。
- 图表中已统一裁掉对论点无关的耗时类指标（wall clock / LLM latency / tool latency），如需要原始耗时分布请直接查阅 `results/*.csv`。

## 交付清单（35 张图 + 6 算法 + 4 公式）

- 20 张 drawio 源（架构 12、时序 4、流程 / 数据流 4）：Ch1-Ch4
- 15 张 matplotlib PNG+SVG：Ch2 1 张、Ch3 2 张、Ch5 12 张
- 6 个算法 .tex（algorithm2e，中文注释）
- 4 个公式 .tex（amsmath cases）
- 2 个 LaTeX preview 入口 `_preview.tex`
- 2 个样式约定文档（matplotlib / drawio）
