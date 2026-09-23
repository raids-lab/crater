# CLI COMMANDS REFERENCE (The Contract)

**职责划分**：本文档是 **Crater CLI 指令级契约** 的权威来源，包含全局通用规范，以及各命令章节对选项、处理逻辑、人类可读与 JSON 输出的行为与字段定义。开发时要完成哪些工作、这些工作如何在仓库与流程中落实，见 **[SPEC.md](./SPEC.md)**。现有代码如何组织、模块与调用链如何协作，见 **[ARCHITECTURE.md](./ARCHITECTURE.md)**。阶段性开发完成后的审查流程、检查重点与反馈方式见 **[REVIEW.md](./REVIEW.md)**。实现必须与本文档对各命令的约定一致，且不得违反 SPEC 中的跨命令公共约定。

## 全局通用规范 (Global Requirements)

为了确保对 AI Agent、CI/CD 环境以及普通开发者的友好性，**所有命令（无论是否具备交互逻辑）必须统一支持以下全局选项：**

- `--json`: 
  - **行为**: 强制开启 `--no-interactive`，输出纯净的 JSON 至 `stdout`。实现会在 Cobra 解析参数**之前**按 pflag bool flag 语义预扫描 `os.Args` 是否包含 `--json` 或 `--json=<bool>`，因此 **`--json` 可出现在参数序列任意位置**；即使因未知 flag 等导致解析阶段失败，**错误输出仍可按 JSON 模式**写到 stderr。空格分隔的 `--json false` 不属于支持形式，等价于 `--json` 后跟普通参数 `false`。
  - **Stdout**: 输出**格式化后的 JSON (Pretty-printed, 带缩进和换行)**，确保既对人类可读，又可被 `jq` 等工具解析。禁止包含任何非 JSON 的装饰性文字。
  - **成功体**：信封（顶层字段与 `data` 约束）见 **[SPEC.md](./SPEC.md)**「命令结果：错误与成功」；**`data` 不出现 `http_status`**。各命令章节**只**写本命令 `--json` 时 **`data` 含哪些键**；可选 **`message`**（**i18n**）；成功体**不得**使用与错误体相同的 **`category` / `code`**。
- `--no-interactive`:
  - **行为**: 彻底禁用所有交互式 Prompt（如密码输入、确认提示、上下键选择等）。
  - **约束**: 如果缺少必要信息，立即报错并返回非零退出码。
- `--help, -h`:
  - **行为**: 显示当前命令或子命令的帮助信息。

根命令另外提供以下仅限根级使用的版本选项，不会由子命令继承：

- `crater --version` / `crater -v`:
  - **行为**：不要求登录、不使用已保存 token、不访问 Crater 平台。为选择显示语言，仍可能读取本地 `state.json` 的 `language` 字段。以单行输出当前 CLI 产品版本与 7 位短 commit SHA，格式为 `Crater CLI version <product-version>, build <short-commit>`。无法确定 commit 时使用 `unknown`。
  - **约束**：不接受位置参数，也不能与 `--json` 同时使用；脚本或 Agent 需要结构化构建信息时应使用 `crater version --json`。

CLI 发出的平台请求带 `User-Agent: crater-cli/<product-version>` 与 `X-Crater-API-Version: <api-version>`，仅供平台诊断，不代表后端会据此改变或拒绝请求。普通业务命令不自动执行 API 兼容性握手。

### 公共列表分页 (List Pagination)

采用公共列表分页契约的命令会显式提供以下选项；未提供这些选项的低基数列表不受本节影响：

- `--page` (int, default `1`): 当前页，从 `1` 开始。
- `--page-size` (int, default `15`): 每页数量。公共上限为 `200`；`crater download ls` 及兼容的 `model-download ls`/管理员列表上限为 `100`。
- `--all-pages` (bool): 从第一页开始返回全部筛选结果；此时 `--page` 不决定起始页。服务端分页命令在未显式提供 `--page-size` 时使用该端点允许的最大批量，显式提供的合法值仍优先。

共同语义：

- `--page` 必须大于等于 `1`；`--page-size` 必须在对应命令允许范围内。分页参数与状态、类型等领域筛选参数会在发请求前一起校验；存在多个问题时按「用法错误聚合」返回一次 `usage_error`。
- 服务端分页命令把分页和服务端支持的筛选参数传给 API；本地分页命令先取得端点返回的完整数组，再筛选、稳定排序（若命令另有排序约定）并分页。混合分页命令需要本地筛选时，会先顺序读取全部服务端候选页，再本地筛选并重新分页。
- 默认 JSON 成功体在 `data` 中同时包含资源数组与 `pagination: {"page": N, "page_size": N, "total": N}`；`total` 是筛选后的总数。`--all-pages` 返回完整数组并省略 `pagination`。
- 默认表格只展示当前页，并在末尾显示当前页码和筛选后的总数；`--all-pages` 不显示分页摘要。

### 错误处理规范 (Error Handling)

所有错误必须通过 `stderr` 输出，其格式受 `--json` 影响：

1. **默认模式**: 首行 `Error:`，正文为 `err.Error()`（`*clierror.Error` 即 `Message`）。正文**允许多行**；`internal/output` 对正文**按行**统一加两格基础缩进，行首若另有空格（如列表 `  -`）会与基础缩进**叠加**。不要求整段仅占一行。
2. **JSON 模式**: 输出格式化（缩进）的结构化 JSON 对象，便于人类阅读；`message` 内换行以 `\n` 转义保留在字符串中。
   - **Schema**:
     ```json
     {
       "category": "usage_error | api_error | system_error | cancelled",
       "code": "ERR_NOT_FOUND_404 | ERR_UNAUTHORIZED_401 | …（见 SPEC）",
       "message": "Human readable message",
       "context": { "key": "value" } 
     }
     ```
   - **错误码定义**: 以 `pkg/errorcodes/codes.go` 为准。**`api_error`** 的 **`code`** 通常须与 **HTTP** 显式对应，命名形如 **`ERR_NOT_FOUND_404`**、**`ERR_SERVER_INTERNAL_5XX`** 等；`crater compatibility` 对握手接口 404 的领域化映射见本命令章节。完整约定见 **[SPEC.md](./SPEC.md)**「命令结果：错误与成功」中 `api_error` 与 HTTP 小节。
   - **退出码**: 出错时非零退出；具体数值由 `Execute` 根据 `*clierror.Error` 的 `category` 映射（实现为 `pkg/errorcodes.ExitCodeForCategory`：`usage_error`→2，`cancelled`→3，`api_error`→4，`system_error`→5；非 `*clierror.Error` 的错误→1）。命令实现里不必自行 `os.Exit`。

---

## 本地 CLI 版本 (`version`)

### `crater version`

- **描述**：显示当前本地 CLI 二进制的产品版本、源码提交、构建信息、Go 运行时以及 API 兼容版本。不要求登录、不使用已保存 token，也不访问 Crater 平台；为选择显示语言，仍可能读取本地 `state.json` 的 `language` 字段。
- **位置参数**：无；出现任何位置参数均返回 `usage_error`。
- **选项**：仅使用全局选项。
- **默认输出**：在 `Crater CLI:` 标题下，按固定顺序显示产品版本、CLI API 版本、CLI 最低支持的后端 API 版本、Go 版本、完整 commit SHA、UTC 构建时间、`OS/ARCH` 和构建类型。未注入且无法可靠确定的值显示为 `unknown`；本地开发构建的产品版本默认为 `dev`、构建类型默认为 `development`。
- **`--json` 成功体的 `data`**：仅包含 `version`，其结构为：
  ```json
  {
    "version": {
      "product_version": "1.2.3",
      "commit_sha": "0123456789abcdef0123456789abcdef01234567",
      "build_type": "release",
      "build_time": "2026-09-07T08:30:00Z",
      "go_version": "go1.25.4",
      "os": "linux",
      "arch": "amd64",
      "api_version": 1,
      "min_supported_backend_api_version": 1
    }
  }
  ```
- **状态**：[x] Completed

---

## API 兼容性诊断 (compatibility)

### `crater compatibility`

- **描述**：显式检查当前 CLI 与指定 Crater 平台是否支持对方的 API 版本；请求无需登录凭据，也不会由其他命令自动执行，但仍必须能够确定目标平台地址。
- **位置参数**：无；出现任何位置参数均返回 `usage_error`。
- **选项**：
  - `--platform, -p <URL>`（条件必填）：待检查的平台地址；不提供时使用当前激活身份保存的平台地址。若没有当前激活身份，则必须显式提供该选项。
- **处理逻辑**：
  - 未提供 `--platform` 且没有当前激活身份时，不发起 HTTP 请求，返回 `usage_error` + `ERR_MISSING_REQUIRED_FLAG`，进程退出码为 `2`；提示用户通过 `--platform <URL>` 指定目标平台，并明确该操作无需登录。
  - 调用公开的 `GET /api/cli/compatibility` 一次，并比较 CLI / 后端各自的当前 API 版本与最低支持的对方版本。
  - 双方均满足最低版本时成功；默认模式输出平台、状态、双方 API 版本，以及后端产品版本、7 位短提交 SHA、构建类型和 UTC 构建时间。
  - 后端构建信息仅用于诊断和展示，不参与 API 兼容性判断。兼容性接口仅返回 API 版本字段的早期实现仍可使用；缺失的后端构建字段显示为“未知”，不会因此把响应判定为非法。
  - 明确不兼容时返回 `api_error` + `ERR_API_VERSION_MISMATCH`，进程退出码为 `4`。人类可读错误按“具体情况 / 版本信息 / 使用建议”分段展示：具体指出哪个最低版本条件不满足，列出 CLI 产品版本、CLI / 后端 API 版本和双方最低支持版本，并说明用户仍可继续使用 CLI，但部分操作可能因 API 契约不一致而失败。
  - 握手响应明确包含版本字段但值为 `0` 时，将 `0` 视为早于首个正式契约的旧版本并参与比较，而不是泛化为“兼容性信息无效”。例如后端明确报告 `apiVersion: 0` 且 CLI 最低支持后端版本为 `1` 时，按“后端版本过旧”输出上述完整诊断，并在最后的排查手段中建议降级 CLI。字段缺失或负数才视为非法响应。
  - 版本诊断只提示潜在风险，不把后续业务错误直接归因于 API 版本。实际操作失败时应先根据该操作自己的报错排查命令输入、本地配置、认证、网络连接和平台状态等更常见原因，也应考虑 CLI 自身（尤其开发版本）的缺陷。只有排除这些原因且仍有证据指向 API 契约不兼容时，才把调整 CLI 版本作为最后的排查手段。
  - 最后的 CLI 版本调整按检测方向决定：CLI API 版本低于后端最低要求时尝试升级 CLI；后端 API 版本低于 CLI 最低要求时尝试降级 CLI；两项限制同时不满足时分别解释升级和降级可解决的约束，并提示向平台管理员确认该部署对应的 CLI 版本。
  - 该接口返回 HTTP 404 时，视为目标后端早于当前 CLI 的兼容性检查命令：返回 `api_error` + `ERR_API_VERSION_MISMATCH`，后端版本显示为未知。排除更常见原因后，最后才建议尝试降级 CLI 或联系平台管理员。原始 `http_status: 404` 与响应 `msg` 保留在错误 context 中。此特殊映射只适用于本命令，不改变其他命令的 404 语义。
  - 其他 HTTP 错误、网络失败或响应非法时保留实际错误，不伪装成版本不兼容。
  - 该公开接口不读取或发送已保存 Token；当前未登录或没有激活身份时，可通过 `--platform` 指定目标平台后检查。
- **`--json` 成功体的 `data`**：仅包含 `compatibility`，其结构为：
  ```json
  {
    "platform_url": "https://crater.example.com",
    "status": "compatible",
    "cli": {
      "product_version": "1.0.0",
      "api_version": 1,
      "min_supported_backend_api_version": 1
    },
    "backend": {
      "product_version": "1.1.1",
      "short_commit_sha": "f42b0c2",
      "build_type": "release",
      "build_time": "2026-07-26T08:30:00Z",
      "api_version": 1,
      "min_supported_cli_api_version": 1
    }
  }
  ```
- **`--json` 不兼容错误的 `context`**：`platform_url`、`status`、`cli_product_version`、`cli_api_version`、`cli_min_supported_backend_api_version`、`backend_product_version`、`backend_short_commit_sha`、`backend_build_type`、`backend_build_time`、`backend_api_version`、`backend_min_supported_cli_api_version`。接口明确报告的 `0` 保留为数值 `0`，并按最低版本条件产生对应状态。缺失的构建信息使用空字符串表示未知，不参与兼容性判断。若握手接口返回 404，无法获知的后端 API 版本字段也以 `0` 占位、构建字段为空字符串，但 `status` 为 `unknown`，并额外包含 `reason: "compatibility_endpoint_not_found"`、`http_status: 404` 与原始 `msg`；因此调用方应结合 `status` / `reason` 区分“明确报告的旧版本 0”和“未知值占位 0”。
- **状态**：[x] Completed

---

## 1. 配置模块 (config)

### `crater config language`
- **描述**: 切换 CLI 的显示语言。
- **位置参数**:
  - `[LANG]` (positional, optional): 目标语言代码，如 `en` 或 `zh-CN`。
- **处理逻辑**:
  - **交互式切换 (默认)**: 如果未提供位置参数 `[LANG]`，且处于交互模式，则弹出列表供用户选择。
  - **直接设置**: 如果提供了有效的位置参数 `[LANG]`，则直接更新配置。
  - **非交互式约束**: 在 `--no-interactive` 模式下，必须提供位置参数 `[LANG]`，否则报错。
  - **验证**: 仅支持受支持的语言代码（目前为 `en`, `zh-CN`）。
- **预期行为**:
  - 更新 `state.json` 中的 `language` 字段。
  - 立即应用新语言展示成功提示。
- **`--json` 的 `data`**：`language`（字符串，目标语言代码）。
- **状态**: [ ] Pending

---

## 2. 补全模块 (completion)

本模块负责为不同 shell 提供 Tab 补全能力。主命令为 `completion`，并提供等价别名 `comp`：两者参数与行为完全一致。

当前实现状态：`bash` 与 `zsh` 已落地（含 `install/uninstall`）。**PowerShell（`pwsh`）下的 Tab 补全当前不在产品范围内**：CLI 不提供 `completion powershell` 子命令，亦不约定 `__complete powershell`。在 Windows 上若需要 Tab 补全，请使用 **Git Bash** 并按 **bash** 路径安装（`completion install bash`）。实现与工程约定见 [SPEC.md](./SPEC.md) 与 [ARCHITECTURE.md](./ARCHITECTURE.md)；**指令契约以本节为准**。

### `crater completion <SHELL>` / `crater comp <SHELL>`
- **描述**: 输出指定 shell 的补全脚本到 stdout（无副作用）。
- **位置参数**:
  - `<SHELL>` (positional, required): `bash | zsh`。
- **处理逻辑**:
  - 生成并输出该 shell 的补全脚本内容。
  - 不执行安装/卸载；不修改任何本地文件或 shell 配置。
- **输出约束**:
  - 默认模式：stdout 为脚本内容本身（可能包含多行），不得夹杂其他装饰性文字。
  - `--json`：stdout 输出成功信封 JSON；脚本内容放入 `data.script`（字符串）。
- **`--json` 的 `data`**：`shell`（字符串）、`script`（字符串）。
- **状态**: bash/zsh [x] Completed

### `crater completion install <SHELL>` / `crater comp install <SHELL>`
- **描述**: 安装/更新指定 shell 的补全脚本（有副作用，幂等）。
- **位置参数**:
  - `<SHELL>` (positional, required): `bash | zsh`。
- **选项**:
  - `--yes, -y` (bool): 跳过确认并直接执行安装/更新。
- **处理逻辑**:
  - **zsh/bash（当前实现）**：在用户的 `~/.zshrc` / `~/.bashrc` 中写入/更新一段带固定起止标记（marker）的内联补全块；不额外落盘 `~/.zsh/completions/_crater` 或 `~/.bash/completions/crater.bash` 等独立脚本文件。
  - 若 marker 块已存在，则替换块内脚本为当前版本（重复执行应安全、可重复）。
  - 交互模式下可展示将要写入/修改的位置并请求确认；`--no-interactive` 下必须提供 `--yes`，否则报错。
- **`--json` 的 `data`**：
  - `shell`（字符串）
  - `installed_paths`（字符串数组：当前为被修改的 rc 文件路径，例如 `~/.zshrc`）
  - `updated`（布尔：**marker 块在本次调用前已存在**，或 **本次为替换已有块**（非“首次追加块”）时为 `true`；首次向 rc 追加 marker 块时为 `false`）
  - `inserted_zshrc` / `inserted_bashrc`（布尔：是否通过“追加新块”完成安装；若为 `false` 通常表示替换了既有 marker 块）
- **状态**: bash/zsh [x] Completed

### `crater completion uninstall <SHELL>` / `crater comp uninstall <SHELL>`
- **描述**: 卸载指定 shell 的补全脚本（有副作用，幂等）。
- **位置参数**:
  - `<SHELL>` (positional, required): `bash | zsh`。
- **选项**:
  - `--yes, -y` (bool): 跳过确认并直接执行卸载。
- **处理逻辑**:
  - **zsh/bash（当前实现）**：从 `~/.zshrc` / `~/.bashrc` 中移除由 `install` 写入的 marker 块（只移除 crater 写入片段；若未找到块则无副作用）。
  - 交互模式下可展示将要删除/回滚的对象并请求确认；`--no-interactive` 下必须提供 `--yes`，否则报错。
- **`--json` 的 `data`**：`shell`（字符串）、`removed_paths`（字符串数组：**实际发生写回修改**的 rc 文件路径；不是“删除整个文件”的语义）。
- **状态**: bash/zsh [x] Completed

---

## 3. 认证模块 (auth)

需要访问平台的命令会读取当前 `active_context` 对应 `auth_infos` 条目中的 `token`。`auth ls` / `switch` / `rm` / `logout` 不读取 token。已有激活身份但该条目没有 `token`（例如从旧 Keyring 升级后尚未重新登录）时，访问平台的命令失败并提示重新 `auth login`。

### `crater auth login`
- **描述**: 登录到一个 Crater platform 实例并获取 Token。
- **选项**:
  - `--platform, -p` (string, required): 平台基础 URL。
  - `--mode, -m` (string): 认证模式，可选 `ldap` 或 `normal`，默认为 `ldap`。
  - `--username, -u` (string): 用户名（不填则进入交互模式）。
  - `--password` (string): 密码（仅限非交互模式或脚本使用，不推荐在普通 shell 中直接输入）。
- **处理逻辑**:
  - **联合键 (Composite Key)**: 使用 `(PlatformURL, Username, Mode)` 三元组作为唯一标识符。
  - **覆盖与追加 (Upsert)**: 
    - 如果三元组完全一致，则视为同一个认证环境，更新其 Token、UID、昵称等元数据。
    - 如果三元组中任一项不同，则视为新的认证环境并追加到配置中。
  - **非交互式约束**: 在开启 `--no-interactive` 或 `--json` 时，如果未提供 `--password`，程序将直接报错而非使用空密码。
- **预期行为**:
  - 调用 `/api/auth/login` 接口。
  - 成功后将 access token 明文写入 `state.json` 中对应 `auth_infos` 条目的 `token` 字段（与身份摘要同一文件，权限 `0600`）。
  - 更新 `auth_infos` 列表，并自动将该环境设为 `active_context`。
- **`--json` 的 `data`**：`user`（身份摘要对象，字段与 `auth_infos` 条目一致，但**不含** `token`）。
- **状态**: [x] Completed

### `crater auth switch`
- **描述**: 切换当前激活的认证上下文（平台与身份）。
- **选项**:
  - `--platform, -p` (string): 目标平台 URL。
  - `--username, -u` (string): 目标用户名。
  - `--mode, -m` (string): 目标认证方式。
- **处理逻辑**:
  - **选项自由度**: 支持提供任意数量 (0-3) 个过滤用选项。
  - **自动推断与智能切换**:
    - 程序根据提供的选项在已保存的 `auth_infos` 中进行筛选。
    - **排除当前**: 筛选逻辑会优先排除当前已经处于激活状态 (`active`) 的上下文。
    - **快速切换**: 如果筛选并排除后，剩余候选项仅剩 **一个**，则无需确认直接切换。
  - **多项冲突处理 (Selection Logic)**:
    - 如果筛选后候选项仍有 **多个**:
      - **交互模式 (默认)**: 在终端展示候选列表，允许用户通过上下方向键（或输入序号）交互式选择目标。
      - **非交互模式 (`--no-interactive`)**: 抛出错误，并列出所有匹配的候选项，要求用户提供更精确的选项组合。
  - **独立性**: 该命令**仅**负责切换认证环境，不会改变全局的视图模式（View Mode）。
- **预期行为**:
  - 更新 `state.json` 中的 `active_context` 字段。
- **`--json` 的 `data`**：`active`（对象，与 `state.json` 中 `active_context` 同形：`platform_url`、`username`、`method`）。
- **状态**: [x] Completed

### `crater auth ls`
- **描述**: 列出所有已保存的认证上下文。
- **选项**:
  - `--platform, -p` (string): 按平台 URL 过滤。
  - `--username, -u` (string): 按用户名过滤。
  - `--mode, -m` (string): 按认证方式过滤。
- **预期行为**:
  - 读取 `state.json` 中的 `auth_infos`，并根据选项进行筛选。
  - 在控制台以表格形式展示匹配的登录信息。
  - 标记出当前激活的 (`active`) 上下文。
- **输出格式**:
  - 表格形式显示: `ACTIVE`, `PLATFORM`, `USERNAME`, `METHOD`, `PRIVILEGE` (该身份在平台的权限级别)。
- **`--json` 的 `data`**：`active_context`（对象）、`auth_infos`（数组，筛选后的条目；字段与磁盘 `auth_infos` 相同，但**不含** `token`）。
- **状态**: [x] Completed

### `crater auth rm`
- **描述**: 删除指定的认证上下文。
- **选项**:
  - `--platform, -p` (string): 过滤待删除的平台。
  - `--username, -u` (string): 过滤待删除的用户。
  - `--mode, -m` (string): 过滤待删除的认证方式。
  - `--yes, -y` (bool): 强制删除，跳过交互式确认。
- **处理逻辑**:
  - **筛选机制**: 筛选出所有匹配给定选项的上下文。
  - **安全确认**: 
    - **交互模式**: 在终端列出所有匹配项，并要求用户确认是否删除。
    - **非交互模式 (`--no-interactive`)**: 必须配合 `-y` 选项，否则报错并拒绝执行。
  - **清理逻辑**: 
    - 从 `state.json` 的 `auth_infos` 中移除对应条目（其中的 `token` 一并删除）。
    - 如果删除的是当前 `active` 的上下文，则将 `active_context` 置为空。
- **`--json` 的 `data`**：`removed_count`（整数）。
- **状态**: [x] Completed

### `crater auth logout`
- **描述**: 登出并注销当前激活的认证上下文。
- **选项**:
  - `--yes, -y` (bool): 强制登出，跳过交互式确认。
- **处理逻辑**:
  - **对象锁定**: 仅针对当前 `active_context` 进行操作。
  - **安全确认**: 
    - **交互模式**: 确认是否登出当前用户。
    - **非交互模式 (`--no-interactive`)**: 必须配合 `-y` 选项，否则报错。
  - **清理逻辑**: 
    - 从 `state.json` 的 `auth_infos` 列表中移除当前激活项（其中的 `token` 一并删除）。
  - **后续行为 (Auto-Switch)**:
    - 如果列表中仍有其他已保存的认证上下文，则**自动切换**到列表中的第一项作为新的 `active_context`。
    - 如果列表为空，则清空 `active_context`。
- **`--json` 的 `data`**：`next_active`（对象，与 `active_context` 同形；登出后若已无激活项则为各字段空字符串的同一结构）。
- **状态**: [x] Completed

---

## 4. 下载模块 (download)

本模块负责通过 CLI 向 Crater 平台提交模型和数据集下载任务。下载行为在平台侧执行：CLI 只负责读取当前激活账号、提交请求、展示任务信息；不会把模型或数据集直接下载到本机。

### `crater download create`
- **描述**: 创建模型或数据集下载任务。
- **选项**:
  - `--name` (string, required): 资源名称，格式为 `owner/name`，例如 `qwen/Qwen2.5-Coder-7B-Instruct`。
  - `--category` (string, required): 下载类别，可选 `model` 或 `dataset`。
  - `--source` (string): 下载来源，可选 `modelscope` / `ms` 或 `huggingface` / `hf`，默认为 `modelscope`；`ms` 与 `hf` 仅为 CLI 简写，发送给平台前会规范化为全拼。
  - `--revision` (string): 可选的分支、tag 或 revision。
  - `--token` (string): 可选的访问令牌，用于 gated/private 仓库。该值仅随本次请求发送给平台，不写入 CLI 本地配置，不在成功/错误输出中展示。
  - `--token-env` (string): 从指定环境变量读取可选访问令牌。与 `--token`、`--token-stdin` 互斥。
  - `--token-stdin` (bool): 从 stdin 读取可选访问令牌。与 `--token`、`--token-env` 互斥。
  - `--wait` (bool): 提交后轮询任务，直到状态进入 `Ready`、`Failed` 或 `Paused`。
  - `--poll-interval` (duration): `--wait` 的轮询间隔，默认 `5s`。
  - `--timeout` (duration): `--wait` 的最长等待时间，默认 `0` 表示不超时。
- **处理逻辑**:
  - 本地校验 `--name`、`--category`、`--source`；可在请求前发现的问题必须聚合为单个 `usage_error`。
  - 若提供 `--token-env` 或 `--token-stdin`，CLI 读取 token 后仅用于本次请求；不得在输出中展示 token。
  - 读取当前激活身份的 token；若未登录或该身份没有保存 token，返回错误（见认证模块）。
  - 调用平台接口创建下载任务。后端根据 `category` 自动选择平台侧目标目录（模型为 `public/Models`，数据集为 `public/Datasets`）。
  - `name + category` 标识平台中的唯一公共资源。若已有 Ready、等待中、下载中或暂停中的记录，后端会复用该记录，即使请求的来源或版本不同；CLI 使用后端返回的实际任务信息，默认输出实际来源、状态与路径，JSON 输出还包含实际版本。
  - `--source` 和 `--revision` 只在创建新任务或重新发起失败下载时决定下载内容。仓库不存在、版本错误或无权访问等源站错误由下载 Job 异步发现，任务随后进入 Failed；可查看任务日志并在其他来源存在同名资源时重新提交。
- **输出格式**:
  - 默认模式：stdout 展示下载任务的 ID、名称、类别、来源、状态与目标路径。
  - `--json`：stdout 输出成功信封 JSON。
- **`--json` 的 `data`**：
  - `download`（对象，保留后端下载任务摘要，包括基础状态/进度、请求者与当前用户关系、操作权限，以及来源元数据字段）
- **状态**: [x] Completed

### `crater download model <NAME>` / `crater download dataset <NAME>`
- **描述**: 创建模型或数据集下载任务的快捷形式。
- **位置参数**:
  - `<NAME>` (positional, required): 资源名称，格式为 `owner/name`。
- **选项**:
  - 与 `download create` 相同，但不需要 `--name` 与 `--category`；类别由子命令固定为 `model` 或 `dataset`。
- **`--json` 的 `data`**：同 `download create`。
- **状态**: [x] Completed

### `crater download ls`
- **描述**: 列出当前用户的下载任务。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--category` (string): 可选过滤类别，`model` 或 `dataset`。
  - `--status` (string): 服务端按任务状态过滤：`Pending | Downloading | Paused | Ready | Failed`。
  - `--search` (string): 服务端按下载任务名称搜索。
  - `--page` (int, default `1`): 服务端页码，必须大于等于 `1`。
  - `--page-size` (int, default `15`): 服务端每页数量，范围 `1..100`。
  - `--all-pages` (bool): 从第一页顺序获取全部服务端分页。
- **处理逻辑**:
  - 调用 `/api/v1/model-download/models/downloads`，分页和筛选均由服务端执行。
  - `--all-pages` 保留第一页响应中的状态汇总，并合并各页任务。
- **输出格式**:
  - 默认模式：表格展示 `ID`、`NAME`、`CATEGORY`、`SOURCE`、`STATUS`、`PATH`。
  - `--json`：stdout 输出成功信封 JSON。
- **`--json` 的 `data`**：`downloads`（当前页或完整下载任务数组）、`summary`（后端状态汇总）；非 `--all-pages` 时还包含 `pagination`。
- **状态**: [x] Completed

### `crater download get <ID>`
- **描述**: 查看单个下载任务详情。
- **位置参数**:
  - `<ID>` (positional, required): 下载任务 ID。
- **`--json` 的 `data`**：`download`（下载任务对象）。
- **状态**: [x] Completed

### `crater download logs <ID>`
- **描述**: 查看下载任务日志。
- **位置参数**:
  - `<ID>` (positional, required): 下载任务 ID。
- **选项**:
  - `--follow` (bool): 持续轮询日志，直到任务状态进入 `Ready`、`Failed` 或 `Paused`。`--follow` 不支持与 `--json` 同时使用。
  - `--poll-interval` (duration): `--follow` 的轮询间隔，默认 `5s`。
- **输出格式**:
  - 默认模式：stdout 输出日志文本本身。
  - `--json`：stdout 输出成功信封 JSON。
- **`--json` 的 `data`**：`logs`（字符串）。
- **状态**: [x] Completed

### `crater download pause <ID>` / `resume <ID>` / `retry <ID>`
- **描述**: 暂停、恢复或重试下载任务。
- **位置参数**:
  - `<ID>` (positional, required): 下载任务 ID。
- **`--json` 的 `data`**：`download`（操作后的下载任务对象）。
- **状态**: [x] Completed

### `crater download rm <ID>`
- **描述**: 移除下载任务。该操作会删除当前用户与下载记录的关联；后端会在无人引用时软删除下载记录并保留已下载文件。
- **位置参数**:
  - `<ID>` (positional, required): 下载任务 ID。
- **选项**:
  - `--yes, -y` (bool): 跳过确认。
- **处理逻辑**:
  - 交互模式下需要确认；`--no-interactive` 下必须提供 `--yes`。
- **`--json` 的 `data`**：`id`（下载任务 ID）、`message`（后端返回消息）。
- **状态**: [x] Completed

---

## 5. 节点模块 (node)

本模块提供集群节点信息的只读查询能力。所有命令均要求已有 active credentials。

### `crater node ls`
- **描述**: 列出当前平台可见的集群节点。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--name` (string): 按节点名称子串本地过滤。
  - `--status` (string): 按节点状态本地过滤，例如 `Ready`、`NotReady`、`Unschedulable`、`Occupied`。
  - `--arch` (string): 按 CPU 架构本地过滤，例如 `amd64`、`arm64`。
  - `--gpu` (string): 按 GPU 资源名或型号关键词本地过滤，例如 `a100`、`v100`、`nvidia.com/a100`。
  - `--gpu-available` (bool): 仅显示存在匹配空闲 GPU 的节点；必须与 `--gpu` 一起使用。
- **处理逻辑**:
  - 调用 `/api/v1/nodes`。
  - 所有过滤均在 CLI 本地完成，不改变平台状态。
  - 默认模式以表格展示节点名称、状态、角色、架构、地址、CPU/内存使用和作业数。
- **`--json` 的 `data`**：`nodes`（数组，元素与平台节点摘要响应一致）。
- **状态**: [x] Completed

### `crater node get <name>`
- **描述**: 查看单个节点详情。
- **位置参数**:
  - `<name>` (positional, required): 节点名称。
- **处理逻辑**:
  - 调用 `/api/v1/nodes/{name}`。
  - 缺少 `<name>` 时返回 `usage_error`。
- **`--json` 的 `data`**：`node`（对象，平台节点详情响应）。
- **状态**: [x] Completed

### `crater node pods <name>`
- **描述**: 查看指定节点上的 Pod。
- **位置参数**:
  - `<name>` (positional, required): 节点名称。
- **选项**:
  - `--namespace` (string): 仅显示指定命名空间中的 Pod；未使用 `--all-namespaces` 时必填。
  - `--all-namespaces` (bool): 显示所有命名空间；与 `--namespace` 互斥。
  - `--status` (string): 按 Pod 阶段过滤：`Pending | Running | Succeeded | Failed | Unknown`。
  - `--type` (string): 按控制器类型过滤：`batch.volcano.sh/v1alpha1/Job | aisystem.github.com/v1alpha1/AIJob`。
  - `--search` (string): 按 Pod 名称子串过滤。
  - `--page` (int, default `1`) / `--page-size` (int, default `15`, max `200`): 对过滤后的列表分页。
  - `--all-pages` (bool): 不截断，返回全部过滤结果。
- **处理逻辑**:
  - 调用 `/api/v1/nodes/{name}/pods`。
  - 平台当前没有向普通用户暴露作业命名空间配置，因此 CLI 不猜测固定默认值；必须显式选择 `--namespace` 或 `--all-namespaces`。
  - CLI 会从 owner reference 补齐缺失的控制器类型，先在本地执行命名空间、状态、类型和名称过滤，再按 Pod 名称、命名空间稳定排序并分页。
  - 默认模式以表格展示 Pod 名称、命名空间、IP、状态、类型和资源。
- **`--json` 的 `data`**：`pods`（当前页数组）和 `pagination`（`page`、`page_size`、过滤后的 `total`）；`--all-pages` 时省略 `pagination`。
- **状态**: [x] Completed

### `crater node gpu <name>`
- **描述**: 查看指定节点的 GPU 信息。
- **位置参数**:
  - `<name>` (positional, required): 节点名称。
- **处理逻辑**:
  - 调用 `/api/v1/nodes/{name}/gpu`。
  - 默认模式展示节点 GPU 总量和设备摘要。
- **`--json` 的 `data`**：`gpu`（对象，平台节点 GPU 响应）。
- **状态**: [x] Completed

---

## 6. 作业模块 (job)

本模块覆盖前端作业页面的通用 Volcano 作业能力：列表、详情、Pods、日志、事件、YAML、模板、Jupyter/WebIDE 访问凭据、SSH、快照、告警、删除/停止、基础创建，以及管理员锁定/保留/清理操作。所有访问平台的命令都使用当前 `auth` active context 的 token。管理员能力统一放在 `crater admin job ...` 下，不使用普通命令加 `--admin`。

### `crater job ls`
- **描述**: 列出当前账号可见的作业。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--all` (bool): 调用 `/api/v1/vcjobs/all`，列出当前身份可见且位于 `--days` 回看窗口内的作业。
  - `--user` (string): 调用 `/api/v1/vcjobs/user/{username}`，列出指定用户且位于 `--days` 回看窗口内的作业。
  - `--days` (int): 覆盖当前路由的回溯天数；`-1` 表示不按时间过滤。小于 `-1` 的值返回 `usage_error`。不指定时，默认自视图不限制时间，`--all`/管理员视图回看 7 天，`--user` 回看 30 天。
  - `--search` (string): 服务端按作业名称、所有者或账户搜索，最多 128 个 Unicode 字符。
  - `--status` (string slice): 服务端过滤作业状态，可重复或逗号分隔，最多 20 项。
  - `--type` (string slice): 服务端过滤作业类型，可重复或逗号分隔，最多 20 项；类型为 `jupyter | webide | custom | pytorch | tensorflow | kuberay | deepspeed | openmpi`。
  - `--schedule` (string slice): 服务端过滤调度类型，可重复或逗号分隔，值为 `normal | backfill`，最多 20 项。
  - `--node` (string): 服务端过滤运行在指定节点上的作业。
  - `--owner` (string): 本地按用户名或作业响应中的 owner 精确筛选。
  - `--from` / `--to` (string): 本地按 `createdAt` 时间范围筛选，支持 RFC3339 或 `YYYY-MM-DD`。
  - `--interactive` (bool): 服务端只返回交互式作业（`jupyter` / `webide`）。
  - `--batch` (bool): 服务端只返回非交互式作业。
  - `--page` (int, default `1`) / `--page-size` (int, default `15`, max `200`): 请求指定服务端分页。
  - `--sort` (string): 最多 3 个逗号分隔的服务端排序字段，字段前加 `-` 表示降序；支持 `name | jobName | owner | queue | jobType | scheduleType | status | billedPointsTotal | createdAt | startedAt | completedAt`，不允许重复字段。
  - `--all-pages` (bool): 顺序请求全部服务端分页。
- **处理逻辑**:
  - 默认调用 `/api/v1/vcjobs`，列出当前用户和当前账户下的作业。
  - `--user` 优先于 `--all`；`--days` 可覆盖任一列表路由的默认回看窗口。
  - `--interactive` 与 `--batch` 互斥。
  - 未使用 `--owner`、`--from`、`--to` 时保留服务端页和 `total`；使用任一本地筛选时，CLI 以每批最多 `200` 条读取全部服务端候选页，完成本地筛选后按用户请求的 `--page-size` 重新分页。
  - 默认模式以表格展示名称、平台作业名、类型、状态、队列、节点和资源。
- **`--json` 的 `data`**：`jobs`（当前页数组）和 `pagination`。需要本地过滤时，CLI 先以较大批次取回候选项、完成过滤，再对过滤结果分页；只有 `--all-pages` 会返回完整过滤结果并省略 `pagination`。
- **状态**: [x] Completed

### `crater job get|pods|events|yaml|template <name>`
- **描述**:
  - `get`: 查看单个作业详情。
  - `pods`: 查看指定作业的 Pod 列表。
  - `events`: 查看指定作业的 Kubernetes 事件。
  - `yaml`: 查看指定作业的 YAML。
  - `template`: 输出作业模板内容。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名，对应前端详情页路径中的作业名。
- **`pods` 选项**:
  - `--status` (string): 本地按 Pod 阶段过滤：`Pending | Running | Succeeded | Failed | Unknown`。
  - `--search` (string): 本地按 Pod 名称子串过滤。
  - `--page` (int, default `1`) / `--page-size` (int, default `15`, max `200`): 对筛选后的 Pod 分页。
  - `--all-pages` (bool): 返回指定作业的全部筛选后 Pod。
- **处理逻辑**:
  - `get` 调用 `/api/v1/vcjobs/{name}/detail`。
  - `pods` 调用 `/api/v1/vcjobs/{name}/pods`，按名称稳定排序后本地分页。
  - `events` 调用 `/api/v1/vcjobs/{name}/event`。
  - `yaml` 调用 `/api/v1/vcjobs/{name}/yaml`，默认模式直接输出 YAML 字符串到 stdout。
  - `template` 调用 `/api/v1/vcjobs/{name}/template`。
  - 这些作业专用端点按平台作业名定位资源，不接受或伪造默认 namespace。`get` 的 `namespace`、`pods` 中每个 Pod 的 `namespace`，以及事件/YAML 中的 namespace 信息均保留后端真实值。
  - 缺少 `<name>` 时返回 `usage_error`。
  - `get` 的人类可读详情显式展示后端返回的 namespace；`pods` 默认模式以表格展示 Pod 名称、命名空间、节点、IP、阶段和资源。
- **`--json` 的 `data`**：
  - `get`: `job`
  - `pods`: `pods`；非 `--all-pages` 时还包含 `pagination`
  - `events`: `events`
  - `yaml`: `yaml`
  - `template`: `template`
- **状态**: [x] Completed

### `crater job logs <name>`
- **描述**: 根据平台作业名自动解析 Pod 和容器并输出日志，不需要手工执行 `job pods`、`pod containers`、`pod logs` 三条命令。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名。
- **选项**:
  - `--pod` (string): 指定一个属于该作业的 Pod。
  - `--all-pods` (bool): 输出作业全部 Pod 的日志；与 `--pod` 互斥。
  - `--container` / `-c` (string): 指定容器。
  - `--all-containers` (bool): 输出所选 Pod 中全部容器的日志；与 `--container` 互斥。
  - `--tail` (int64, default `0`): 最近日志行数；`0` 表示全部，负数返回 `usage_error`。
  - `--timestamps` (bool): 包含 Kubernetes 日志时间戳。
  - `--previous` / `-p` (bool): 获取上一个已终止容器实例的日志。
  - `--follow` / `-f` (bool): 实时跟随日志，仅支持一个 Pod 和一个容器；不能与 `--previous` 或 `--json` 同时使用。
  - `--prefix` (bool): 仅为文本输出的每行添加 `[pod/container]` 前缀。选择多个日志来源时自动启用前缀；JSON 已提供独立的 `pod` 和 `container` 字段，不受该选项影响。
- **选择逻辑**:
  - 单 Pod、单普通容器会自动选择；默认忽略 init container。
  - Pod 仅包含 init container 时不会自动回退；必须使用 `--container` 或 `--all-containers` 显式选择。
  - 多 Pod 必须使用 `--pod` 或 `--all-pods`。
  - Pod 中存在多个普通容器时必须使用 `--container` 或 `--all-containers`。
  - 所有候选项和多来源输出均按 Pod、容器名称稳定排序。
- **`--json` 的 `data`**：`logs`（数组）；每项固定包含 `namespace`、`pod`、`container`、`content`。`--prefix` 仅影响文本输出，不修改 JSON 内容。
- **状态**: [x] Completed

### `crater job token|secret|ssh|snapshot|alert|delete <name>`
- **描述**:
  - `token`: 获取运行中 Jupyter 作业的 URL 与 token。
  - `secret`: 获取运行中 WebIDE 作业的 URL 与密码。
  - `ssh`: 为运行中作业开启 SSH，并输出 `host:port`。
  - `snapshot`: 为 Jupyter 或 Custom 作业创建镜像快照。
  - `alert`: 切换作业告警状态。
  - `delete`: 停止或删除自己的作业。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名。
- **`delete` 选项与确认**:
  - `--yes` / `-y` (bool): 跳过删除确认。
  - 交互模式默认二次确认；`--json` 或 `--no-interactive` 下必须显式提供 `--yes`，否则返回 `usage_error`。
- **`--json` 的 `data`**：
  - `token` / `secret`: `token`
  - `ssh`: `ssh`
  - `snapshot` / `alert` / `delete`: `message`
- **状态**: [x] Completed

### `crater job create jupyter|webide`
- **描述**: 创建交互式作业。支持 flags 构造常用请求，也支持 `--file` 传入完整 JSON 请求体。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--file` (string): 从文件读取完整 JSON 请求体；提供后忽略其它创建 flags。文件必须只包含一个 JSON 对象，未知字段会在发起请求前被拒绝。
  - `--name` (string, required without `--file`): 显示名称。
  - `--image` (string, required without `--file`): 镜像地址。
  - `--arch` (stringSlice): 镜像架构。
  - `--cpu` (float, default `1`): CPU 请求量，必须大于等于 0。
  - `--memory` (string, required without `--file`): 内存请求量，不能为负数。
  - `--gpu` (int, default `0`): GPU 数量，必须大于等于 0。
  - `--gpu-resource` (string, default `nvidia.com/gpu`): GPU 资源名；`--gpu > 0` 时必填。
  - `--schedule` (string): `normal | backfill`。
  - `--env` (stringArray): `KEY=VALUE`，可重复。
  - `--volume` (stringArray): `subPath:mountPath`，可重复；转换为后端 `volumeMounts` 的工作区类型（`type=1`）。
  - `--dataset` (stringArray): `datasetID:mountPath`，可重复；转换为后端实际消费的 `volumeMounts` 数据集类型（`type=2`），`datasetID` 必须大于 0。
  - `--selector` (stringArray): `key=Operator:value1,value2`，可重复。
  - `--forward` (stringArray): `name:port`，可重复；名称使用 1–20 个小写字母，端口范围为 1–65535。
  - `--template`、`--alert`、`--cpu-pinning`。
- **本地校验**: CPU、Memory、GPU 不允许负数；缺少 `name`、`image`、`memory` 会在发起请求前聚合报错。
- **`--json` 的 `data`**：`job`（后端返回的 Volcano Job 对象）。
- **状态**: [x] Completed

### `crater job create custom`
- **描述**: 创建单机自定义训练作业。支持 flags 构造常用请求，也支持 `--file` 传入完整 JSON 请求体。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **额外选项**:
  - `--working-dir` (string, default `/workspace`): 工作目录。
  - `--command` (string): 容器中执行的命令。
  - `--shell` (string, default `sh`): `--command` 使用的 shell。
- **其余选项与校验**: 同 `jupyter|webide`。
- **`--json` 的 `data`**：`job`。
- **状态**: [x] Completed

### `crater job create tensorflow|pytorch --file <json>`
- **描述**: 创建 TensorFlow 或 PyTorch 分布式作业。由于后端 `tasks[]` 结构较复杂，CLI 要求通过 `--file` 传入与前端/后端 DTO 对齐的完整 JSON 请求体。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **本地校验**:
  - `name` 必填。
  - `tasks` 至少一个。
  - 每个 task 的 `name`、`image.imageLink` 必填。
  - 每个 task 的 `replicas` 必须大于 0。
  - 每个 task 的资源值不能为负数。
  - 请求文件只允许后端 DTO 中存在的字段；TensorFlow / PyTorch 不允许 `scheduleType=0`（backfill）。
- **`--json` 的 `data`**：`job`。
- **状态**: [x] Completed

### `crater admin job ls`
- **描述**: 使用 `/api/v1/admin/vcjobs` 或 `/api/v1/admin/vcjobs/user/{username}` 列出管理员可见作业。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--user` (string): 列出指定用户作业。
  - `--days` (int): 回看天数；`-1` 表示全部，默认 `0`。
  - `--search`、`--status`、`--type`、`--node`、`--owner`、`--from`、`--to`、`--interactive`、`--batch`、`--sort`: 与 `crater job ls` 相同；其中 `owner/from/to` 为本地筛选，其余支持的筛选和排序传给管理员列表 API。
  - `--page` (int, default `1`) / `--page-size` (int, default `15`, max `200`) / `--all-pages`: 与 `crater job ls` 使用相同的服务端/本地混合分页语义。
- **`--json` 的 `data`**：`jobs`（当前页数组）和 `pagination`；`--all-pages` 时省略 `pagination`。
- **状态**: [x] Completed

### `crater admin job delete <name>`
- **描述**: 使用 `/api/v1/admin/vcjobs/{name}` 管理员删除作业。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名。
- **选项与确认**:
  - `--yes` / `-y` (bool): 跳过删除确认。
  - 交互模式默认二次确认；`--json` 或 `--no-interactive` 下必须显式提供 `--yes`。
- **`--json` 的 `data`**：`message`。
- **状态**: [x] Completed

### `crater admin job lock|unlock|keep <name>`
- **描述**:
  - `lock`: 设置清理锁定时长或永久锁定。
  - `unlock`: 清除清理锁定。
  - `keep`: 切换低 GPU 利用率清理时的保留状态。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名。
- **`lock` 选项**:
  - `--permanent` (bool): 永久锁定。
  - `--days` / `--hours` / `--minutes` (int): 锁定时长；非永久锁定时至少一个大于 0。
- **`--json` 的 `data`**：`message`。
- **状态**: [x] Completed

### `crater admin job clean ...`
- **描述**: 管理员作业清理操作，对应前端 admin jobs 的清理接口。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **子命令**:
  - `waiting-jupyter --wait-minutes N`: 取消等待超过阈值的 Jupyter 作业；`N` 必须大于 0。
  - `waiting-custom --wait-minutes N`: 取消等待超过阈值的 Custom 作业；`N` 必须大于 0。
  - `long-running --batch-days N --interactive-days N`: 清理长时间运行作业；两个天数阈值都必须大于 0，避免后端把缺失阈值按 0 天处理。
  - `low-gpu --time-range N --wait-time N [--util N]`: 清理低 GPU 利用率作业；时间单位均为分钟，`time-range` 与 `wait-time` 必须大于 0，`util` 必须在 0–100 之间。
- **确认**:
  - 每个清理子命令都支持 `--yes` / `-y` 跳过确认。
  - 交互模式默认二次确认；`--json` 或 `--no-interactive` 下必须显式提供 `--yes`。
- **`--json` 的 `data`**：`cleanup`（含 `reminded` 与 `deleted`）。
- **状态**: [x] Completed

---

## 7. 镜像模块 (image)

本模块提供容器镜像、镜像构建、分享、CUDA base image 和 Harbor 项目管理能力。所有命令均要求已有 active credentials。用户可操作资源使用 `crater image ...`；管理员/平台级资源统一使用 `crater admin image ...`，不得使用 `--admin` 切换。

### `crater image ls` / `crater admin image ls`
- **描述**:
  - `crater image ls`: 列出当前账号可见的镜像。
  - `crater admin image ls`: 列出管理员可见的全部镜像。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--available` (bool, user only): 调用 `/api/v1/images/available`，列出创建作业时可选择的镜像。
  - `--type` (string, user only): 本地过滤镜像适用的作业类型。
  - `--arch` (string): 本地过滤镜像架构，例如 `linux/amd64`。
  - `--visibility` (string): 本地过滤镜像可见性：`Public | Private | UserShare | AccountShare`。
  - `--owner` (string): 按所有者用户名或昵称子串本地过滤。
  - `--search` (string): 按镜像地址或描述子串本地过滤。
  - `--page` (int, default `1`) / `--page-size` (int, default `15`, max `200`) / `--all-pages`: 使用公共本地分页契约。
- **处理逻辑**:
  - 用户命令默认调用 `/api/v1/images/image`，`--available` 改用 `/api/v1/images/available`；管理员命令调用管理员镜像列表接口。
  - 所有过滤均在 CLI 本地完成；过滤后保持接口返回顺序并分页，不改变平台状态。
  - 默认模式以表格展示 ID、镜像地址、类型、可见性、架构和所有者。
- **`--json` 的 `data`**：`images`（当前页或完整数组）；非 `--all-pages` 时还包含 `pagination`。
- **状态**: [x] Completed

### Image Build Commands
- `crater image build ls [--page N] [--page-size N] [--all-pages]`: `/api/v1/images/kaniko`
- `crater image build get <name>`: `/api/v1/images/getbyname?name=...`
- `crater image build template <name>`: `/api/v1/images/template?name=...`
- `crater image build pod <id>`: `/api/v1/images/podname?id=...`
- `crater image build pip-apt --name NAME --tag TAG --image BASE [--packages TEXT] [--requirements TEXT]`
- `crater image build dockerfile --name NAME --tag TAG (--dockerfile TEXT | --file PATH)`
- `crater image build envd --name NAME --tag TAG (--envd TEXT | --file PATH) [--build-source EnvdAdvanced|EnvdRaw]`
- `crater image build remove --ids 1,2`
- Admin variants:
  - `crater admin image build-ls [--page N] [--page-size N] [--all-pages]`
  - `crater admin image build-remove --ids 1,2`
- 两个构建列表保持接口返回顺序后在本地分页；`page-size` 默认 `15`、最大 `200`。
- JSON payload keys: `builds`（列表非 `--all-pages` 时同时有 `pagination`）、`build`、`template`、`pod`、`message`.

### Image Record Commands
- `crater image upload --image IMAGE [--type jupyter|webide|custom|pytorch|tensorflow]`
- `crater image delete <id>`
- `crater image delete-many --ids 1,2`
- `crater image description <id> --description TEXT`
- `crater image type <id> --type jupyter|webide|custom|pytorch|tensorflow`
- `crater image tags <id> --tags a,b`
- `crater image arch <id> --archs linux/amd64,linux/arm64`
- `crater image valid --links image-a,image-b`
- Admin variants:
  - `crater admin image ls`（分页和筛选契约见上文）
  - `crater admin image delete-many --ids 1,2`
  - `crater admin image description <id> --description TEXT`
  - `crater admin image type <id> --type jupyter|webide|custom|pytorch|tensorflow`
  - `crater admin image tags <id> --tags a,b`
  - `crater admin image arch <id> --archs linux/amd64`
  - `crater admin image public <id>`
- `type=all` is accepted only as a local list filter, not as a writable image task type.
- JSON payload keys: `images`, `message`, `invalid_pairs`.

### Image Share, CUDA, Harbor, And Quota Commands
- `crater image share ls <image-id>`: `/api/v1/images/share?imageID=...`
- `crater image share users <image-id> [--name NAME]`: `/api/v1/images/user`
- `crater image share accounts <image-id>`: `/api/v1/images/account`
- `crater image share add <image-id> --share-type user|account --ids 1,2`
- `crater image share remove <image-id> --share-type user|account --target-id ID`
- `crater image cuda ls`
- `crater admin image cuda add --image-label LABEL --label TEXT --value IMAGE`
- `crater admin image cuda delete <id>`
- `crater image harbor info`
- `crater image harbor credential --yes`
- `crater image quota get|set --size BYTES`
- Harbor credential output contains sensitive data and requires explicit `--yes` in every mode.
- JSON payload keys: `grants`, `users`, `accounts`, `cuda_base_images`, `harbor`, `credential`, `quota`, `message`.

---

## 7. Additional Read Modules

This section records the read-only API surface covered by the CLI after the broader read-interface audit. All commands require active credentials. User-visible reads stay under their resource domain. Administrator-only reads are explicitly under `crater admin ...` and require platform administrator permissions.

### Account And Queue Reads
- `crater account ls`: `/api/v1/accounts`.
- `crater account get <name>`: `/api/v1/accounts/by-name/{name}`.
- `crater account members <id>`: `/api/v1/accounts/{id}/users`.
- `crater account users-out <id>`: `/api/v1/accounts/{id}/users/out`.
- `crater account billing config <id>`: `/api/v1/accounts/{id}/billing/config`.
- `crater account billing members <id>`: `/api/v1/accounts/{id}/billing/members`.
- `crater admin account ls|get|members|users-out|quota`: `/api/v1/admin/accounts...`.
- `crater admin account billing config|members <id>`: `/api/v1/admin/accounts/{id}/billing/...`.
- JSON payload keys: `accounts`, `account`, `members`, `users`, `quota`, `billing_config`.

### Resource Reads
- `crater resource ls [--with-vendor-domain]`: `/api/v1/resources`.
- `crater resource networks <id>`: `/api/v1/resources/{id}/networks`.
- `crater resource vgpu <id>`: `/api/v1/resources/{id}/vgpu`.
- `crater resource prices`: `/api/v1/resources/billing/prices`.
- `crater admin resource networks|vgpu <id>`: `/api/v1/admin/resources/{id}/...`.
- JSON payload keys: `resources`, `networks`, `vgpu`, `prices`.

### Dataset And Template Reads
- `crater dataset ls`: `/api/v1/dataset/mydataset`.
- `crater dataset get <id>`: `/api/v1/dataset/detail/{id}`.
- `crater dataset users <id>` / `queues <id>`: current share relationship reads.
- `crater dataset users-out <id>` / `queues-out <id>`: unshared user/account candidate reads.
- `crater admin dataset ls`: `/api/v1/admin/dataset/alldataset`.
- `crater template ls`: `/api/v1/jobtemplate/list`.
- `crater template get <id>`: `/api/v1/jobtemplate/{id}`.
- JSON payload keys: `datasets`, `dataset`, `users`, `queues`, `templates`, `template`.

### Model Download Reads
- `crater model-download ls [--category model|dataset] [--status STATUS] [--search TEXT] [--page N] [--page-size N] [--all-pages]`: `/api/v1/model-download/models/downloads`；这是 `download ls` 的兼容入口，复用相同的服务端分页、筛选和 JSON 契约，`page-size` 默认 `15`、最大 `100`。
- `crater model-download get <id>`: `/api/v1/model-download/models/downloads/{id}`.
- `crater model-download logs <id>`: `/api/v1/model-download/models/downloads/{id}/logs`.
- `crater admin model-download ls [--category model|dataset] [--status STATUS] [--search TEXT] [--page N] [--page-size N] [--all-pages]`: `/api/v1/admin/model-download/models/downloads`；后端返回数组，CLI 完成本地筛选后分页，`page-size` 默认 `15`、最大 `100`。
- 列表 JSON payload 包含 `downloads` 与当前页 `pagination`；普通列表还包含后端 `summary`。`--all-pages` 省略 `pagination`。其他 payload keys: `download`, `logs`.

### Context, Billing, User, And Approval Reads
- `crater context prequeue|quota|resources|billing`: `/api/v1/context/...` summary reads used by the portal.
- `crater billing status`, `summary`, `prices`, `job <name>`.
- `crater billing jobs [--all | --user USER] [--days N] [--search TEXT] [--page N] [--page-size N] [--all-pages]` 与 `crater admin billing jobs [--user USER] ...`：
  - `--days` 默认 `30` 且必须大于等于 `-1`；普通命令仅在 `--all` / `--user` 路由中传递 days，并保留 `--user` 优先于 `--all` 的语义；管理员命令始终传递 days。
  - `--search` 大小写不敏感地匹配展示名或平台作业名；CLI 按平台作业名、展示名、计费点数稳定排序后本地分页。
  - `page-size` 默认 `15`、最大 `200`；JSON 使用 `data.billing`，非 `--all-pages` 时同时有 `data.pagination`。
- `crater user get <username>`, `email-verified`.
- `crater order ls [--status STATUS] [--type TYPE] [--search TEXT] [--page N] [--page-size N] [--all-pages]`, `get <id>`, `by-name <name>`。
- `crater admin order ls` 支持相同的本地筛选和分页，并额外支持 `--creator TEXT`。普通用户 `order ls` 不提供 creator filter；两者的 `--search` 均匹配工单名称或创建者信息。
- 工单列表默认按 `Pending`、`Approved`、`Rejected`、`Canceled` 分组，同状态按创建时间和 ID 倒序排列。
- `crater admin user ls [--base] [--search TEXT] [--role Guest|User|Admin] [--status Pending|Active|Inactive] [--page N] [--page-size N] [--all-pages]`；搜索覆盖用户名与显示名，默认按 ID 倒序。`--base` 与 `--role` / `--status` 互斥。
- 上述工单和用户列表均先本地筛选再分页，`page-size` 默认 `15`、最大 `200`；JSON 当前页输出同时包含 `data.pagination`。
- `crater admin billing status|jobs`, `crater admin order get <id>`, `crater admin user billing summary|accounts <username>`.

### Approval Order Writes
- User-visible commands stay under `crater order ...`:
  - `crater order submit --name NAME --type job|dataset --reason TEXT [--type-id ID] [--hours N]`.
  - `crater order edit <id> [--name NAME] [--type job|dataset] [--type-id ID] [--reason TEXT] [--hours N]`.
  - `crater order cancel <id> --yes`.
- Administrator review commands stay under `crater admin order ...`:
  - `crater admin order approve <id> [--review-notes TEXT]`.
  - `crater admin order approve <id> --lock [--permanent | --days N --hours N --minutes N] [--review-notes TEXT]`.
  - `crater admin order reject <id> --review-notes TEXT`.
  - `crater admin order check --yes`.
- `order edit` first reads the current order and preserves fields that were not explicitly provided, so absent flags do not clear existing content.
- `admin order approve|reject` use the admin review API and only send review status/notes. The backend derives `reviewerID` from the active token and preserves the original order content.
- Lock duration flags must be non-negative. Unless `--permanent` is set, `--lock` requires a positive duration.
- `--json` success payloads use `data.message`; `approve --lock --json` also includes `data.lock_message`.

### Pod And Non-Volcano Job Diagnostics
- 普通用户直接诊断 Job Pod 时必须显式指定真实命名空间：
  - `crater pod containers|events|ingresses|nodeports <pod> --namespace NAMESPACE`
  - `crater pod logs <pod> <container> --namespace NAMESPACE [--tail N] [--timestamps] [--previous]`
- 为兼容旧脚本，仍接受显式 namespace 的旧位置参数形式：`... <namespace> <pod>` 与 `logs <namespace> <pod> <container>`。同一次调用不能同时提供旧位置参数 namespace 和 `--namespace`，否则返回 `usage_error`。
- 上述命令覆盖 `/api/v1/namespaces/...` diagnostic GET APIs；`pod logs` 会先解码后端 Base64 载荷再输出原始日志文本。平台未向普通用户暴露全局作业命名空间配置，CLI 不硬编码或猜测默认值；`crater job get|pods|logs|events|yaml` 始终使用作业 API 返回的真实 namespace。
- Job-level log streaming is available through `crater job logs --follow`; terminal websocket APIs are intentionally not part of this CLI.
- AIJob/SPJob reads are intentionally not exposed in this PR because their backend identifier contracts differ from Volcano job names and need a dedicated CLI design.

### Interfaces Not Exposed As General Read CLI
- Sensitive credential reads (`/token`, `/secret`, Harbor credential APIs) are not exposed in the broad read surface.
- WebSocket and terminal endpoints are not exposed because they are interactive rather than stable one-shot reads.
- The untracked local `inference-services` API is not documented here until that backend/frontend feature lands in the branch base.
- Public health, Swagger, Prometheus metrics, and generic WebDAV operations are left to their domain-specific tools rather than this read CLI surface.

### Admin-Only Read Coverage
- `crater admin system-config llm|gpu-analysis|prequeue`: `/api/v1/admin/system-config/{llm,gpu-analysis,prequeue}`.
- `crater admin queue-quotas`: `/api/v1/admin/queue-quotas`.
- `crater admin gpu-analyses`: `/api/v1/admin/gpu-analysis`.
- `crater admin operation-logs [--page N] [--limit N] [--operator USER] [--operation-type TYPE] [--target TARGET] [--start-time TIME] [--end-time TIME]`: `/api/v1/admin/operation-logs`.
- `crater admin cronjobs`: `/api/v1/admin/operations/cronjob`.
- `crater admin whitelist`: `/api/v1/admin/operations/whitelist`.
- These commands surface existing admin GET APIs only. They do not perform update/delete/reconcile actions.

---

## 8. 远端文件模块 (file)

本模块面向普通用户访问 storage service 暴露的逻辑文件空间。远端路径不是本机路径，只允许以 `user`、`public` 或 `account` 为首段；CLI 会规范化安全的 `.`、重复分隔符和首尾分隔符，逐段进行 URL 编码，保留合法的空格与非 ASCII 文件名，并在请求前拒绝任何 `..` 段、反斜杠和控制字符。

### `crater file ls [remote-path]`

- **描述**：列出当前用户可见的远端文件或目录。
- **位置参数**：
  - `[remote-path]`（可选）：逻辑远端目录。省略时列出可见根目录；可用根为 `user`、`public`、`account`。
- **处理逻辑**：
  - 调用 `GET /api/ss/files` 或 `GET /api/ss/files/*path`。
  - 目录排在普通文件之前，同类型条目按名称稳定排序。
  - 空目录返回稳定的空列表。
  - 本命令只读取普通用户文件视图，不会切换到管理员接口。
- **输出格式**：
  - 默认模式：表格展示 `NAME`、`TYPE`、`SIZE`、`MODIFIED`；目录的大小显示为 `-`。
  - `--json`：stdout 输出成功信封 JSON。
- **`--json` 的 `data`**：
  - `files`（数组）：文件条目，每项包含 `name`、`size`、`isdir`、`modifytime`。
- **状态**：[x] Completed

下载、上传、创建目录、移动和删除不属于本命令范围，由各自独立的文件命令契约定义。
