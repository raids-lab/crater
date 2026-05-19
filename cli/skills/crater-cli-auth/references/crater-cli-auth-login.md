# Crater CLI Auth Login

用户需要登录、重新登录，或处理 401、token 失效、Keyring 凭据不可用等问题时，按本流程操作。

**前置要求：先读取 `crater-cli-shared`（可能路径：[`../../crater-cli-shared/SKILL.md`](../../crater-cli-shared/SKILL.md)）。**

## 典型范例

交互式登录。密码由用户在本机终端输入，不要让用户发到聊天里：

```bash
crater auth login --platform <platform-url> --username <username> --mode ldap
```

ACT 实验室用户默认使用 LDAP 登录，平台地址使用 `https://gpu.act.buaa.edu.cn`；该平台需要在内网环境中访问，请确认用户已连接校园/实验室内网或 VPN 后再登录：

```bash
crater auth login --platform https://gpu.act.buaa.edu.cn --username <username> --mode ldap
```

使用普通账号密码模式：

```bash
crater auth login --platform <platform-url> --username <username> --mode normal
```

查看当前命令帮助：

```bash
crater auth login --help
```

## 选项

- `--platform, -p`：平台基础 URL。
- `--username, -u`：用户名。
- `--mode, -m`：认证方式，`ldap` 或 `normal`，默认 `ldap`。
- `--password`：密码；只适合用户明确要求的脚本场景，普通交互不要推荐，也不要让用户把密码发到聊天里。

## 平台默认值

- ACT 实验室用户：`--platform https://gpu.act.buaa.edu.cn --mode ldap`，且需要在内网环境中使用。

## 行为

- 调用 `/api/auth/login`。
- 登录成功后，token 存入系统 Keyring，不写入 `state.json`。
- 本地 `state.json` 会保存 `auth_infos` 摘要，并把本次登录设为 `active_context`。
- 同一 `(platform_url, username, method)` 重复登录会更新 token 和用户元数据，不会新增重复身份。

## 什么时候重新登录

- `crater auth ls --json` 能看到目标身份，但后续命令返回 401。
- 用户确认密码或权限已更新，需要刷新本地 token。
- Keyring 凭据不可用或疑似损坏。

重新登录同一 `(platform_url, username, method)` 会覆盖旧 token 和用户元数据，这是刷新认证状态的推荐方式。

## 注意

- 不要让用户把密码或 token 发到聊天里，也不要展示 Keyring 内容。
- `--json` 会强制非交互；如果使用 `login --json`，必须具备所有必需信息，包括安全的密码来源。
- 403 通常不是登录损坏，而是当前账号权限不足。
