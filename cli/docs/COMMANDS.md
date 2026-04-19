# CLI COMMANDS REFERENCE (The Contract)

这个文档是 Crater CLI 的唯一事实来源。所有代码实现必须严格遵循此处的定义。

## 全局通用规范 (Global Requirements)

为了确保对 AI Agent、CI/CD 环境以及普通开发者的友好性，**所有命令（无论是否具备交互逻辑）必须统一支持以下全局选项：**

- `--json`: 
  - **行为**: 强制开启 `--no-interactive`，输出纯净的 JSON 至 `stdout`。
  - **Stdout**: 输出**格式化后的 JSON (Pretty-printed, 带缩进和换行)**，确保既对人类可读，又可被 `jq` 等工具解析。禁止包含任何非 JSON 的装饰性文字。
- `--no-interactive`:
  - **行为**: 彻底禁用所有交互式 Prompt（如密码输入、确认提示、上下键选择等）。
  - **约束**: 如果缺少必要信息，立即报错并返回非零退出码。
- `--help, -h`:
  - **行为**: 显示当前命令或子命令的帮助信息。

### 错误处理规范 (Error Handling)

所有错误必须通过 `stderr` 输出，其格式受 `--json` 影响：

1. **默认模式**: 输出人类可读的 `Error: <message>`。
2. **JSON 模式**: 输出单行结构化 JSON 对象。
   - **Schema**:
     ```json
     {
       "category": "usage_error | api_error | system_error",
       "code": "ERR_XXX_XXX",
       "message": "Human readable message",
       "context": { "key": "value" } 
     }
     ```
   - **错误码定义**: 统一参考源码中的 `pkg/errorcodes/codes.go`。
   - **退出码**: 发生任何错误时，进程必须以非零状态码（通常为 1）退出。

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
- **状态**: [ ] Pending

---

## 2. 认证模块 (auth)

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
- **状态**: [ ] Pending

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
- **状态**: [ ] Pending

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
- **状态**: [ ] Pending

---