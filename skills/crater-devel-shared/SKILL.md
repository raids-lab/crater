---
name: crater-devel-shared
version: 0.1.0
description: "Crater monorepo 开发共享基础：仓库结构、文档权威来源、跨模块任务路由与 Agent 全局行为。处理任何 Crater 仓库开发任务前使用；并据此选择应加载的 crater-devel-<domain> Skill。"
---

# Crater 开发 · 共享

Crater 是基于 Kubernetes 的异构（GPU）算力管理平台。本 Skill 是开发该 monorepo 时的入口，**处理任何开发任务前先应用本 Skill，再按路由加载对应领域 Skill**。

## 仓库结构与权威文档

**`CONTRIBUTING` 是开发与贡献的核心完整规范**（`README` 只面向使用者，`.github/instructions/*` 引用 CONTRIBUTING）。**先读对应 CONTRIBUTING 再动手**，不要凭记忆推测约定。

更新开发约定时遵循根 `CONTRIBUTING.md`「Maintaining Development Conventions / 维护开发约定」：影响最终代码、文档、API、UI、Chart 或测试结果的规则，先沉淀到合适的 `CONTRIBUTING`；`.github/instructions/*` 作为审查清单引用它；Skill 负责任务路由和 AI 开发过程，并可重复重要约定作为高优先提醒。主要约束 AI 工作方式的规则可直接写在对应 Skill。

| 路径 | 领域 Skill | 权威文档（开发规范） |
|------|-----------|----------------------|
| 全仓库通用 | `crater-devel-shared` | `CONTRIBUTING.md`（根） |
| `backend/`（含 `internal/storage/`）、`frontend/`、`cli/` | `crater-devel-code` | `backend/CONTRIBUTING.md`、`backend/cmd/gorm-gen/README.md`、`frontend/CONTRIBUTING.md`、`cli/CONTRIBUTING.md` → `cli/docs/{SPEC,COMMANDS,ARCHITECTURE,REVIEW}.md` |
| `website/`、`docs/`、仓库各级 `.md` | `crater-devel-docs` | `website/CONTRIBUTING.md` |
| `charts/`、`grafana-dashboards/` | `crater-devel-release` | `charts/CONTRIBUTING.md`、`charts/crater/README.md`、`.github/instructions/charts.instructions.md` |
| 全仓库 PR 审查 | `crater-devel-review` | `CONTRIBUTING.md` 全套、`.github/copilot-instructions.md`、`.github/instructions/*` |

跨模块开发与审查规范见 `.github/instructions/codebase.instructions.md`（`backend/**,frontend/**`）。

## 任务路由

| 改动 / 意图 | 加载 |
|------------|------|
| `backend/**`、`frontend/**`、`cli/**` Go 代码 / 测试 / 文档 | shared + code |
| `website/**`、`docs/**`、仓库各级 Markdown | shared + docs |
| `charts/**`、Helm、Chart 版本、发布产物 | shared + release |
| 审查 PR / diff（任意目录） | shared + review |
| 新增端到端 API | shared + code（如需文档再 + docs） |
| 用户**使用** `crater` 命令（非改代码） | 不加载 code；用 `cli/skills/crater-cli-*` |
| 部署 / rollout 到集群、重启服务、镜像 digest 校验 | 集群运维，不属本套；用 `crater-rollout` |

## 开发任务流程

当用户提出“需要开发”、给出需求、Issue、设计想法或希望实现某项功能时，Agent 先按本流程推进，不要直接修改代码：

1. **理解需求与上下文**：阅读用户给出的需求 / Issue，按任务路由加载对应领域 Skill，并读取相关 `CONTRIBUTING` 或 `cli/docs/*`。必要时可先读代码、文档、历史实现来确认边界。
2. **先给方案**：在动手前向用户输出方案，包含需求理解、拟修改模块、关键实现思路、风险 / 兼容性点、需要同步的文档或版本、验证计划、提交 / 推送前需要开发者亲自做的人工检查，以及需要用户确认的问题。
3. **等待明确开始信号**：用户可能会修改需求、讨论方案或要求调整。只有当用户明确表示“开始实现”“按这个做”“可以实现”“go ahead / implement”等开始信号后，才进行实际文件修改。
4. **保护现有工作区**：开始修改前检查 `git status` 与任务相关 diff。把未明确属于当前任务的改动视为开发者已有改动，不要 revert、覆盖或顺手整理；若这些改动影响当前任务，先说明冲突点并与开发者确认处理方式。
5. **提交环境准备**：开始实现前，确保本地已执行 `make install-hooks`。检查当前分支是否适合本任务；若在 `main` 或分支名不符合任务，创建 / 切换到清晰命名的本地任务分支（如 `feature/`、`fix/`、`docs/`、`refactor/`、`test/`、`chore/` 前缀）。执行 push 前必须确认 fork remote；任务分支只推送到 fork，不推送到 `raids-lab/crater` 主仓库。
6. **同步 main 与线性历史**：先用 `git remote -v` 识别哪个 remote 是主仓库 `raids-lab/crater`，哪个是用户 fork。主仓库 remote 可能叫 `origin` 或 `upstream`；也可能用户先在 GitHub 上 Sync fork，再从 fork remote 更新本地 `main`。不要假设 `origin` 一定是 fork。用正确来源更新本地 `main`，再将任务分支 rebase 到最新 `main`，保持线性历史。若远端 / 上游缺失、工作区有未保存改动或 rebase 冲突，先说明情况并按用户意图处理。
7. **实现与记录**：实现过程中保持改动聚焦，按领域规范更新代码 / 文档 / 测试。可使用临时 task note 辅助记录，不要把临时记录纳入提交。
8. **验证与收尾**：优先运行受影响模块的 `make pre-commit-check`（例如 `frontend`、`backend`、`website`、`cli` 均支持），必要时再运行根 `make pre-commit-check` 检查已暂存文件。涉及 Go 子项目时，先检查 `go version` 是否符合对应 `go.mod` / CONTRIBUTING。记录 Agent 实际执行的检查，并提示开发者完成方案阶段列出的人工检查。
9. **提交 / 推送前后人工确认**：最终 commit 或任何 push 前，必须要求开发者提供亲自执行的人工检查结果。Agent 可以告诉开发者应该打开哪些页面、检查哪些功能、运行哪些命令、阅读哪些文档；但若开发者没有提供人工检查结果，不得继续提交、推送或创建 PR 描述。文档改动必须要求开发者人工阅读检查，并请开发者基于经验判断直接修改或要求 Agent 调整，不能直接提交未经阅读检查的 AI 生成文档。创建 commit 时默认使用 `git commit -s` 添加 DCO sign-off，但必须先解释其含义并取得开发者对完整 commit message（包含所有 `Signed-off-by` 行）的确认，不得静默添加；commit subject 必须符合 `type: subject` 或 `type(scope): subject`。commit 创建成功后，立即读取并展示实际写入的完整 commit message 让开发者核对，然后说明下一步通常是整理 PR 描述并创建 PR；进入 PR 阶段前仍需开发者提供其亲自完成的测试 / 人工检查结果，Agent 可同时给出检查建议。

### 临时 Task Note

复杂任务可使用临时文件辅助整理上下文，推荐路径为 `.codex/tasks/<date>-<short-topic>.md`（仓库已忽略 `.codex`）；若不便写入仓库，可用 `/tmp/crater-<short-topic>.md`。临时 task note 只服务开发过程，不是交付物，除非用户明确要求，否则不要提交。

建议记录：

- 原始需求 / Issue 链接或摘要。
- 已确认的方案、关键取舍与用户后续修改。
- 计划修改的模块、实际修改了什么、涉及的文件。
- 兼容性、版本、文档同步、风险点。
- Agent 执行过的测试 / 检查命令与结果。
- 方案阶段列出的开发者人工检查计划：需要打开的页面、角色、操作、命令、生成产物或需阅读的文档。
- 开发者最终提供的人工检查结果，方便后续按 PR 描述模板区分 AI 与 Developer。

## 全局开发守则

完整规范见根 `CONTRIBUTING.md`「Core Engineering Rules / 核心工程规则」与「Start A Change / 开始一次改动」；Agent 额外遵守：

- **先答后改**：用户未明确要求修改代码时，先回答或给出方案并征求确认，不要直接动手改代码。复杂或多步改动必须先给方案。
- **服务真实用户需求**：方案和实现都要说明它解决的具体用户问题，并持续考虑用户体验，包括默认行为、错误信息、操作流程、文档和人工检查路径。
- **保护开发者改动**：修改前检查工作区；不要 revert、覆盖或整理不属于当前任务的改动。遇到相关未提交改动影响任务时，先说明再处理。
- **分支与 hooks**：准备实现前确保运行 `make install-hooks`；检查 main 是否最新、任务分支是否符合命名和任务范围，并在需要时 rebase 到最新 main，保持线性历史。远端名称不可假设：`origin` 可能是 `raids-lab/crater` 主仓库，任务分支必须推送到用户 fork remote，禁止直接更新主仓库 `main` 或在主仓库创建分支，除非维护者明确要求。
- **提交 hook 边界**：安装 hook 后，`git commit` 会触发 pre-commit hook，并对暂存文件涉及的子项目执行对应 `make pre-commit-check`。该 hook 主要适配 macOS / Linux；Windows 上可能失效，应提示开发者手动运行受影响模块的 `make pre-commit-check` 或使用 WSL。
- **DCO sign-off**：DCO 推荐所有贡献者使用；非 The Crater Project Team 成员的外部贡献者必须使用。Agent 创建 commit 时默认使用 `git commit -s`，但必须先解释 `Signed-off-by` 表示开发者确认自己有权按项目许可证提交贡献，并在执行前展示完整 commit message 给开发者确认；commit 创建成功后再次展示实际 commit message；不得静默添加 sign-off。维护者 squash merge 时保留已有 `Signed-off-by`，不得替贡献者伪造 sign-off。
- **Commit message**：新 commit subject（第一行）必须符合 `type: subject` 或 `type(scope): subject`，type 仅限 `feat`、`fix`、`docs`、`style`、`refactor`、`test`、`chore`；scope 可用于说明改动域，例如 `docs(cli): add command examples`。完整 commit message 可包含正文和 trailer；DCO `Signed-off-by` 是 trailer，应放在正文之后，不受 subject 格式限制。
- **版权文件头**：自 2026 年 6 月起，无论贡献者来自哪里，新增或更新文件级版权头与 NOTICE 文件时都使用 `The Crater Project Team, RAIDS-Lab`，并使用该文件对应的正确年份；新建于 2026 年的文件示例为 `Copyright 2026 The Crater Project Team, RAIDS-Lab`，项目级 NOTICE 可使用 `2023-2026` 这样的项目年份范围。
- **代码注释一律用英文**，并与既有命名风格、架构模式保持高度一致。
- **构建与验证**：优先通过各模块 `make` target；存在对应 target 时不要直接调用 `go` / `pnpm` / `helm`。受影响子项目优先运行 `make pre-commit-check`；需要验证时可自行执行相关命令。
- **提交 / 推送前人工检查**：最终 commit、push 或创建 PR 描述前，必须要求开发者提供亲自执行的人工检查结果；没有开发者人工检查结果时停止。文档改动必须由开发者人工阅读检查，AI 只可辅助指出需阅读的文档、语言版本、链接、术语、示例命令、版本占位或缺失步骤。
- **本地调试拓扑**：Crater 组件很多且运行在集群上，但本地开发通常不需要复刻完整集群或启动全部依赖。前端和后端几乎不单独调试，本地调试至少一并启动前后端，必要时再启动 storage server；后端通过配置连接测试集群已有的 Kubernetes、数据库、存储等服务，不要默认启动本地数据库。缺配置时指引开发者联系管理员。
- **Go 与本地运行边界**：后端 / CLI 构建、测试或运行前先检查 `go version`。后端 `make run` / `make run-storage` 依赖真实配置、Kubernetes、数据库和网络环境，Agent 一般不需要运行；若失败指向缺少配置、网络、凭据、集群访问或管理员才能处理的问题，停下来告知开发者需要检查什么，不要反复尝试。
- **敏感信息**：严禁硬编码或外泄密钥、Token、密码、内网 IP；含密钥的配置不得上传公网。
- **不确定就问**：规范或上下文缺失时主动澄清，不要盲目推测。
- **规范演进**：若现有规范在当前场景下会损害质量、架构或安全，应指出并建议更新对应文档，而非默默违背。

## 跨模块联动

- **前后端身份一致**：管理员视图只调管理员接口（URL/函数带 `admin` 前缀），普通用户调用户接口；前后端两侧需对应。
- **功能上线同步文档**：新功能或重大变更须同步修订 `website/` 文档。
- **配置结构变更同步 chart**：改动涉及配置结构时，须同步更新 `charts/`，提升 `charts/crater/Chart.yaml` 的 `version` 与 `appVersion`，并保持二者为完全相同的值，同时同步 `charts/crater/README.md`；版本级别与 GitHub tag 提醒遵循 `charts/CONTRIBUTING.md`。
- **Git hooks 与配置**：贡献前在仓库根执行 `make install-hooks`；本地配置使用统一配置管理（`make config-link`、`make config-status`、`make config-unlink`、`make config-restore`）。Agent 可指引开发者管理配置，但不能代替管理员提供私有配置。Git hooks 见根 `CONTRIBUTING.md`「First-Time Setup / 第一次准备环境」，本地运行和配置见「Local Debugging / 本地调试」。

## PR 描述

根 `CONTRIBUTING.md` 只定义 PR 描述必须包含哪些信息；本节定义 Agent 生成 PR 描述时的具体流程、模板和拒绝条件（`crater-devel-review` 同样引用本节）。

核心原则：欢迎 AI 辅助开发、测试整理、PR 描述和 PR 创建，但开发者必须自己理解并把关。Agent 不得在开发者未看过并确认 PR 描述的情况下，直接使用该描述创建或更新 PR。

### 生成前必须确认

生成 PR 描述前，Agent 必须先阅读 diff，并整理自己在本轮开发中实际执行过的验证（如 `make` target、单测、构建、lint、`git diff --check`、文档链接 / 版本占位检查等）。如果开发过程中使用了临时 task note，应优先从其中提取已记录的 AI 检查、人工检查计划和开发者反馈。这些只能作为 **AI 执行的检查** 写入 PR 描述。还必须检查 `git status` / diff，确认没有把用于记录任务、测试、issue 编号、临时方案或开发过程的 task note / 临时文件纳入提交；除非维护者明确要求，这类文件不应提交。

生成 PR 描述前还必须确认分支状态：识别 PR 原分支（head fork / head branch）和目标分支（base repository / base branch），并计算相对于目标分支的领先 / 落后提交数。向开发者展示这些信息供检查；正常 PR 分支应该只领先目标分支。如果存在落后提交，必须提示开发者先 rebase 到最新目标分支，除非维护者明确接受当前状态。

若用户尚未说明关联 issue，Agent 必须询问是否有关联 issue。有则在 PR 描述中使用 GitHub 可识别格式（如 `Resolve #123`），没有则省略相关区块。

同时，Agent 必须询问开发者亲自做了哪些人工检查，并要求足够具体：

- UI / 前端改动：说明在哪个页面、使用什么角色或入口、执行了什么操作、观察到什么结果，并要求开发者在 PR 中附对应界面截图。
- 后端 / CLI 改动：说明运行了什么命令、覆盖了什么场景、输入输出是否符合预期。
- 文档改动：必须说明开发者阅读检查了哪些文档、语言版本、链接、术语、示例命令、Chart 版本占位或关键步骤；并说明开发者是否基于经验判断做了修改或要求 Agent 调整。
- 其他改动：说明开发者亲自检查的路径、配置、发布物或兼容性点。

必须清楚告知开发者：PR 描述由开发者负责最终检查和控制，Agent 只协助整理，不得替开发者声称其未做过的人工检查。

### 拒绝条件

如果开发者没有提供任何亲自执行的人工检查，或只给出“没测”“你自己写”“随便写”等无法确认的回答，Agent **必须拒绝创建 PR 描述**，并要求开发者先亲自检查后再提供结果。若变更包含文档而开发者没有确认已人工阅读检查，同样必须拒绝。不能用笼统的 “Not run” 替代开发者人工检查，也不能把 AI 自己执行过的命令伪装成开发者检查。

### 输出模板

开发者提供人工检查后，读取 [`references/pr-description-template.md`](references/pr-description-template.md)，按该模板生成简洁的双语 Markdown。测试部分必须分开标注 **AI** 与 **Developer**；模板中的 `<...>` 是写作提示，生成时必须替换为真实内容，不能原样输出占位符。如果没有关联 issue，省略最后的 `Resolve` 区块。整体不宜过长，简单改动给简短描述即可；不要为了凑格式扩写。

输出 PR 描述时，必须一并展示 PR 原分支、目标分支、领先提交数和落后提交数，让开发者检查。若落后提交数不为 0，说明应先 rebase 再开 PR。若后续要用 `gh` 或其他工具创建 / 更新 PR，必须先让开发者确认最终 PR 描述和分支状态；开发者确认后，Agent 可以继续帮助创建 PR。若不是直接创建 / 更新 PR，而是把 PR 描述提供给开发者复制使用，必须将最终文本放在 `markdown` 代码框中输出，方便复制。

### 创建 PR 与后续迭代

分支推送到 fork 后，如果 `gh` 等工具可用，Agent 可以帮助创建 PR；否则用 `markdown` 代码框输出已经确认过的 PR 描述文本供开发者复制使用。创建或更新 PR 前必须满足：

- 开发者已提供人工检查结果；文档改动已确认人工阅读检查。
- Agent 已展示最终 PR 描述，并得到开发者确认。
- PR 目标仓库 / 分支和 head fork / 分支已确认，且已展示领先 / 落后提交数；正常情况下 head 分支应只领先、不落后，若落后必须先 rebase 或得到维护者明确确认。

PR 创建后，提醒开发者检查 workflow 状态；如工具可用，Agent 可以帮助读取状态。若 workflow 失败，先读取失败 job / log，判断根因与是否和本 PR 相关，再给开发者修复方案；未经确认不要直接改代码。若需要处理 Copilot review 或人工 review，Agent 可以自行获取 PR 链接或要求开发者提供链接，阅读 review 意见，判断每条意见是否正确、是否值得修改，再向开发者给出修改方案。未经开发者讨论和确认，不要直接按 review 意见修改代码。
