# Crater CLI Review Guide

**职责划分**：本文档说明一个阶段的 CLI 开发完成后，人类 reviewer 与 AI reviewer 应如何审查 `cli/` 目录下的代码、文档、测试与 Skills 变更。本文档只定义**审查流程、检查重点与问题反馈方式**；指令级行为以 **[COMMANDS.md](./COMMANDS.md)** 为准，开发约定与跨命令公共契约以 **[SPEC.md](./SPEC.md)** 为准，代码组织与实现叙事以 **[ARCHITECTURE.md](./ARCHITECTURE.md)** 为准。发现偏差时，优先指出应回到哪份文档修正或对齐，避免在 review 中重新定义一套做法。

本文档面向人类 reviewer、AI reviewer 与提交者。目标不是替代测试，而是在合并前确认实现、文档与用户可见行为没有漂移。

---

## 审查时机

在以下情况完成后，应按本文档进行 review：

- 新增或修改命令、flag、位置参数、输出、错误、退出码或交互行为。
- 修改 `internal/api`、`internal/state`、`internal/credential`、`internal/output`、`internal/completion`、`internal/i18n` 等影响多个命令的公共模块。
- 新增或修改快照测试、测试沙箱、golden 文件或 Makefile 测试入口。
- 新增或修改 `cli/skills/`、`cli/skill-template/` 或面向 AI Agent 的调用说明。
- 执行阶段性开发收尾，即使改动看似只在一个命令域内。

## 审查输入

开始 review 前，先确定本次变更的范围：

- 查看 `git diff --stat` 与具体 diff，确认所有 `cli/` 变更都属于本阶段目标。
- 阅读与变更相关的 `COMMANDS.md` 命令章节；若改动涉及公共契约，再阅读 `SPEC.md` 对应章节。
- 若实现方式、模块边界或调用链发生变化，阅读 `ARCHITECTURE.md` 对应章节。
- 若改动涉及 Skills，阅读相关 `SKILL.md`、`references/` 与 `cli/skill-template/`。

若发现代码已实现但文档没有对应依据，先按“文档漂移”处理，而不是假定代码是新的权威来源。

## 核心检查

### 1. 文档与行为一致

- `cli/docs/COMMANDS.md` 是指令级契约的权威来源。
- `cli/docs/SPEC.md` 是开发约定与跨命令公共契约的权威来源。
- `cli/docs/ARCHITECTURE.md` 说明实现结构与模块边界。
- `cli/docs/REVIEW.md` 说明审查流程与检查重点。
- 用户可见命令、参数、默认值、交互路径、JSON 字段、人类可读输出、错误分类与状态说明，必须能在 `COMMANDS.md` 找到对应约定。
- 跨命令公共行为必须符合 `SPEC.md`，尤其是 `--json`、`--no-interactive`、错误信封、成功信封、退出码、快照测试、i18n、Tab 补全与测试沙箱。
- 若代码行为合理但文档缺失，应要求补充对应文档；若文档过时，应指出应更新 `COMMANDS.md`、`SPEC.md` 或 `ARCHITECTURE.md` 的位置。
- 若发现实现、测试或 Skills 与上述文档冲突，应指出应回到哪份文档修正或对齐，不要在 review 评论中重新定义一套命令契约或实现规则。

### 2. 命令契约

- `--json` 输出必须是纯 JSON；成功写 stdout，错误写 stderr。
- `--json` 必须强制非交互；解析阶段失败也应遵守 JSON 错误输出。
- `--no-interactive` 下缺少必要信息时应直接失败，不得进入 prompt。
- 本地可判定的多个用法错误应尽量聚合后一次返回；具体适用边界见 `SPEC.md`「用法错误聚合」。
- `api_error` 的错误码、HTTP 档位与 context 字段应与 `SPEC.md`「api_error 与 HTTP」一致。
- 成功 JSON 的 `data` 字段只能包含 `COMMANDS.md` 对应命令章节允许的键，且不得泄漏 token、密码等敏感信息。

### 3. 输出与错误实现

- 成功 JSON 应通过 `internal/output` 提供的成功信封与 JSON 写出函数生成；命令代码不应自行拼接 JSON、直接调用 `json.Marshal` 写 stdout，或为单个命令另造信封结构。
- 错误输出应通过 `RunE` 返回错误，再由根命令统一渲染；命令实现不应绕过公共错误处理自行写 stderr、调用 `os.Exit`，或在多个位置分散决定退出码。
- 需要稳定错误分类、错误码与 context 时，应返回 `*clierror.Error` 或使用既有 helper 映射；访问平台失败时应走 `cmd/errors.go` 中的 API 错误映射，而不是在命令内临时判断 HTTP 状态码。
- 人类可读成功输出可以由命令写 stdout，但应使用 i18n 文案，并与 `COMMANDS.md` 对应章节保持一致。
- 错误信息应足够具体：说明哪个参数、状态、平台响应或本地条件导致失败；在不泄漏敏感信息的前提下，尽量提供用户或 Agent 可据此修正下一次调用的事实。
- 多行错误、聚合错误与 JSON 错误 context 应保持可读、可解析；不要只返回底层库的模糊错误，也不要把 Go 类型、原始 error 对象或不可 JSON 化的值塞入 context。

### 4. 外部访问与模块边界

- `cmd/` 负责命令编排、flag 读取、成功输出与错误返回，不应散落 HTTP path、Keyring 细节、状态文件路径或自定义 JSON 编码。
- `internal/api` 只处理 HTTP 与 API 语义，不应输出 stdout/stderr、调用 i18n 或构造 `*clierror.Error`。
- `internal/output` 负责渲染，不负责业务判断或进程退出。
- 访问 Crater 后端应通过 `internal/api` 的客户端、path 常量与领域方法；review 时应留意是否出现临时 HTTP client、手写 URL path、重复 DTO 或绕过统一错误类型的实现。
- 读写本地状态应通过统一的 session/state 入口；不应在命令代码里直接拼配置目录、读写 `state.json`，或绕开测试沙箱。
- 读写 token、密码等敏感凭据应通过 `internal/credential` 或现有 session 抽象；不应在命令代码、测试或 golden 中暴露 token 明文，也不应直接调用 Keyring 库绕过封装。
- 测试或补全路径中的网络、存储与凭据访问应遵守 sandbox 和快路径约束；新增访问点需要确认不会触达真实 HOME、真实 Keyring 或真实网络。
- 新增公共逻辑应放在既有职责匹配的包中；若需要改变边界，应同步更新 `ARCHITECTURE.md`。

### 5. 测试与可复现性

- 新增或改变用户可见输出、错误分支、`--json` 行为、`--no-interactive` 行为时，应有快照测试或明确说明为何不需要。
- 新增纯逻辑、解析、筛选、映射、补全或 sandbox 行为时，应有对应单元测试或说明风险。
- 阶段性开发收尾应确认已按 `SPEC.md`「快照测试」与 `cli/Makefile` 中定义的入口完成相应测试；若未运行，应说明原因与剩余风险。
- golden 文件必须由规定命令生成，不得手工编辑；若 golden diff 很大，应确认是契约变化而不是环境漂移。
- 快照测试不得依赖真实 HOME、真实 Keyring、真实网络、真实登录态或开发者本机状态；隔离要求见 `SPEC.md`「快照测试」与 `ARCHITECTURE.md`「测试沙箱」。
- Review golden 文件时应直接读取 `cli/testdata/snapshots/**/*.txtar`，不要只相信测试通过；重点检查 argv、stdout、stderr 与 exit 是否共同表达了预期契约。
- 对带 `--json` 的 golden 用例，stdout 或 stderr 中对应输出必须是合法 JSON，且不得混入提示语、表格、help 页面或其它装饰性文本。
- 对失败用例，exit 应与错误 category 匹配；例如用法错误不应返回成功退出码，网络或 API 错误不应被误归为用法错误。
- 对非交互用例，argv 应显式体现 `--no-interactive` 或由 `--json` 推导；缺少必要信息时不应出现 prompt 文案或挂起式交互输出。
- golden 中的错误消息应足够帮助定位问题；若只出现泛化的 `failed`、底层库英文错误或缺少字段名，应要求补齐更可操作的错误文案或 context。
- golden 中不得出现 token、密码、真实用户名、真实平台地址等敏感或本机相关信息；若出现，说明测试隔离或输出过滤存在问题。

### 6. 多语言与人类体验

- 面向人类的帮助、提示、成功与错误消息应走 i18n；新增命令域应补齐对应 catalog。
- 中英文输出都应可理解，且 JSON 键名保持稳定英文。
- 错误信息应帮助用户或 Agent 下一步修正问题；不要用内部实现细节替代可操作事实。
- 安全确认、交互式选择、取消操作等路径应同时考虑人类用户与 AI Agent。

### 7. Skills 与 Agent 友好性

- Skill 只能指导 AI 如何调用既有 CLI 契约，不应单独定义命令行为。
- 领域 Skill 应保持精炼；复杂流程放入同目录 `references/`。
- 变更全局调用规则时，应检查 `crater-cli-shared` 及相关领域 Skill 是否需要同步更新。
- 修改已分发 Skill 内容时，应按 `SPEC.md` 的约定评估是否提升 frontmatter `version`。

## 反馈格式

Review 反馈应优先列出会导致行为错误、契约漂移、测试失效或安全风险的问题。严重程度按所使用的 review 工具或团队流程表达即可；本文档不另行定义分级。每条问题尽量包含：

- 位置：文件与行号，必要时补充相关文档章节。
- 事实：当前代码或文档实际表现。
- 影响：它会造成哪类行为错误、契约漂移、测试风险或维护风险。
- 期望：应与哪份文档保持一致，或应先更新哪份文档再让代码与测试跟随。

不要把新的命令细则、API 约定或模块设计直接写进 review 评论作为权威来源。若确实需要调整契约，应要求提交者更新对应文档，再让代码与测试跟随。

## 收尾标准

一个阶段的 CLI 开发可视为通过 review，至少需要满足：

- 代码、`COMMANDS.md`、`SPEC.md`、`ARCHITECTURE.md` 与相关 Skills 之间没有已知冲突。
- 用户可见行为有契约依据，契约变化有测试或明确的风险说明。
- 快照与单元测试覆盖了本阶段主要风险，且测试结果可复现。
- Review 中必须在本阶段修复的问题已完成；剩余问题有明确归属与后续处理计划。
