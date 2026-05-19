package i18n

// config command domain: descriptions and language selection prompts/messages.
var catalogConfig = map[Language]map[string]string{
	En: {
		// Command Descriptions
		"config_short":            "Configure CLI settings",
		"config_long":             "Manage CLI configuration such as display language.",
		"config_language_short":   "Change CLI display language",
		"lang_short":              "Change CLI display language",
		"select_language":         "Select language:",
		"lang_switch_success":     "Language switched to: %s",
		"lang_completion_label_current": "%s (current)",
	},
	ZhCN: {
		// Command Descriptions
		"config_short":            "配置 CLI 设置",
		"config_long":             "管理 CLI 配置，如显示语言。",
		"config_language_short":   "修改 CLI 显示语言",
		"lang_short":              "修改 CLI 显示语言",
		"select_language":         "选择语言：",
		"lang_switch_success":     "语言已切换为：%s",
		"lang_completion_label_current": "%s（当前）",
	},
}

