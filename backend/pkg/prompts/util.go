package prompts

import (
	"bytes"
	"embed"
	"text/template"
)

// Render 是一个通用的渲染函数
// 它接收一个 embed.FS（由具体的子包提供）和文件名
func Render(fs embed.FS, name string, data any) (string, error) {
	tmpl, err := template.ParseFS(fs, name)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
