package output

// stderr.go：将错误渲染到 stderr（人类可读或格式化 JSON），与 success.go 对称。
// 人类可读模式下在「Error:」下对 Message 的每一行统一添加两格缩进；Message 内自带的前导空格会保留并叠加。

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

// humanErrorBodyIndent 为「Error:」之下每一行正文统一添加的基础缩进（两格）。
// 调用方可在 Message 内自行再加空格（例如列表 `  -`），本层会在每一行前再叠加上述缩进。
const humanErrorBodyIndent = "  "

type errorResponse struct {
	Category string                 `json:"category"`
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

const errorContextEncodeFailedMessage = "错误 context JSON 化失败，请联系开发者修复错误 context"

// WriteError 将 err 写入 writer（通常为 stderr：人类可读或格式化 JSON）。
// 注意：该函数只负责渲染，不负责计算退出码或 os.Exit。
func WriteError(w io.Writer, jsonEnabled bool, err error) {
	if jsonEnabled {
		writeErrorJSON(w, err)
		return
	}

	writeHumanError(w, err)
}

func writeErrorJSON(w io.Writer, err error) {
	category := errorcodes.CategorySystem
	code := errorcodes.ErrCommandExecution
	var context map[string]interface{}

	var ce *clierror.Error
	if errors.As(err, &ce) {
		category = ce.Category
		code = ce.Code
		context = ce.Context
	}

	resp := errorResponse{
		Category: category,
		Code:     code,
		Message:  err.Error(),
		Context:  context,
	}
	b, marshalErr := MarshalJSONPretty(resp)
	if marshalErr == nil {
		_, _ = w.Write(b)
		return
	}
	if resp.Context != nil {
		resp.Context = map[string]interface{}{
			"msg":          errorContextEncodeFailedMessage,
			"encode_error": marshalErr.Error(),
		}
		if b, err2 := MarshalJSONPretty(resp); err2 == nil {
			_, _ = w.Write(b)
			return
		}
	}
	resp = errorResponse{
		Category: errorcodes.CategorySystem,
		Code:     errorcodes.ErrJSONEncodeFailed,
		Message:  errorContextEncodeFailedMessage,
	}
	b, _ = MarshalJSONPretty(resp)
	_, _ = w.Write(b)
}

func writeHumanError(w io.Writer, err error) {
	msg := strings.ReplaceAll(err.Error(), "\r\n", "\n")
	msg = strings.ReplaceAll(msg, "\r", "\n")
	msg = strings.TrimRight(msg, "\n")

	fmt.Fprintln(w, "Error:")
	if msg == "" {
		return
	}
	for _, line := range strings.Split(msg, "\n") {
		fmt.Fprint(w, humanErrorBodyIndent+line+"\n")
	}
}
