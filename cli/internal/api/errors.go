package api

import (
	"fmt"
	"net/http"
)

// CodePermissionDenied is the Crater API business code for a forbidden action.
const CodePermissionDenied = 40301

// RequestError 表示已收到 HTTP 响应但请求未成功，供命令层映射为 *clierror.Error（含 http_status、crater_code、msg）。
type RequestError struct {
	HTTPStatus    int
	CraterCode    int
	Msg           string
	DuringRefresh bool
}

// ShouldSignInAgain distinguishes credential failures from other HTTP 401s.
func (e *RequestError) ShouldSignInAgain() bool {
	return e.HTTPStatus == http.StatusUnauthorized &&
		(e.CraterCode == tokenExpiredCode || e.CraterCode == tokenInvalidCode)
}

// SessionStateError indicates a failed check or update of saved credentials.
type SessionStateError struct{ Cause error }

func (e *SessionStateError) Error() string { return e.Cause.Error() }
func (e *SessionStateError) Unwrap() error { return e.Cause }

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
