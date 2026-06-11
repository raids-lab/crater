[English](../../CONTRIBUTING.md) | [简体中文](CONTRIBUTING.md)

# 为 Crater 贡献

本文档是参与 Crater monorepo 开发的起点。它先说明全仓库通用流程，再引导你进入具体模块的开发规范。

如果你第一次接触 Crater，建议按顺序读一遍本文档。日常开发时，把它当作流程清单；真正修改某个模块前，再进入对应模块的 `CONTRIBUTING`。

## 文档体系

- **README** 面向使用者，说明 Crater 是什么、怎么用、如何部署。
- **CONTRIBUTING** 面向贡献者。根文档定义全仓库流程，模块文档定义模块内工程约束。
- **`.github/instructions/*`** 是 Copilot 审查清单，引用 `CONTRIBUTING`，可以重复重要 MUST / SHOULD 检查项，但不是独立规则源。
- **`skills/`** 是 Agent 工作流，用于路由 AI 开发任务并重复关键规则；影响代码、文档、API、UI、Chart 或测试结果的开发约束必须先写入 `CONTRIBUTING`。
- **`cli/docs/*`** 是 CLI 指令契约与实现叙事，入口是 `cli/CONTRIBUTING.zh-CN.md`。

## 选择模块规范

修改文件前，先打开对应模块的规范：

| 改动范围 | 必读规范 |
|----------|----------|
| `backend/`（含 `internal/storage/`） | [backend/CONTRIBUTING.zh-CN.md](../../backend/CONTRIBUTING.zh-CN.md) |
| `frontend/` | [frontend/CONTRIBUTING.zh-CN.md](../../frontend/CONTRIBUTING.zh-CN.md) |
| `website/`、`docs/` | [website/CONTRIBUTING.zh-CN.md](../../website/CONTRIBUTING.zh-CN.md) |
| `cli/` | [cli/CONTRIBUTING.zh-CN.md](../../cli/CONTRIBUTING.zh-CN.md)，再读相关 `cli/docs/*` |
| `charts/` | [charts/CONTRIBUTING.zh-CN.md](../../charts/CONTRIBUTING.zh-CN.md) |

跨模块改动需要同时遵守所有相关模块规范，以及本文末尾的跨模块规则。

## 第一次准备环境

下文命令块分两类：

- **按序执行**：首次环境准备时按顺序运行，通常只需做一次（如克隆、配置远端、安装钩子）。
- **示例命令**：说明日常 Git 操作的写法；占位符（如 `YOUR_USERNAME`、`feature/your-feature-name`）和上下文需按实际情况替换，**勿整段复制执行**。

### Fork

我们禁止在主仓库（`raids-lab/crater`）中创建新的开发分支，也禁止直接向 `main` 提交。因此贡献者必须先 Fork 仓库，在自己的 fork 上开发，再通过 Pull Request 合入主仓库。

在浏览器打开主仓库 [https://github.com/raids-lab/crater](https://github.com/raids-lab/crater)，登录 GitHub 后点击右上角 **Fork**，将仓库 Fork 到你自己的账号下。

### 克隆与配置远端

主仓库是 `https://github.com/raids-lab/crater`。新任务分支和 PR 分支必须推送到你的 fork，不要推送到主仓库；不要直接向主仓库的 `main` 提交，也不要在主仓库创建新分支。

**推荐方式：克隆主仓库，添加 `myfork` 指向你的 fork**

**按序执行**（首次配置远端时运行一次）：

```bash
git clone https://github.com/raids-lab/crater.git
cd crater
git remote add myfork https://github.com/YOUR_USERNAME/crater.git
git remote -v
```

约定：`origin` = 主仓库（`raids-lab/crater`），`myfork` = 你的 fork（将 `YOUR_USERNAME` 换成你的 GitHub 用户名）。

**示例命令**（同步 `main` 与推送任务分支的写法；按当前分支与远端名替换后使用）：

```bash
git checkout main
git fetch origin
git rebase origin/main
git push myfork main
```

```bash
git push myfork feature/your-feature-name
```

**备选方式：克隆 fork，再添加主仓库为 `origin`**

若已克隆 fork（此时默认 `origin` 指向你的 fork），可将其重命名并补上主仓库。

**按序执行**（首次配置远端时运行一次）：

```bash
git clone https://github.com/YOUR_USERNAME/crater.git
cd crater
git remote rename origin myfork
git remote add origin https://github.com/raids-lab/crater.git
git remote -v
```

之后同样从 `origin` 拉取、向 `myfork` 推送；命令写法见推荐方式中的**示例命令**。

**备选方式：只保留 fork 作为远端**

若不想配置两个远端，克隆 fork 后维持默认设置，仅让 `origin` 指向你的 fork。

**按序执行**（克隆仓库，通常只需一次）：

```bash
git clone https://github.com/YOUR_USERNAME/crater.git
cd crater
```

在 GitHub 网页上对 fork 执行 **Sync fork** 后，**示例命令**（更新本地 `main`）：

```bash
git checkout main
git pull origin main
```

**示例命令**（推送任务分支）：

```bash
git push origin feature/your-feature-name
```

### 安装 Git 钩子

安装仓库自带的 pre-commit 钩子，使 `git commit` 在创建提交前自动进入受影响的子项目并执行对应的 `make pre-commit-check`。该 hook 主要适配 macOS 和 Linux 的 POSIX shell 环境；在 Windows 上可能失效，Windows 贡献者应手动运行相关模块检查，或使用 WSL。

**按序执行**（克隆仓库后运行一次）：

```bash
make install-hooks
```

## 本地调试

### 调试拓扑

Crater 部署时由多个运行在 Kubernetes 上的组件协作，但本地开发通常不需要复刻完整集群，也不需要把所有依赖都在本机启动。前端和后端几乎不适合单独调试；本地调试时至少应一并启动前端和后端。只有在修改 storage-server 行为或存储路径时，才需要按需启动 `backend` 的 storage server。

本地服务应通过运行配置连接项目已有的测试集群服务，例如 Kubernetes、PostgreSQL、存储、镜像仓库、网络或外部集成。除非任务明确要求做隔离的基础设施工作，否则不要另外启动本地数据库，也不要模拟完整集群。环境相关配置应由开发者或管理员提供，并且不得进入 Git。

### 运行配置

部分后端 / 前端运行配置与具体环境相关，可能需要管理员提供 Kubernetes 配置、数据库、网络或外部集成凭据。不要编造私有配置，也不要提交它们。

可以把本地配置集中到一个目录，再通过软链放到各模块：

```text
config/
├── backend/
│   ├── .debug.env
│   ├── kubeconfig
│   └── debug-config.yaml
└── frontend/
    └── .env.development
```

常用 target：

- `make config-link CONFIG_DIR=~/develop/crater/config`：创建软链，已存在文件会备份为 `.bak`
- `make config-status`：查看配置文件状态
- `make config-unlink`：仅移除软链
- `make config-restore`：从 `.bak` 恢复

各模块的具体环境说明见模块规范。

## 开始一次改动

日常 Git 操作同样遵循上文两类命令：**按序执行**与**示例命令**（勿整段复制示例块）。

### 先理解需求再修改

复杂或跨模块工作，先在 issue、设计说明或 PR 讨论中说明方案，再开始实现。

### 从正确来源同步本地 `main`

若采用双远端配置（`origin` + `myfork`），**示例命令**（从主仓库更新本地 `main`）：

```bash
git checkout main
git fetch origin
git rebase origin/main
```

若只使用 fork 作为远端，先在 GitHub 上对 fork 执行 **Sync fork**，**示例命令**：

```bash
git checkout main
git pull origin main
```

### 分支前缀与 Commit type

创建任务分支前，先根据下表确定本次改动的 **type**，并在分支名中使用对应前缀（如 `feature/`、`fix/`）。后续 commit message 也使用同一套 type。

| type | 分支前缀 | 含义 |
|------|----------|------|
| `feat` | `feature/` | 新功能 |
| `fix` | `fix/` | 缺陷修复 |
| `docs` | `docs/` | 文档变更 |
| `style` | `style/` | 代码风格 / 格式 |
| `refactor` | `refactor/` | 重构 |
| `test` | `test/` | 测试相关 |
| `chore` | `chore/` | 构建 / 工具链 |

Commit message 格式为 `type(scope): subject`，其中 `scope` 可选。

### 在本地创建任务分支

如果当前在 `main`，或当前分支与任务不匹配，按上表选择前缀，创建或切换到命名清晰的任务分支。除非维护者另有要求，分支名格式为 `<前缀><简短描述>`，描述部分使用小写英文与连字符（如 `feature/add-job-submission-form`）。

**示例命令**：

```bash
git checkout -b feature/your-feature-name
```

已存在的工作分支应 rebase 到最新本地 `main`，保持线性历史。优先 rebase，不要把 `main` merge 进功能分支。

### 按模块规范实现

优先复用现有架构、helper API 和代码风格，保持改动聚焦。

## 核心工程规则

所有模块都适用：

- **功能服务真实用户需求**。功能应解决具体用户问题，而不是只增加实现表面。设计行为、错误信息、操作流程、默认值和文档时，都要持续考虑用户体验。
- **构建、lint、测试优先使用 `make`**。存在 Makefile target 时，优先使用模块 target，而不是直接调用 `go`、`pnpm` 或 `helm`。
- **Go 构建 / 测试前检查工具链**。涉及 `backend/`、`cli/` 等 Go 子项目时，先运行 `go version`，确认它与对应子项目 `go.mod` / 模块规范一致。
- **代码注释一律使用英文**，并与既有命名风格和架构模式保持一致。
- **严禁提交敏感信息**：密钥、Token、密码、内网 IP、kubeconfig、证书和生产凭据都不能提交。
- **不确定就问**。规范或上下文缺失时主动澄清，不要盲目猜测。
- **规范可演进**。如果现有规则在当前场景下会损害质量、架构或安全，应指出并建议更新对应文档，而不是默默违背。

## 提交或推送前验证

### 运行相关自动化检查

根 `pre-commit` 检查暂存文件。**示例命令**（将 `<your-files>` 换成实际路径）：

```bash
git add <your-files>
make pre-commit-check
```

也可以在子项目内运行完整检查（按需进入对应目录，**非**一次性整段执行）：

```bash
cd frontend && make pre-commit-check
cd backend && make pre-commit-check
cd website && make pre-commit-check
cd cli && make pre-commit-check
```

### 要求开发者人工检查

自动化检查或 AI 执行的检查不能替代开发者判断。最终 commit 或任何 push 前，开发者必须亲自检查受影响的功能、页面、命令、生成产物或文档，并判断改动是否可以提交。

Agent 可以辅助列出应该打开哪些页面、使用什么角色、执行哪些操作、尝试哪些命令或阅读哪些文档。Agent 也可以把自己执行过的检查和要求开发者执行的人工检查记录到临时 task note，之后生成 PR 描述时复用。

文档改动必须由开发者人工阅读检查后才能提交或推送。不要直接提交未经阅读检查的 AI 生成文档；AI 生成文档常会漏掉关键背景、步骤或运维细节。开发者应根据经验直接修改文档，或要求 Agent 调整后再提交。

## Commit 与推送

若整个 PR 只有一个 commit，推荐 commit message 与分支名保持一致：将分支名中的 `/` 替换为 `:`，`-` 替换为空格。例如分支 `docs/contributing-command-hints` 对应 `docs:contributing command hints`；分支 `feature/portal-add-job-form` 对应 `feature:portal add job form`。

若有多个 commit，则每个 commit 分别按上表编写 `type(scope): subject` 格式的消息。

执行过 `make install-hooks` 后，`git commit` 会触发已安装的 pre-commit hook。该 hook 会检查暂存文件，进入受影响的子项目，并在允许提交前执行对应的 `make pre-commit-check`。这个提交时自动检查主要面向 macOS 和 Linux；在 Windows 上可能无法正常工作。Windows 环境下请手动执行受影响模块的 `make pre-commit-check`，或使用 WSL。

提交时明确指定文件或目录，避免 `git add .`。**示例命令**：

```bash
git add backend/internal/handler/user.go
git commit -m "feat(portal): add job submission form"
git push myfork feature/your-feature-name
```

任务分支只推送到 fork 远端（双远端配置下为 `myfork`；若只配置了 fork 作为 `origin`，则推送到 `origin`）。

## 创建并迭代 Pull Request

欢迎 AI 辅助开发，但开发者必须自己理解并把关。创建或更新 PR 前，开发者必须检查最终 diff、验证结果和 PR 描述。

PR 描述必须使用**双语 Markdown**，并覆盖：

- **变更意图**：一句话概述本 PR，必要时说明动机。
- **核心改动**：按「做了什么」归类，不要只按文件罗列。
- **测试验证**：只列实际执行过的检查，并清晰区分自动化 / AI 检查与开发者人工检查。
- **截图**：涉及前端 / UI 改动时必须附相应界面截图，展示受影响界面状态。
- **其他说明**：可选，记录特殊风险、迁移说明、上线说明或兼容性说明。
- **关联 ISSUE**：如适用，以 GitHub 可识别的方式逐行列出，例如 `Resolve #208`。

简单改动保持简短即可。Agent 生成 PR 描述的具体流程见 `skills/crater-devel-shared/SKILL.md`；模板与示例见 `skills/crater-devel-shared/references/pr-description-template.md`。

分支推送后，如果 `gh` 等工具可用，Agent 可以帮助创建 PR；也可以用代码框输出 PR 描述文本，交给开发者自行使用。无论哪种方式，Agent 都必须先把 PR 描述展示给开发者，并取得确认后，才能用它创建或更新 PR。

PR 创建后，需要检查 workflow 状态。PR 也可能需要和 Copilot review 或人工 review 进行多轮迭代。Agent 可以自行获取 PR 链接，或要求开发者提供链接；随后阅读 review 意见，判断每条意见是否正确、是否值得修改，再提出修改方案。未经开发者讨论和确认，不要直接按 review 意见修改代码。

## 跨模块规则

- **前后端身份一致**：管理员视图只调管理员接口（URL / 函数名带 `admin` 前缀），普通用户调用户接口；前后端两侧需对应。
- **功能上线同步文档**：新功能或重大变更须同步修订 `website/` 文档。
- **配置结构变更同步 chart**：改动涉及配置结构时，须同步更新 `charts/`，提升 `charts/crater/Chart.yaml` 的 `version` 与 `appVersion`，并保持二者为完全相同的值，同时同步 `charts/crater/README.md`。版本提升级别与 GitHub tag 提醒遵循 `charts/CONTRIBUTING.zh-CN.md`。

## 维护开发约定

新增或修改开发约定时，必须保持权威来源清晰：

- 开发约束与期望工程结果写入合适的 `CONTRIBUTING`：全仓库规则写根文档，模块规则写模块文档，CLI 行为契约写入 `cli/docs/*`。
- `.github/instructions/*` 用作审查清单并引用 `CONTRIBUTING`；可以重复重要 MUST / SHOULD 检查项，但不能成为独立规则源。
- `skills/` 用于 Agent 任务路由、AI 开发流程与操作指引。Skill 应引用相关 `CONTRIBUTING`，并可把重要约定重复为高优先提醒。
- 若规则主要约束 AI 开发过程，可写在对应 Skill；若规则影响最终代码、文档、API、UI、Chart 或测试结果，则必须先沉淀到 `CONTRIBUTING`。

## Agent Skills

仓库在 `skills/` 下随附面向开发者的 Agent Skills，指导 AI Agent 完成 Crater 开发任务。功能、边界与各 Skill 职责见 [skills/README.md](../../skills/README.md)。

从 GitHub 列出可用 Skill：

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/skills -l
```

为受支持的 Agent 全局安装全部 Crater 开发 Skill：

```bash
npx skills add https://github.com/raids-lab/crater/tree/main/skills -g --all
```
