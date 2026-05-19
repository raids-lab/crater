package output

import "encoding/json"

// JSONIndent 为 CLI 结构化输出（stdout / stderr 的 --json）统一使用的缩进。
const JSONIndent = "  "

// MarshalJSONPretty 将 v 编码为带缩进与尾随换行的 JSON 字节（便于人类阅读；message 内换行仍为 \n 转义）。
func MarshalJSONPretty(v interface{}) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", JSONIndent)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
