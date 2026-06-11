[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

# 为 Crater 文档贡献

本文档覆盖 `website/` 中的产品文档，以及 `docs/`、各模块 `.md` 等仓库内开发者文档。请先阅读仓库根 [CONTRIBUTING](../docs/zh-CN/CONTRIBUTING.md)，了解全仓库流程、PR 要求和文档人工阅读检查规则。

新增、移动或修改用户文档、部署文档、排障文档、贡献者文档或生成文档引用时，使用本文档。

## 选择正确的文档位置

每次文档改动都先判断读者是谁。

- **`website/`** 面向平台用户：集群用户和集群管理员。这里的管理员也属于平台用户。部署、使用、管理、排障和运维指导放在这里。除非为了说明用户可观察行为，否则避免加入代码级实现细节。
- **`docs/` 和模块级 Markdown** 面向需要和代码打交道的开发者 / 贡献者。架构说明、开发流程、实现取舍、维护说明和贡献者设计记录放在这里。
- **README** 保持面向使用者且简洁，不要变成内部开发手册。

如果文档放错位置，应移动到合适位置，而不是复制出多套相互竞争的说明。

## 本地运行文档站

修改 `website/` 时：

- 期望 Node.js v22+、pnpm v10+。

```bash
cd website
pnpm install
make run
```

常用 target：

| 命令 | 用途 |
|------|------|
| `make run` | 启动本地文档站 |
| `make build` | 构建文档站 |
| `make pre-commit-check` | 提交前运行文档检查 |

## 写作规则

- 文档必须与当前代码和 Chart 行为一致。
- 面向目标读者写作。平台用户需要操作步骤和可观察行为；贡献者需要源码路径、架构和维护上下文。
- 优先写清晰流程，而不是泛泛描述。必要时包含前置条件、预期结果、回滚或排障说明。
- 避免泄露密钥、内部 endpoint、私有集群名称或凭据。
- 文档改动必须按根 CONTRIBUTING 要求由开发者人工阅读检查后才能提交或推送。

## 术语

Crater 术语必须保持一致：

- **Account** / **账户** 特指 Crater 中的调度队列 / 计费账户概念，不要与通用登录账号混用。
- 已有明确产品含义的术语，应在 website 文档、UI 文案和贡献者文档之间保持一致。

## Chart 版本占位

`website/` 中涉及 `oci://ghcr.io/raids-lab/crater` 的 Helm 部署 / 安装 / 升级命令时，严禁硬编码 Chart 版本。

- 在需要 Chart 版本的部署命令附近放 `<CraterChartVersionNotice />`。
- bash/yaml 示例中使用 `<chart-version>`。
- 不要写 `--version 1.2.3` 这类字面版本示例。
- Chart 配置详情页使用 `<ChartBadge />`。

`docs/` 和贡献者文档没有 website 组件注入能力。需要写部署命令时，优先使用 `<chart-version>` 这类占位符。

## 提交文档改动前

- 确认文档位置符合目标读者。
- 涉及文档站时，在 `website/` 运行 `make pre-commit-check`。
- 检查链接、示例、术语和 Chart 版本占位。
- 要求开发者人工阅读变更文档，并判断关键背景、步骤和运维细节是否完整。
- 如果前端 / UI 文档涉及用户可见行为，PR 中按需附截图。
