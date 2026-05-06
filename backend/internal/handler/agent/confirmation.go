package agent

import (
	"encoding/json"
	"fmt"
	"strings"

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
	case agentToolCreateWebIDE:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateWebIDEJobForm(rawArgs)
	case agentToolCreateTrain:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateTrainingJobForm(token, rawArgs)
	case agentToolCreateCustom:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateCustomJobForm(token, rawArgs)
	case agentToolCreatePytorch:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreatePytorchJobForm(rawArgs)
	case agentToolCreateTensorflow:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateTensorflowJobForm(rawArgs)
	case agentToolCreateImage:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateImageBuildForm(rawArgs)
	case agentToolRegisterImage:
		confirmation.Interaction = "form"
		confirmation.Form = buildRegisterExternalImageForm(rawArgs)
	}
	switch toolName {
	case agentToolManageAccess:
		confirmation.RiskLevel = "medium"
	case toolNotifyJobOwner:
		confirmation.RiskLevel = "medium"
	case agentToolRegisterImage:
		confirmation.RiskLevel = "medium"
	case agentToolK8sScaleWL, agentToolK8sLabelNode, agentToolK8sTaintNode:
		confirmation.RiskLevel = "high"
	case agentToolCreateImage, agentToolManageBuild:
		confirmation.RiskLevel = "high"
	case agentToolUncordonNode:
		confirmation.RiskLevel = "medium"
	case agentToolCordonNode, agentToolRestartWL:
		confirmation.RiskLevel = "high"
	case agentToolDrainNode, agentToolDeletePod, agentToolRunKubectl, agentToolAdminCommand:
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
			{
				Key:          "forwards_text",
				Label:        "附加端口暴露",
				Type:         "textarea",
				Description:  "可选。每行一个规则，格式 name:port[:ingress|nodeport]，例如 grpc:19530 或 http:9091:nodeport。",
				DefaultValue: formatForwardTextDefault(args),
			},
		},
	}
}

func buildCreateWebIDEJobForm(rawArgs json.RawMessage) *AgentToolForm {
	form := buildCreateJupyterJobForm(rawArgs)
	if form == nil {
		return nil
	}
	form.Title = "补全 WebIDE 作业配置"
	form.Description = "Agent 已生成一个 WebIDE 作业草案。你可以在这里确认镜像、资源和附加端口暴露后提交。"
	form.SubmitLabel = "提交 WebIDE 作业"
	return form
}

func buildCreateTrainingJobForm(token util.JWTMessage, rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)

	defaultWorkingDir := "/workspace"
	if strings.TrimSpace(token.Username) != "" {
		defaultWorkingDir = fmt.Sprintf("/home/%s", token.Username)
	}

	return &AgentToolForm{
		Title:       "补全自定义作业配置",
		Description: "Agent 已生成一个自定义作业草案，你可以在这里补全镜像、命令、资源和附加端口暴露后再提交。",
		SubmitLabel: "提交自定义作业",
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
			{
				Key:          "forwards_text",
				Label:        "附加端口暴露",
				Type:         "textarea",
				Description:  "可选。每行一个规则，格式 name:port[:ingress|nodeport]，例如 grpc:19530 或 http:9091:nodeport。",
				DefaultValue: formatForwardTextDefault(args),
			},
		},
	}
}

func buildCreateCustomJobForm(token util.JWTMessage, rawArgs json.RawMessage) *AgentToolForm {
	return buildCreateTrainingJobForm(token, rawArgs)
}

func buildCreatePytorchJobForm(rawArgs json.RawMessage) *AgentToolForm {
	return buildCreateDistributedJobForm(
		rawArgs,
		"补全 PyTorch 分布式作业配置",
		"Agent 已生成一个 PyTorch 分布式作业草案。你可以在这里检查任务拓扑、镜像、资源与附加端口暴露后提交。",
		"提交 PyTorch 作业",
		defaultPytorchTasksJSON(),
	)
}

func buildCreateTensorflowJobForm(rawArgs json.RawMessage) *AgentToolForm {
	return buildCreateDistributedJobForm(
		rawArgs,
		"补全 TensorFlow 分布式作业配置",
		"Agent 已生成一个 TensorFlow 分布式作业草案。你可以在这里检查任务拓扑、镜像、资源与附加端口暴露后提交。",
		"提交 TensorFlow 作业",
		defaultTensorflowTasksJSON(),
	)
}

func buildCreateDistributedJobForm(
	rawArgs json.RawMessage,
	title string,
	description string,
	submitLabel string,
	defaultTasks string,
) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	return &AgentToolForm{
		Title:       title,
		Description: description,
		SubmitLabel: submitLabel,
		Fields: []AgentToolField{
			{Key: "name", Label: "作业名称", Type: "text", Required: true, DefaultValue: getToolArgString(args, "name", "")},
			{
				Key:          "tasks_json",
				Label:        "任务拓扑 JSON",
				Type:         "textarea",
				Required:     true,
				Description:  "填写 JSON 数组。每个任务可包含 name / replicas / image_link / command / working_dir / cpu / memory / gpu_count / gpu_model / ports。",
				DefaultValue: formatDistributedTasksDefault(args, defaultTasks),
			},
			{
				Key:          "forwards_text",
				Label:        "附加端口暴露",
				Type:         "textarea",
				Description:  "可选。每行一个规则，格式 name:port[:ingress|nodeport]，例如 dashboard:6006:ingress。",
				DefaultValue: formatForwardTextDefault(args),
			},
		},
	}
}

func formatForwardTextDefault(args map[string]any) string {
	items, ok := args["forwards"].([]any)
	if ok {
		lines := make([]string, 0, len(items))
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := getToolArgString(entry, "name", "")
			port := getToolArgInt(entry, "port", 0)
			if name == "" || port <= 0 {
				continue
			}
			forwardType, _ := normalizeForwardTypeValue(entry["type"])
			typeLabel := "ingress"
			if forwardType == agentForwardTypeNodePort {
				typeLabel = "nodeport"
			}
			lines = append(lines, fmt.Sprintf("%s:%d:%s", name, port, typeLabel))
		}
		if len(lines) > 0 {
			return strings.Join(lines, "\n")
		}
	}
	return getToolArgString(args, "forwards_text", "")
}

func formatDistributedTasksDefault(args map[string]any, fallback string) string {
	if raw, ok := args["tasks"]; ok {
		if encoded, err := json.MarshalIndent(raw, "", "  "); err == nil {
			return string(encoded)
		}
	}
	if value := getToolArgString(args, "tasks_json", ""); value != "" {
		return value
	}
	return fallback
}

func defaultPytorchTasksJSON() string {
	return `[
  {
    "name": "master",
    "replicas": 1,
    "image_link": "crater-harbor.example.com/platform/pytorch:latest",
    "command": "torchrun --nnodes=2 --nproc_per_node=8 --node_rank=0 --master_addr=${MASTER_ADDR} --master_port=23456 train.py",
    "working_dir": "/workspace",
    "cpu": "8",
    "memory": "32Gi",
    "gpu_count": 8,
    "gpu_model": "a100",
    "ports": [{"name": "master", "port": 23456}]
  },
  {
    "name": "worker",
    "replicas": 1,
    "image_link": "crater-harbor.example.com/platform/pytorch:latest",
    "command": "torchrun --nnodes=2 --nproc_per_node=8 --node_rank=1 --master_addr=${MASTER_ADDR} --master_port=23456 train.py",
    "working_dir": "/workspace",
    "cpu": "8",
    "memory": "32Gi",
    "gpu_count": 8,
    "gpu_model": "a100"
  }
]`
}

func defaultTensorflowTasksJSON() string {
	return `[
  {
    "name": "chief",
    "replicas": 1,
    "image_link": "crater-harbor.example.com/platform/tensorflow:latest",
    "command": "python train.py --task chief",
    "working_dir": "/workspace",
    "cpu": "8",
    "memory": "32Gi",
    "gpu_count": 1,
    "gpu_model": "a100",
    "ports": [{"name": "grpc", "port": 2222}]
  },
  {
    "name": "worker",
    "replicas": 2,
    "image_link": "crater-harbor.example.com/platform/tensorflow:latest",
    "command": "python train.py --task worker",
    "working_dir": "/workspace",
    "cpu": "8",
    "memory": "32Gi",
    "gpu_count": 1,
    "gpu_model": "a100"
  }
]`
}

func buildCreateImageBuildForm(rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	mode := getToolArgString(args, "mode", string(agentImageBuildModePipApt))
	fields := []AgentToolField{
		{
			Key:          "mode",
			Label:        "构建模式",
			Type:         "select",
			Required:     true,
			DefaultValue: mode,
			Options: []AgentToolFieldOption{
				{Value: string(agentImageBuildModePipApt), Label: "Pip + APT"},
				{Value: string(agentImageBuildModeDocker), Label: "Dockerfile"},
				{Value: string(agentImageBuildModeEnvd), Label: "Envd"},
				{Value: string(agentImageBuildModeEnvdRaw), Label: "Envd Raw"},
			},
		},
		{Key: "description", Label: "描述", Type: "text", Required: true, DefaultValue: getToolArgString(args, "description", "")},
		{Key: "image_name", Label: "镜像名", Type: "text", DefaultValue: getToolArgString(args, "image_name", "")},
		{Key: "image_tag", Label: "镜像标签", Type: "text", DefaultValue: getToolArgString(args, "image_tag", "")},
		{Key: "tags_csv", Label: "标签", Type: "text", Description: "多个标签请用英文逗号分隔。", DefaultValue: strings.Join(getToolArgStringSlice(args, "tags"), ", ")},
		{
			Key:          "archs_csv",
			Label:        "架构",
			Type:         "text",
			Description:  "多个架构请用英文逗号分隔，默认 linux/amd64。",
			DefaultValue: strings.Join(parseCSVBackedSlice(args, "archs", "archs_csv", []string{"linux/amd64"}), ", "),
		},
	}

	switch mode {
	case string(agentImageBuildModeDocker):
		fields = append(fields, AgentToolField{
			Key:          "dockerfile",
			Label:        "Dockerfile",
			Type:         "textarea",
			Required:     true,
			DefaultValue: getToolArgString(args, "dockerfile", ""),
		})
	case string(agentImageBuildModeEnvd):
		fields = append(fields,
			AgentToolField{Key: "python_version", Label: "Python 版本", Type: "text", Required: true, DefaultValue: getToolArgString(args, "python_version", getToolArgString(args, "python", "3.10"))},
			AgentToolField{Key: "cuda_base", Label: "CUDA 基础镜像", Type: "text", Required: true, Description: "可直接填 list_cuda_base_images 返回的 value 或 imageLabel。", DefaultValue: getToolArgString(args, "cuda_base", getToolArgString(args, "base", ""))},
			AgentToolField{Key: "requirements", Label: "Python 依赖", Type: "textarea", DefaultValue: getToolArgString(args, "requirements", "")},
			AgentToolField{Key: "apt_packages_text", Label: "APT 包", Type: "text", Description: "多个包可用空格或逗号分隔。", DefaultValue: strings.Join(parseCSVBackedSlice(args, "apt_packages", "apt_packages_text", nil), " ")},
			AgentToolField{
				Key:          "enable_jupyter",
				Label:        "启用 Jupyter",
				Type:         "select",
				DefaultValue: fmt.Sprintf("%t", getToolArgBool(args, "enable_jupyter")),
				Options: []AgentToolFieldOption{
					{Value: "true", Label: "是"},
					{Value: "false", Label: "否"},
				},
			},
			AgentToolField{
				Key:          "enable_zsh",
				Label:        "启用 Zsh",
				Type:         "select",
				DefaultValue: fmt.Sprintf("%t", getToolArgBool(args, "enable_zsh")),
				Options: []AgentToolFieldOption{
					{Value: "true", Label: "是"},
					{Value: "false", Label: "否"},
				},
			},
		)
	case string(agentImageBuildModeEnvdRaw):
		fields = append(fields, AgentToolField{
			Key:          "envd_script",
			Label:        "Envd 脚本",
			Type:         "textarea",
			Required:     true,
			DefaultValue: getToolArgString(args, "envd_script", getToolArgString(args, "envd", "")),
		})
	default:
		fields = append(fields,
			AgentToolField{Key: "base_image", Label: "基础镜像", Type: "text", Required: true, DefaultValue: getToolArgString(args, "base_image", getToolArgString(args, "image", ""))},
			AgentToolField{Key: "requirements", Label: "Python 依赖", Type: "textarea", DefaultValue: getToolArgString(args, "requirements", "")},
			AgentToolField{Key: "apt_packages_text", Label: "APT 包", Type: "text", Description: "多个包可用空格或逗号分隔。", DefaultValue: strings.Join(parseCSVBackedSlice(args, "apt_packages", "apt_packages_text", nil), " ")},
		)
	}

	return &AgentToolForm{
		Title:       "补全镜像构建参数",
		Description: "Agent 已生成一个镜像构建草案。请确认构建模式、描述和模式相关参数后提交。",
		SubmitLabel: "提交镜像构建",
		Fields:      fields,
	}
}

func buildRegisterExternalImageForm(rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	return &AgentToolForm{
		Title:       "登记外部镜像",
		Description: "将 Harbor / OCI 中已存在的镜像注册到平台，后续可被作业和分享流程复用。",
		SubmitLabel: "登记镜像",
		Fields: []AgentToolField{
			{Key: "image_link", Label: "镜像链接", Type: "text", Required: true, DefaultValue: getToolArgString(args, "image_link", "")},
			{Key: "description", Label: "描述", Type: "text", Required: true, DefaultValue: getToolArgString(args, "description", "")},
			{
				Key:          "task_type",
				Label:        "任务类型",
				Type:         "select",
				Required:     true,
				DefaultValue: getToolArgString(args, "task_type", "custom"),
				Options: []AgentToolFieldOption{
					{Value: "custom", Label: "custom"},
					{Value: "jupyter", Label: "jupyter"},
					{Value: "pytorch", Label: "pytorch"},
					{Value: "tensorflow", Label: "tensorflow"},
					{Value: "deepspeed", Label: "deepspeed"},
					{Value: "openmpi", Label: "openmpi"},
					{Value: "ray", Label: "ray"},
				},
			},
			{Key: "tags_csv", Label: "标签", Type: "text", Description: "多个标签请用英文逗号分隔。", DefaultValue: strings.Join(getToolArgStringSlice(args, "tags"), ", ")},
			{Key: "archs_csv", Label: "架构", Type: "text", Description: "多个架构请用英文逗号分隔，默认 linux/amd64。", DefaultValue: strings.Join(parseCSVBackedSlice(args, "archs", "archs_csv", []string{"linux/amd64"}), ", ")},
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
	case agentToolCreateWebIDE:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的 WebIDE 作业"
		}
		return fmt.Sprintf("创建 WebIDE 作业 %s", name)
	case agentToolCreateImage:
		mode := getToolArgString(args, "mode", "")
		description := getToolArgString(args, "description", "")
		imageName := getToolArgString(args, "image_name", "")
		imageTag := getToolArgString(args, "image_tag", "")
		target := imageName
		if imageTag != "" {
			target = fmt.Sprintf("%s:%s", imageName, imageTag)
		}
		if target == "" {
			target = "自动命名镜像"
		}
		if description == "" {
			description = "未填写描述"
		}
		return fmt.Sprintf("创建镜像构建任务（模式=%s，目标=%s，描述=%s）", mode, target, description)
	case agentToolManageBuild:
		action := getToolArgString(args, "action", "")
		if buildID := getToolArgInt(args, "build_id", 0); buildID > 0 {
			return fmt.Sprintf("对镜像构建 #%d 执行动作 %s", buildID, action)
		}
		if imagePackName := getToolArgString(args, "imagepack_name", getToolArgString(args, "image_pack_name", "")); imagePackName != "" {
			return fmt.Sprintf("对镜像构建 %s 执行动作 %s", imagePackName, action)
		}
		return fmt.Sprintf("执行镜像构建动作 %s", action)
	case agentToolRegisterImage:
		imageLink := getToolArgString(args, "image_link", "")
		taskType := getToolArgString(args, "task_type", "custom")
		return fmt.Sprintf("登记外部镜像 %s（任务类型=%s）", imageLink, taskType)
	case agentToolManageAccess:
		action := getToolArgString(args, "action", "")
		targetType := getToolArgString(args, "target_type", "")
		targets := parseFlexibleTargets(args)
		imageRef := getToolArgString(args, "image_link", "")
		if imageID := getToolArgInt(args, "image_id", 0); imageID > 0 {
			imageRef = fmt.Sprintf("image#%d", imageID)
		}
		return fmt.Sprintf("对镜像 %s 执行 %s %s 权限，目标：%s", imageRef, action, targetType, strings.Join(targets, "，"))
	case agentToolCreateTrain:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的自定义作业"
		}
		return fmt.Sprintf("创建自定义作业 %s", name)
	case agentToolCreateCustom:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的自定义作业"
		}
		return fmt.Sprintf("创建自定义作业 %s", name)
	case agentToolCreatePytorch:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的 PyTorch 分布式作业"
		}
		return fmt.Sprintf("创建 PyTorch 分布式作业 %s", name)
	case agentToolCreateTensorflow:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的 TensorFlow 分布式作业"
		}
		return fmt.Sprintf("创建 TensorFlow 分布式作业 %s", name)
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
	case agentToolK8sScaleWL:
		kind := getToolArgString(args, "kind", "")
		name := getToolArgString(args, "name", "")
		namespace := getToolArgString(args, "namespace", "")
		replicas := getToolArgInt(args, "replicas", -1)
		target := name
		if namespace != "" && name != "" {
			target = fmt.Sprintf("%s/%s", namespace, name)
		}
		if target == "" {
			target = "<未指定工作负载>"
		}
		return strings.Join([]string{
			fmt.Sprintf("操作：将 %s %s 缩放到 %d 个副本。", kind, target, replicas),
			"影响：Pod 数量会立即变化；扩容可能触发新的资源占用，缩容可能中断正在运行的工作负载。",
			"建议先确认：目标副本数、当前业务负载、以及资源配额是否允许此次调整。",
		}, "\n")
	case agentToolK8sLabelNode:
		nodeName := getToolArgString(args, "node_name", "")
		key := getToolArgString(args, "key", "")
		value := getToolArgString(args, "value", "")
		return strings.Join([]string{
			fmt.Sprintf("操作：为节点 %s 设置标签 %s=%s。", nodeName, key, value),
			"影响：调度规则、节点选择器、拓扑约束可能立即变化。",
			"建议先确认：是否存在依赖该标签的工作负载、调度策略或自动化控制器。",
		}, "\n")
	case agentToolK8sTaintNode:
		nodeName := getToolArgString(args, "node_name", "")
		key := getToolArgString(args, "key", "")
		value := getToolArgString(args, "value", "")
		effect := getToolArgString(args, "effect", "NoSchedule")
		return strings.Join([]string{
			fmt.Sprintf("操作：为节点 %s 添加 taint %s=%s:%s。", nodeName, key, value, effect),
			"影响：不匹配 toleration 的 Pod 可能无法调度，NoExecute 还可能驱逐现有 Pod。",
			"建议先确认：受影响工作负载是否已配置 toleration，以及是否允许触发迁移/驱逐。",
		}, "\n")
	case agentToolRunKubectl:
		command := getToolArgString(args, "command", "")
		reason := getToolArgString(args, "reason", "")
		return strings.Join([]string{
			fmt.Sprintf("操作：执行 kubectl 写命令：%s", command),
			"影响：该命令可能直接修改集群对象，风险高于结构化运维工具。",
			fmt.Sprintf("原因：%s", reason),
			"建议先确认：目标资源、命令副作用、回退手段和变更窗口都已明确。",
		}, "\n")
	case agentToolAdminCommand:
		command := getToolArgString(args, "command", "")
		reason := getToolArgString(args, "reason", "")
		return strings.Join([]string{
			fmt.Sprintf("操作：执行管理命令：%s", command),
			"影响：该命令可能通过 helm / velero / istioctl / psql 等工具直接修改平台配置、数据或运行态。",
			fmt.Sprintf("原因：%s", reason),
			"建议先确认：命令目标、预期变更、副作用和回退步骤都已明确。",
		}, "\n")
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
		subject := getToolArgString(args, "subject", "作业通知")
		if jobNames != "" {
			return fmt.Sprintf("向作业所有者发送邮件通知：%s（主题：%s）", jobNames, subject)
		}
		return fmt.Sprintf("向作业所有者发送邮件通知（主题：%s）", subject)
	default:
		return fmt.Sprintf("执行操作 %s", toolName)
	}
}

func (mgr *AgentMgr) buildToolOutcomeMessage(toolName, status string, result any, fallback string) string {
	if status == "rejected" {
		switch toolName {
		case agentToolResubmitJob:
			return "已取消重新提交作业。"
		case agentToolStopJob:
			return "已取消停止作业。"
		case agentToolDeleteJob:
			return "已取消删除作业。"
		case agentToolCreateJupyter:
			return "已取消创建 Jupyter 作业。"
		case agentToolCreateWebIDE:
			return "已取消创建 WebIDE 作业。"
		case agentToolCreateTrain:
			return "已取消创建训练作业。"
		case agentToolCreateCustom:
			return "已取消创建自定义作业。"
		case agentToolCreatePytorch:
			return "已取消创建 PyTorch 作业。"
		case agentToolCreateTensorflow:
			return "已取消创建 TensorFlow 作业。"
		case agentToolCreateImage:
			return "已取消创建镜像构建任务。"
		case agentToolManageBuild:
			return "已取消镜像构建管理操作。"
		case agentToolRegisterImage:
			return "已取消登记外部镜像。"
		case agentToolManageAccess:
			return "已取消镜像权限变更。"
		case toolBatchStopJobs:
			return "已取消批量停止作业。"
		case toolNotifyJobOwner:
			return "已取消发送作业通知邮件。"
		case toolMarkAuditHandled:
			return "已取消标记审计条目。"
		case agentToolCordonNode:
			return "已取消封锁节点。"
		case agentToolUncordonNode:
			return "已取消解除节点封锁。"
		case agentToolDrainNode:
			return "已取消驱逐节点。"
		case agentToolDeletePod:
			return "已取消删除 Pod。"
		case agentToolRestartWL:
			return "已取消重启工作负载。"
		case agentToolK8sScaleWL:
			return "已取消调整工作负载副本数。"
		case agentToolK8sLabelNode:
			return "已取消修改节点标签。"
		case agentToolK8sTaintNode:
			return "已取消添加节点 taint。"
		case agentToolRunKubectl:
			return "已取消执行 kubectl 命令。"
		case agentToolAdminCommand:
			return "已取消执行管理命令。"
		default:
			return "已取消该操作。"
		}
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
	case agentToolCreateImage:
		if imagePackName, _ := resultMap["imagePackName"].(string); imagePackName != "" {
			return fmt.Sprintf("已提交镜像构建任务 %s。", imagePackName)
		}
		return "已提交新的镜像构建任务。"
	case agentToolManageBuild:
		action, _ := resultMap["action"].(string)
		if imagePackName, _ := resultMap["imagePackName"].(string); imagePackName != "" {
			switch action {
			case "cancel":
				return fmt.Sprintf("已请求取消镜像构建 %s。", imagePackName)
			case "delete":
				return fmt.Sprintf("已删除镜像构建 %s。", imagePackName)
			}
		}
		return "已更新镜像构建状态。"
	case agentToolRegisterImage:
		if image, ok := resultMap["image"].(map[string]any); ok {
			if imageLink, _ := image["imageLink"].(string); imageLink != "" {
				return fmt.Sprintf("已登记外部镜像 %s。", imageLink)
			}
		}
		return "已登记外部镜像。"
	case agentToolManageAccess:
		action, _ := resultMap["action"].(string)
		targetType, _ := resultMap["targetType"].(string)
		return fmt.Sprintf("已完成镜像权限%s，目标类型为 %s。", action, targetType)
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
		if name == "" {
			name, _ = resultMap["pod_name"].(string)
		}
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
	case agentToolK8sScaleWL:
		kind, _ := resultMap["kind"].(string)
		name, _ := resultMap["name"].(string)
		replicas, _ := resultMap["replicas"].(float64)
		if kind != "" && name != "" && replicas >= 0 {
			return fmt.Sprintf("已将 %s %s 调整到 %.0f 个副本。", kind, name, replicas)
		}
		return "已调整工作负载副本数。"
	case agentToolK8sLabelNode:
		nodeName, _ := resultMap["node_name"].(string)
		label, _ := resultMap["label"].(string)
		if nodeName != "" && label != "" {
			return fmt.Sprintf("已为节点 %s 设置标签 %s。", nodeName, label)
		}
		return "已更新节点标签。"
	case agentToolK8sTaintNode:
		nodeName, _ := resultMap["node_name"].(string)
		taint, _ := resultMap["taint"].(string)
		if nodeName != "" && taint != "" {
			return fmt.Sprintf("已为节点 %s 设置 taint %s。", nodeName, taint)
		}
		return "已更新节点 taint。"
	case agentToolRunKubectl:
		return "已执行 kubectl 管理命令。"
	case agentToolAdminCommand:
		return "已执行管理命令。"
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
		if sent, _ := resultMap["sent"].(float64); sent > 0 {
			return fmt.Sprintf("已向 %.0f 个作业所有者发送邮件通知。", sent)
		}
		return "作业通知邮件处理完成。"
	default:
		if fallback != "" {
			return fallback
		}
		return "操作已完成。"
	}
}
