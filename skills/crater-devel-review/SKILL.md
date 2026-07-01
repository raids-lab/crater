---
name: crater-devel-review
version: 0.1.0
description: "Crater 全仓库代码审查与 PR 描述：纵览 backend/frontend/cli/charts/docs/website 的变更，按核心规范/优化建议分级反馈，并生成双语 PR 描述。用户要求审查 PR、diff、变更或撰写 PR 描述时使用；开始前须应用 crater-devel-shared。"
---

# Crater 开发 · Review

**开始前先应用 [`crater-devel-shared`](../crater-devel-shared/SKILL.md)。** 本 Skill 用于帮助开发者在提交或 PR 前纵览 monorepo 变更、发现问题并撰写 PR 描述。开发规范的完整权威来源是各级 `CONTRIBUTING`；`.github/instructions/*` 是引用 CONTRIBUTING 的审查 MUST/SHOULD 清单。GitHub Copilot code review 阶段的行内 / 总览评论格式由 `.github/copilot-instructions.md` 负责，本 Skill 不强制遵循那套输出形态。

## 分级与语气

- 评价、总结、统计与描述一律用**简体中文**。
- 每条问题标注等级：
  - **【核心规范】**（MUST）：违反视为阻断性问题。
  - **【优化建议】**（SHOULD）：参考改进方向。
- 作为资深导师：礼貌、专业、有建设性；指出问题同时肯定亮点。若某规范在当前场景反而损害质量，明确指出并建议修订对应指令文档。

## 按目录加载细则

审查命中的目录时，读取对应权威文档并据此判断，不要在评论里另立一套约定：

| 变更目录 | 审查依据（规范权威 + 审查清单） |
|----------|----------------------------------|
| `backend/**` | `backend/CONTRIBUTING.md` + `.github/instructions/codebase.instructions.md` |
| `frontend/**` | `frontend/CONTRIBUTING.md` + `.github/instructions/codebase.instructions.md` |
| `cli/**` | `cli/CONTRIBUTING.md`；**先读 `cli/docs/REVIEW.md`**（CLI 审查入口），契约见 `COMMANDS/SPEC/ARCHITECTURE.md` |
| `charts/**` | `charts/CONTRIBUTING.md` + `.github/instructions/charts.instructions.md` |
| `docs/**`、`website/**` | `website/CONTRIBUTING.md` + `.github/instructions/docs.instructions.md` |

## 核心规范检查要点（跨目录速查）

- **安全**：无硬编码密钥/Token/密码/内网 IP；DAO 与存储层无 SQL 字符串拼接。
- **后端**：管理员接口入 `Admin` 路由、用户接口入 `Protected` 路由；外部 API 变更同步 `swag`；HTTP 状态码符合 RESTful；错误信息为清晰英文。
- **作业模板**：作业配置字段、模板序列化或克隆作业 payload 变更须判断是否应阻断旧模板 / 旧导出配置；需要阻断时提升对应前端 `MetadataForm*` version，旧配置仍需可用时补充兼容处理与验证。
- **前端**：无硬编码文本（接入 i18n）；翻译 key 使用英文语义 key 并放入合适 domain，不得新增中文 key；新增 / 修改可翻译文案须同步所有语言 `translation.json`，保证翻译准确和专有名词一致；身份判断用 `useIsAdmin()`；非幂等操作执行前须有确认弹出框，说明对象和后果；改动 `ui-custom/` 等高复用组件须评估影响范围与兼容性（审查时列为核心规范，要求开发者确认）；不易理解的输入 / 配置项须提供帮助图标与 hover tooltip；前端 / UI 改动的 PR 描述须包含相应界面截图。
- **Chart**：`values.yaml` / 模板 / 依赖 / 配置项行为或默认值变更须提升 `charts/crater/Chart.yaml` 的 `version` 与 `appVersion`，二者必须保持完全相同的值，并同步 `charts/crater/README.md`；values-only 可 patch，前后端 API 变化影响应用契约须 minor，不主动 major；版本发布变化须提醒创建 GitHub tag；新增项含英文注释。
- **文档**：`website/` 应面向平台用户（集群用户与集群管理员）承载部署、使用、管理、排障等产品文档；`docs/` 和仓库各级 `.md` 默认面向开发者 / 贡献者。归档须正确；术语准确（Account=调度队列）；`website/` 部署命令无硬编码 Chart 版本，命令附近用 `<CraterChartVersionNotice />`，代码块内用 `<chart-version>`，Chart 配置页用 `<ChartBadge />`。
- **验证记录**：后端 / CLI 构建或测试应先确认本地 `go version` 符合对应 `go.mod` / CONTRIBUTING；后端 `make run` 依赖真实配置和网络环境，缺少配置、凭据或集群访问时应要求开发者检查，不要求 Agent 反复尝试。
- **人工检查**：最终提交 / 推送 / PR 描述前必须有开发者亲自执行的人工检查结果；文档改动必须由开发者人工阅读检查，不能只依赖 AI 生成或 AI 自检。
- **分支与 PR 准备**：提交 / 推送 / PR 前检查当前分支、目标 base、领先 / 落后提交数和修改文件清单；任务分支落后目标 base 时，先提示需要 rebase 或取得维护者明确接受。
- **联动**：功能上线同步 `website/` 文档；配置结构变更同步 `charts/`。

## 审查输出

- 面向开发者自查输出问题清单即可，不要求拆成 GitHub Copilot 的“行内评论”和“总览评论”。
- 优先列出会阻断提交 / PR 的 **【核心规范】** 问题，再列 **【优化建议】**；能定位到文件和行号时给出位置，不能定位时说明影响范围。
- 同类重复问题合并说明，指出代表位置和需要统一检查的范围；复杂问题给修复方向，不堆砌大段代码。
- 若没有发现问题，明确说明未发现阻断项，并列出仍建议开发者人工确认的风险点或测试缺口。
- 给修改建议代码时提醒：直接套用 AI 建议可能导致 Workflow 失败，推荐本地参考修改并测试。
- 审查力度保持克制：后端优化建议仅在明显架构违规、重大性能隐患或高价值最佳实践时提出；`website/` 空格、基础校对等琐碎问题交由 Workflow，不必评论。

## 人工检查建议

审查结束时，结合 diff 告诉开发者还需要亲自检查什么，而不是只说“请测试”：

- UI / 前端改动：列出应打开的页面、角色、入口、关键操作和需要截图的状态；如果改了复用组件、hook、表单控件或 `ui-custom/`，指出可能被牵连的其它页面或流程，要求开发者抽查代表性页面。
- 后端 / CLI 改动：列出建议运行的命令、API 场景、输入输出、权限身份或错误分支。
- 文档改动：列出需要人工阅读的文档、语言版本、链接、术语、示例命令、Chart 版本占位或关键步骤；提醒开发者 AI 生成文档不能直接提交，必须人工阅读判断。
- Chart / 配置改动：列出需要核对的 values、README、版本号、tag、模板渲染或敏感配置项。
- 提交前检查 `git status` / diff，确认没有把用于记录任务、测试、issue 编号、临时方案或开发过程的 task note / 临时文件纳入提交；除非维护者明确要求，这类文件不应提交。
- PR 前要求开发者查看最终文件清单、关键 diff 和 PR 描述；若审查发现需要修改的问题，先迭代处理，再生成或更新 PR 描述。

## PR 描述生成

根 `CONTRIBUTING.md`「Open And Iterate On A Pull Request / 创建并迭代 Pull Request」定义 PR 描述必须包含的信息；**具体生成流程、模板、AI / 开发者测试分组与拒绝条件以 `crater-devel-shared` 的「PR 描述」为准**。

生成 PR 描述前，必须询问开发者亲自做过哪些人工检查，并要求其提供具体页面操作、命令场景或文档阅读检查。若用户尚未说明关联 issue，必须询问是否有关联 issue；有则按 GitHub 可识别格式写入（如 `Resolve #123`），没有则省略。文档改动必须要求开发者说明已阅读检查的文档、语言版本、链接、术语、示例命令、版本占位或关键步骤，并确认是否需要调整。Agent 可以加入自己实际执行过的验证，但必须标注为 AI 执行；如果开发者不提供亲自检查结果，或文档改动没有开发者阅读检查结果，拒绝创建 PR 描述。

若需要创建或更新 PR，必须先把最终 PR 描述展示给开发者并取得确认；不能把开发者没看过的 AI 生成文本直接用于 PR。

## PR 后续迭代

创建 PR 后提醒开发者检查 workflow 状态；如工具可用，可帮助读取 workflow 状态和 PR 链接。若 workflow 失败，先读取失败 job / log，判断根因与是否和本 PR 相关，再给开发者修复方案。处理 Copilot review 或人工 review 时，先读取意见并判断其是否正确、是否值得修改，再给开发者修改方案。未经开发者讨论和确认，不要直接按 workflow / review 意见修改代码。
