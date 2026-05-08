# Crater CLI Auth Identities

用户需要查看已保存身份、确认当前身份、筛选身份，或切换 `active_context` 时，按本流程操作。

**前置要求：先读取 `crater-cli-shared`（可能路径：[`../../crater-cli-shared/SKILL.md`](../../crater-cli-shared/SKILL.md)）。**

## 典型范例

查看当前激活身份和全部保存身份：

```bash
crater auth ls
crater auth ls --json
```

按任意数量的条件筛选；可以只提供一个条件，也可以组合多个条件：

```bash
crater auth ls --platform <platform-url>
crater auth ls --username <username>
crater auth ls --platform <platform-url> --username <username> --mode ldap
```

切换身份时也可以提供 0-3 个条件。条件越少，越可能匹配多个候选；条件越多，越容易精确切换：

```bash
crater auth switch
crater auth switch --username <username>
crater auth switch --platform <platform-url> --username <username> --mode ldap
```

查看当前命令帮助：

```bash
crater auth ls --help
crater auth switch --help
```

## `crater auth ls`

用途：列出本地保存的认证上下文，并标记当前 active 身份。

选项：

- `--platform, -p`：按平台 URL 过滤。
- `--username, -u`：按用户名过滤。
- `--mode, -m`：按认证方式过滤。

默认输出列：

- `ACTIVE`
- `PLATFORM`
- `USERNAME`
- `METHOD`
- `PRIVILEGE`

`--json` 成功数据：

- `data.active_context`：当前激活三元组。
- `data.auth_infos`：筛选后的身份摘要数组。

## `crater auth switch`

用途：切换当前激活的认证上下文。

**注意：该命令会修改用户本地认证状态。执行前必须确认用户确实要切换当前身份。**

选项：

- `--platform, -p`：目标平台 URL。
- `--username, -u`：目标用户名。
- `--mode, -m`：目标认证方式。

行为：

- 根据给定选项筛选 `auth_infos`；`--platform`、`--username`、`--mode` 都是可选过滤条件，可提供任意数量。
- 筛选时排除当前已经激活的身份。
- 若只剩一个候选，则直接切换。
- 若有多个候选，交互模式下让用户选择；非交互模式下失败，并要求提供更精确条件。

`--json` 成功数据：

- `data.active`：新的 `active_context` 对象。

## 常见判断

- 要确认“当前登录的是谁”：用 `crater auth ls --json`，看 `active_context`。
- 要切到另一个已保存身份：先 `ls` 再 `switch`，避免误切。
- `switch` 失败且提示多个候选时，补齐更精确的 `--platform`、`--username`、`--mode`。
- `auth_infos` 中没有目标身份时，需要先登录，而不是切换。
- 如果用户只是想查看当前身份，不要执行 `switch`。
