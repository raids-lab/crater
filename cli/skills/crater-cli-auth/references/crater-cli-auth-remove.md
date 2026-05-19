# Crater CLI Auth Remove

用户需要登出当前身份、删除指定身份，或理解 `logout` 和 `rm` 的区别时，按本流程操作。

**前置要求：先读取 `crater-cli-shared`（可能路径：[`../../crater-cli-shared/SKILL.md`](../../crater-cli-shared/SKILL.md)）。**

**CRITICAL — `logout` 和 `rm` 都会修改用户本地认证状态。执行前必须确认用户意图；不要静默追加 `--yes`。**

## 典型范例

交互式登出当前身份：

```bash
crater auth logout
```

非交互式登出当前身份：

```bash
crater auth logout --yes --no-interactive
```

删除指定身份：

```bash
crater auth rm --platform <platform-url> --username <username> --mode ldap
```

非交互式删除指定身份：

```bash
crater auth rm --platform <platform-url> --username <username> --mode ldap --yes --no-interactive
```

查看当前命令帮助：

```bash
crater auth logout --help
crater auth rm --help
```

## `logout` 和 `rm` 的区别

- `logout` 只作用于当前 `active_context`。
- `rm` 删除所有匹配过滤条件的身份，可以删除非当前身份，也可能一次匹配多个身份。
- 如果用户说“退出当前账号”，用 `logout`。
- 如果用户说“删除某个平台/某个用户/某种认证方式的保存凭据”，用 `rm`。

## `crater auth logout`

用途：登出当前激活身份。

选项：

- `--yes, -y`：跳过确认。

行为：

- 删除当前 active 身份对应的 Keyring token。
- 从 `auth_infos` 移除当前 active 身份。
- 如果仍有其他身份，自动切换到列表中第一项；否则清空 `active_context`。
- `--no-interactive` 下必须同时传 `--yes`。

`--json` 成功数据：

- `data.next_active`：登出后的新 active 身份；无 active 时为空对象字段。

## `crater auth rm`

用途：删除指定的一个或多个认证上下文。

选项：

- `--platform, -p`：过滤待删除的平台。
- `--username, -u`：过滤待删除的用户。
- `--mode, -m`：过滤待删除的认证方式。
- `--yes, -y`：跳过确认。

行为：

- 删除所有匹配过滤条件的身份。
- 同时删除对应 Keyring token。
- 如果删除的是当前 active 身份，则清空 `active_context`。
- `--no-interactive` 下必须同时传 `--yes`。

`--json` 成功数据：

- `data.removed_count`：删除数量。

## 安全注意

- 不要主动加 `--yes`，除非用户明确要求非交互执行或确认跳过提示。
- 执行 `rm` 前，如果条件可能匹配多个身份，应先建议 `crater auth ls` 预览。
- 如果用户只是想“检查登录状态”，不要执行 `logout` 或 `rm`。
