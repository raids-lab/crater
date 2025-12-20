package gpu_analysis

import (
	"embed"

	"github.com/raids-lab/crater/pkg/prompts"
)

//go:embed *.prompt
var f embed.FS

func GetSystemPrompt(data any) (string, error) {
	// 调用父包导出的 Render 函数
	return prompts.Render(f, "system.prompt", data)
}

func GetPhase1Prompt(data any) (string, error) {
	// 调用父包导出的 Render 函数
	return prompts.Render(f, "phase1.prompt", data)
}

func GetPhase2Prompt(data any) (string, error) {
	return prompts.Render(f, "phase2.prompt", data)
}
