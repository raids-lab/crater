package clierror

// Error 是 CLI 内部使用的结构化错误类型。
//
// - `cmd` 负责构造并返回该错误
// - `cmd/root.go` 负责基于 Category 计算退出码
// - `internal/output` 负责将其渲染为 stderr 文本或 JSON
type Error struct {
	Category string
	Code     string
	Message  string
	Context  map[string]interface{}
}

func (e *Error) Error() string {
	return e.Message
}

