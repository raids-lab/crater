package agent

import (
	"sort"

	"github.com/raids-lab/crater/internal/util"
)

var agentUserTools = []string{
	agentToolAnalyzeQueue,
	agentToolCheckQuota,
	agentToolCreateCustom,
	agentToolCreateJupyter,
	agentToolCreatePytorch,
	agentToolCreateTensorflow,
	agentToolCreateWebIDE,
	agentToolDeleteJob,
	agentToolDiagnoseJob,
	agentToolGetDiagnosticCtx,
	agentToolGetJobDetail,
	agentToolGetJobEvents,
	agentToolGetJobLogs,
	agentToolGetJobTemplates,
	agentToolListGPUModels,
	agentToolListImages,
	agentToolListUserJobs,
	agentToolQueryJobMetrics,
	agentToolRealtimeCapacity,
	agentToolResubmitJob,
	agentToolResourceRecommend,
	agentToolSearchSimilarFail,
	agentToolStopJob,
}

var agentConfirmToolSet = map[string]struct{}{
	agentToolCreateCustom:     {},
	agentToolCreateJupyter:    {},
	agentToolCreatePytorch:    {},
	agentToolCreateTensorflow: {},
	agentToolCreateWebIDE:     {},
	agentToolDeleteJob:        {},
	agentToolResubmitJob:      {},
	agentToolStopJob:          {},
}

//nolint:gocyclo // Tool descriptions are intentionally centralized for capability export.
func agentToolCompactDescription(toolName string) string {
	switch toolName {
	case agentToolGetJobDetail:
		return "读取作业状态、资源、时间线和终止信息"
	case agentToolGetJobEvents:
		return "读取作业相关 Kubernetes 事件"
	case agentToolGetJobLogs:
		return "读取作业日志尾部或按关键词过滤"
	case agentToolDiagnoseJob:
		return "执行规则诊断并输出故障分类和根因"
	case agentToolGetDiagnosticCtx:
		return "读取完整诊断上下文"
	case agentToolSearchSimilarFail:
		return "检索相似历史失败案例"
	case agentToolQueryJobMetrics:
		return "读取 GPU、CPU、内存等监控指标"
	case agentToolAnalyzeQueue:
		return "分析 Pending 或排队原因"
	case agentToolRealtimeCapacity:
		return "读取实时资源容量概览"
	case agentToolListImages:
		return "列出当前可见镜像"
	case agentToolListGPUModels:
		return "列出当前可用 GPU 型号和数量"
	case agentToolCheckQuota:
		return "查看账户配额使用情况"
	case agentToolListUserJobs:
		return "列出当前用户近期作业"
	case agentToolGetJobTemplates:
		return "列出平台提供的作业模板"
	case agentToolResourceRecommend:
		return "根据任务描述推荐 CPU/GPU/内存配置"
	case agentToolCreateCustom:
		return "创建自定义作业，需要确认"
	case agentToolCreateJupyter:
		return "创建 Jupyter 作业，需要确认"
	case agentToolCreatePytorch:
		return "创建 PyTorch 分布式作业，需要确认"
	case agentToolCreateTensorflow:
		return "创建 TensorFlow 分布式作业，需要确认"
	case agentToolCreateWebIDE:
		return "创建 WebIDE 作业，需要确认"
	case agentToolDeleteJob:
		return "删除作业，需要确认"
	case agentToolResubmitJob:
		return "重新提交已有作业，需要确认"
	case agentToolStopJob:
		return "停止作业，需要确认"
	default:
		return "平台工具"
	}
}

func buildAgentToolCatalog(enabledTools []string) []map[string]any {
	catalog := make([]map[string]any, 0, len(enabledTools))
	for _, toolName := range enabledTools {
		mode := "read_only"
		if isAgentConfirmTool(toolName) {
			mode = "confirm"
		}
		catalog = append(catalog, map[string]any{
			"name":        toolName,
			"mode":        mode,
			"description": agentToolCompactDescription(toolName),
		})
	}
	return catalog
}

func (mgr *AgentMgr) buildAgentCapabilities(token util.JWTMessage, page map[string]any) map[string]any {
	return buildAgentCapabilitiesWithCatalog(token, page)
}

func buildAgentCapabilitiesWithCatalog(token util.JWTMessage, page map[string]any) map[string]any {
	pageRoute, _ := page["route"].(string)
	pageURL, _ := page["url"].(string)
	pageScope := agentPageScopeForToken(token, page)
	enabledTools := append([]string(nil), agentUserTools...)
	sort.Strings(enabledTools)

	confirmTools := make([]string, 0, len(enabledTools))
	for _, name := range enabledTools {
		if _, ok := agentConfirmToolSet[name]; ok {
			confirmTools = append(confirmTools, name)
		}
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
	}
}
