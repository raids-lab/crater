package cmd

import (
	"errors"
	"net/http"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

// apiCodeForHTTP 将 HTTP 状态映射为 SPEC 约定的 ERR_* 字符串（含档位后缀）。
func apiCodeForHTTP(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return errorcodes.ErrUnauthorized401
	case http.StatusForbidden:
		return errorcodes.ErrForbidden403
	case http.StatusNotFound:
		return errorcodes.ErrNotFound404
	default:
		if status >= 500 && status <= 599 {
			return errorcodes.ErrServerInternal5XX
		}
		if status >= 400 && status < 500 {
			return errorcodes.ErrClient4XX
		}
	}
	return errorcodes.ErrAPIOther
}

func apiContextFromRequest(e *api.RequestError) map[string]interface{} {
	ctx := map[string]interface{}{
		"http_status": e.HTTPStatus,
	}
	if e.CraterCode != 0 {
		ctx["crater_code"] = e.CraterCode
	}
	if e.Msg != "" {
		ctx["msg"] = e.Msg
	}
	return ctx
}

// cliErrFromAPIRequest 将 internal/api 的 RequestError 转为带 Context 的 *clierror.Error。
func cliErrFromAPIRequest(e *api.RequestError) *clierror.Error {
	upstream := e.Msg
	if upstream == "" {
		upstream = "-"
	}
	return &clierror.Error{
		Category: errorcodes.CategoryAPI,
		Code:     apiCodeForHTTP(e.HTTPStatus),
		Message:  i18n.T("err_api_request", e.HTTPStatus, upstream),
		Context:  apiContextFromRequest(e),
	}
}

// cliErrFromAPINetwork 将 NetworkError 转为 *clierror.Error（无 http_status）。
func cliErrFromAPINetwork(e *api.NetworkError) *clierror.Error {
	return &clierror.Error{
		Category: errorcodes.CategoryAPI,
		Code:     errorcodes.ErrNetworkFailure,
		Message:  i18n.T("err_network", e.Cause),
		Context:  map[string]interface{}{"msg": e.Error()},
	}
}

// cliErrFromLoginAPI 映射 Login 返回的 api 错误；未知类型退化为 api_error + ERR_API_OTHER。
func cliErrFromLoginAPI(err error) *clierror.Error {
	var req *api.RequestError
	if errors.As(err, &req) {
		return cliErrFromAPIRequest(req)
	}
	var net *api.NetworkError
	if errors.As(err, &net) {
		return cliErrFromAPINetwork(net)
	}
	return &clierror.Error{
		Category: errorcodes.CategoryAPI,
		Code:     errorcodes.ErrAPIOther,
		Message:  i18n.T("err_api_other", err.Error()),
		Context:  map[string]interface{}{"msg": err.Error()},
	}
}

func errOperationCancelled() *clierror.Error {
	return &clierror.Error{
		Category: errorcodes.CategoryCancelled,
		Code:     errorcodes.ErrOperationCancelled,
		Message:  i18n.T("err_operation_cancelled"),
	}
}

// errSurveyOrSame 将 Survey 在终端被中断（如 Ctrl+C）映射为用户取消；其它错误原样返回。
func errSurveyOrSame(err error) error {
	if err == nil {
		return nil
	}
	if err.Error() == "interrupt" {
		return errOperationCancelled()
	}
	return err
}
