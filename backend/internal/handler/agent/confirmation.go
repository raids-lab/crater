package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

func (mgr *AgentMgr) buildToolConfirmation(token util.JWTMessage, toolName string, rawArgs json.RawMessage) AgentToolConfirmation {
	confirmation := AgentToolConfirmation{
		ToolName:    toolName,
		RiskLevel:   "high",
		Interaction: "approval",
	}
	switch toolName {
	case agentToolResubmitJob:
		confirmation.Interaction = "form"
		confirmation.Form = buildResubmitJobForm(rawArgs)
	case agentToolCreateJupyter:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateJupyterJobForm(rawArgs)
	case agentToolCreateTrain:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateTrainingJobForm(token, rawArgs)
	}
	confirmation.Description = mgr.buildConfirmationDescription(toolName, rawArgs)
	return confirmation
}

func buildResubmitJobForm(rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	var gpuCountDefault any
	if rawValue, ok := args["gpu_count"]; ok && rawValue != nil {
		gpuCountDefault = getToolArgInt(args, "gpu_count", 0)
	}

	return &AgentToolForm{
		Title:       "检查并重新提交作业",
		Description: "Agent 已定位到待重提的作业。你可以在这里修改显示名称和资源；留空的字段会沿用原配置。",
		SubmitLabel: "重新提交作业",
		Fields: []AgentToolField{
			{
				Key:          "name",
				Label:        "显示名称",
				Type:         "text",
				Description:  "留空则沿用原作业显示名称；系统内部 jobName 仍会自动生成。",
				DefaultValue: getToolArgString(args, "name", ""),
			},
			{
				Key:          "cpu",
				Label:        "CPU",
				Type:         "text",
				Description:  "留空则沿用原配置，例如 4 或 8000m。",
				DefaultValue: getToolArgString(args, "cpu", ""),
			},
			{
				Key:          "memory",
				Label:        "内存",
				Type:         "text",
				Description:  "留空则沿用原配置，例如 32Gi。",
				DefaultValue: getToolArgString(args, "memory", ""),
			},
			{
				Key:          "gpu_count",
				Label:        "GPU 数量",
				Type:         "number",
				Description:  "留空则沿用原配置；填 0 表示不申请 GPU。",
				DefaultValue: gpuCountDefault,
			},
			{
				Key:          "gpu_model",
				Label:        "GPU 型号",
				Type:         "text",
				Description:  "留空则沿用原配置，例如 v100 / a100。",
				DefaultValue: getToolArgString(args, "gpu_model", ""),
			},
		},
	}
}

func buildCreateJupyterJobForm(rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	var gpuCountDefault any
	if rawValue, ok := args["gpu_count"]; ok && rawValue != nil {
		gpuCountDefault = getToolArgInt(args, "gpu_count", 0)
	}

	return &AgentToolForm{
		Title:       "补全 Jupyter 作业配置",
		Description: "Agent 已生成一个 Jupyter 作业草案。你可以在这里确认或补全镜像与资源配置后提交。",
		SubmitLabel: "提交 Jupyter 作业",
		Fields: []AgentToolField{
			{Key: "name", Label: "作业名称", Type: "text", Required: true, DefaultValue: getToolArgString(args, "name", "")},
			{
				Key:          "image_link",
				Label:        "镜像",
				Type:         "text",
				Required:     true,
				Placeholder:  "registry/project/image:tag",
				DefaultValue: getToolArgString(args, "image_link", ""),
			},
			{
				Key:          "cpu",
				Label:        "CPU",
				Type:         "text",
				Required:     true,
				Description:  "默认 2。",
				DefaultValue: getToolArgString(args, "cpu", "2"),
			},
			{
				Key:          "memory",
				Label:        "内存",
				Type:         "text",
				Required:     true,
				Description:  "默认 8Gi。",
				DefaultValue: getToolArgString(args, "memory", "8Gi"),
			},
			{
				Key:          "gpu_count",
				Label:        "GPU 数量",
				Type:         "number",
				Description:  "可选，填 0 或留空表示不申请 GPU。",
				DefaultValue: gpuCountDefault,
			},
			{
				Key:          "gpu_model",
				Label:        "GPU 型号",
				Type:         "text",
				Description:  "可选，例如 v100 / a100。",
				DefaultValue: getToolArgString(args, "gpu_model", ""),
			},
		},
	}
}

func buildCreateTrainingJobForm(token util.JWTMessage, rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)

	defaultWorkingDir := "/workspace"
	if strings.TrimSpace(token.Username) != "" {
		defaultWorkingDir = fmt.Sprintf("/home/%s", token.Username)
	}

	return &AgentToolForm{
		Title:       "补全训练作业配置",
		Description: "Agent 已生成一个新作业草案，你可以在这里补全镜像、命令和资源后再提交。",
		SubmitLabel: "提交训练作业",
		Fields: []AgentToolField{
			{Key: "name", Label: "作业名称", Type: "text", Required: true, DefaultValue: getToolArgString(args, "name", "")},
			{Key: "image_link", Label: "镜像", Type: "text", Required: true, Placeholder: "registry/project/image:tag", DefaultValue: getToolArgString(args, "image_link", "")},
			{Key: "command", Label: "启动命令", Type: "textarea", Required: true, Placeholder: "python train.py --config ...", DefaultValue: getToolArgString(args, "command", "")},
			{Key: "working_dir", Label: "工作目录", Type: "text", Required: true, DefaultValue: getToolArgString(args, "working_dir", defaultWorkingDir)},
			{
				Key: "shell", Label: "Shell", Type: "select", DefaultValue: getToolArgString(args, "shell", "bash"),
				Options: []AgentToolFieldOption{
					{Value: "bash", Label: "bash"},
					{Value: "sh", Label: "sh"},
					{Value: "zsh", Label: "zsh"},
				},
			},
			{Key: "cpu", Label: "CPU", Type: "text", Required: true, DefaultValue: getToolArgString(args, "cpu", "4")},
			{Key: "memory", Label: "内存", Type: "text", Required: true, DefaultValue: getToolArgString(args, "memory", "16Gi")},
			{Key: "gpu_count", Label: "GPU 数量", Type: "number", DefaultValue: getToolArgInt(args, "gpu_count", 0)},
			{Key: "gpu_model", Label: "GPU 型号", Type: "text", Placeholder: "如 v100 / a100", DefaultValue: getToolArgString(args, "gpu_model", "")},
		},
	}
}

func (mgr *AgentMgr) buildConfirmationDescription(toolName string, rawArgs json.RawMessage) string {
	args := parseToolArgsMap(rawArgs)
	switch toolName {
	case agentToolStopJob:
		if jobName, ok := args["job_name"].(string); ok && jobName != "" {
			return fmt.Sprintf("停止作业 %s", jobName)
		}
		return "停止当前作业"
	case agentToolDeleteJob:
		if jobName, ok := args["job_name"].(string); ok && jobName != "" {
			return fmt.Sprintf("删除作业 %s", jobName)
		}
		return "删除当前作业"
	case agentToolResubmitJob:
		if jobName, ok := args["job_name"].(string); ok && jobName != "" {
			parts := []string{}
			if name := getToolArgString(args, "name", ""); name != "" {
				parts = append(parts, fmt.Sprintf("显示名称=%s", name))
			}
			if cpu := getToolArgString(args, "cpu", ""); cpu != "" {
				parts = append(parts, fmt.Sprintf("CPU=%s", cpu))
			}
			if memory := getToolArgString(args, "memory", ""); memory != "" {
				parts = append(parts, fmt.Sprintf("内存=%s", memory))
			}
			if gpuCount := getToolArgInt(args, "gpu_count", 0); gpuCount > 0 {
				gpuText := fmt.Sprintf("GPU=%d", gpuCount)
				if gpuModel := getToolArgString(args, "gpu_model", ""); gpuModel != "" {
					gpuText += fmt.Sprintf(" (%s)", gpuModel)
				}
				parts = append(parts, gpuText)
			}
			if len(parts) == 0 {
				return fmt.Sprintf("重新提交作业 %s", jobName)
			}
			return fmt.Sprintf("重新提交作业 %s，并应用覆盖：%s", jobName, strings.Join(parts, "，"))
		}
		return "重新提交当前作业"
	case agentToolCreateJupyter:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的 Jupyter 作业"
		}
		parts := []string{}
		if imageLink := getToolArgString(args, "image_link", ""); imageLink != "" {
			parts = append(parts, fmt.Sprintf("镜像=%s", imageLink))
		}
		if cpu := getToolArgString(args, "cpu", ""); cpu != "" {
			parts = append(parts, fmt.Sprintf("CPU=%s", cpu))
		}
		if memory := getToolArgString(args, "memory", ""); memory != "" {
			parts = append(parts, fmt.Sprintf("内存=%s", memory))
		}
		if gpuCount := getToolArgInt(args, "gpu_count", 0); gpuCount > 0 {
			gpuText := fmt.Sprintf("GPU=%d", gpuCount)
			if gpuModel := getToolArgString(args, "gpu_model", ""); gpuModel != "" {
				gpuText += fmt.Sprintf(" (%s)", gpuModel)
			}
			parts = append(parts, gpuText)
		}
		if len(parts) == 0 {
			return fmt.Sprintf("创建 Jupyter 作业 %s", name)
		}
		return fmt.Sprintf("创建 Jupyter 作业 %s：%s", name, strings.Join(parts, "，"))
	case agentToolCreateTrain:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的训练作业"
		}
		return fmt.Sprintf("创建训练作业 %s", name)
	case toolMarkAuditHandled:
		itemIDs := getToolArgString(args, "item_ids", "")
		if itemIDs != "" {
			return fmt.Sprintf("标记审计条目 %s 为已处理", itemIDs)
		}
		return "标记审计条目为已处理"
	case toolBatchStopJobs:
		jobNames := getToolArgString(args, "job_names", "")
		if jobNames != "" {
			return fmt.Sprintf("批量停止作业：%s", jobNames)
		}
		return "批量停止作业"
	case toolNotifyJobOwner:
		jobNames := getToolArgString(args, "job_names", "")
		if jobNames != "" {
			return fmt.Sprintf("通知作业所有者：%s", jobNames)
		}
		return "通知作业所有者"
	default:
		return fmt.Sprintf("执行操作 %s", toolName)
	}
}

func (mgr *AgentMgr) buildToolOutcomeMessage(toolName, status string, result any, fallback string) string {
	if status == "rejected" {
		return "已取消该操作。"
	}
	if status == agentToolStatusError {
		if fallback != "" {
			return fmt.Sprintf("%s 执行失败：%s", toolName, fallback)
		}
		return fmt.Sprintf("%s 执行失败。", toolName)
	}

	resultMap, _ := result.(map[string]any)
	switch toolName {
	case agentToolStopJob:
		if jobName, _ := resultMap["jobName"].(string); jobName != "" {
			return fmt.Sprintf("已停止作业 %s。", jobName)
		}
		return "已停止目标作业。"
	case agentToolDeleteJob:
		if jobName, _ := resultMap["jobName"].(string); jobName != "" {
			return fmt.Sprintf("已删除作业 %s。", jobName)
		}
		return "已删除目标作业。"
	case agentToolResubmitJob:
		sourceJobName, _ := resultMap["sourceJobName"].(string)
		jobName, _ := resultMap["jobName"].(string)
		if sourceJobName != "" && jobName != "" {
			return fmt.Sprintf("已基于 %s 重新提交作业 %s。", sourceJobName, jobName)
		}
		return "已重新提交作业。"
	case agentToolCreateJupyter:
		return "已提交新的 Jupyter 作业。"
	case agentToolCreateTrain:
		return "已提交新的训练作业。"
	case toolMarkAuditHandled:
		if updated, _ := resultMap["updated"].(float64); updated > 0 {
			return fmt.Sprintf("已标记 %.0f 条审计条目为已处理。", updated)
		}
		return "已标记审计条目为已处理。"
	case toolBatchStopJobs:
		if total, _ := resultMap["total"].(float64); total > 0 {
			return fmt.Sprintf("已批量停止 %.0f 个作业。", total)
		}
		return "已批量停止作业。"
	case toolNotifyJobOwner:
		if notified, _ := resultMap["notified"].(float64); notified > 0 {
			return fmt.Sprintf("已通知 %.0f 位作业所有者。", notified)
		}
		return "已通知作业所有者。"
	default:
		if fallback != "" {
			return fallback
		}
		return "操作已完成。"
	}
}

func (mgr *AgentMgr) persistAssistantToolMessage(ctx context.Context, sessionID, toolName, status, content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	metadata, _ := json.Marshal(map[string]any{
		"source":   "tool_confirmation",
		"toolName": toolName,
		"status":   status,
	})
	_ = mgr.agentService.SaveMessage(ctx, &model.AgentMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   content,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	})
}
