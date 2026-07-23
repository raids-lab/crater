package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/raids-lab/crater/internal/util"
)

func (mgr *AgentMgr) buildToolConfirmation(_ util.JWTMessage, toolName string, rawArgs json.RawMessage) AgentToolConfirmation {
	confirmation := AgentToolConfirmation{
		ToolName:              toolName,
		RiskLevel:             "high",
		Interaction:           "approval",
		Description:           mgr.buildConfirmationDescription(toolName, rawArgs),
		PermissionExplanation: buildToolPermissionExplanation(toolName),
		RiskExplanation:       buildToolRiskExplanation(toolName),
		AffectedResources:     inferToolAffectedResources(rawArgs),
	}
	switch toolName {
	case agentToolCreateJupyter, agentToolCreateWebIDE, agentToolCreateCustom,
		agentToolCreatePytorch, agentToolCreateTensorflow, agentToolResubmitJob:
		confirmation.Interaction = "form"
		confirmation.Form = buildJobForm(toolName, rawArgs)
	}
	return confirmation
}

func buildToolPermissionExplanation(toolName string) string {
	switch toolName {
	case agentToolCreateJupyter, agentToolCreateWebIDE, agentToolCreateCustom,
		agentToolCreatePytorch, agentToolCreateTensorflow:
		return "需要使用你的作业创建权限提交新的工作负载，并申请表单中的资源。"
	case agentToolResubmitJob:
		return "需要读取原作业配置，并基于确认后的表单重新提交一个新作业。"
	case agentToolStopJob, agentToolDeleteJob:
		return "需要确认目标作业属于你或当前身份有权管理该作业。"
	default:
		return "这是一个需要显式确认的写操作；系统会在你确认后才以当前登录身份执行。"
	}
}

func buildToolRiskExplanation(toolName string) string {
	switch toolName {
	case agentToolDeleteJob:
		return "删除作业会停止相关工作负载并清除记录，请确认目标作业无误。"
	case agentToolStopJob:
		return "停止作业会中断当前运行任务，但不会创建新作业。"
	case agentToolResubmitJob:
		return "重提会创建新作业并重新申请资源；原作业不会被自动删除。"
	case agentToolCreateJupyter, agentToolCreateWebIDE, agentToolCreateCustom,
		agentToolCreatePytorch, agentToolCreateTensorflow:
		return "创建作业会占用账户配额和集群资源，配置错误可能导致排队或启动失败。"
	default:
		return "确认后系统会执行该写操作。"
	}
}

func inferToolAffectedResources(rawArgs json.RawMessage) []string {
	args := parseToolArgsMap(rawArgs)
	resources := make([]string, 0, 2)
	for _, key := range []string{"job_name", "jobName", "name"} {
		value := strings.TrimSpace(getToolArgString(args, key, ""))
		if value != "" {
			resources = append(resources, fmt.Sprintf("作业: %s", value))
			break
		}
	}
	return resources
}

func (mgr *AgentMgr) buildConfirmationDescription(toolName string, rawArgs json.RawMessage) string {
	args := parseToolArgsMap(rawArgs)
	switch toolName {
	case agentToolDeleteJob:
		return fmt.Sprintf("删除作业 %s", getToolArgString(args, "job_name", getToolArgString(args, "jobName", "")))
	case agentToolStopJob:
		return fmt.Sprintf("停止作业 %s", getToolArgString(args, "job_name", getToolArgString(args, "jobName", "")))
	case agentToolResubmitJob:
		return fmt.Sprintf("重新提交作业 %s", getToolArgString(args, "job_name", getToolArgString(args, "jobName", "")))
	case agentToolCreateJupyter:
		return "创建 Jupyter 作业"
	case agentToolCreateWebIDE:
		return "创建 WebIDE 作业"
	case agentToolCreateCustom:
		return "创建自定义作业"
	case agentToolCreatePytorch:
		return "创建 PyTorch 作业"
	case agentToolCreateTensorflow:
		return "创建 TensorFlow 作业"
	default:
		return toolName
	}
}

func buildJobForm(toolName string, rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	fields := []AgentToolField{
		{
			Key:          "name",
			Label:        "作业名称",
			Type:         "text",
			Required:     toolName != agentToolResubmitJob,
			DefaultValue: firstArg(args, "name", "job_name", "jobName"),
		},
		{
			Key:          "image_link",
			Label:        "镜像",
			Type:         "text",
			Required:     toolName != agentToolResubmitJob,
			DefaultValue: firstArg(args, "image_link", "image", "imageLink"),
		},
		{
			Key:          "cpu",
			Label:        "CPU",
			Type:         "text",
			Required:     false,
			DefaultValue: firstArg(args, "cpu"),
		},
		{
			Key:          "memory",
			Label:        "内存",
			Type:         "text",
			Required:     false,
			DefaultValue: firstArg(args, "memory"),
		},
		{
			Key:          "gpu_count",
			Label:        "GPU 数量",
			Type:         "number",
			Required:     false,
			DefaultValue: firstArg(args, "gpu_count", "gpu"),
		},
		{
			Key:          "gpu_model",
			Label:        "GPU 型号",
			Type:         "select",
			Required:     false,
			DefaultValue: firstArg(args, "gpu_model", "gpuModel"),
			Options: []AgentToolFieldOption{
				{Label: "不指定", Value: ""},
				{Label: "V100", Value: "v100"},
				{Label: "A100", Value: "a100"},
				{Label: "H100", Value: "h100"},
				{Label: "L40S", Value: "l40s"},
				{Label: "RTX4090", Value: "rtx4090"},
			},
		},
	}
	if toolName == agentToolCreateCustom || toolName == agentToolCreatePytorch || toolName == agentToolCreateTensorflow {
		fields = append(fields, AgentToolField{
			Key:          "command",
			Label:        "启动命令",
			Type:         "textarea",
			Required:     toolName == agentToolCreateCustom,
			DefaultValue: firstArg(args, "command"),
		})
	}
	return &AgentToolForm{
		Title:       "确认作业配置",
		Description: "可在执行前调整关键作业参数。",
		Fields:      fields,
		SubmitLabel: "确认执行",
	}
}

func firstArg(args map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(getToolArgString(args, key, "")); value != "" {
			return value
		}
	}
	return ""
}
