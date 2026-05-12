# 面向智算平台的运维智能体设计与实现

> 本科毕业设计论文（2026 年 5 月 11 日修订版）。本目录汇总论文全部章节、配图源文件与说明。

## 故事主线（一句话总结）

**智算平台用户在面对作业失败、配额不足、资源选择等问题时缺少的不是原始数据，而是"对多源证据进行面向用户解释的交互中介"**。本文设计并实现了多智能体运维系统 **Mops**，用「协调器状态机 + 五角色 + 共享工件 + 工具声明执行分离 + 安全四闸门」五项机制把"问题理解-证据检索-结论生成-人工确认-执行与审计"串成统一闭环，并以离线 85 场景 + Crater 平台真实部署双重验证了其有效性。

## 论文章节索引

| 文件 | 章节 | 估算字数 | 核心图表 |
|------|------|---------|----------|
| [abstract.md](abstract.md) | 摘要（中英文） | 1 200 | — |
| [chapter1-introduction.md](chapter1-introduction.md) | 第 1 章 绪论 | 3 200 | 图 1-2 研究现状综述 |
| [chapter2-background.md](chapter2-background.md) | 第 2 章 相关概念与技术 | 3 800 | 图 2-1 Crater 平台架构 · 图 2-3 LLM 范式对比 · 图 2-4 多智能体范式对比 |
| [chapter3-design.md](chapter3-design.md) | 第 3 章 多智能体协作框架设计 | 7 000 | 图 3-1 总体架构 · 图 3-2 角色协作 · 图 3-3 MASState · 图 3-4 状态机 · 图 3-5 上下文 · 图 3-6 工具体系 · 图 3-7 安全闸门 |
| [chapter4-implementation.md](chapter4-implementation.md) | 第 4 章 系统实现 | 7 200 | 图 4-1 三服务部署 · 图 4-3 ER 模型 · 图 4-4 SSE 时序 |
| [chapter5-experiments.md](chapter5-experiments.md) | 第 5 章 实验与评估 | 5 500 | 图 5-1 三方法对比 · 图 5-2 数据集分布 · 图 5-4 双案例时序 |
| [conclusion.md](conclusion.md) | 结论与展望 | 700 | — |
| [references.md](references.md) | 参考文献 (40 条) | — | — |

正文中文合计约 **2.8 万字**，超过本科毕业论文 2.5 万字目标。

## 配图清单（共 16 张，全部中文 · drawio 源文件）

### 第 1 章
| 图编号 | 名称 | 源文件 |
|--------|------|--------|
| 图 1-2 | 智能运维研究演进与本文工作定位 | [figures/fig1-2-research-landscape.drawio](figures/fig1-2-research-landscape.drawio) |

### 第 2 章
| 图编号 | 名称 | 源文件 |
|--------|------|--------|
| 图 2-1 | Crater 智算平台总体架构（本文基础系统） | [figures/fig2-1-crater-platform.drawio](figures/fig2-1-crater-platform.drawio) |
| 图 2-3 | LLM 推理范式对比：CoT · ReAct · PS · Mops | [figures/fig2-3-llm-paradigms.drawio](figures/fig2-3-llm-paradigms.drawio) |
| 图 2-4 | 多智能体协作范式对比与本文选型 | [figures/fig2-4-mas-patterns.drawio](figures/fig2-4-mas-patterns.drawio) |

### 第 3 章（核心设计）
| 图编号 | 名称 | 源文件 |
|--------|------|--------|
| 图 3-1 | Mops 总体分层架构（用户·业务·Runtime·平台四层） | [figures/fig3-1-mops-architecture.drawio](figures/fig3-1-mops-architecture.drawio) |
| 图 3-2 | 多智能体角色分工与含 replan / reassign / HITL 的协作流 | [figures/fig3-2-mas-roles.drawio](figures/fig3-2-mas-roles.drawio) |
| 图 3-3 | MASState 共享状态对象与工件流转 | [figures/fig3-3-masstate.drawio](figures/fig3-3-masstate.drawio) |
| 图 3-4 | Coordinator 状态机：replan / reassign / retry / HITL 完整转移图 | [figures/fig3-4-state-machine.drawio](figures/fig3-4-state-machine.drawio) |
| 图 3-5 | 三类上下文（用户/页面/会话）的获取、维护与注入路径 | [figures/fig3-5-context-layers.drawio](figures/fig3-5-context-layers.drawio) |
| 图 3-6 | 工具体系总览：五元组 · 声明-执行分离 · 分类 · 权限矩阵 | [figures/fig3-6-tools-overview.drawio](figures/fig3-6-tools-overview.drawio) |
| 图 3-7 | 安全控制四闸门：权限过滤 → 操作确认 → 续接 → 审计 | [figures/fig3-7-security-gates.drawio](figures/fig3-7-security-gates.drawio) |

### 第 4 章（系统实现）
| 图编号 | 名称 | 源文件 |
|--------|------|--------|
| 图 4-1 | Mops 三服务部署架构（Frontend · Backend · Runtime） | [figures/fig4-1-system-architecture.drawio](figures/fig4-1-system-architecture.drawio) |
| 图 4-3 | 数据持久化 ER 模型（PostgreSQL · 6 张新表 + 关联） | [figures/fig4-3-er-diagram.drawio](figures/fig4-3-er-diagram.drawio) |
| 图 4-4 | SSE 流式输出与高风险操作 Pause-Confirm-Resume 时序 | [figures/fig4-4-sse-confirmation.drawio](figures/fig4-4-sse-confirmation.drawio) |

### 第 5 章（实验与评估）
| 图编号 | 名称 | 源文件 |
|--------|------|--------|
| 图 5-1 | Mops · PS · ReAct 三方法整体效果对比（柱状 + 表格） | [figures/fig5-1-results-overview.drawio](figures/fig5-1-results-overview.drawio) |
| 图 5-2 | CraterOps-85 数据集分布（任务 × 难度热图） | [figures/fig5-2-dataset-distribution.drawio](figures/fig5-2-dataset-distribution.drawio) |
| 图 5-4 | 典型案例时序：OOM 诊断（单轮）vs. 批量关停（多轮+确认） | [figures/fig5-4-case-studies.drawio](figures/fig5-4-case-studies.drawio) |

## LaTeX 伪代码（穿插于各章节）

为使关键机制可形式化呈现，各章节穿插 9 段 LaTeX 算法伪代码：

| 算法编号 | 名称 | 所在章节 |
|----------|------|----------|
| Algorithm 3.1 | MAS 主循环（Coordinator 状态机驱动） | 3.3.2 |
| Algorithm 3.2 | Verifier 三类裁决与回退分派 | 3.3.2 |
| Algorithm 3.3 | StateView 角色视图过滤 | 3.4.2 |
| Algorithm 3.4 | 工具权限过滤（零信任三原则） | 3.5.2 |
| Algorithm 3.5 | Pause-Confirm-Resume 续接 | 3.6 |
| Algorithm 4.1 | SSE 事件流式路由 | 4.2.2 |
| Algorithm 4.2 | GoBackendToolExecutor 执行与审计 | 4.3.3 |
| Algorithm 4.3 | 评分计算（六维加权 + 难度归一化） | 4.4.2 |
| Algorithm 4.4 | 巡检定时任务编排 | 4.3.4 |

## 数据来源声明

- **代码实现**：`/Users/moon/GolandProjects/crater/crater-agent/` 主分支 `feature/crater-agent`
- **平台说明**：《Crater 平台系统能力与功能模块说明》（51 页 PDF）
- **运行数据（主实验）**：`results/exp30-qwen3-coder-plus-newkey-prod-{mops,ps,react}-20260505-v1/`
- **运行数据（跨模型）**：`results/exp30-qwen35flash0223-oldkey-20260503-v1/`
- **消融数据**：`results/20260429-164642-mas-internal-ablation-glm51/`
- **前期成果**：本科毕设中期检查报告 · AgentOps4Crater 工作沟通纪要 · `docs/thesis/`

## 创新点回顾（C1-C4）

- **C1 智算运维任务建模与三层上下文工程**：把运维任务抽象为 $(actor, page, capabilities)$ 三元组；工具形式化为 $(name, schema, executor, permission, confirm)$ 五元组；上下文按用户/页面/会话分层注入；定义诊断/运维/查询/提交四类任务与三级风险。

- **C2 工件共享 + 状态机的多智能体协作（含 replan/reassign）**：协调器/规划器/探索器/执行器/验证器五角色，通过 MASState 共享工件；Verifier 触发三类回退（补充探索/replan/retry），Coordinator 触发重新指派；硬上界 + 无进展守护 + 工具签名去重三道终止保险。

- **C3 工具声明-执行分离的跨平台体系**：88 个工具的声明层平台无关，执行层通过 GoBackend / Local / Mock 三个适配器实例化；零信任原则确保 LLM 输出无法绕过权限边界。

- **C4 安全控制四闸门与 HITL 续接**：权限过滤 → 操作确认 → workflow checkpoint 续接 → 审计落库；Runtime 在暂停期间完全无状态，支持水平扩展与失败重连。
