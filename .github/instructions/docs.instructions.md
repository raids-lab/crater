---
applyTo: "**/*.md,docs/**,website/**"
---

# 仓库文档及文档网站开发与审查规范

此文档针对 `docs/`、`website/` 及仓库各级 Markdown 文档变更提供**审查指引**。

**完整文档规范以 [`website/CONTRIBUTING.md`](../../website/CONTRIBUTING.md) 为权威来源**（文档分类、术语、Chart 版本占位、写作规范）；本文档不重复其全部内容，仅固化最重要的拦截项与审查特有补充。

## 核心规范 (Core Requirements)

以下为高优先拦截项（完整规则见 `website/CONTRIBUTING.md`）：

- **分类归档**: `website/` 放面向平台用户的产品文档（集群用户与集群管理员，含部署、使用、管理、排障）；`docs/` 和仓库各级 `.md` 默认放面向开发者 / 贡献者的技术或维护文档。存放位置必须与读者和职能相符。
- **术语准确性**: 严禁混淆核心术语，如 **“账户 (Account)”** 特指调度队列概念，不可与通用账号概念混用。
- **一致性与语言**: 文档描述必须与代码实际实现一致；严禁引起歧义的严重拼写或笔误。
- **人工阅读检查**: 文档改动在提交 / 推送 / PR 描述前必须由开发者人工阅读检查；AI 生成或 AI 自检不能替代开发者基于经验判断补齐关键信息、步骤或运维细节。
- **Chart 版本规范 (仅限 website/)**: 凡涉及 `oci://ghcr.io/raids-lab/crater` 的 Helm 部署 / 安装 / 升级命令**严禁硬编码版本号**；命令附近须放 `<CraterChartVersionNotice />`，代码块内须用 `<chart-version>` 占位符，Chart 配置详情页用 `<ChartBadge />`。审查 `website/` 文件时必须检查硬编码版本号，发现即标注 **【核心规范】**。

## 优化建议 (Optimization Suggestions)

排版质量、表述精炼、信息完备性，以及 `docs/` 开发文档优先用 `<chart-version>` 占位符等建议，详见 `website/CONTRIBUTING.md` 的写作规范；审查时据此提出 SHOULD 级建议。

## 协作说明
- `website/` 目录的微小格式问题（如空格、基础校对）由 Workflow 自动处理，AI 无需在此类琐碎问题上消耗精力。
- **版本规范执行**:
  - 修改 `website/` 文档时，AI 应优先通过 `<CraterChartVersionNotice />` 引导用户关注版本匹配的重要性（Chart 版本必须与镜像版本对应）。
  - 意识到 `website/` 具备自动注入能力，而 `docs/` 仅具备占位能力。
