package i18n

// Root command + global flags.
var catalogRoot = map[Language]map[string]string{
	En: {
		// Command Descriptions
		"root_short": "Crater CLI - Official command line tool for Crater platform",
		"root_long":  "Crater CLI is an AI-friendly command line tool designed for both humans and AI Agents to interact with the Crater platform.",

		// Flag Descriptions
		"flag_json":           "Output in raw JSON format",
		"flag_no-interactive": "Disable interactive prompts",
		"flag_help":           "Help for crater",
	},
	ZhCN: {
		// Command Descriptions
		"root_short": "Crater CLI - Crater 平台官方命令行工具",
		"root_long":  "Crater CLI 是一款对 AI 友好的命令行工具，旨在为人类和 AI Agent 提供与 Crater 平台交互的能力。",

		// Flag Descriptions
		"flag_json":           "以原始 JSON 格式输出",
		"flag_no-interactive": "禁用交互式提示",
		"flag_help":           "显示帮助信息",
	},
}

