package i18n

// Human-readable errors (message strings). Keep codes stable elsewhere.
var catalogErrors = map[Language]map[string]string{
	En: {
		"err_not_found":            "no saved credentials match the criteria",
		"err_missing_required":     "%s is required (--%s)",
		"err_missing_password":     "password is required (--password) in non-interactive mode",
		"err_unauthorized":         "unauthorized: %s",
		"err_api_request":          "HTTP %d: %s",
		"err_network":              "could not reach server: %v",
		"err_api_other":            "API error (unclassified): %s",
		"err_json_encode":          "failed to write JSON output: %s",
		"err_config_write":         "failed to write config: %s",
		"err_no_active":            "no active credentials found",
		"err_operation_cancelled":  "operation cancelled",
		"err_invalid_lang":         "invalid language code: %s",
		"err_multiple_matches":     "multiple credentials match, please be more specific:",
		"err_missing_language_arg": "language code is required in non-interactive mode",
	},
	ZhCN: {
		"err_not_found":            "未找到匹配的已保存账号",
		"err_missing_required":     "缺少必要参数：%s (--%s)",
		"err_missing_password":     "非交互模式下必须提供密码 (--password)",
		"err_unauthorized":         "认证失败：%s",
		"err_api_request":          "请求失败（HTTP %d）：%s",
		"err_network":              "无法连接服务器：%v",
		"err_api_other":            "接口错误（未分类）：%s",
		"err_json_encode":          "JSON 输出写入失败：%s",
		"err_config_write":         "配置文件写入失败：%s",
		"err_no_active":            "未找到当前激活的账号",
		"err_operation_cancelled":  "操作已取消",
		"err_invalid_lang":         "无效的语言代码：%s",
		"err_multiple_matches":     "存在多个匹配的已保存账号，请提供更具体的信息：",
		"err_missing_language_arg": "非交互模式下必须提供语言代码",
	},
}

