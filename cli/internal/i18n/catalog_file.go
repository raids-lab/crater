package i18n

var catalogFile = map[Language]map[string]string{
	En: {
		"file_short":             "View remote files",
		"file_long":              "List files in user, public, and account storage spaces.",
		"file_ls_short":          "List remote files",
		"err_file_path_invalid":  "invalid remote path %q",
		"err_file_path_root":     "remote path must start with user, public, or account: %q",
		"file_type_directory":    "directory",
		"file_type_regular":      "file",
		"file_table_size":        "SIZE",
		"file_table_modified":    "MODIFIED",
		"file_root_user_desc":    "Your private user storage.",
		"file_root_public_desc":  "Shared public storage.",
		"file_root_account_desc": "Storage for the current account.",
	},
	ZhCN: {
		"file_short":             "查看远端文件",
		"file_long":              "列出用户、公共及当前账户存储空间中的文件。",
		"file_ls_short":          "列出远端文件",
		"err_file_path_invalid":  "无效的远端路径 %q",
		"err_file_path_root":     "远端路径必须以 user、public 或 account 开头：%q",
		"file_type_directory":    "目录",
		"file_type_regular":      "文件",
		"file_table_size":        "大小",
		"file_table_modified":    "修改时间",
		"file_root_user_desc":    "当前用户的私有存储空间",
		"file_root_public_desc":  "共享公共存储空间",
		"file_root_account_desc": "当前账户的存储空间",
	},
}
