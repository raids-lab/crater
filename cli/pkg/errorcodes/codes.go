package errorcodes

// Category 是稳定的错误大类，用于 JSON 输出与进程退出码映射（二者一一对应）。
// 新增大类应极少发生；细分原因请用 Code 字段表达。
const (
	// CategoryUsage 表示调用方输入或 CLI 使用方式问题：缺参、非法 flag、在当前状态下
	// 无法继续（例如未找到匹配项）等。修复方式通常是改命令行或配置，而非重试网络。
	CategoryUsage = "usage_error"

	// CategoryAPI 表示与远端 Crater 平台交互失败：HTTP/业务错误、登录鉴权失败、
	// 请求阶段的网络不可达（当前实现里 Login 等 API 错误均归此类）。
	CategoryAPI = "api_error"

	// CategorySystem 表示本机环境与本地依赖问题：读写配置文件、系统安全存储（Keyring）、
	// 本地权限等。不是「业务 API 返回 4xx/5xx」那类问题。
	CategorySystem = "system_error"

	// CategoryCancelled 表示用户在交互中主动中止（例如确认选 No），与可重试的
	// 网络错误或参数错误区分。
	CategoryCancelled = "cancelled"
)

// Process exit codes（与 Category 一一对应，便于 shell 脚本分支）。
// 未识别为 *CLIError 的错误使用 ExitFailure。
// [TODO]尚未实现
const (
	ExitSuccess   = 0
	ExitFailure   = 1
	ExitUsage     = 2
	ExitCancelled = 3
	ExitAPI       = 4
	ExitSystem    = 5
)

// ExitCodeForCategory returns the exit status for a CLI error category string.
func ExitCodeForCategory(category string) int {
	switch category {
	case CategoryUsage:
		return ExitUsage
	case CategoryAPI:
		return ExitAPI
	case CategorySystem:
		return ExitSystem
	case CategoryCancelled:
		return ExitCancelled
	default:
		return ExitFailure
	}
}

// Error Codes
const (
	// Usage Errors
	ErrMissingRequiredFlag    = "ERR_MISSING_REQUIRED_FLAG"
	ErrInvalidFlagValue       = "ERR_INVALID_FLAG_VALUE"
	ErrUnknownCommand         = "ERR_UNKNOWN_COMMAND"
	ErrMutuallyExclusiveFlags = "ERR_MUTUALLY_EXCLUSIVE_FLAGS"

	// API Errors
	ErrUnauthorized       = "ERR_UNAUTHORIZED"
	ErrForbidden          = "ERR_FORBIDDEN"
	ErrNotFound           = "ERR_NOT_FOUND"
	ErrServerInternal     = "ERR_SERVER_INTERNAL"
	ErrAPIVersionMismatch = "ERR_API_VERSION_MISMATCH"

	// System Errors
	ErrNetworkFailure     = "ERR_NETWORK_FAILURE"
	ErrConfigWriteFailed  = "ERR_CONFIG_WRITE_FAILED"
	ErrSecureStorageError = "ERR_SECURE_STORAGE_ERROR"
	ErrBinaryNotFound     = "ERR_BINARY_NOT_FOUND"
	ErrCommandExecution   = "ERR_COMMAND_EXECUTION"

	// User cancellation
	ErrOperationCancelled = "ERR_OPERATION_CANCELLED"
)
