package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
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

// cliErrFromAPI 将 internal/api 返回的错误映射为 CLI 稳定错误契约。
// 已知 API 错误会按 RequestError / NetworkError 保留结构化事实；未知错误退化为 ERR_API_OTHER。
func cliErrFromAPI(err error) *clierror.Error {
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

// errUnknownSubcommand returns a docker/git-style short error for a mistyped subcommand.
//
// - No usage is printed automatically (root sets SilenceUsage/SilenceErrors).
// - Exit code is derived from CategoryUsage (ExitUsage).
// - When --json is enabled, the error is rendered as JSON by internal/output.
func errUnknownSubcommand(parent *cobra.Command, typed string) *clierror.Error {
	if parent == nil {
		return &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrUnknownCommand,
			Message:  fmt.Sprintf("unknown command %q", typed),
		}
	}

	// Match Cobra's style, but keep output under our error/rendering pipeline.
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "unknown command %q for %q", typed, parent.CommandPath())

	// Cobra only computes Levenshtein suggestions when SuggestionsMinimumDistance > 0.
	// The upstream default (when unset) is effectively 2, but that default is applied
	// in an internal helper, not in SuggestionsFor itself. We mirror that behavior here.
	if parent.SuggestionsMinimumDistance <= 0 {
		parent.SuggestionsMinimumDistance = 2
	}
	if suggestions := parent.SuggestionsFor(typed); len(suggestions) > 0 {
		b.WriteString("\n\nDid you mean this?\n")
		for _, s := range suggestions {
			_, _ = fmt.Fprintf(&b, "\t%v\n", s)
		}
	}

	// Keep this line stable for users and snapshots.
	_, _ = fmt.Fprintf(&b, "\nRun %q for usage.", parent.CommandPath()+" --help")

	return &clierror.Error{
		Category: errorcodes.CategoryUsage,
		Code:     errorcodes.ErrUnknownCommand,
		Message:  b.String(),
	}
}
