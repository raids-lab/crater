package i18n

// completion 命令族：描述、安装提示与错误。
var catalogCompletion = map[Language]map[string]string{
	En: {
		"completion_short": "Generate or install shell completion scripts",
		"completion_long":  "Manage shell tab completion for Crater (zsh supported in this phase).",

		"completion_zsh_short": "Print zsh completion snippet",
		"completion_zsh_long":  "Print the zsh rc snippet to enable crater tab completion (delegates to crater __complete).",

		"completion_bash_short": "Print bash completion snippet",
		"completion_bash_long":  "Print the bash rc snippet to enable crater tab completion (delegates to crater __complete).",

		"completion_install_short": "Install completion scripts for a shell",
		"completion_install_long":  "Install completion to user-writable locations (no admin rights).",

		"completion_install_zsh_short": "Enable zsh completion by updating ~/.zshrc",
		"completion_install_zsh_long":  "Write an inline completion block into ~/.zshrc (idempotent, uninstallable).",

		"completion_install_bash_short": "Enable bash completion by updating ~/.bashrc",
		"completion_install_bash_long":  "Write an inline completion block into ~/.bashrc (idempotent, uninstallable).",

		"completion_uninstall_short": "Remove installed completion scripts",
		"completion_uninstall_long":   "Remove files written by crater completion install.",

		"completion_uninstall_zsh_short": "Disable zsh completion by updating ~/.zshrc",
		"completion_uninstall_zsh_long":   "Remove the crater completion marker block from ~/.zshrc.",

		"completion_uninstall_bash_short": "Disable bash completion by updating ~/.bashrc",
		"completion_uninstall_bash_long":  "Remove the crater completion marker block from ~/.bashrc.",

		"completion_install_plan": "This will modify your zsh config:\n\n  - Update your zsh rc file:\n      %s\n\nThe following block will be added or updated in your ~/.zshrc:\n\n%s\n",
		"completion_install_confirm": "Proceed? [y/N]: ",
		"completion_install_done":    "Enabled zsh completion by updating:\n  - %s\n",
		"completion_install_next_steps": "To activate in current shell, run:\n  source ~/.zshrc\nOr open a new terminal session.\n",
		"completion_install_uninstall_hint": "To uninstall, run:\n  crater completion uninstall zsh\n",
		"completion_install_uninstall_hint_bash": "To uninstall, run:\n  crater completion uninstall bash\n",

		"completion_uninstall_plan": "This will remove Crater zsh completion:\n\n  - Remove the marker block from:\n      %s\n",
		"completion_uninstall_confirm": "Proceed? [y/N]: ",
		"completion_uninstall_done":    "Removed:\n%s\n",
		"completion_uninstall_next_steps": "To ensure changes take effect, run:\n  source ~/.zshrc\nOr open a new terminal session.\n",

		"completion_bash_install_plan": "This will modify your bash config:\n\n  - Update your bash rc file:\n      %s\n\nThe following block will be added or updated in your ~/.bashrc:\n\n%s\n",
		"completion_bash_install_done": "Enabled bash completion by updating:\n  - %s\n",
		"completion_bash_install_next_steps": "To activate in current shell, run:\n  source ~/.bashrc\nOr open a new terminal session.\n",
		"completion_bash_uninstall_plan": "This will remove Crater bash completion:\n\n  - Remove the marker block from:\n      %s\n",
		"completion_bash_uninstall_next_steps": "To ensure changes take effect, run:\n  source ~/.bashrc\nOr open a new terminal session.\n",
		"err_completion_unsupported":   "unsupported shell for this build: %s (supported: zsh, bash)",
	},
	ZhCN: {
		"completion_short": "生成或安装 shell 补全脚本",
		"completion_long":  "管理 Crater 的 Tab 补全（当前阶段仅支持 zsh）。",

		"completion_zsh_short": "输出 zsh 补全片段",
		"completion_zsh_long":  "输出写入 ~/.zshrc 的补全片段，通过 crater __complete 快路径获取候选。",

		"completion_bash_short": "输出 bash 补全片段",
		"completion_bash_long":  "输出写入 ~/.bashrc 的补全片段，通过 crater __complete 快路径获取候选。",

		"completion_install_short": "为指定 shell 安装补全脚本",
		"completion_install_long":  "安装到用户可写目录（无需提权）。",

		"completion_install_zsh_short": "通过更新 ~/.zshrc 启用 zsh 补全",
		"completion_install_zsh_long":  "向 ~/.zshrc 写入内联补全 block（幂等、可卸载）。",

		"completion_install_bash_short": "通过更新 ~/.bashrc 启用 bash 补全",
		"completion_install_bash_long":  "向 ~/.bashrc 写入内联补全 block（幂等、可卸载）。",

		"completion_uninstall_short": "卸载已安装的补全脚本",
		"completion_uninstall_long":  "删除由 crater completion install 写入的文件。",

		"completion_uninstall_zsh_short": "通过更新 ~/.zshrc 禁用 zsh 补全",
		"completion_uninstall_zsh_long":  "从 ~/.zshrc 移除 crater completion marker block。",

		"completion_uninstall_bash_short": "通过更新 ~/.bashrc 禁用 bash 补全",
		"completion_uninstall_bash_long":  "从 ~/.bashrc 移除 crater completion marker block。",

		"completion_install_plan": "本操作将修改你的 zsh 配置：\n\n  - 更新你的 zsh rc 文件：\n      %s\n\n将向 ~/.zshrc 追加或更新如下 block：\n\n%s\n",
		"completion_install_confirm": "是否继续？[y/N]: ",
		"completion_install_done":    "已通过更新以下文件启用 zsh 补全：\n  - %s\n",
		"completion_install_next_steps": "如需在当前 shell 立刻生效，请执行：\n  source ~/.zshrc\n或重新打开一个终端会话。\n",
		"completion_install_uninstall_hint": "如需卸载，请执行：\n  crater completion uninstall zsh\n",
		"completion_install_uninstall_hint_bash": "如需卸载，请执行：\n  crater completion uninstall bash\n",

		"completion_uninstall_plan": "本操作将移除 Crater 的 zsh 补全：\n\n  - 从以下文件移除 marker block：\n      %s\n",
		"completion_uninstall_confirm": "是否继续？[y/N]: ",
		"completion_uninstall_done":    "已移除：\n%s\n",
		"completion_uninstall_next_steps": "为确保变更生效，请执行：\n  source ~/.zshrc\n或重新打开一个终端会话。\n",

		"completion_bash_install_plan": "本操作将修改你的 bash 配置：\n\n  - 更新你的 bash rc 文件：\n      %s\n\n将向 ~/.bashrc 追加或更新如下 block：\n\n%s\n",
		"completion_bash_install_done": "已通过更新以下文件启用 bash 补全：\n  - %s\n",
		"completion_bash_install_next_steps": "如需在当前 shell 立刻生效，请执行：\n  source ~/.bashrc\n或重新打开一个终端会话。\n",
		"completion_bash_uninstall_plan": "本操作将移除 Crater 的 bash 补全：\n\n  - 从以下文件移除 marker block：\n      %s\n",
		"completion_bash_uninstall_next_steps": "为确保变更生效，请执行：\n  source ~/.bashrc\n或重新打开一个终端会话。\n",
		"err_completion_unsupported":   "当前构建不支持的 shell：%s（支持：zsh、bash）",
	},
}
