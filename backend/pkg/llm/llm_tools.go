//nolint:gocritic,gocyclo,mnd,lll,staticcheck,unused // Storage governance tool wiring keeps prompts, handlers, and K8s traversal centralized.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/monitor"
)

const (
	labelKeyTaskUser = "crater.raids.io/task-user"
)

// ---- 业务响应类型 ----

type LLMDecisionResponse struct {
	Reason        string `json:"reason"`
	AllowExpand   bool   `json:"allow_expand"`
	ExpandBytes   int64  `json:"expand_bytes"`
	FreezeNewJobs bool   `json:"freeze_new_jobs"`
}

type PlatformCapacityResponse struct {
	TotalCapacity     int64 `json:"total_capacity"`
	UsedCapacity      int64 `json:"used_capacity"`
	AvailableCapacity int64 `json:"available_capacity"`
}

type TenantPodResponse struct {
	PodName      string `json:"pod_name"`
	Phase        string `json:"phase"`
	GPURrequests int    `json:"gpu_requests"`
}

type TenantPodsResponse struct {
	TenantID string              `json:"tenant_id"`
	Pods     []TenantPodResponse `json:"pods"`
}

type PodDetailsResponse struct {
	PodName         string    `json:"pod_name"`
	StartTime       time.Time `json:"start_time"`
	RunningTime     int       `json:"running_time"`
	ContainerImages []string  `json:"container_images"`
	RestartCount    int       `json:"restart_count"`
	GPUModel        string    `json:"gpu_model"`
	GPUCount        int       `json:"gpu_count"`
	CPUCount        int       `json:"cpu_count"`
	MemorySize      string    `json:"memory_size"`
}

type ComputeQuotaResponse struct {
	TenantID      string `json:"tenant_id"`
	GPULimit      int    `json:"gpu_limit"`
	GPURequest    int    `json:"gpu_request"`
	CPULimit      int    `json:"cpu_limit"`
	CPURequest    int    `json:"cpu_request"`
	MemoryLimit   string `json:"memory_limit"`
	MemoryRequest string `json:"memory_request"`
}

type UsageTrend struct {
	Timestamp  time.Time `json:"timestamp"`
	UsageBytes int64     `json:"usage_bytes"`
}

type TenantStorageTrendResponse struct {
	TenantID     string       `json:"tenant_id"`
	CurrentUsage int64        `json:"current_usage"`
	History      []UsageTrend `json:"history"`
}

// ---- DeepSeek / OpenAI 兼容类型 ----

type dsToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type dsTool struct {
	Type     string         `json:"type"` // "function"
	Function dsToolFunction `json:"function"`
}

type dsToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type dsToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function dsToolCallFunction `json:"function"`
}

type dsMessage struct {
	Role       string       `json:"role"`
	Content    string       `json:"content,omitempty"`
	ToolCalls  []dsToolCall `json:"tool_calls,omitempty"`
	ToolCallID string       `json:"tool_call_id,omitempty"`
}

type dsRequest struct {
	Model     string      `json:"model"`
	Messages  []dsMessage `json:"messages"`
	Tools     []dsTool    `json:"tools,omitempty"`
	MaxTokens int         `json:"max_tokens,omitempty"`
}

type dsResponse struct {
	Choices []struct {
		Message      dsMessage `json:"message"`
		FinishReason string    `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ---- 工具定义 ----

func getTools() []dsTool {
	return []dsTool{
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "query_platform_capacity",
				Description: "获取整个平台（硬限制 8TB）的总量、已用量、可用量",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
					"required":   []string{},
				},
			},
		},
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "list_tenant_pods",
				Description: "列出租户当前活跃的 Pod 列表",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tenant_id": map[string]any{"type": "string", "description": "租户 ID"},
					},
					"required": []string{"tenant_id"},
				},
			},
		},
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "inspect_pod_details",
				Description: "深入查看某个特定 Pod 的启动时间、镜像和重启次数、GPU型号、个数，CPU核数和内存大小",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tenant_id": map[string]any{"type": "string", "description": "租户 ID"},
						"pod_name":  map[string]any{"type": "string", "description": "Pod 名称"},
					},
					"required": []string{"tenant_id", "pod_name"},
				},
			},
		},
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "get_tenant_compute_quota",
				Description: "获取租户当前所有活跃 Pod 的 GPU/CPU/内存请求与限制总量",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tenant_id": map[string]any{"type": "string", "description": "租户 ID"},
					},
					"required": []string{"tenant_id"},
				},
			},
		},
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "query_tenant_storage_trend",
				Description: "获取指定租户当前的真实存储占用以及最近的几次历史记录，用于推导增长斜率",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tenant_id": map[string]any{"type": "string", "description": "租户 ID"},
					},
					"required": []string{"tenant_id"},
				},
			},
		},
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "query_pod_realtime_metrics",
				Description: "通过 Prometheus 查询指定 Pod 过去 5 分钟的真实资源利用率（CPU 核数、内存 MB、GPU 利用率 %、GPU 显存 MB）。用于检测僵尸作业：gpu_data_available=true 时以 gpu_util_percent 判断，gpu_data_available=false 时以 cpu_cores 判断（< 0.05 核视为进程挂死）",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"pod_name": map[string]any{"type": "string", "description": "Pod 名称"},
					},
					"required": []string{"pod_name"},
				},
			},
		},
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "query_pod_gpu_history",
				Description: "查询指定 Pod 在整个生命周期（或最近 24 小时）内的 GPU 历史利用率，返回平均值和最大值。用于区分两种低 GPU 利用率场景：(1) 作业曾经高强度使用 GPU（max_util_ever > 50%），当前低利用率说明正处于落盘/IO 阶段，是有价值的作业；(2) 从未有过高 GPU 利用率（max_util_ever ≈ 0%），则可能是僵尸作业或纯 CPU 作业。data_available=false 表示 DCGM 未采集到该 Pod 数据",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"pod_name":       map[string]any{"type": "string", "description": "Pod 名称"},
						"duration_hours": map[string]any{"type": "number", "description": "查询历史时长（小时），默认 24。建议传入作业已运行时长以覆盖完整生命周期"},
					},
					"required": []string{"pod_name"},
				},
			},
		},
		{
			Type: "function",
			Function: dsToolFunction{
				Name:        "diagnose_prometheus",
				Description: "诊断 Prometheus 连通性与 DCGM 监控可用性。返回：Prometheus 是否可达、DCGM 指标是否存在、当前 namespace 下被 DCGM 追踪的 Pod 列表，以及（可选）指定 Pod 的原始指标查询结果。当 query_pod_realtime_metrics 返回 gpu_data_available=false 时必须调用此工具排查原因",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"pod_name": map[string]any{"type": "string", "description": "（可选）需要额外查询实时指标的 Pod 名称"},
					},
					"required": []string{},
				},
			},
		},
	}
}

// ---- Agent 主循环 ----

func AskAgentForDecision(clientset kubernetes.Interface, restConfig *rest.Config, tenantID string, promClient monitor.PrometheusInterface) (*LLMDecisionResponse, error) {
	ctx := context.Background()
	llmConfig, err := GetStorageDecisionProviderConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载存储决策 LLM 配置失败: %w", err)
	}
	skillText, skillSource, err := loadStorageAgentSkill()
	if err != nil {
		return nil, fmt.Errorf("加载 storage agent skill 失败: %w", err)
	}

	const (
		systemPrompt = `你是一个平台运维 AI 助手，负责分析租户存储告警并给出临时扩容决策。

【背景知识】
GPU 密集型训练完成后，作业通常进入 IO 密集型落盘阶段（写 checkpoint / 模型参数）。
此阶段特征：GPU 利用率降至 0、CPU 极低（IO bound）、存储快速增长。
这是有价值的正常行为，必须优先保护。区分落盘作业与僵尸作业的关键依据是 GPU 历史记录：
  · 落盘作业：历史上曾有过高 GPU 利用率（max_util_ever > 50%），说明完成了真实的 GPU 计算
  · 僵尸作业：整个生命周期 GPU 利用率始终接近 0，从未进行有效计算

【工具调用策略 — 请严格按序执行，禁止冗余调用】

第一步（必做）：调用 query_tenant_storage_trend。
  从 history 记录计算增长速率（growth_rate）：
    · 将记录按时间排序，用最新与最早的 usage_bytes 之差除以时间跨度，得到 growth_rate（字节/小时）
    · 不足 2 条记录时视为"增长趋势未知"

▸ usage_ratio >= 100%（已超出配额）：
  → freeze_new_jobs=true，allow_expand=false，立即输出 JSON。

▸ 90% <= usage_ratio < 100%（接近配额）：

  第二步：调用 query_platform_capacity 确认平台剩余空间。
    · 平台空间不足（可用量 <= 当前配额 20%）：
        → allow_expand=false，freeze_new_jobs=false
        → reason 说明平台空间不足，建议管理员释放集群存储，输出 JSON。

  第三步（平台空间充足时）：判断是否需要作业层面分析。
    · 增长平缓（growth_rate < 配额 1%/小时）或趋势未知：
        → 属于正常数据积累，allow_expand=true，expand_bytes=配额的 20%，freeze_new_jobs=false，输出 JSON。
    · 增长较快（growth_rate >= 配额 1%/小时）：
        → 进入第四步，分析活跃作业以辅助决策。

  第四步（仅在增长较快时执行）：
    → 调用 list_tenant_pods 获取活跃 Pod 列表
    → 对每个 gpu_requests > 0 的 Pod，调用 query_pod_gpu_history（duration_hours 传入该 Pod 的预估运行时长，默认 24）
    → 根据 GPU 历史数据判断作业性质：

      · max_util_ever_percent > 50%（曾进行 GPU 密集计算）：
          → 判定为【落盘作业】，当前低利用率是正常落盘行为
          → allow_expand=true，expand_bytes=配额的 50%（为落盘预留充足空间）
          → reason 中注明："检测到作业 [pod名] 历史 GPU 峰值为 X%，当前处于落盘阶段，扩容保护训练成果"

      · max_util_ever_percent <= 50% 且 data_available=true（DCGM 有数据但 GPU 从未高负载）：
          → 判定为【可疑作业】，在 reason 中说明情况，建议人工排查
          → 但仍 allow_expand=true，expand_bytes=配额的 20%（不确定时优先保护用户）
          → reason 中注明："作业 [pod名] GPU 历史峰值仅 X%，未见明显 GPU 计算，存储增长原因待排查"

      · data_available=false（DCGM 无数据，无法获取 GPU 历史）：
          → 无法区分，保守但偏向保护：allow_expand=true，expand_bytes=配额的 20%
          → reason 中注明 GPU 历史数据不可用

    → freeze_new_jobs=false，输出 JSON。

▸ usage_ratio < 90%：
  → 无需冻结，无需扩容，立即输出 JSON。

【仅在上述流程中明确要求时才调用对应工具；inspect_pod_details / get_tenant_compute_quota / query_pod_realtime_metrics / diagnose_prometheus 不得主动调用】

完成分析后，仅输出如下格式的 JSON，不要附加任何其他文字：
{"reason": "<决策理由，必须包含 usage_ratio、growth_rate 及关键作业诊断证据>", "allow_expand": true/false, "expand_bytes": <字节数>, "freeze_new_jobs": true/false}`
		maxLoops = 10
	)

	messages := []dsMessage{
		{Role: "system", Content: systemPrompt},
	}
	if skillText != "" {
		klog.Infof("AskAgentForDecision[%s] loaded storage agent skill from %s", tenantID, skillSource)
		messages = append(messages, dsMessage{
			Role:    "system",
			Content: "以下为存储治理领域技能补充，请在工具调用和最终决策时遵循：\n" + skillText,
		})
	}
	messages = append(messages, dsMessage{
		Role:    "user",
		Content: fmt.Sprintf("租户 %s 触发存储告警（使用率超过 90%%），请调用工具进行全面排查，然后给出是否需要临时扩容的决策。", tenantID),
	})

	for i := 0; i < maxLoops; i++ {
		resp, err := callConfiguredLLM(ctx, *llmConfig, dsRequest{
			MaxTokens: 4096,
			Tools:     getTools(),
			Messages:  messages,
		})
		if err != nil {
			return nil, fmt.Errorf("调用配置化 LLM Provider 失败: %w", err)
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		// 无工具调用 → 模型给出了最终决策
		if choice.FinishReason == "stop" || len(choice.Message.ToolCalls) == 0 {
			klog.Infof("AskAgentForDecision[%s] 完整分析结果:\n%s", tenantID, choice.Message.Content)
			raw := extractJSON(choice.Message.Content)
			var decision LLMDecisionResponse
			if err := json.Unmarshal([]byte(raw), &decision); err != nil {
				return nil, fmt.Errorf("解析决策 JSON 失败: %w\n原始响应: %s", err, choice.Message.Content)
			}
			return &decision, nil
		}

		// 执行所有工具调用，收集结果
		for _, toolCall := range choice.Message.ToolCalls {
			result, toolErr := dispatchTool(clientset, restConfig, tenantID, promClient, toolCall)
			if toolErr != nil {
				result = fmt.Sprintf(`{"error": %q}`, toolErr.Error())
			}
			messages = append(messages, dsMessage{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    result,
			})
		}
	}

	return nil, fmt.Errorf("超过最大对话轮数 (%d)，未能得出决策", maxLoops)
}

// dispatchTool 根据工具名分发到对应 handler
func dispatchTool(clientset kubernetes.Interface, restConfig *rest.Config, tenantID string, promClient monitor.PrometheusInterface, toolCall dsToolCall) (string, error) {
	switch toolCall.Function.Name {
	case "query_platform_capacity":
		return HandleQueryPlatformCapacity(clientset, restConfig)

	case "list_tenant_pods":
		return HandleListTenantPods(clientset, tenantID)

	case "inspect_pod_details":
		var args struct {
			TenantID string `json:"tenant_id"`
			PodName  string `json:"pod_name"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("解析 inspect_pod_details 参数失败: %w", err)
		}
		return HandleInspectPodDetails(clientset, args.TenantID, args.PodName)

	case "get_tenant_compute_quota":
		return HandleGetComputeQuota(clientset, tenantID)

	case "query_tenant_storage_trend":
		return HandleQueryTenantStorageTrend(clientset, restConfig, tenantID)

	case "query_pod_realtime_metrics":
		var args struct {
			PodName string `json:"pod_name"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("解析 query_pod_realtime_metrics 参数失败: %w", err)
		}
		return HandleQueryPodRealtimeMetrics(args.PodName, clientset, promClient)

	case "query_pod_gpu_history":
		var args struct {
			PodName       string  `json:"pod_name"`
			DurationHours float64 `json:"duration_hours"`
		}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("解析 query_pod_gpu_history 参数失败: %w", err)
		}
		return HandleQueryPodGPUHistory(args.PodName, args.DurationHours, promClient)

	case "diagnose_prometheus":
		var args struct {
			PodName string `json:"pod_name"`
		}
		// pod_name 是可选参数，忽略解析错误
		_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		result := DiagnosePrometheus(args.PodName, clientset, promClient)
		data, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("序列化诊断结果失败: %w", err)
		}
		return string(data), nil

	default:
		return "", fmt.Errorf("未知工具: %s", toolCall.Function.Name)
	}
}

// ---- Prometheus 诊断 ----

// PrometheusDiagnosis 是 DiagnosePrometheus 的返回结构。
type PrometheusDiagnosis struct {
	PrometheusURL   string   `json:"prometheus_url"`
	Reachable       bool     `json:"reachable"`
	ConnectError    string   `json:"connect_error,omitempty"`
	DCGMAvailable   bool     `json:"dcgm_available"`
	DCGMSeriesCount int      `json:"dcgm_series_count"`
	TrackedPods     []string `json:"tracked_pods_in_namespace"`
	Namespace       string   `json:"namespace"`
	PodMetrics      *struct {
		PodName        string  `json:"pod_name"`
		CPUCores       float64 `json:"cpu_cores"`
		MemoryMB       float64 `json:"memory_mb"`
		GPUUtilPercent float64 `json:"gpu_util_percent"`
		GPUDataFound   bool    `json:"gpu_data_found"`
	} `json:"pod_metrics,omitempty"`
}

// DiagnosePrometheus 验证 Prometheus 连通性、DCGM 指标可用性，
// 并列出 jobNamespace 下被 DCGM 追踪的 Pod。
// podName 非空时额外查询该 Pod 的实时指标。
func DiagnosePrometheus(podName string, _ kubernetes.Interface, promClient monitor.PrometheusInterface) *PrometheusDiagnosis {
	jobNamespace := config.GetConfig().Namespaces.Job

	diag := &PrometheusDiagnosis{
		PrometheusURL: config.GetConfig().PrometheusAPI,
		Namespace:     jobNamespace,
		TrackedPods:   []string{},
	}

	if promClient == nil {
		diag.ConnectError = "Prometheus 客户端未初始化"
		return diag
	}

	// 1. 基础连通性：vector(1) 强制返回向量，任何 Prometheus 均支持
	v, ok, err := promClient.QueryInstant("vector(1)")
	if err != nil {
		diag.ConnectError = fmt.Sprintf("Prometheus 查询失败: %v", err)
		return diag
	}
	if !ok || v != 1 {
		diag.ConnectError = fmt.Sprintf("Prometheus 返回异常值: ok=%v val=%v", ok, v)
		return diag
	}
	diag.Reachable = true

	// 2. 检查 DCGM 是否存在任意时间序列
	if cnt, ok, err := promClient.QueryInstant("count(DCGM_FI_DEV_GPU_UTIL)"); err == nil && ok {
		diag.DCGMAvailable = true
		diag.DCGMSeriesCount = int(cnt)
	}

	// 3. 列出 jobNamespace 下 DCGM 追踪的所有 Pod 名称
	pods := promClient.QueryInstantLabels(
		fmt.Sprintf(`count by (pod) (DCGM_FI_DEV_GPU_UTIL{namespace=%q})`, jobNamespace),
		"pod")
	if pods != nil {
		diag.TrackedPods = pods
	}

	// 4. 如果指定了 Pod，额外查询其实时指标
	if podName != "" {
		pm := &struct {
			PodName        string  `json:"pod_name"`
			CPUCores       float64 `json:"cpu_cores"`
			MemoryMB       float64 `json:"memory_mb"`
			GPUUtilPercent float64 `json:"gpu_util_percent"`
			GPUDataFound   bool    `json:"gpu_data_found"`
		}{PodName: podName}

		if v, ok, _ := promClient.QueryInstant(
			fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{pod=%q,container!=""}[5m]))`, podName),
		); ok {
			pm.CPUCores = v
		}
		if v, ok, _ := promClient.QueryInstant(
			fmt.Sprintf(`sum(container_memory_usage_bytes{pod=%q,container!=""})`, podName),
		); ok {
			pm.MemoryMB = v / 1024 / 1024
		}
		// GPU：先带 namespace，再退化
		for _, q := range []string{
			fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q})`, jobNamespace, podName),
			fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{pod=%q})`, podName),
		} {
			if v, ok, _ := promClient.QueryInstant(q); ok {
				pm.GPUUtilPercent = v
				pm.GPUDataFound = true
				break
			}
		}
		diag.PodMetrics = pm
	}

	return diag
}

// HandleQueryPodRealtimeMetrics 通过 Prometheus 查询 Pod 的真实资源利用率，
// 用于识别高申请低利用的僵尸作业。
func HandleQueryPodRealtimeMetrics(podName string, _ kubernetes.Interface, promClient monitor.PrometheusInterface) (string, error) {
	if promClient == nil {
		return "", fmt.Errorf("Prometheus 客户端未初始化")
	}

	// Job namespace — DCGM 标签与 Kubernetes namespace 绑定，必须传入才能匹配到指标
	jobNamespace := config.GetConfig().Namespaces.Job

	type metricsResult struct {
		PodName          string  `json:"pod_name"`
		Namespace        string  `json:"namespace"`
		CPUCores         float64 `json:"cpu_cores"`
		MemoryMB         float64 `json:"memory_mb"`
		GPUUtilPercent   float64 `json:"gpu_util_percent"`
		GPUMemoryMB      float64 `json:"gpu_memory_mb"`
		GPUDataAvailable bool    `json:"gpu_data_available"`
		Note             string  `json:"note"`
	}

	res := metricsResult{PodName: podName, Namespace: jobNamespace}

	// CPU 使用量（核数，5 分钟均值）
	if v, ok, err := promClient.QueryInstant(
		fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{pod=%q,container!=""}[5m]))`, podName),
	); err != nil {
		klog.Warningf("query_pod_realtime_metrics: CPU 查询失败 pod=%s: %v", podName, err)
	} else if ok {
		res.CPUCores = v
	}

	// 内存使用量（MB）
	if v, ok, err := promClient.QueryInstant(
		fmt.Sprintf(`sum(container_memory_usage_bytes{pod=%q,container!=""})`, podName),
	); err != nil {
		klog.Warningf("query_pod_realtime_metrics: 内存查询失败 pod=%s: %v", podName, err)
	} else if ok {
		res.MemoryMB = v / 1024 / 1024
	}

	// GPU 利用率（%，来自 DCGM_FI_DEV_GPU_UTIL）
	// 优先用 namespace+pod 双标签（与现有 monitor 包一致），若无结果则退化为仅 pod 标签
	gpuUtilQueries := []string{
		fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q})`, jobNamespace, podName),
		fmt.Sprintf(`avg(DCGM_FI_DEV_GPU_UTIL{pod=%q})`, podName),
	}
	for _, q := range gpuUtilQueries {
		v, ok, err := promClient.QueryInstant(q)
		if err != nil {
			klog.Warningf("query_pod_realtime_metrics: GPU 利用率查询失败 pod=%s query=%s: %v", podName, q, err)
			continue
		}
		if ok {
			res.GPUUtilPercent = v
			res.GPUDataAvailable = true
			break
		}
	}

	// GPU 显存使用量（MB，来自 DCGM_FI_DEV_FB_USED）
	gpuMemQueries := []string{
		fmt.Sprintf(`avg(DCGM_FI_DEV_FB_USED{namespace=%q,pod=%q})`, jobNamespace, podName),
		fmt.Sprintf(`avg(DCGM_FI_DEV_FB_USED{pod=%q})`, podName),
	}
	for _, q := range gpuMemQueries {
		v, ok, err := promClient.QueryInstant(q)
		if err != nil {
			klog.Warningf("query_pod_realtime_metrics: GPU 显存查询失败 pod=%s query=%s: %v", podName, q, err)
			continue
		}
		if ok {
			res.GPUMemoryMB = v
			break
		}
	}

	res.Note = "cpu_cores 为过去 5 分钟均值；memory_mb 为当前值；gpu_util_percent/gpu_memory_mb 来自 DCGM（gpu_data_available=false 表示该 Pod 无 GPU 或 DCGM 未采集到数据）"

	data, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("序列化利用率响应失败: %w", err)
	}
	return string(data), nil
}

// HandleQueryPodGPUHistory 查询 Pod 在指定时间窗口内的 GPU 历史利用率，
// 用于区分"正在落盘的有价值作业"（历史上曾高强度用过 GPU）与"僵尸作业"（从未有效使用 GPU）。
func HandleQueryPodGPUHistory(podName string, durationHours float64, promClient monitor.PrometheusInterface) (string, error) {
	if promClient == nil {
		return "", fmt.Errorf("Prometheus 客户端未初始化")
	}
	if durationHours <= 0 {
		durationHours = 24
	}

	jobNamespace := config.GetConfig().Namespaces.Job
	dur := fmt.Sprintf("%.0fh", durationHours)
	// 不足 1 小时时用分钟表示，避免 Prometheus 解析错误
	if durationHours < 1 {
		dur = fmt.Sprintf("%.0fm", durationHours*60)
	}

	type gpuHistoryResult struct {
		PodName       string  `json:"pod_name"`
		DurationHours float64 `json:"duration_hours"`
		AvgUtil       float64 `json:"avg_util_percent"`
		MaxUtil       float64 `json:"max_util_ever_percent"`
		DataAvailable bool    `json:"data_available"`
		Note          string  `json:"note"`
	}

	res := gpuHistoryResult{
		PodName:       podName,
		DurationHours: durationHours,
		Note:          fmt.Sprintf("查询过去 %s 内的 GPU 利用率历史。max_util_ever_percent > 50 表明作业曾进行 GPU 密集型计算（如模型训练），当前低利用率很可能是落盘/IO 阶段", dur),
	}

	// 平均利用率：namespace+pod 优先，退化为仅 pod
	avgQueries := []string{
		fmt.Sprintf(`avg_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[%s])`, jobNamespace, podName, dur),
		fmt.Sprintf(`avg_over_time(DCGM_FI_DEV_GPU_UTIL{pod=%q}[%s])`, podName, dur),
	}
	for _, q := range avgQueries {
		if v, ok, err := promClient.QueryInstant(q); err == nil && ok {
			res.AvgUtil = v
			res.DataAvailable = true
			break
		}
	}

	// 峰值利用率：用 max_over_time 捕获历史最高点
	maxQueries := []string{
		fmt.Sprintf(`max_over_time(DCGM_FI_DEV_GPU_UTIL{namespace=%q,pod=%q}[%s])`, jobNamespace, podName, dur),
		fmt.Sprintf(`max_over_time(DCGM_FI_DEV_GPU_UTIL{pod=%q}[%s])`, podName, dur),
	}
	for _, q := range maxQueries {
		if v, ok, err := promClient.QueryInstant(q); err == nil && ok {
			res.MaxUtil = v
			res.DataAvailable = true
			break
		}
	}

	data, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("序列化 GPU 历史响应失败: %w", err)
	}
	return string(data), nil
}

// ---- Handler 实现 ----

func HandleQueryPlatformCapacity(clientset kubernetes.Interface, restConfig *rest.Config) (string, error) {
	totalCapacity, usedCapacity, err := ceph.GetCraterStorageCapacity(clientset, restConfig, "rook-ceph")
	if err != nil {
		return "", fmt.Errorf("获取平台容量失败: %w", err)
	}
	availableCapacity := ceph.AvailableBytes(totalCapacity, usedCapacity)

	data, err := json.Marshal(map[string]any{
		"total_capacity_bytes":     totalCapacity,
		"used_capacity_bytes":      usedCapacity,
		"available_capacity_bytes": availableCapacity,
		"total_capacity_formatted": formatBytes(totalCapacity),
		"used_capacity_formatted":  formatBytes(usedCapacity),
		"note":                     "所有容量单位均为字节(bytes)，formatted 字段为人类可读格式",
	})
	if err != nil {
		return "", fmt.Errorf("序列化平台容量响应失败: %w", err)
	}
	return string(data), nil
}

func HandleListTenantPods(clientset kubernetes.Interface, tenantID string) (string, error) {
	jobNamespace := config.GetConfig().Namespaces.Job
	pods, err := clientset.CoreV1().Pods(jobNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labelKeyTaskUser, tenantID),
	})
	if err != nil {
		return "", fmt.Errorf("列出 Pod 失败: %w", err)
	}

	podResponses := make([]TenantPodResponse, 0)
	for _, pod := range pods.Items {
		phase := string(pod.Status.Phase)
		if phase != "Running" && phase != "Pending" {
			continue
		}
		gpuRequests := 0
		for _, c := range pod.Spec.Containers {
			for k, v := range c.Resources.Requests {
				if strings.Contains(string(k), "nvidia.com/") {
					gpuRequests += int(v.Value())
				}
			}
		}
		podResponses = append(podResponses, TenantPodResponse{
			PodName:      pod.Name,
			Phase:        phase,
			GPURrequests: gpuRequests,
		})
	}

	data, err := json.Marshal(TenantPodsResponse{TenantID: tenantID, Pods: podResponses})
	if err != nil {
		return "", fmt.Errorf("序列化租户 Pod 列表响应失败: %w", err)
	}
	return string(data), nil
}

func HandleInspectPodDetails(clientset kubernetes.Interface, _ string, podName string) (string, error) {
	jobNamespace := config.GetConfig().Namespaces.Job
	pod, err := clientset.CoreV1().Pods(jobNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("获取 Pod 失败: %w", err)
	}

	var startTime time.Time
	if pod.Status.StartTime != nil {
		startTime = pod.Status.StartTime.Time
	}
	runningMinutes := 0
	if !startTime.IsZero() {
		runningMinutes = int(time.Since(startTime).Minutes())
	}

	images := make([]string, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		images = append(images, c.Image)
	}

	restartCount := 0
	for _, cs := range pod.Status.ContainerStatuses {
		restartCount += int(cs.RestartCount)
	}

	gpuCount := 0
	cpuMillis := int64(0)
	memBytes := int64(0)
	for _, c := range pod.Spec.Containers {
		for k, v := range c.Resources.Requests {
			if strings.Contains(string(k), "nvidia.com/") {
				gpuCount += int(v.Value())
			}
		}
		if cpu, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
			cpuMillis += cpu.MilliValue()
		}
		if mem, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
			memBytes += mem.Value()
		}
	}

	gpuModel := "unknown"
	if pod.Spec.NodeName != "" {
		if node, err := clientset.CoreV1().Nodes().Get(context.TODO(), pod.Spec.NodeName, metav1.GetOptions{}); err == nil {
			for k, v := range node.Labels {
				if strings.Contains(k, "nvidia.com/gpu.product") || k == "gpu-model" {
					gpuModel = v
					break
				}
			}
		}
	}

	data, err := json.Marshal(PodDetailsResponse{
		PodName:         podName,
		StartTime:       startTime,
		RunningTime:     runningMinutes,
		ContainerImages: images,
		RestartCount:    restartCount,
		GPUModel:        gpuModel,
		GPUCount:        gpuCount,
		CPUCount:        int(cpuMillis / 1000),
		MemorySize:      formatBytes(memBytes),
	})
	if err != nil {
		return "", fmt.Errorf("序列化 Pod 详情响应失败: %w", err)
	}
	return string(data), nil
}

func HandleGetComputeQuota(clientset kubernetes.Interface, tenantID string) (string, error) {
	jobNamespace := config.GetConfig().Namespaces.Job
	pods, err := clientset.CoreV1().Pods(jobNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labelKeyTaskUser, tenantID),
	})
	if err != nil {
		return "", fmt.Errorf("列出 Pod 失败: %w", err)
	}

	gpuRequest, gpuLimit := 0, 0
	cpuReqMillis, cpuLimMillis := int64(0), int64(0)
	memReqBytes, memLimBytes := int64(0), int64(0)

	for _, pod := range pods.Items {
		for _, c := range pod.Spec.Containers {
			for k, v := range c.Resources.Requests {
				if strings.Contains(string(k), "nvidia.com/") {
					gpuRequest += int(v.Value())
				}
			}
			for k, v := range c.Resources.Limits {
				if strings.Contains(string(k), "nvidia.com/") {
					gpuLimit += int(v.Value())
				}
			}
			if v, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
				cpuReqMillis += v.MilliValue()
			}
			if v, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
				cpuLimMillis += v.MilliValue()
			}
			if v, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
				memReqBytes += v.Value()
			}
			if v, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
				memLimBytes += v.Value()
			}
		}
	}

	data, err := json.Marshal(ComputeQuotaResponse{
		TenantID:      tenantID,
		GPULimit:      gpuLimit,
		GPURequest:    gpuRequest,
		CPULimit:      int(cpuLimMillis / 1000),
		CPURequest:    int(cpuReqMillis / 1000),
		MemoryLimit:   formatBytes(memLimBytes),
		MemoryRequest: formatBytes(memReqBytes),
	})
	if err != nil {
		return "", fmt.Errorf("序列化计算配额响应失败: %w", err)
	}
	return string(data), nil
}

func HandleQueryTenantStorageTrend(clientset kubernetes.Interface, restConfig *rest.Config, tenantID string) (string, error) {
	db := query.GetDB()

	// 先查基础用户信息和 space_quota
	var userRow struct {
		model.User
		SpaceQuota int64 `gorm:"column:space_quota"`
	}
	if err := db.Model(&model.User{}).
		Select("users.*, users.space_quota").
		Where("name = ?", tenantID).
		First(&userRow).Error; err != nil {
		return "", fmt.Errorf("用户 %s 不存在: %w", tenantID, err)
	}
	user := userRow.User

	// 尝试获取 original_space_quota（临时扩容时才有值），用作理论配额
	// 若迁移未执行列不存在，保持使用 space_quota
	spaceQuota := userRow.SpaceQuota
	var origRow struct {
		OriginalSpaceQuota *int64 `gorm:"column:original_space_quota"`
	}
	if err := db.Raw("SELECT original_space_quota FROM users WHERE id = ?", user.ID).Scan(&origRow).Error; err == nil && origRow.OriginalSpaceQuota != nil {
		spaceQuota = *origRow.OriginalSpaceQuota
	}
	if user.Space == "" {
		return "", fmt.Errorf("用户 %s 的空间路径为空", tenantID)
	}

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}
	currentUsage, err := ceph.GetCephDirectorySize(clientset, restConfig, "rook-ceph", "/user/"+user.Space, prefixConfig)
	if err != nil {
		currentUsage = ceph.UnknownSizeBytes
	}

	var historyRecords []model.TenantUsageHistory
	db.Where("tenant_id = ?", user.ID).Order("recorded_at DESC").Limit(10).Find(&historyRecords)

	type historyItem struct {
		Timestamp           time.Time `json:"timestamp"`
		UsageBytes          int64     `json:"usage_bytes"`
		UsageBytesFormatted string    `json:"usage_bytes_formatted"`
	}
	history := make([]historyItem, 0, len(historyRecords))
	for _, h := range historyRecords {
		history = append(history, historyItem{
			Timestamp:           h.RecordedAt,
			UsageBytes:          h.UsageBytes,
			UsageBytesFormatted: formatBytes(h.UsageBytes),
		})
	}

	usageRatio := ""
	if spaceQuota > 0 && currentUsage >= 0 {
		usageRatio = fmt.Sprintf("%.1f%%", float64(currentUsage)/float64(spaceQuota)*100)
	} else if spaceQuota == -1 {
		usageRatio = "unlimited"
	}

	data, err := json.Marshal(map[string]any{
		"tenant_id":               tenantID,
		"current_usage_bytes":     currentUsage,
		"current_usage_formatted": formatBytes(currentUsage),
		"quota_bytes":             spaceQuota,
		"quota_formatted":         formatBytes(spaceQuota),
		"usage_ratio":             usageRatio,
		"history":                 history,
		"note":                    "所有大小单位均为字节(bytes)，formatted 字段为人类可读格式。quota_bytes=-1 表示无限制。usage_ratio 为当前使用量占配额的百分比。",
	})
	if err != nil {
		return "", fmt.Errorf("序列化租户存储趋势响应失败: %w", err)
	}
	return string(data), nil
}

// ---- 工具函数 ----

// extractJSON 从可能包含分析文字的响应中提取最后一个 JSON 对象
func extractJSON(s string) string {
	start := strings.LastIndex(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(s[start : end+1])
	}
	return strings.TrimSpace(s)
}

func parseGetfattrValue(output, attr string) int64 {
	prefix := attr + "="
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			val := strings.Trim(strings.TrimPrefix(line, prefix), "\"")
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				return n
			}
		}
	}
	return 0
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes <= 0 {
		return "0 B"
	}
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
