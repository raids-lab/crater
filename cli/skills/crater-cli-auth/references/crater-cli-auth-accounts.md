# Crater CLI Project Accounts

用户需要查看当前项目账户／队列，或在同一登录身份下切换项目时，按本流程操作。

**前置要求：先读取 [`crater-cli-shared`](../../crater-cli-shared/SKILL.md)。**

## 查看与切换

```bash
crater account current --json
crater account ls --json
crater account use <account-name> --json
crater account current --json
```

- `account current` 请求平台，返回当前凭据对应的账户上下文；`data.context.queue` 是当前账户／队列名称。
- `account ls` 返回当前用户所属的账户，选择其中的精确 `name` 作为 `account use` 参数，不能用昵称或 ID 替代。
- 用户明确要求切换后执行 `account use <name>`。该命令会更新当前登录身份在本机保存的凭据，后续作业提交、文件访问、配额和计费查询随之使用目标账户。
- `account use` 在 JSON／非交互模式下也必须提供名称；不需要 `--yes`。
- `current` 和 `use` 的 JSON 成功数据包含 `platform_url`、`username`、`context`，均不包含 access token 或 refresh token。
- 切换失败时按共享错误处理规则检查 stderr；不要通过手工编辑 `state.json` 或 JWT 修改项目。

## 选择正确的切换入口

- 同一用户切换项目账户／调度队列：`crater account use <account-name>`。
- 切换平台、登录用户或认证方式：`crater auth switch`。
- 只查看当前项目：`crater account current`，无需执行切换。
