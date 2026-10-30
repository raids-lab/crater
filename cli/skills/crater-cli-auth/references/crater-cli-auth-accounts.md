# Crater CLI Accounts

用户需要查看当前账户／队列，或在同一登录身份下切换账户时，按本流程操作。

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
- 切换失败时按共享错误处理规则检查 stderr；不要通过手工编辑 `state.json` 或 JWT 切换账户。

## 解释账户上下文

`current` 和 `use` 的 `data.context` 使用相同字段。以下数值对应后端的 `Role` 和 `AccessMode` 定义：

| 字段 | 含义 | 数值 |
| --- | --- | --- |
| `queue` | 当前账户／调度队列的名称 | 字符串 |
| `roleQueue` | 当前用户在该账户中的角色 | `1` = 访客，`2` = 普通用户，`3` = 管理员 |
| `rolePlatform` | 当前用户的平台角色 | `1` = 访客，`2` = 普通用户，`3` = 管理员 |
| `accessQueue` | 账户空间访问模式 | `1` = 禁止访问，`2` = 只读，`3` = 读写，`4` = 仅追加 |
| `accessPublic` | 公共空间访问模式 | `1` = 禁止访问，`2` = 只读，`3` = 读写，`4` = 仅追加 |

账户角色描述当前账户内的权限，平台角色描述平台级权限，应分别解释。空间访问模式由对应的 `accessQueue`、`accessPublic` 字段确定，不要根据角色数值推断。

例如 `roleQueue: 2` 表示账户中的普通用户，`accessQueue: 3` 表示账户空间可读写。解释时保留 JSON 字段名与原始数值。遇到未知数值（例如 `roleQueue: 99`）时，保留 `99` 并说明“未知角色值”，不要自行映射为已有角色或访问权限。

## 选择正确的切换入口

- 同一用户切换账户／调度队列：`crater account use <account-name>`。
- 切换平台、登录用户或认证方式：`crater auth switch`。
- 只查看当前账户：`crater account current`，无需执行切换。
