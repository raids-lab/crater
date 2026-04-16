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
	switch toolName {
	case agentToolUncordonNode:
		confirmation.RiskLevel = "medium"
	case agentToolCordonNode, agentToolRestartWL:
		confirmation.RiskLevel = "high"
	case agentToolDrainNode, agentToolDeletePod, agentToolRunOpsScript:
		confirmation.RiskLevel = "critical"
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
	case agentToolCordonNode:
		nodeName := getToolArgString(args, "node_name", "")
		reason := getToolArgString(args, "reason", "")
		target := nodeName
		if target == "" {
			target = "<未指定节点>"
		}
		lines := []string{
			fmt.Sprintf("操作：将节点 %s 标记为不可调度。", target),
			"影响：新 Pod/新作业将不再调度到该节点，当前已运行负载通常不会被立刻驱逐。",
			"建议先确认：该节点是否正用于问题隔离、维护准备或升级前冻结。",
		}
		if reason != "" {
			lines = append(lines, fmt.Sprintf("原因：%s", reason))
		}
		return strings.Join(lines, "\n")
	case agentToolUncordonNode:
		nodeName := getToolArgString(args, "node_name", "")
		reason := getToolArgString(args, "reason", "")
		target := nodeName
		if target == "" {
			target = "<未指定节点>"
		}
		lines := []string{
			fmt.Sprintf("操作：恢复节点 %s 的调度。", target),
			"影响：新的 Pod/作业会重新落到该节点；如果节点仍有硬件或驱动问题，可能再次放大故障范围。",
			"建议先确认：节点 Ready、关键 DaemonSet、GPU/网络/存储相关组件已恢复正常。",
		}
		if reason != "" {
			lines = append(lines, fmt.Sprintf("说明：%s", reason))
		}
		return strings.Join(lines, "\n")
	case agentToolDrainNode:
		nodeName := getToolArgString(args, "node_name", "")
		reason := getToolArgString(args, "reason", "")
		target := nodeName
		if target == "" {
			target = "<未指定节点>"
		}
		lines := []string{
			fmt.Sprintf("操作：排空节点 %s 并禁止调度。", target),
			"影响：该节点上的可驱逐 Pod 会被迁移或中断，训练/服务可能出现短时抖动；受 PDB、local data、daemonset 约束时可能无法完全排空。",
			"建议先确认：关键作业是否已知情，是否允许短时中断，是否已准备后续重启/升级/检修动作。",
		}
		if reason != "" {
			lines = append(lines, fmt.Sprintf("原因：%s", reason))
		}
		return strings.Join(lines, "\n")
	case agentToolDeletePod:
		name := getToolArgString(args, "name", "")
		namespace := getToolArgString(args, "namespace", "")
		target := name
		if target == "" {
			target = "<未指定 Pod>"
		}
		if namespace != "" {
			target = fmt.Sprintf("%s/%s", namespace, name)
		}
		lines := []string{
			fmt.Sprintf("操作：删除 Pod %s。", target),
			"影响：控制器可能立即重建该 Pod；如果问题来自配置、镜像或节点本身，删除后可能仍会失败。未受控制器管理的 Pod 删除后不会自动恢复。",
			"建议先确认：是否已查看事件/日志，是否确认该 Pod 可被安全重建。",
		}
		if force, ok := args["force"].(bool); ok && force {
			lines = append(lines, "附加风险：将使用强制删除，可能跳过优雅退出。")
		}
		return strings.Join(lines, "\n")
	case agentToolRestartWL:
		kind := getToolArgString(args, "kind", "")
		name := getToolArgString(args, "name", "")
		namespace := getToolArgString(args, "namespace", "")
		target := name
		if target == "" {
			target = "<未指定工作负载>"
		}
		if namespace != "" {
			target = fmt.Sprintf("%s/%s", namespace, name)
		}
		kindText := kind
		if kindText == "" {
			kindText = "工作负载"
		}
		lines := []string{
			fmt.Sprintf("操作：滚动重启 %s %s。", kindText, target),
			"影响：对应 Pod 会逐步重建；如果副本数不足、配置错误或镜像有问题，服务/监控可能出现不可用窗口。",
			"建议先确认：当前是否在业务低峰，副本数和就绪策略是否足以承受滚动重启。",
		}
		return strings.Join(lines, "\n")
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
	case agentToolRunOpsScript:
		scriptName := getToolArgString(args, "script_name", "")
		target := scriptName
		if target == "" {
			target = "<未指定脚本>"
		}
		return strings.Join([]string{
			fmt.Sprintf("操作：在沙箱中执行运维脚本 %s。", target),
			"影响：会触发受白名单约束的自动化运维动作，可能修改集群或节点状态。",
			"建议先确认：脚本名称、目标范围、预期副作用和回退方式都已明确。",
		}, "\n")
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
	case agentToolCordonNode:
		if nodeName, _ := resultMap["node_name"].(string); nodeName != "" {
			return fmt.Sprintf("已将节点 %s 标记为不可调度。", nodeName)
		}
		return "已将节点标记为不可调度。"
	case agentToolUncordonNode:
		if nodeName, _ := resultMap["node_name"].(string); nodeName != "" {
			return fmt.Sprintf("已恢复节点 %s 的调度。", nodeName)
		}
		return "已恢复节点调度。"
	case agentToolDrainNode:
		if nodeName, _ := resultMap["node_name"].(string); nodeName != "" {
			return fmt.Sprintf("已开始排空节点 %s。", nodeName)
		}
		return "已开始排空节点。"
	case agentToolDeletePod:
		name, _ := resultMap["name"].(string)
		namespace, _ := resultMap["namespace"].(string)
		if name != "" && namespace != "" {
			return fmt.Sprintf("已删除 Pod %s/%s。", namespace, name)
		}
		if name != "" {
			return fmt.Sprintf("已删除 Pod %s。", name)
		}
		return "已删除目标 Pod。"
	case agentToolRestartWL:
		kind, _ := resultMap["kind"].(string)
		name, _ := resultMap["name"].(string)
		namespace, _ := resultMap["namespace"].(string)
		if kind != "" && name != "" && namespace != "" {
			return fmt.Sprintf("已触发 %s %s/%s 的滚动重启。", kind, namespace, name)
		}
		if kind != "" && name != "" {
			return fmt.Sprintf("已触发 %s %s 的滚动重启。", kind, name)
		}
		return "已触发工作负载滚动重启。"
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
	case agentToolRunOpsScript:
		jobName, _ := resultMap["sandbox_job_name"].(string)
		if jobName != "" {
			return fmt.Sprintf("已提交运维脚本任务 %s，等待沙箱执行结果。", jobName)
		}
		return "已提交运维脚本任务。"
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
