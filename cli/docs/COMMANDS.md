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
   - **错误码定义**: 以 `pkg/errorcodes/codes.go` 为准。**`api_error`** 的 **`code`** 须与 **HTTP** 显式对应，命名形如 **`ERR_NOT_FOUND_404`**、**`ERR_SERVER_INTERNAL_5XX`** 等，完整约定见 **[SPEC.md](./SPEC.md)**「命令结果：错误与成功」中 `api_error` 与 HTTP 小节。
   - **退出码**: 出错时非零退出；具体数值由 `Execute` 根据 `*clierror.Error` 的 `category` 映射（实现为 `pkg/errorcodes.ExitCodeForCategory`：`usage_error`→2，`cancelled`→3，`api_error`→4，`system_error`→5；非 `*clierror.Error` 的错误→1）。命令实现里不必自行 `os.Exit`。

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
  - 成功后将 Token 存入系统 Keyring，键名为 `crater`，子键为 `platform|user|mode`。
  - 更新 `state.json` 中的 `auth_infos` 列表，并自动将该环境设为 `active_context`。
- **`--json` 的 `data`**：`user`（与 CLI 持久化视图一致的用户摘要对象；**不含** token 明文等敏感字段，与实现 `config.AuthInfo` 对齐）。
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
- **`--json` 的 `data`**：`active_context`（对象）、`auth_infos`（数组，筛选后的条目）。
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
    - 从 `state.json` 中移除对应条目。
    - 同时从系统 Keyring 中删除关联的 Token。
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
    - 从 Keyring 中删除当前激活项的 Token。
    - 从 `state.json` 的 `auth_infos` 列表中移除该项。
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
  - `--token` (string): 可选的访问令牌，用于 gated/private 仓库。该值仅随本次请求发送给平台，不写入 CLI 本地配置或 keyring，不在成功/错误输出中展示。
  - `--token-env` (string): 从指定环境变量读取可选访问令牌。与 `--token`、`--token-stdin` 互斥。
  - `--token-stdin` (bool): 从 stdin 读取可选访问令牌。与 `--token`、`--token-env` 互斥。
  - `--wait` (bool): 提交后轮询任务，直到状态进入 `Ready`、`Failed` 或 `Paused`。
  - `--poll-interval` (duration): `--wait` 的轮询间隔，默认 `5s`。
  - `--timeout` (duration): `--wait` 的最长等待时间，默认 `0` 表示不超时。
- **处理逻辑**:
  - 本地校验 `--name`、`--category`、`--source`；可在请求前发现的问题必须聚合为单个 `usage_error`。
  - 若提供 `--token-env` 或 `--token-stdin`，CLI 读取 token 后仅用于本次请求；不得在输出中展示 token。
  - 读取当前激活的认证上下文与 token；若未登录或 token 不可用，返回错误。
  - 调用平台接口创建下载任务。后端根据 `category` 自动选择平台侧目标目录（模型为 `public/Models`，数据集为 `public/Datasets`）。
  - 若平台返回资源已存在或正在下载，CLI 仍按成功处理并展示后端返回的任务信息与消息。
- **输出格式**:
  - 默认模式：stdout 展示下载任务的 ID、名称、类别、来源、状态与目标路径。
  - `--json`：stdout 输出成功信封 JSON。
- **`--json` 的 `data`**：
  - `download`（对象，后端返回的下载任务摘要，字段包括 `id`、`name`、`source`、`category`、`revision`、`path`、`sizeBytes`、`downloadedBytes`、`downloadSpeed`、`status`、`message`、`jobName`、`creatorId`、`referenceCount`、`createdAt`、`updatedAt`）
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
- **输出格式**:
  - 默认模式：表格展示 `ID`、`NAME`、`CATEGORY`、`SOURCE`、`STATUS`、`PATH`。
  - `--json`：stdout 输出成功信封 JSON。
- **`--json` 的 `data`**：`downloads`（下载任务数组）。
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
- **处理逻辑**:
  - 调用 `/api/v1/nodes/{name}/pods`。
  - 默认模式以表格展示 Pod 名称、命名空间、IP、状态、类型和资源。
- **`--json` 的 `data`**：`pods`（数组，平台节点 Pod 响应）。
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

本模块提供 Volcano 作业的只读查询能力。第一版仅使用 `/api/v1/vcjobs` 族接口；`aijobs` / `spjobs` 后续独立扩展。

### `crater job ls`
- **描述**: 列出当前账号可见的作业。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--all` (bool): 调用 `/api/v1/vcjobs/all`，列出当前身份可见且位于 `--days` 回看窗口内的作业。
  - `--user` (string): 调用 `/api/v1/vcjobs/user/{username}`，列出指定用户且位于 `--days` 回看窗口内的作业。
  - `--days` (int): 与 `--all` 或 `--user` 配合使用的回溯天数；`-1` 表示不按时间过滤。小于 `-1` 的值返回 `usage_error`。
  - `--status` (string): 本地过滤作业状态。
  - `--type` (string): 本地过滤作业类型。
  - `--node` (string): 本地过滤运行在指定节点上的作业。
  - `--interactive` (bool): 本地过滤交互式作业（`jupyter` / `webide`）。
  - `--batch` (bool): 本地过滤非交互式作业。
- **处理逻辑**:
  - 默认调用 `/api/v1/vcjobs`，列出当前用户和当前账户下的作业。
  - `--user` 优先于 `--all`；两者都不提供时不使用 `--days`。
  - `--interactive` 与 `--batch` 互斥。
  - 默认模式以表格展示名称、平台作业名、类型、状态、队列、节点和资源。
- **`--json` 的 `data`**：`jobs`（数组，元素与平台作业摘要响应一致，过滤后返回）。
- **状态**: [x] Completed

### `crater job get <name>`
- **描述**: 查看单个作业详情。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名，对应前端详情页路径中的作业名。
- **处理逻辑**:
  - 调用 `/api/v1/vcjobs/{name}/detail`。
  - 缺少 `<name>` 时返回 `usage_error`。
- **`--json` 的 `data`**：`job`（对象，平台作业详情响应）。
- **状态**: [x] Completed

### `crater job pods <name>`
- **描述**: 查看指定作业的 Pod 列表。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名。
- **处理逻辑**:
  - 调用 `/api/v1/vcjobs/{name}/pods`。
  - 默认模式以表格展示 Pod 名称、命名空间、节点、IP、阶段和资源。
- **`--json` 的 `data`**：`pods`（数组，平台作业 Pod 响应）。
- **状态**: [x] Completed

### `crater job events <name>`
- **描述**: 查看指定作业的 Kubernetes 事件。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名。
- **处理逻辑**:
  - 调用 `/api/v1/vcjobs/{name}/event`。
  - 默认模式逐行打印事件对象摘要；脚本化使用建议加 `--json`。
- **`--json` 的 `data`**：`events`（数组，平台事件响应）。
- **状态**: [x] Completed

### `crater job yaml <name>`
- **描述**: 查看指定作业的 YAML。
- **位置参数**:
  - `<name>` (positional, required): 平台作业名。
- **处理逻辑**:
  - 调用 `/api/v1/vcjobs/{name}/yaml`。
  - 默认模式直接输出 YAML 字符串到 stdout，不添加表格或装饰文本。
- **`--json` 的 `data`**：`yaml`（字符串）。
- **状态**: [x] Completed

---

## 7. 镜像模块 (image)

本模块提供容器镜像、镜像构建、分享、CUDA base image 和 Harbor 项目管理能力。所有命令均要求已有 active credentials。用户可操作资源使用 `crater image ...`；管理员/平台级资源统一使用 `crater admin image ...`，不得使用 `--admin` 切换。

### `crater image ls`
- **描述**: 列出当前账号可见的镜像。
- **位置参数**: 无；如果提供任何位置参数，返回 `usage_error`。
- **选项**:
  - `--available` (bool): 调用 `/api/v1/images/available`，列出创建作业时可选择的镜像。
  - `--type` (string): 本地过滤镜像适用的作业类型。
  - `--arch` (string): 本地过滤镜像架构，例如 `linux/amd64`。
  - `--visibility` (string): 本地过滤镜像可见性，例如 `Public`、`Private`、`UserShare`、`AccountShare`。
  - `--owner` (string): 按所有者用户名或昵称子串本地过滤。
  - `--search` (string): 按镜像地址或描述子串本地过滤。
- **处理逻辑**:
  - 默认调用 `/api/v1/images/image`。
  - 所有过滤均在 CLI 本地完成，不改变平台状态。
  - 默认模式以表格展示 ID、镜像地址、类型、可见性、架构和所有者。
- **`--json` 的 `data`**：`images`（数组，元素与平台镜像响应一致，过滤后返回）。
- **状态**: [x] Completed

### Image Build Commands
- `crater image build ls`: `/api/v1/images/kaniko`
- `crater image build get <name>`: `/api/v1/images/getbyname?name=...`
- `crater image build template <name>`: `/api/v1/images/template?name=...`
- `crater image build pod <id>`: `/api/v1/images/podname?id=...`
- `crater image build pip-apt --name NAME --tag TAG --image BASE [--packages TEXT] [--requirements TEXT]`
- `crater image build dockerfile --name NAME --tag TAG (--dockerfile TEXT | --file PATH)`
- `crater image build envd --name NAME --tag TAG (--envd TEXT | --file PATH) [--build-source EnvdAdvanced|EnvdRaw]`
- `crater image build remove --ids 1,2`
- Admin variants:
  - `crater admin image build-ls`
  - `crater admin image build-remove --ids 1,2`
- JSON payload keys: `builds`, `build`, `template`, `pod`, `message`.

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
  - `crater admin image ls`
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
- `crater image cuda ls|add|delete`
- `crater image harbor info`
- `crater image harbor credential --yes`
- `crater image quota get|set --size BYTES`
- Harbor credential output contains sensitive data and requires explicit `--yes`.
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
- `crater model-download ls [--category model|dataset]`: `/api/v1/model-download/models/downloads`.
- `crater model-download get <id>`: `/api/v1/model-download/models/downloads/{id}`.
- `crater model-download logs <id>`: `/api/v1/model-download/models/downloads/{id}/logs`.
- `crater admin model-download ls`: `/api/v1/admin/model-download/models/downloads`.
- JSON payload keys: `downloads`, `download`, `logs`.

### Context, Billing, User, And Approval Reads
- `crater context prequeue|quota|resources|billing`: `/api/v1/context/...` summary reads used by the portal.
- `crater billing status`, `summary`, `prices`, `jobs [--all|--user USER --days N]`, `job <name>`.
- `crater user get <username>`, `email-verified`.
- `crater order ls`, `get <id>`, `by-name <name>`.
- `crater admin billing status|jobs`, `crater admin order ls|get <id>`, `crater admin user ls`, `crater admin user billing summary|accounts <username>`.

### Pod And Non-Volcano Job Diagnostics
- `crater pod containers|events|ingresses|nodeports <namespace> <pod>` and `crater pod logs <namespace> <pod> <container> [--tail N] [--timestamps] [--previous]` cover `/api/v1/namespaces/...` diagnostic GET APIs. Log streaming and terminal websocket APIs are intentionally not part of this read CLI.
- AIJob/SPJob reads are intentionally not exposed in this PR because their backend identifier contracts differ from Volcano job names and need a dedicated CLI design.

### Interfaces Not Exposed As General Read CLI
- Sensitive credential reads (`/token`, `/secret`, Harbor credential APIs) are not exposed in the broad read surface.
- WebSocket, terminal, and log streaming endpoints are not exposed because they are interactive/streaming rather than stable one-shot reads.
- The untracked local `inference-services` API is not documented here until that backend/frontend feature lands in the branch base.
- Public health, Swagger, Prometheus metrics, and low-level WebDAV file listing are left to their domain-specific tools rather than this first read CLI pass.

### Admin-Only Read Coverage
- `crater admin system-config llm|gpu-analysis|prequeue`: `/api/v1/admin/system-config/{llm,gpu-analysis,prequeue}`.
- `crater admin queue-quotas`: `/api/v1/admin/queue-quotas`.
- `crater admin gpu-analyses`: `/api/v1/admin/gpu-analysis`.
- `crater admin operation-logs [--page N] [--limit N] [--operator USER] [--operation-type TYPE] [--target TARGET] [--start-time TIME] [--end-time TIME]`: `/api/v1/admin/operation-logs`.
- `crater admin cronjobs`: `/api/v1/admin/operations/cronjob`.
- `crater admin whitelist`: `/api/v1/admin/operations/whitelist`.
- These commands surface existing admin GET APIs only. They do not perform update/delete/reconcile actions.
