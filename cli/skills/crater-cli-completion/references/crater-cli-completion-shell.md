# Crater CLI Completion Shell

用户需要输出、安装、更新或卸载 shell Tab 补全，或排查补全不可用时，按本流程操作。

**前置要求：先读取 `crater-cli-shared`（可能路径：[`../../crater-cli-shared/SKILL.md`](../../crater-cli-shared/SKILL.md)）。**

**CRITICAL — `install` 和 `uninstall` 会修改用户的 shell rc 文件。执行前必须确认用户意图；不要静默追加 `--yes`。**

## 典型范例

输出 zsh 补全脚本，不修改文件：

```bash
crater completion zsh
```

输出 bash 补全脚本，不修改文件：

```bash
crater completion bash
```

安装或更新 zsh 补全：

```bash
crater completion install zsh
```

安装或更新 bash 补全：

```bash
crater completion install bash
```

非交互安装：

```bash
crater completion install zsh --yes --no-interactive
```

卸载补全：

```bash
crater completion uninstall zsh
crater completion uninstall bash
```

查看帮助：

```bash
crater completion --help
crater completion install --help
crater completion uninstall --help
```

## 命令说明

`crater completion <SHELL>` 输出指定 shell 的补全脚本到 stdout，无副作用。

- `<SHELL>`：`bash` 或 `zsh`。
- `--json` 成功数据：`data.shell`、`data.script`。

`crater completion install <SHELL>` 安装或更新补全脚本。

- `zsh`：写入或更新 `~/.zshrc` 中带 marker 的 Crater 补全块。
- `bash`：写入或更新 `~/.bashrc` 中带 marker 的 Crater 补全块。
- marker 起止行为固定为：
  ```text
  # >>> crater completion >>>
  ...
  # <<< crater completion <<<
  ```
- 重复执行是幂等的：已有 marker 块时会替换块内脚本。
- `--no-interactive` 下必须同时提供 `--yes`。
- `--json` 成功数据：`data.shell`、`data.installed_paths`、`data.updated`，以及 shell 对应的 `data.inserted_zshrc` 或 `data.inserted_bashrc`。

`crater completion uninstall <SHELL>` 卸载补全脚本。

- 只移除 Crater 写入的 marker 块，不删除整个 rc 文件。
- `--no-interactive` 下必须同时提供 `--yes`。
- `--json` 成功数据：`data.shell`、`data.removed_paths`。

`crater comp` 是 `crater completion` 的等价别名，参数和行为一致。

## 判断规则

- 用户只是想查看脚本：使用 `crater completion <shell>`。
- 用户想启用 Tab 补全：使用 `crater completion install <shell>`。
- 用户想移除补全配置：使用 `crater completion uninstall <shell>`。
- 用户使用 Windows 且需要补全：当前不支持 PowerShell；建议 Git Bash + bash 路径。
- 用户没有说明 shell：询问使用 `bash` 还是 `zsh`，不要猜测并修改 rc 文件。

## AI 是否应使用补全探测候选

- 常规情况下不要把补全当作命令发现或业务候选查询接口；优先使用 `--help`、领域 skill、`--json` 输出和明确的业务命令。
- `crater __complete ...` 是 shell 钩子的内部快路径，不作为稳定用户接口。只有在排查补全脚本或补全候选异常时，才围绕它分析。
- 若需要知道某个命令当前支持的静态参数，先运行 `crater <command> --help`。

## 检查安装是否成功

当用户要求检查补全是否安装成功时，按 shell 检查对应 rc 文件：

- `zsh`：检查 `~/.zshrc`。
- `bash`：检查 `~/.bashrc`。

判断标准：

- 文件中存在 `# >>> crater completion >>>` 和 `# <<< crater completion <<<` 两个 marker。
- marker 块内包含调用当前 `crater` 可执行文件的补全脚本。
- 如果用户是在本地开发环境测试刚编译出的 CLI，确认 marker 块中的可执行文件路径是否符合预期，避免仍指向全局旧版本。

安装命令的 `--json` 输出也可辅助判断：

- `data.installed_paths`：本次写入的 rc 文件路径。
- `data.updated`：是否替换了已有 marker 块。
- `data.inserted_zshrc` / `data.inserted_bashrc`：是否本次追加了新块。

检查完成后，提示用户**重启终端**再测试 Tab 补全。仅执行 `source ~/.zshrc` 或 `source ~/.bashrc` 有时不能完全刷新当前 shell 的补全状态，尤其是补全函数或 shell 缓存已初始化时。

## 安全注意

- 安装和卸载都会修改用户 shell 配置文件；执行前说明会改哪个 rc 文件，并等待用户确认。
- 不要删除用户 rc 文件中的非 Crater 内容。
- 非交互使用 `--yes` 前必须确认用户已同意跳过提示。

## 排查重点

- 安装后无效：先检查对应 rc 文件中的 marker 块和可执行文件路径，再提示用户重启终端；不要只依赖 `source`。
- 非交互失败：补充 `--yes`。
- shell 不支持：仅 `bash` / `zsh`；不支持 PowerShell。
- rc 文件写入失败：检查 HOME、rc 文件权限或文件系统权限。
