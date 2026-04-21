package api

import "fmt"

// RequestError 表示已收到 HTTP 响应但请求未成功，供命令层映射为 *clierror.Error（含 http_status、crater_code、msg）。
type RequestError struct {
	HTTPStatus int
	CraterCode int
	Msg        string
}

func (e *RequestError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("API error (%d): %s", e.HTTPStatus, e.Msg)
	}
	return fmt.Sprintf("API error: status code %d", e.HTTPStatus)
}

// NetworkError 表示未拿到有效 HTTP 语义（如连接失败、超时），不应设置 http_status Context。
type NetworkError struct {
	Cause error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %v", e.Cause)
}

func (e *NetworkError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
