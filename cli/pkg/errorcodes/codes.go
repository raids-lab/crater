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
// 未识别为 *clierror.Error 的错误使用 ExitFailure。
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

	// API Errors（与 HTTP 语义对齐的后缀约定，见 docs/SPEC.md）
	ErrUnauthorized401   = "ERR_UNAUTHORIZED_401"
	ErrForbidden403      = "ERR_FORBIDDEN_403"
	ErrNotFound404       = "ERR_NOT_FOUND_404"
	ErrClient4XX         = "ERR_CLIENT_4XX" // 除 401/403/404 外的客户端 4xx（含 400、409 等），与 _5XX 对称
	ErrServerInternal5XX = "ERR_SERVER_INTERNAL_5XX"
	// ErrAPIOther 表示 api_error，但 HTTP 档位不属于已约定的 401/403/404、4xx 桶、5xx 桶（如 1xx/2xx/3xx/0），或错误类型无法归类时的后备。
	ErrAPIOther           = "ERR_API_OTHER"
	ErrAPIVersionMismatch = "ERR_API_VERSION_MISMATCH"
	// ErrNetworkFailure 表示 API 请求在传输阶段失败，未拿到有效 HTTP 语义。
	ErrNetworkFailure = "ERR_NETWORK_FAILURE"

	// ErrNotFound 用于 usage_error 等本地状态「未找到」，不带 HTTP 后缀。
	ErrNotFound = "ERR_NOT_FOUND"

	// System Errors
	ErrConfigWriteFailed  = "ERR_CONFIG_WRITE_FAILED"
	ErrSecureStorageError = "ERR_SECURE_STORAGE_ERROR"
	ErrBinaryNotFound     = "ERR_BINARY_NOT_FOUND"
	ErrJSONEncodeFailed   = "ERR_JSON_ENCODE_FAILED" // 成功体写 stdout 时 json.Encode 失败
	ErrCommandExecution   = "ERR_COMMAND_EXECUTION"

	// User cancellation
	ErrOperationCancelled = "ERR_OPERATION_CANCELLED"
)
