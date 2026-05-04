package i18n

// auth command domain: descriptions, prompts, tables, and human-readable messages.
var catalogAuth = map[Language]map[string]string{
	En: {
		// Command Descriptions
		"auth_short":        "Authentication and credentials management",
		"auth_long":         "Manage login sessions and switch between different saved credentials (platform, identity, and method).",
		"auth_login_short":  "Log in to a Crater platform instance",
		"auth_login_long":   "Authenticate with a platform using Platform URL, Username, and Method.",
		"auth_switch_short": "Switch active credentials",
		"auth_switch_long":  "Switch between different saved platform credentials with fuzzy matching and interactive selection.",
		"auth_ls_short":     "List all saved credentials",
		"auth_rm_short":     "Remove saved credentials",
		"auth_logout_short": "Log out of the active saved credentials",

		// Flag Descriptions
		"flag_platform":  "Platform base URL",
		"flag_username":  "Username",
		"flag_mode":      "Authentication mode (ldap | normal)",
		"flag_password":  "Password (non-interactive only)",
		"flag_yes":       "Force operation without confirmation",

		// Completion candidate descriptions
		"auth_mode_ldap_desc":   "Authenticate via an LDAP server (recommended for ACT Lab clusters).",
		"auth_mode_normal_desc": "Authenticate against Crater's internal database.",

		// Prompts
		"prompt_platform": "Platform URL: ",
		"prompt_username": "Username: ",
		"prompt_password": "Password: ",

		// Human-readable messages
		"login_success":           "Successfully logged into:\n  %s\nAs:\n  %s (%s, %s)",
		"switch_success":          "Switched active credentials to:\n  %s (%s, %s, %s)",
		"current_active_context":  "Current active credentials:\n  %s (%s, %s, %s)",
		"no_active_context":       "No active credentials are set.",
		"logout_confirm":          "Log out of these credentials %s (%s, %s, %s)?",
		"logout_success_switched": "Logged out.\nAutomatically switched credentials to:\n  %s (%s, %s, %s)",
		"logout_success_no_other": "Logged out.\nNo other saved credentials are available.",
		"remove_confirm_title":    "The following credentials will be removed:",
		"remove_confirm_ask":      "Are you sure you want to remove these credentials?",
		"remove_success":          "Successfully removed:\n  %d credentials",
		"select_context":          "Select the credentials to switch to:",

		// Table headers / labels
		"table_active":    "ACTIVE",
		"table_platform":  "PLATFORM",
		"table_username":  "USERNAME",
		"table_method":    "METHOD",
		"table_privilege": "PRIVILEGE",
	},
	ZhCN: {
		// Command Descriptions
		"auth_short":        "认证与账号管理",
		"auth_long":         "管理登录会话，并在不同账号之间切换。",
		"auth_login_short":  "登录到 Crater 平台实例",
		"auth_login_long":   "使用平台 URL、用户名和认证方式进行身份验证。",
		"auth_switch_short": "切换激活的账号",
		"auth_switch_long":  "在已保存的账号之间切换，支持模糊匹配和交互式选择。",
		"auth_ls_short":     "列出所有已保存的账号",
		"auth_rm_short":     "删除已保存的账号",
		"auth_logout_short": "登出当前激活的账号",

		// Flag Descriptions
		"flag_platform":  "平台基础 URL",
		"flag_username":  "用户名",
		"flag_mode":      "认证模式 (ldap | normal)",
		"flag_password":  "密码 (仅限非交互模式)",
		"flag_yes":       "强制执行操作，无需确认",

		// Completion candidate descriptions
		"auth_mode_ldap_desc":   "通过 LDAP 服务器进行认证（推荐用于 ACT 实验室集群）",
		"auth_mode_normal_desc": "通过 Crater 内置数据库进行认证",

		// Prompts
		"prompt_platform": "平台 URL: ",
		"prompt_username": "用户名: ",
		"prompt_password": "密码: ",

		// Human-readable messages
		"login_success":           "成功登录到：\n  %s\n身份为：\n  %s (%s, %s)",
		"switch_success":          "已切换到以下账号：\n  %s (%s, %s, %s)",
		"current_active_context":  "当前激活的账号：\n  %s (%s, %s, %s)",
		"no_active_context":       "当前未设置激活的账号。",
		"logout_confirm":          "是否登出当前账号 %s (%s, %s, %s)？",
		"logout_success_switched": "已登出。已自动切换到以下账号：\n  %s (%s, %s, %s)",
		"logout_success_no_other": "已登出。没有其他可用的账号。",
		"remove_confirm_title":    "以下账号将被删除：",
		"remove_confirm_ask":      "确定要删除这些账号吗？",
		"remove_success":          "成功删除：\n  %d 个账号",
		"select_context":          "请选择要切换到的账号：",

		// Table headers / labels
		"table_active":    "激活",
		"table_platform":  "平台",
		"table_username":  "用户名",
		"table_method":    "认证方式",
		"table_privilege": "权限级别",
	},
}

