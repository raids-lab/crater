# Crater CLI 开发规范和约束

**职责划分**：本文档说明开发 CLI 时要**完成哪些工作**、这些工作**如何落实**，以及须遵守的工程约定与**跨命令公共契约**。各命令的选项、分命令行为与人类可读 / JSON 输出的**指令级细则**以 **[COMMANDS.md](./COMMANDS.md)** 为准，勿在本规范中逐条重复定义。现有代码如何实现、为何如此拆分、模块与调用链如何协作等**实现叙事**写在 **[ARCHITECTURE.md](./ARCHITECTURE.md)**。人类与 AI 增补内容时请分清三者，勿将实现叙事或指令级契约混入本规范。

本规范面向人类与 AI 开发者，须严格遵守。修改意见请通过 ISSUE 或 PR 更新本文档。

---

## 总体

### 名词优先

命令路径遵循 **名词优先** 原则，即域名词在前，动词在后，满足 `crater <domain> <verb>` 的形式。

域名词使用单数形式（如 `image`、`job`），与 API 资源名保持一致。

**例外**：常用命令的简写或模仿 Unix 传统命令名可不遵守此约束。

### 文档驱动

一方面，整个开发过程中必须严格遵守本规范文档；另一方面，使用[指令文档](./COMMANDS.md)驱动新指令的开发和已有指令的修改。

推荐首先在指令文档中详尽的约束指令的**描述、选项、处理逻辑和具体行为**，然后让 AI 根据本规范文档和指令文档进行开发。

需要保证指令的具体行为，和指令文档中所描述的严格一致。

### AI Agent Skills（分发约定）

`cli/skills/` 存放面向平台用户及其 AI Agent 的可分发 Skills，用于指导 AI 安全、稳定地调用 `crater` 命令；它们不是开发者内部规则，也不替代 [COMMANDS.md](./COMMANDS.md) 的命令契约。

- 顶层 Skill 按用户任务域组织，命名使用 `crater-cli-<domain>`；共享基础规则使用 `crater-cli-shared`。
- `crater-cli-shared` 承载全局调用规则（如 `--json`、`--no-interactive`、`--help`、错误输出、敏感信息与确认规则）；领域 Skill 应在开头要求先读取或应用它。
- 领域 Skill 的 `SKILL.md` 保持精炼，负责适用场景、核心概念、安全边界、工作流索引和常用范例；具体流程放在同目录的 `references/` 下。
- `references/` 按用户任务场景拆分，而不是机械按子命令拆分；例如查看身份与切换身份可以放在同一参考文档中。
- Skill 中可以给出相对路径作为定位提示，同时保留 Skill 或 reference 名称，兼容不同 AI 工具的路径解析能力。
- 新增或调整 Skill 时，应避免写入面向 Skill 作者的元说明；措辞应直接面向执行任务的 AI Agent。
- 变更跨命令公共契约时（如错误码、退出码、`--json` 形状、全局选项、安全确认规则），必须同步更新 `crater-cli-shared` 及其 references。
- 修改任何已分发 Skill 内容时，必须提升该 Skill frontmatter 中的 `version`；若变更 shared 规则影响多个领域，也应同步评估相关领域 Skill 是否需要提升版本。

`cli/skill-template/` 存放编写领域 Skill 与 reference 的模板；模板仅用于维护分发内容，不参与 CLI 运行时行为。

新增 Skill 时：

- 先阅读 `cli/skill-template/skill-template.md` 与 `cli/skill-template/reference-template.md`，再参考 `cli/skills/` 下已有 Skill 的实际写法。
- 先判断是否需要新增顶层领域 Skill；若只是现有领域内的新场景，优先在该领域的 `references/` 下新增或更新场景文档。
- 新领域应同时补齐 `SKILL.md` 中的适用场景、安全边界、工作流参考和常用范例；涉及复杂流程时再拆出一个或多个 reference。
- 若新增命令改变了用户可见行为，仍须先更新 [COMMANDS.md](./COMMANDS.md)；Skill 只能说明 Agent 如何调用既有契约，不能单独定义命令行为。

---

## API

本节只规定**开发者**在扩展或修改「与 Crater 平台的 HTTP 交互」时必须怎么做。包分工与实现叙事见 [ARCHITECTURE.md](./ARCHITECTURE.md)。

### 与平台 HTTP 相关的实现约束

- 所有对外请求的 path 常量写在 `internal/api/paths.go`，禁止在方法体内手写未集中管理的 `"/api/..."` 片段。
- 通用客户端、`Response[T]`、与连接/传输层直接相关的入口（如 `NewClient`、`SetToken`）放在 `internal/api/client.go`；新增域接口时，该域的 DTO、小接口（如 `AuthClient`）、`NewXxxClient` 与同域的 `(*Client)` 方法放在**同一域文件**（如 `auth.go`），避免在 `cmd` 或无关包中散落 HTTP 细节。
- `internal/api` 的导出方法只返回业务数据与 `error`；使用本包错误类型表达「已收到 HTTP 响应但未成功」「网络层失败」等语义。**禁止**在 `internal/api` 内构造 `*clierror.Error`、向 stdout/stderr 输出、或调用 `i18n`。
- 通过 `cmd/errors.go` 的 `cliErrFromAPI` 等 helper 将 `internal/api` 返回的错误映射为 `*clierror.Error`，并在 `RunE` 中 `return`；命令实现不得自行分散处理 API 错误码映射。HTTP 档位与 `Code` 的对应关系遵循本文档「命令结果」中的 `api_error` 约定。
- 仅在本地开发或自测需要**伪造传输层结果**时，使用环境变量 `CRATER_TEST_SANDBOX_HTTP`（如 `timeout` / `error404` 等）；**允许取值与行为以架构文档「网络通信」节为准**。契约验证与前后端联调须使用真实服务或后端提供的 mock，不得把模拟响应当作平台契约。

---

## Tab 补全（开发者约定）

本节说明：**开发新命令或改交互时，为适配当前 Tab 补全机制开发者需要做什么**。端到端行为见 [ARCHITECTURE.md](./ARCHITECTURE.md)；对外子命令契约见 [COMMANDS.md](./COMMANDS.md)。

**产品范围（与实现一致）**：当前仅 **bash / zsh**；不包含 PowerShell（`pwsh`）。Windows 上若需 Tab 补全，以 **Git Bash + bash** 路径为准。

### 默认行为

- 子命令名：从 Cobra 命令树枚举可见子命令（尊重 `Hidden` 等）。
- flag 名（如 `--json`、`--platform`、`-m`）：从 `pflag` 元数据生成候选。

只要命令树和 flags 正确挂在对应的 `*cobra.Command` 上，就会有基本补全。

### 什么时候需要注册

当候选无法仅靠命令树静态得出时（枚举值、读取 `state.json`、按前缀过滤、业务语义候选等），在对应命令域的 `cmd/*.go` 的 `init()` 中注册动态补全函数。

当前只保留两类局部注册点（不引入命令级接管接口；`Completer` 方案已移除）：

- 位置参数候选：`completion.RegisterPositional(...)`（参考实现：`cmd/config.go` 的 `init()`）
- flag 值候选：`completion.RegisterFlagValue(...)`（参考实现：`cmd/auth.go` 的 `init()`）

### RegisterPositional（位置参数）

`completion.RegisterPositional(commandKey, argIndex, fn)`：

- `commandKey`：不含根命令名 `crater`；与 `cobra.Command.CommandPath()` 按空格拆分后去掉第一段一致。例如 `crater config language` → `[]string{"config","language"}`。
- `argIndex`：从 0 开始，表示该子命令下第 \(N\) 个纯位置参数。引擎会跳过已出现的 flag 及其取值后再计数。
- `fn(ctx)`：返回 `[]completion.Candidate`，其中 `Value` 为插入 token（不翻译，应稳定），`Description` 为可选说明（zsh 展示，bash 通常忽略）。按前缀过滤时可用 `completion.CurrentWordPrefix(ctx)` 取当前正在补全词中的已输入片段（可能为空）。

实例：`crater config language [LANG]` 在 `cmd/config.go` 的 `init()` 里注册 `argIndex=0`，候选来自 `i18n.GetSupportedLanguages()`（Value），展示名来自 `i18n.GetLanguageDisplay()`（Description），并按 `CurrentWordPrefix` 过滤。

### RegisterFlagValue（flag 取值）

`completion.RegisterFlagValue(commandKey, flagName, fn)`：

- `commandKey`：同上（不含 `crater`）。
- `flagName`：Cobra 的长选项名（不含 `--`），例如 `mode` / `platform`。
- `fn(ctx)`：返回 `[]Candidate`；`Value` 为值 token（不翻译）。值前缀过滤可用 `completion.CurrentWordPrefix(ctx)`（与归一后的当前槽位一致，可能为空）。

引擎会自动适配多种输入形态，并路由到同一个 `RegisterFlagValue`（实现见 `internal/completion/engine.go` 的 `completeFlagValues`，对应三种分支：行内长选项、flag 与值相邻两词、bash 下 flag / `=` / 值 三词）：

- `--flag=valuePrefix`（**长选项** `--…` 与 `=` 在同一 token；短选项 `-f=value` 若未被拆成多词，当前不归入本路由）
- `--flag valuePrefix` / `-f valuePrefix`（flag 与值分两词）
- `--flag = valuePrefix` / `-f = valuePrefix`（`=` 为独立 token；常见于 bash：`COMP_WORDBREAKS` 含 `=`）

`fn(ctx)` 收到的上下文中，引擎已将当前槽位规范为正在补全的**纯值前缀**（不含 `--flag=`）；`completion.CurrentWordPrefix(ctx)` 与该字符串一致。

### 工程约束

- 补全查询路径不得发起网络请求；读取本地配置需轻量且可快速返回。
- 补全子进程 stdout 只输出 shell 协议要求的候选行，不得混入业务成功 JSON 信封或无关提示。

---

## 快照测试（Snapshot / Golden）

本节规定：为保证 CLI 在多语言、`--json`、错误处理与退出码等关键行为上的稳定性，仓库使用 **快照测试（golden）** 作为强约束的一部分。实现细节与代码组织见 [ARCHITECTURE.md](./ARCHITECTURE.md) 的「快照测试」小节。

### 范围与原则

- 快照测试优先覆盖**错误处理**与分支稳定的输出（含 `--no-interactive`、`--json`、未知子命令等），避免依赖开发者本机状态（如 keyring、真实网络、真实登录态）。
- 快照必须在**受控环境**中运行：固定语言（如 `CRATER_LANG`）、固定 locale（`LANG/LC_ALL`）、隔离 HOME（临时目录），并避免交互式输入。
- golden 文件的内容应**人类可读**且便于 diff；更新 golden 需明确且可审阅，不允许在 CI 中静默改写。
- **稳定性优先**：只有在不同机器、不同开发者环境下能够稳定复现的输出，才允许进入 golden。凡依赖本地 keyring/凭据、真实网络、真实平台数据、用户目录残留状态等的结果，默认**不得**写入 golden；应启用测试沙箱（见本节下文与 [ARCHITECTURE.md](./ARCHITECTURE.md)）、隔离 HOME、注入假实现或仅做非快照测试。

### 测试沙箱（环境变量约束，强制约定）

快照与可复现测试通过“网络 + 存储”两类沙箱开关实现隔离（语义是**强约束**）：

- **网络（阻止真实请求）**：`CRATER_TEST_SANDBOX_HTTP`（如 `timeout` / `error404`）
- **存储（阻止真实存储访问）**：`CRATER_TEST_SANDBOX=1`（快照测试默认开启）

两者的**共同目的**：

- 避免测试过程中修改或影响外部环境（开发者真实配置/凭据、真实网络）。
- 避免测试受外部环境影响（登录态、网络波动、残留文件），保持稳定可复现。

**边界**：沙箱不是完整的进程隔离；语言与 locale（如 `CRATER_LANG`、`LANG/LC_ALL`）仍可能影响行为与输出，快照 harness 必须显式固定。

**实现与扩展**：沙箱在代码中的落点、取值与具体行为，以及“如有必要可提供假实现”的方式，见 [ARCHITECTURE.md](./ARCHITECTURE.md) 的「测试沙箱（Sandbox）」与「传输层模拟」小节。新增配置或新增存储访问时必须确保不破坏沙箱约束。

### 必测场景（第一优先级）

每个新增或改动较大的命令域，至少补齐以下快照用例（可按命令重要性裁剪，但不得遗漏“未知子命令 + `--json`”这一类全局行为）：

- **参数错误（usage_error）**：缺参、非法值、互斥选项等；尤其在 `--no-interactive` 下要验证“缺信息即失败”的输出与退出码。
- **命令拼写错误（usage_error）**：根命令与各命令域下的未知子命令；输出采用短报错（不自动刷整页用法，提示 `... --help`，可选 `Did you mean this?`），并同时覆盖 `--json` 与非 `--json`。
- **网络失败（api_error，访问平台的命令）**：使用 `CRATER_TEST_SANDBOX_HTTP`（如 `timeout` / `error404`）模拟“参数解析成功但不发真实请求”的失败输出；该类失败主要用于证明命令已正确解析并进入请求路径，同时避免依赖真实网络与真实 token。

### 目录与命名（强制约定）

- golden 文件目录：`cli/testdata/snapshots/<domain>/`。
- golden 文件格式：`txtar`（单文件包含多用例段落）。
- golden 文件名：`<stem>.<lang>.txtar`，其中 `lang` 为 `en` / `zh-CN`。

### 运行方式（强制约定）

本规范强调**测试分层与职责**，具体命令入口以 `cli/Makefile` 为准（例如 `unit-test` / `snapshot-check` / `snapshot-update` / `test`）。

其中快照测试的实现入口与用例定义由 `cli/test/snapshots/**` 下的测试代码驱动。**golden 快照测试默认开启存储沙箱（`CRATER_TEST_SANDBOX=1`）**；除非在文档中明确说明并给出替代隔离方案，否则不得在快照 harness 中关闭该默认设置。

**禁止手改 golden**：`cli/testdata/snapshots/**/*.txtar` 须通过 `make snapshot-update`（或 `UPDATE_SNAPSHOTS=1 go test ./test/snapshots/...`）由测试运行生成；不得直接编辑 golden 文本冒充快照结果。

新增命令或新增包时：

- 若引入了关键的**与进程无关的纯逻辑**（例如解析、映射、筛选、补全引擎等），应视需要补充对应的**代码单元测试**（包内 `_test.go`）。
- 若引入了新的用户可见输出/错误分支或重要组合路径，应补充对应的**快照测试**（golden），以锁定 CLI 合约并避免回归。

## 命令结果：错误与成功

**相关章节**：[COMMANDS.md](./COMMANDS.md)、[ARCHITECTURE.md](./ARCHITECTURE.md)。

### 责任路由

分工如下。

| 位置 | 失败时 | 成功时 |
|------|--------|--------|
| `cmd/*` 各命令 `RunE` | 编排业务；失败则 `return err`。常规路径不要自行往 stderr 写错误 JSON，由 `Execute` → `handleError` 写 stderr 并退出。 | 在 `return nil` 之前完成成功输出，且只在此处写成功内容：`--json` 时用 `internal/output` 的 `WriteSuccessJSON`、`SuccessEnvelope`；否则用 `fmt` 与 `i18n` 写 stdout。`root` 不会替你拼成功 JSON。 |
| `cmd/root.go` 的 `Execute` | `rootCmd.Execute()` 非 nil 时调用 `handleError(err)`，再经 `exitCodeFor` 使用 `pkg/errorcodes.ExitCodeForCategory` 映射退出码并 `os.Exit`。 | `RunE` 成功返回后不再写 stdout；业务成功输出已在各命令内写完。 |

### 失败：`return` 什么

需要稳定错误 JSON 与退出码时，优先 `return` 已组装好的 `*clierror.Error`：`Category`、`Code`（`pkg/errorcodes`）、`Message`（`i18n`）、可选 `Context`。脚本应消费 `Context` 等结构化字段，勿依赖解析自然语言 `Message`。`Message` **允许多行**；人类可读 stderr 由 `internal/output` 在 `Error:` 下对每行统一加基础缩进，行内额外空格由你写入 `Message` 即可，会与基础缩进叠加（见 [COMMANDS.md](./COMMANDS.md)）。

交互过程中用户中止（如 Ctrl+C，无论 `survey` 还是其它 TTY 读入）须统一 `return` `cancelled` / `ERR_OPERATION_CANCELLED`（退出码 3，经 `cmd/errors.go` 的 `errSurveyOrSame` 等集中映射），不得以 130 或未分类错误结束。

`Context` 只能放 JSON 可序列化的事实值（如字符串、数字、布尔值、数组、对象，以及 `nil`），禁止放入 `error`、函数、channel、包含循环引用的对象或其它无法被 `encoding/json` 编码的值。与后端响应相关的错误应倾向于直接提供后端返回的事实字段（如 `http_status`、`crater_code`、`msg` 等），不要把原始响应对象或 Go 错误对象塞入 `Context`。若开发者违反该约定导致错误 `Context` 无法 JSON 化，渲染层会保留原始 `category` / `code` / `message`，并将 `context` 替换为“错误 context JSON 化失败，请联系开发者修复”的诊断信息，以保证 stderr 仍是合法 JSON。

若不是 `*clierror.Error`：JSON 模式下退化为 `system_error` 与 `ERR_COMMAND_EXECUTION`，`message` 取自 `err.Error()`；字段名与整体形状仍须满足 COMMANDS「错误处理规范」。

### 用法错误聚合（本地校验）

对**可在发起副作用之前本地判定**的用法问题（缺 flag、非法枚举值、互斥选项等），命令应**先收集、再一次报错**，避免「修一个参数再运行又冒出下一个」的 fail-fast 体验。该约定适用于各命令域；具体命令哪些字段在非交互下必填、交互下是否 prompt，写在 [COMMANDS.md](./COMMANDS.md) 的指令级说明中，聚合机制本身以本节为准。

**适用范围**

- 优先用于 `--no-interactive` 或 `--json`（二者均会强制非交互）下、依赖多个 flag 或本地规则的场景。
- 仅汇总**尚未发请求、未写存储**前能确定的错误；API / 网络 / Keyring 等失败仍按单条 `api_error` / `system_error` 返回。
- 交互模式下：缺项一般改为 survey / prompt；用户在提示后仍提交空值时，用「不能为空」类文案（`err_prompt_empty`），**不要**写成「非交互模式下必须提供 `--…`」。

**实现**

- 在 `cmd/errors.go` 用 `usageIssue` 描述单条问题，经 `errUsageFromIssues` 合并为一条 `*clierror.Error`（`category` 为 `usage_error`）。
- 顶层 `code` 由 `primaryUsageCode` 从各条 `usageIssue.Code` 推导（全为缺参时用 `ERR_MISSING_REQUIRED_FLAG`，全为非法值时用 `ERR_INVALID_FLAG_VALUE`，混合时取 `ERR_INVALID_FLAG_VALUE`）。
- `message` 为多行文本（每条一行），人类可读 stderr 按既有规则缩进（见 COMMANDS「错误处理规范」）。
- `--json`：**仅一条**问题时与往常相同，**不**附加 `context.issues`；**两条及以上**时在 `context.issues` 中给出数组，元素含 `field`（可选）、`code`、`message`，供脚本与 Agent 解析；仍只输出**一个** JSON 错误对象、一次退出（`usage_error` → 2）。

**测试**

- 多问题聚合路径须有快照或单元测试覆盖；golden 须通过 `make snapshot-update` 生成，禁止手改（见上文「快照测试」）。

### `api_error` 与 HTTP：公共 `Code` 档位

凡 `Category == api_error` 且语义来自**已收到的** HTTP 响应时，`Code` 必须按下表与状态码档位一致，以便跨命令对脚本暴露同一套档位。`usage_error`、`system_error` 等不必带 `_401` 这类后缀。无 HTTP 结果时不得使用带 HTTP 后缀的 `Code`，不得伪造 `http_status`；超时、DNS、连接失败等用 `pkg/errorcodes/codes.go` 中的非 HTTP 后缀码。

| HTTP | `Code`（名与值一致） |
|------|----------------------|
| 401 | `ERR_UNAUTHORIZED_401` |
| 403 | `ERR_FORBIDDEN_403` |
| 404 | `ERR_NOT_FOUND_404` |
| 其它 4xx（非上列） | `ERR_CLIENT_4XX` |
| 500–599 | `ERR_SERVER_INTERNAL_5XX` |
| 其它或无法归类 | `ERR_API_OTHER` |

有 HTTP 时，`Context` 建议包含整数 `http_status`；能读到响应体时再带 `crater_code`、`msg` 等事实。无 HTTP 响应时不得写入 `http_status`；超时、DNS、连接失败等传输层失败使用 `ERR_NETWORK_FAILURE`，并只在 `Context` 中放可序列化的错误事实（如 `msg`）。映射以 `cmd/errors.go` 的 `cliErrFromAPI` / `apiCodeForHTTP` 为准；新增档位须同步修改 `pkg/errorcodes/codes.go`、上表，以及 COMMANDS 中引用本节的表述。

### 成功：`--json` 的公共形状

在 stdout / stderr 只出合法 JSON、禁止混入装饰性文字等前提下，须满足 COMMANDS 全局与各命令对成功 / 错误体的规定。编码统一经 `internal/output.MarshalJSONPretty`（两空格缩进、尾随换行），成功与失败路径均适用，禁止在命令内自行 `json.Marshal` 紧凑输出。

实现上：

- 使用 `output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(payload))`。顶层 `status` 使用常量 `output.JSONSuccessStatus`；业务载荷只进 `data` 对象。成功体不使用错误体的 `category` / `code`；`data` 不得含 `http_status`。
- `data` 允许哪些键、是否可为 `{}`、顶层是否带 `message`、敏感字段能否出现等，**只以 COMMANDS 为准**，本规范不列举。

### 成功：非 `--json`

不用顶层信封。`fmt` 与 `i18n` 写人类可读输出至 stdout；字段不必与 JSON 形态一一对应。

### 成功路径上的 JSON 编码失败

`WriteSuccessJSON` 内编码失败时 `return` `*clierror.Error`：`system_error` 与 `ERR_JSON_ENCODE_FAILED`，经 `handleError` 以错误 JSON 写 stderr，进程非零退出。

### 退出码与 `--json` 预解析

`Category` 到进程退出码的映射以 `pkg/errorcodes.ExitCodeForCategory` 为唯一实现源；数值须与 COMMANDS「错误处理规范」一致。`RunE` 返回非 `*clierror.Error` 的退化行为同上节。各命令内禁止 `os.Exit`。若要调整映射，改 `pkg/errorcodes` 并同步 **COMMANDS** 与本文档涉及处。

`Execute` 在 Cobra 解析前对 `os.Args` 做 `--json` 预扫描并与 `viper` 同步，使解析阶段失败时仍可按 JSON 写 stderr；**该行为的产品定义在 COMMANDS 全局 `--json`，实现须与之完全一致**。

### 日志

[TODO] 增加日志系统

---

## 人文关怀

### 多语言支持

单一二进制支持中文（`zh-CN`）与英文（`en`）。所有面向人类的文案（帮助信息、提示语、成功/失败消息等）必须走 `internal/i18n`；`--json` 模式下仅要求 JSON 键名为英文，但错误体 `message` 与成功体可选 `message` 仍应随当前语言变化（见上文「命令结果：错误与成功」与 COMMANDS 全局约定）。

**翻译文件分布**

翻译 key 按命令域拆分在 `internal/i18n/catalog_*.go` 中，并统一注册到 `internal/i18n/i18n.go`（`mergeCatalogs()`）：
- `catalog_root.go`：根命令、全局选项
- `catalog_auth.go`：`auth` 命令族
- `catalog_config.go`：`config` 命令族
- `catalog_completion.go`：`completion` / `comp` 命令族
- `catalog_errors.go`：通用错误消息

新增命令域时：新增 `catalog_<域>.go`，并在 `mergeCatalogs()` 注册。

**Key 规范**

- 命令描述：`<域>_<命令>_short` 与 `<域>_<命令>_long`
- 选项描述：`flag_<选项名>`（沿用 `flag_` 前缀）
- 提示语：`prompt_<字段>`
- 表格标题：`table_<列名>`
- 结果消息：`<动作>_<结果>`

**关于 “cmd 中硬编码文案” 的约定（重要）**

在 `cmd/*.go` 中，命令的 `Short` / `Long` 允许出现硬编码英文作为**开发期占位与提示**；但**发布前必须提供**对应的 i18n key（`*_short` / `*_long`），不得依赖硬编码英文作为最终文案来源。运行时，CLI 会在执行入口根据当前语言统一覆盖命令树的 `Short` / `Long` 与 flag 的 `Usage`。

换句话说：不要以 `cmd/*.go` 中看到的 `Short` / `Long` 字面量作为最终显示文案的权威来源；权威来源是对应的 i18n key。关于覆盖发生的具体时机与实现机制，见 `ARCHITECTURE.md` 中的 i18n/help 初始化与更新章节。

**用法（按职责拆分）**

- 命令文案的权威来源：在 `internal/i18n/catalog_<域>.go` 中维护 `*_short` / `*_long` 等 key。
- 交互提示、成功/失败消息：在 `RunE` 或辅助函数中使用 `i18n.T(...)` 输出（stdout/stderr 规则见本规范上文与 COMMANDS）。
- 选项 `Usage`：在定义 flag 时使用 `i18n.T("flag_<name>")`。
