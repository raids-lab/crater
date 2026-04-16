package agent

import (
	"sort"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

var agentUserTools = []string{
	agentToolAnalyzeQueue,
	agentToolCheckQuota,
	agentToolCreateJupyter,
	agentToolCreateTrain,
	agentToolDeleteJob,
	agentToolDetectIdleJobs,
	agentToolDiagnoseJob,
	agentToolGetDiagnosticCtx,
	agentToolGetFailureStats,
	agentToolGetHealthOverview,
	agentToolGetJobDetail,
	agentToolGetJobEvents,
	agentToolGetJobLogs,
	agentToolGetJobTemplates,
	agentToolListCudaBase,
	agentToolListGPUModels,
	agentToolListImages,
	agentToolListUserJobs,
	agentToolQueryJobMetrics,
	agentToolRealtimeCapacity,
	agentToolRecommendImages,
	agentToolResubmitJob,
	agentToolResourceRecommend,
	agentToolSearchSimilarFail,
	agentToolStopJob,
	agentToolK8sGetEvents,
	agentToolK8sDescribe,
	agentToolK8sPodLogs,
}

var agentAdminTools = []string{
	agentToolGetClusterHealth,
	agentToolGetClusterReport,
	agentToolGetNodeDetail,
	agentToolListClusterJobs,
	agentToolListClusterNodes,
	agentToolListStoragePVCs,
	agentToolGetPVCDetail,
	agentToolGetPVCEvents,
	agentToolInspectJobStorage,
	agentToolStorageCapacity,
	agentToolNodeNetwork,
	agentToolDiagnoseJobNet,
	agentToolWebSearch,
	agentToolSandboxGrep,
	agentToolRuntimeSummary,
	agentToolK8sListNodes,
	agentToolK8sListPods,
	agentToolPromQuery,
	agentToolHarborCheck,
	agentToolRunOpsScript,
	agentToolCordonNode,
	agentToolUncordonNode,
	agentToolDrainNode,
	agentToolDeletePod,
	agentToolRestartWL,
	toolBatchStopJobs,
	toolNotifyJobOwner,
}

var agentConfirmToolSet = map[string]struct{}{
	agentToolResubmitJob:   {},
	agentToolStopJob:       {},
	agentToolDeleteJob:     {},
	agentToolCreateJupyter: {},
	agentToolCreateTrain:   {},
	agentToolRunOpsScript:  {},
	agentToolCordonNode:    {},
	agentToolUncordonNode:  {},
	agentToolDrainNode:     {},
	agentToolDeletePod:     {},
	agentToolRestartWL:     {},
	toolMarkAuditHandled:   {},
	toolBatchStopJobs:      {},
	toolNotifyJobOwner:     {},
}

func agentToolCompactDescription(toolName string) string {
	switch toolName {
	case agentToolGetJobDetail:
		return "读取单个作业的状态、资源、时间线和终止信息"
	case agentToolGetJobEvents:
		return "读取作业相关 Kubernetes 事件"
	case agentToolGetJobLogs:
		return "读取作业日志尾部或按关键词过滤"
	case agentToolDiagnoseJob:
		return "执行规则诊断并输出故障分类和根因"
	case agentToolGetDiagnosticCtx:
		return "读取完整诊断上下文，信息量更大"
	case agentToolSearchSimilarFail:
		return "检索相似历史失败案例"
	case agentToolQueryJobMetrics:
		return "读取 GPU/CPU/内存等监控指标"
	case agentToolAnalyzeQueue:
		return "分析 Pending 或排队原因"
	case agentToolRealtimeCapacity:
		return "读取集群实时资源容量概览"
	case agentToolListImages:
		return "列出当前可见镜像"
	case agentToolListCudaBase:
		return "列出 CUDA 基础镜像"
	case agentToolListGPUModels:
		return "列出当前可用 GPU 型号和数量"
	case agentToolRecommendImages:
		return "为训练任务推荐候选镜像"
	case agentToolCheckQuota:
		return "查看账户配额使用情况"
	case agentToolGetHealthOverview:
		return "读取当前用户作业健康概览"
	case agentToolListUserJobs:
		return "列出当前用户近期作业"
	case agentToolDetectIdleJobs:
		return "检测当前账户下长期低利用率作业"
	case agentToolGetJobTemplates:
		return "列出平台提供的作业模板"
	case agentToolGetFailureStats:
		return "统计近期失败作业类型分布"
	case agentToolGetClusterReport:
		return "管理员读取聚合后的集群健康报告"
	case agentToolGetAdminOpsReport:
		return "管理员读取智能运维分析报告，聚合成功/失败/闲置任务与资源差异"
	case agentToolListStoragePVCs:
		return "管理员读取存储 PVC 列表与状态摘要"
	case agentToolGetPVCDetail:
		return "管理员读取单个 PVC 详情"
	case agentToolGetPVCEvents:
		return "管理员读取 PVC 相关事件"
	case agentToolInspectJobStorage:
		return "管理员诊断作业挂载、卷声明与存储关联"
	case agentToolStorageCapacity:
		return "管理员读取 PVC 容量与存储类聚合视图"
	case agentToolNodeNetwork:
		return "管理员读取节点网络状态摘要"
	case agentToolDiagnoseJobNet:
		return "管理员诊断分布式作业网络问题（事件/日志/节点分布）"
	case agentToolWebSearch:
		return "管理员执行受白名单约束的外网文档检索"
	case agentToolSandboxGrep:
		return "管理员在受限沙箱目录执行内容检索"
	case agentToolRuntimeSummary:
		return "读取 agent 运行时配置摘要（本地工具、k8s、prometheus 等）"
	case agentToolK8sListNodes:
		return "通过 agent 侧 kubeconfig 直接列出节点摘要"
	case agentToolK8sListPods:
		return "通过 agent 侧 kubeconfig 直接列出 Pod 摘要"
	case agentToolK8sGetEvents:
		return "通过 agent 侧 kubeconfig 直接查询 Kubernetes 事件"
	case agentToolK8sDescribe:
		return "通过 agent 侧 kubeconfig 直接执行 kubectl describe"
	case agentToolK8sPodLogs:
		return "通过 agent 侧 kubeconfig 直接读取 Pod 日志"
	case agentToolPromQuery:
		return "通过 agent 侧 Prometheus API 执行指标查询"
	case agentToolHarborCheck:
		return "检查 Harbor/OCI Registry 健康状态以及目标镜像是否存在"
	case agentToolRunOpsScript:
		return "管理员提交白名单运维脚本（需确认）"
	case agentToolResourceRecommend:
		return "根据任务描述推荐 CPU/GPU/内存配置"
	case agentToolGetNodeDetail:
		return "管理员读取单个节点详情"
	case agentToolGetClusterHealth:
		return "管理员读取集群健康概览"
	case agentToolListClusterJobs:
		return "管理员读取集群近期作业"
	case agentToolListClusterNodes:
		return "管理员读取节点摘要"
	case agentToolResubmitJob:
		return "重新提交已有作业，需要确认"
	case agentToolStopJob:
		return "停止作业，需要确认"
	case agentToolDeleteJob:
		return "删除作业，需要确认"
	case agentToolCreateJupyter:
		return "创建 Jupyter 作业，需要确认"
	case agentToolCreateTrain:
		return "创建训练作业，需要确认"
	case agentToolCordonNode:
		return "将节点标记为不可调度，需要确认"
	case agentToolUncordonNode:
		return "恢复节点调度，需要确认"
	case agentToolDrainNode:
		return "排空节点，需要确认"
	case agentToolDeletePod:
		return "删除 Pod 以触发重建，需要确认"
	case agentToolRestartWL:
		return "滚动重启 Deployment/StatefulSet/DaemonSet，需要确认"
	case toolGetLatestAuditReport:
		return "查看最近审计报告"
	case toolListAuditItems:
		return "筛选审计条目"
	case toolSaveAuditReport:
		return "保存审计报告"
	case toolMarkAuditHandled:
		return "标记审计已处理"
	case toolBatchStopJobs:
		return "批量停止作业"
	case toolNotifyJobOwner:
		return "通知作业所有者"
	default:
		return "平台工具"
	}
}

func buildAgentToolCatalog(enabledTools []string) []map[string]any {
	catalog := make([]map[string]any, 0, len(enabledTools))
	for _, toolName := range enabledTools {
		catalog = append(catalog, map[string]any{
			"name":        toolName,
			"mode":        map[bool]string{true: "confirm", false: "read_only"}[isAgentConfirmTool(toolName)],
			"description": agentToolCompactDescription(toolName),
		})
	}
	return catalog
}

func buildAgentCapabilities(token util.JWTMessage, page map[string]any) map[string]any {
	enabledSet := make(map[string]struct{}, len(agentUserTools)+len(agentAdminTools))
	addTools := func(names ...string) {
		for _, name := range names {
			if name == "" {
				continue
			}
			enabledSet[name] = struct{}{}
		}
	}

	addTools(agentUserTools...)
	if token.RolePlatform == model.RoleAdmin {
		addTools(agentAdminTools...)
	}

	enabledTools := make([]string, 0, len(enabledSet))
	for name := range enabledSet {
		enabledTools = append(enabledTools, name)
	}
	sort.Strings(enabledTools)

	confirmTools := make([]string, 0, len(agentConfirmToolSet))
	for _, name := range enabledTools {
		if _, ok := agentConfirmToolSet[name]; ok {
			confirmTools = append(confirmTools, name)
		}
	}

	pageRoute, _ := page["route"].(string)
	pageURL, _ := page["url"].(string)
	pageScope := "user"
	if token.RolePlatform == model.RoleAdmin && (strings.HasPrefix(pageRoute, "/admin") || strings.HasPrefix(pageURL, "/admin")) {
		pageScope = "admin"
	}

	return map[string]any{
		"enabled_tools": enabledTools,
		"confirm_tools": confirmTools,
		"tool_catalog":  buildAgentToolCatalog(enabledTools),
		"surface": map[string]any{
			"page_scope": pageScope,
			"page_route": pageRoute,
			"page_url":   pageURL,
		},
		"role_policies": map[string]any{
			"coordinator": "负责路由、整合和澄清，允许少量只读取证",
			"planner":     "只读规划，可参考上下文和工具目录，不得执行写工具",
			"explorer":    "只读探索与检索，不得执行写工具",
			"executor":    "负责真正工具执行，写工具必须走确认流",
			"verifier":    "只读验证与挑战结论，不得执行写工具",
			"guide":       "帮助/说明型回答，不执行写工具",
			"general":     "通用平台回答，默认不执行写工具",
		},
	}
}
