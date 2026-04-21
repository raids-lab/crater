package output

import (
	"encoding/json"
	"io"

	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

// JSONSuccessStatus 是成功 JSON 顶层 status 的固定取值。
const JSONSuccessStatus = "OK"

// SuccessEnvelope 返回 {"status":"OK","data": data}。
func SuccessEnvelope(data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"status": JSONSuccessStatus,
		"data":   data,
	}
}

// WriteSuccessJSON 将成功体写入 writer（Pretty-printed）。
// 编码失败时返回 *clierror.Error（system_error + ERR_JSON_ENCODE_FAILED）。
func WriteSuccessJSON(w io.Writer, v interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrJSONEncodeFailed,
			Message:  i18n.T("err_json_encode", err.Error()),
			Context:  map[string]interface{}{"msg": err.Error()},
		}
	}
	return nil
}

