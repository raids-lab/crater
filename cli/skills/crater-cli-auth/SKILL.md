---
name: crater-cli-auth
version: 1.1.0
description: "Crater CLI 认证域：指导 AI Agent 帮用户登录、重新登录、查看和切换已保存身份、删除凭据、登出当前身份，以及排查 token、session、active_context、401/403、未登录等认证问题。用户提到 crater auth、login、logout、switch、ls、rm、session、token、active context、state.json、未登录、认证失败、权限错误时使用。"
metadata:
  requires:
    bins: ["crater"]
---

# Crater CLI 认证

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

通过 `crater auth` 命令帮助用户处理认证状态时，遵守本规则。

## 适用场景

- 用户需要登录或重新登录 Crater platform。
- 用户需要查看当前身份、列出已保存身份，或切换 `active_context`。
- 用户需要登出当前身份，或删除某个已保存认证上下文。
- 用户遇到未登录、token 失效、401/403、权限不足等认证相关问题。

## 安全原则

- **禁止要求用户把密码、token、cookie 或 `state.json` 内容发到聊天里。**
- 查看登录状态用 `crater auth ls --json`，不要读取或粘贴本地 `state.json`；该文件含明文 token。
- `login`、`logout`、`rm`、`switch` 都会改变用户本地认证状态；执行前必须确认用户意图。
- 删除或登出需要确认时，只有在用户明确同意跳过确认或要求非交互执行时才添加 `--yes` / `-y`。
- 普通登录优先让用户在本机终端交互式输入密码，不要推荐把密码明文写入 `--password`。

## 认证模型

Crater CLI 使用 `(platform_url, username, method)` 三元组标识一个已保存认证上下文。

- `auth_infos`：保存在 `state.json` 中的本地身份记录，磁盘上含 `token` 明文。
- `active_context`：当前激活的三元组，后续需要认证的命令默认使用它。
- `token`：登录接口返回的访问令牌，写在对应 `auth_infos` 条目里。`auth ls` / `auth login` 的命令输出会去掉该字段。
- `method`：认证方式，目前支持 `ldap` 与 `normal`，默认 `ldap`。

## 工作流参考

- 登录、重新登录、401/token 失效处理：读取 `crater-cli-auth-login`（可能路径：[`references/crater-cli-auth-login.md`](references/crater-cli-auth-login.md)）。
- 查看当前身份、筛选身份、切换 `active_context`：读取 `crater-cli-auth-identities`（可能路径：[`references/crater-cli-auth-identities.md`](references/crater-cli-auth-identities.md)）。
- 登出当前身份、删除指定身份、区分 `logout` 与 `rm`：读取 `crater-cli-auth-remove`（可能路径：[`references/crater-cli-auth-remove.md`](references/crater-cli-auth-remove.md)）。

## 常用范例

```bash
crater auth ls --json
crater auth login --platform <platform-url> --username <username> --mode ldap
crater auth login --platform https://gpu.act.buaa.edu.cn --username <username> --mode ldap
crater auth switch --platform <platform-url> --username <username> --mode ldap
crater auth logout
crater auth rm --platform <platform-url> --username <username> --mode ldap
```

## 排查顺序

1. 先用 `crater auth ls --json` 查看 `active_context` 是否为空、目标身份是否存在、`method` 是否匹配。`auth ls` 输出故意不含 `token`，不能用来判断磁盘上有没有 token。
2. 若访问平台的命令在发请求前失败：`category=usage_error`，`code=ERR_NOT_FOUND`，没有 `http_status`，且 `message` 为 `no token saved for these credentials; please log in again` / `当前账号未保存 token，请重新登录`：身份还在，只是本地没有 token（常见于从旧 Keyring 升级后尚未重新登录）。不要读取 `state.json`。对同一 `(platform_url, username, method)` 执行 `crater auth login`。详细对照见 `crater-cli-shared` 的错误排查 reference。
3. 如果命令返回 401（`api_error` / `ERR_UNAUTHORIZED_401`），建议重新登录同一三元组。
4. 如果命令返回 403，优先判断当前账号在平台上的权限不足，而不是本地登录状态损坏。
5. 如果切换失败，检查过滤条件是否匹配多个身份；非交互模式下需要补齐更精确的 `--platform`、`--username`、`--mode`。
6. 如果删除或登出在 `--no-interactive` 下失败，检查是否缺少 `--yes`。
