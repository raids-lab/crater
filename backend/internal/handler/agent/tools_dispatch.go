package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func isAgentReadOnlyTool(toolName string) bool {
	switch toolName {
	case agentToolGetJobDetail,
		agentToolGetJobEvents,
		agentToolGetJobLogs,
		agentToolDiagnoseJob,
		agentToolGetDiagnosticCtx,
		agentToolSearchSimilarFail,
		agentToolQueryJobMetrics,
		agentToolAnalyzeQueue,
		agentToolRealtimeCapacity,
		agentToolListImages,
		agentToolListCudaBase,
		agentToolListGPUModels,
		agentToolRecommendImages,
		agentToolListImageBuilds,
		agentToolGetImageBuild,
		agentToolGetImageAccess,
		agentToolCheckQuota,
		agentToolGetHealthOverview,
		agentToolListUserJobs,
		agentToolGetClusterHealth,
		agentToolListClusterJobs,
		agentToolListClusterNodes,
		agentToolDetectIdleJobs,
		agentToolGetJobTemplates,
		agentToolGetFailureStats,
		agentToolGetClusterReport,
		agentToolGetAdminOpsReport,
		agentToolResourceRecommend,
		agentToolGetNodeDetail,
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
		agentToolK8sGetEvents,
		agentToolK8sDescribe,
		agentToolK8sPodLogs,
		agentToolK8sGetService,
		agentToolK8sGetEndpoints,
		agentToolK8sGetIngress,
		agentToolPromQuery,
		agentToolHarborCheck,
		toolGetLatestAuditReport,
		toolListAuditItems,
		toolSaveAuditReport,
		toolGetApprovalHistory:
		return true
	default:
		return false
	}
}

func isAgentConfirmTool(toolName string) bool {
	switch toolName {
	case agentToolResubmitJob, agentToolStopJob, agentToolDeleteJob,
		agentToolCreateJupyter, agentToolCreateWebIDE, agentToolCreateTrain, agentToolCreateCustom,
		agentToolCreatePytorch, agentToolCreateTensorflow,
		agentToolCreateImage, agentToolManageBuild, agentToolRegisterImage, agentToolManageAccess,
		agentToolCordonNode, agentToolUncordonNode, agentToolDrainNode, agentToolDeletePod, agentToolRestartWL,
		agentToolK8sScaleWL, agentToolK8sLabelNode, agentToolK8sTaintNode, agentToolRunKubectl, agentToolAdminCommand,
		toolMarkAuditHandled, toolBatchStopJobs, toolNotifyJobOwner:
		return true
	default:
		return false
	}
}

func isAgentAutoActionTool(toolName string) bool {
	switch toolName {
	default:
		return false
	}
}

func isAgentAdminOnlyTool(toolName string) bool {
	switch toolName {
	case agentToolGetClusterHealth,
		agentToolGetClusterReport,
		agentToolGetNodeDetail,
		agentToolListClusterJobs,
		agentToolListClusterNodes,
		agentToolGetAdminOpsReport,
		agentToolListStoragePVCs,
		agentToolGetPVCDetail,
		agentToolGetPVCEvents,
		agentToolInspectJobStorage,
		agentToolStorageCapacity,
		agentToolNodeNetwork,
		agentToolDiagnoseJobNet,
		agentToolSandboxGrep,
		agentToolRuntimeSummary,
		agentToolK8sListNodes,
		agentToolPromQuery,
		agentToolHarborCheck,
		agentToolCordonNode,
		agentToolUncordonNode,
		agentToolDrainNode,
		agentToolDeletePod,
		agentToolRestartWL,
		agentToolK8sScaleWL,
		agentToolK8sLabelNode,
		agentToolK8sTaintNode,
		agentToolRunKubectl,
		agentToolAdminCommand,
		toolGetLatestAuditReport,
		toolListAuditItems,
		toolSaveAuditReport,
		toolMarkAuditHandled,
		toolBatchStopJobs,
		toolNotifyJobOwner:
		return true
	default:
		return false
	}
}

func normalizeAgentRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "coordinator", "planner", "explorer", "executor", "verifier", "guide", "general", "single_agent":
		return strings.TrimSpace(strings.ToLower(role))
	default:
		return "single_agent"
	}
}

func normalizeInternalToolRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "admin", "system_admin", "platform_admin":
		return "admin"
	default:
		return ""
	}
}

func normalizeRequestedSessionSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "ops_audit":
		return "ops_audit"
	case "system":
		return "system"
	case "benchmark":
		return "benchmark"
	default:
		return "chat"
	}
}

func defaultInternalSessionTitle(source, toolName, providedTitle string) string {
	title := strings.TrimSpace(providedTitle)
	if title != "" {
		return title
	}

	prefix := "[system]"
	switch source {
	case "ops_audit":
		prefix = "[audit]"
	case "benchmark":
		prefix = "[benchmark]"
	}

	name := strings.TrimSpace(toolName)
	if name == "" {
		name = "internal-task"
	}
	return fmt.Sprintf("%s %s", prefix, name)
}

func toolCallAuditSourceForSessionSource(sessionSource string) string {
	if normalizeRequestedSessionSource(sessionSource) == "benchmark" {
		return "benchmark"
	}
	return "backend"
}

func toolCallAuditSourceForExecution(sessionSource, executionBackend string) string {
	if normalizeRequestedSessionSource(sessionSource) == "benchmark" {
		return "benchmark"
	}
	if strings.TrimSpace(strings.ToLower(executionBackend)) == "python_local" {
		return "local"
	}
	return "backend"
}

func normalizeExecutionBackend(toolName, executionBackend string) string {
	normalized := strings.TrimSpace(strings.ToLower(executionBackend))
	if !isAgentConfirmTool(toolName) {
		return ""
	}
	switch normalized {
	case "", "backend":
		return "backend"
	case "python_local":
		return "python_local"
	default:
		return "backend"
	}
}

func (mgr *AgentMgr) ensureInternalAuditSession(c *gin.Context, req ExecuteToolRequest) error {
	if req.InternalContext == nil {
		return nil
	}

	source := normalizeRequestedSessionSource(req.SessionSource)
	if source == "chat" {
		source = "system"
	}

	_, _, err := mgr.agentService.GetOrCreateSessionWithSource(
		c.Request.Context(),
		req.SessionID,
		0,
		0,
		defaultInternalSessionTitle(source, req.ToolName, req.SessionTitle),
		nil,
		source,
	)
	return err
}

func (mgr *AgentMgr) resolveToolExecutionToken(c *gin.Context, req ExecuteToolRequest) (util.JWTMessage, error) {
	if req.InternalContext != nil {
		if normalizeInternalToolRole(req.InternalContext.Role) != "admin" {
			return util.JWTMessage{}, fmt.Errorf("unsupported internal tool role")
		}
		username := strings.TrimSpace(req.InternalContext.Username)
		if username == "" {
			username = "agent-pipeline"
		}
		accountName := strings.TrimSpace(req.InternalContext.AccountName)
		if accountName == "" {
			accountName = "system"
		}
		return util.JWTMessage{
			Username:     username,
			AccountName:  accountName,
			RoleAccount:  model.RoleAdmin,
			RolePlatform: model.RoleAdmin,
		}, nil
	}

	session, err := mgr.agentService.GetSession(c.Request.Context(), req.SessionID)
	if err != nil {
		return util.JWTMessage{}, fmt.Errorf("session not found")
	}
	return mgr.getSessionToken(c.Request.Context(), session)
}

func validateAgentToolAccess(agentRole string, toolName string) error {
	role := normalizeAgentRole(agentRole)

	switch role {
	case "coordinator", "planner", "explorer", "verifier", "guide", "general":
		if isAgentReadOnlyTool(toolName) {
			return nil
		}
		if isAgentAutoActionTool(toolName) {
			return fmt.Errorf("agent role '%s' cannot execute auto-action tools", role)
		}
		if isAgentConfirmTool(toolName) {
			return fmt.Errorf("agent role '%s' cannot execute confirmation tools", role)
		}
		return fmt.Errorf("agent role '%s' can only execute read-only tools", role)
	case "executor", "single_agent":
		if isAgentReadOnlyTool(toolName) || isAgentConfirmTool(toolName) || isAgentAutoActionTool(toolName) {
			return nil
		}
		return fmt.Errorf("tool '%s' is not supported", toolName)
	default:
		return fmt.Errorf("agent role '%s' is not allowed to execute tools", role)
	}
}

// ExecuteTool godoc
// @Summary Execute a named tool (called by the Python Agent service)
// @Description Routes tool_name to the appropriate internal handler. Write tools return confirmation_required.
// @Tags agent
// @Accept json
// @Produce json
// @Param request body ExecuteToolRequest true "Tool execution request"
// @Success 200 {object} AgentToolResponse
// @Router /api/agent/tools/execute [post]
//

func (mgr *AgentMgr) ExecuteTool(c *gin.Context) {
	if !mgr.isInternalToolRequestAuthorized(c) {
		resputil.HTTPError(c, http.StatusUnauthorized, "invalid internal agent token", resputil.TokenInvalid)
		return
	}

	var req ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	req.AgentRole = normalizeAgentRole(req.AgentRole)
	if accessErr := validateAgentToolAccess(req.AgentRole, req.ToolName); accessErr != nil {
		resputil.HTTPError(c, http.StatusForbidden, accessErr.Error(), resputil.TokenInvalid)
		return
	}
	if err := mgr.ensureInternalAuditSession(c, req); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create internal audit session: %v", err), resputil.NotSpecified)
		return
	}

	sessionToken, tokenErr := mgr.resolveToolExecutionToken(c, req)
	if tokenErr != nil {
		resputil.Error(c, fmt.Sprintf("failed to resolve tool actor: %v", tokenErr), resputil.NotSpecified)
		return
	}
	if isAgentAdminOnlyTool(req.ToolName) && sessionToken.RolePlatform != model.RoleAdmin {
		resputil.HTTPError(c, http.StatusForbidden, "admin privileges required for this tool", resputil.TokenInvalid)
		return
	}

	if isAgentConfirmTool(req.ToolName) {
		start := time.Now()
		executionBackend := normalizeExecutionBackend(req.ToolName, req.ExecutionBackend)
		confirmation := mgr.buildToolConfirmation(sessionToken, req.ToolName, req.ToolArgs)
		pendingResult, _ := json.Marshal(map[string]any{
			"description":       confirmation.Description,
			"riskLevel":         confirmation.RiskLevel,
			"interaction":       confirmation.Interaction,
			"form":              confirmation.Form,
			"execution_backend": executionBackend,
		})
		toolCallRecord := &model.AgentToolCall{
			SessionID:        req.SessionID,
			TurnID:           req.TurnID,
			ToolCallID:       req.ToolCallID,
			AgentID:          req.AgentID,
			AgentRole:        req.AgentRole,
			Source:           toolCallAuditSourceForExecution(req.SessionSource, executionBackend),
			ToolName:         req.ToolName,
			ToolArgs:         datatypes.JSON(req.ToolArgs),
			ToolResult:       pendingResult,
			ResultStatus:     agentToolStatusAwaitConfirm,
			ExecutionBackend: executionBackend,
			CreatedAt:        time.Now(),
		}
		toolCall, createErr := mgr.agentService.CreateToolCall(c.Request.Context(), toolCallRecord)
		if createErr != nil {
			resputil.Error(c, fmt.Sprintf("failed to create pending tool call: %v", createErr), resputil.NotSpecified)
			return
		}
		resputil.Success(c, AgentToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     agentToolStatusConfirmationRequired,
			Confirmation: &AgentToolConfirmation{
				ConfirmID:   strconv.FormatUint(uint64(toolCall.ID), 10),
				ToolName:    req.ToolName,
				Description: confirmation.Description,
				RiskLevel:   confirmation.RiskLevel,
				Interaction: confirmation.Interaction,
				Form:        confirmation.Form,
			},
			LatencyMs: int(time.Since(start).Milliseconds()),
		})
		return
	}

	start := time.Now()

	var result any
	var execErr error
	if isAgentAutoActionTool(req.ToolName) {
		result, execErr = mgr.executeAutoActionTool(c, sessionToken, req)
	} else {
		result, execErr = mgr.executeReadTool(c, sessionToken, req)
	}
	latencyMs := int(time.Since(start).Milliseconds())

	status := agentToolStatusSuccess
	var resultBytes json.RawMessage
	var errMsg string

	if execErr != nil {
		status = agentToolStatusError
		errMsg = execErr.Error()
		errJSON, _ := json.Marshal(map[string]string{"error": errMsg})
		resultBytes = errJSON
	} else {
		resultBytes, _ = json.Marshal(result)
	}

	mgr.agentService.LogToolCallAsync(
		req.SessionID, req.ToolName,
		req.ToolArgs, resultBytes,
		status, latencyMs, req.TurnID, req.ToolCallID, req.AgentID, req.AgentRole,
		toolCallAuditSourceForSessionSource(req.SessionSource),
	)

	resputil.Success(c, AgentToolResponse{
		ToolCallID: req.ToolCallID,
		Status:     status,
		Result:     resultBytes,
		Message:    errMsg,
		LatencyMs:  latencyMs,
	})
}

func (mgr *AgentMgr) executeAutoActionTool(_ *gin.Context, token util.JWTMessage, req ExecuteToolRequest) (any, error) {
	switch req.ToolName {
	default:
		return nil, fmt.Errorf("auto-action tool '%s' is not supported", req.ToolName)
	}
}

func (mgr *AgentMgr) executeReadTool(c *gin.Context, token util.JWTMessage, req ExecuteToolRequest) (any, error) {
	switch req.ToolName {
	case agentToolGetJobDetail:
		return mgr.toolGetJobDetail(c, token, req.ToolArgs)
	case agentToolGetJobEvents:
		return mgr.toolGetJobEvents(c, token, req.ToolArgs)
	case agentToolGetJobLogs:
		return mgr.toolGetJobLogs(c, token, req.ToolArgs)
	case agentToolDiagnoseJob:
		return mgr.toolDiagnoseJob(c, token, req.ToolArgs)
	case agentToolGetDiagnosticCtx:
		return mgr.toolGetDiagnosticContext(c, token, req.ToolArgs)
	case agentToolSearchSimilarFail:
		return mgr.toolSearchSimilarFailures(c, token, req.ToolArgs)
	case agentToolQueryJobMetrics:
		return mgr.toolQueryJobMetrics(c, token, req.ToolArgs)
	case agentToolAnalyzeQueue:
		return mgr.toolAnalyzeQueueStatus(c, token, req.ToolArgs)
	case agentToolRealtimeCapacity:
		return mgr.toolGetRealtimeCapacity(c, token, req.ToolArgs)
	case agentToolListImages:
		return mgr.toolListAvailableImages(c, token, req.ToolArgs)
	case agentToolListCudaBase:
		return mgr.toolListCudaBaseImages(c, token, req.ToolArgs)
	case agentToolListGPUModels:
		return mgr.toolListAvailableGPUModels(c, token, req.ToolArgs)
	case agentToolRecommendImages:
		return mgr.toolRecommendTrainingImages(c, token, req.ToolArgs)
	case agentToolListImageBuilds:
		return mgr.toolListImageBuilds(c, token, req.ToolArgs)
	case agentToolGetImageBuild:
		return mgr.toolGetImageBuildDetail(c, token, req.ToolArgs)
	case agentToolGetImageAccess:
		return mgr.toolGetImageAccessDetail(c, token, req.ToolArgs)
	case agentToolCheckQuota:
		return mgr.toolCheckQuota(c, token, req.ToolArgs)
	case agentToolGetHealthOverview:
		return mgr.toolGetHealthOverview(c, token, req.ToolArgs)
	case agentToolListUserJobs:
		return mgr.toolListUserJobs(c, token, req.ToolArgs)
	case agentToolGetClusterHealth:
		return mgr.toolGetClusterHealthOverview(c, token, req.ToolArgs)
	case agentToolListClusterJobs:
		return mgr.toolListClusterJobs(c, token, req.ToolArgs)
	case agentToolListClusterNodes:
		return mgr.toolListClusterNodes(c, token)
	case agentToolDetectIdleJobs:
		return mgr.toolDetectIdleJobs(c, token, req.ToolArgs)
	case agentToolGetJobTemplates:
		return mgr.toolGetJobTemplates(c, token, req.ToolArgs)
	case agentToolGetFailureStats:
		return mgr.toolGetFailureStatistics(c, token, req.ToolArgs)
	case agentToolGetClusterReport:
		return mgr.toolGetClusterHealthReport(c, token, req.ToolArgs)
	case agentToolGetAdminOpsReport:
		return mgr.toolGetAdminOpsReport(c, token, req.ToolArgs)
	case agentToolResourceRecommend:
		return mgr.toolGetResourceRecommendation(c, token, req.ToolArgs)
	case agentToolGetNodeDetail:
		return mgr.toolGetNodeDetail(c, token, req.ToolArgs)
	case agentToolListStoragePVCs:
		return mgr.toolListStoragePVCs(c, token, req.ToolArgs)
	case agentToolGetPVCDetail:
		return mgr.toolGetPVCDetail(c, token, req.ToolArgs)
	case agentToolGetPVCEvents:
		return mgr.toolGetPVCEvents(c, token, req.ToolArgs)
	case agentToolInspectJobStorage:
		return mgr.toolInspectJobStorage(c, token, req.ToolArgs)
	case agentToolStorageCapacity:
		return mgr.toolGetStorageCapacityOverview(c, token, req.ToolArgs)
	case agentToolNodeNetwork:
		return mgr.toolGetNodeNetworkSummary(c, token, req.ToolArgs)
	case agentToolDiagnoseJobNet:
		return mgr.toolDiagnoseDistributedJobNetwork(c, token, req.ToolArgs)
	case agentToolWebSearch:
		return mgr.toolWebSearch(c, token, req.ToolArgs)
	case agentToolSandboxGrep:
		return mgr.toolSandboxGrep(c, token, req.ToolArgs)
	case toolGetLatestAuditReport:
		return mgr.toolGetLatestAuditReport(c, token, req.ToolArgs)
	case toolListAuditItems:
		return mgr.toolListAuditItems(c, token, req.ToolArgs)
	case toolSaveAuditReport:
		return mgr.toolSaveAuditReport(c, token, req.ToolArgs)
	case toolGetApprovalHistory:
		return mgr.toolGetApprovalHistory(c, token, req.ToolArgs)
	case agentToolK8sListPods:
		return mgr.toolK8sListPods(c, token, req.ToolArgs)
	case agentToolK8sGetService:
		return mgr.toolK8sGetService(c, token, req.ToolArgs)
	case agentToolK8sGetEndpoints:
		return mgr.toolK8sGetEndpoints(c, token, req.ToolArgs)
	case agentToolK8sGetIngress:
		return mgr.toolK8sGetIngress(c, token, req.ToolArgs)
	default:
		return nil, fmt.Errorf("tool '%s' is not yet implemented", req.ToolName)
	}
}

func (mgr *AgentMgr) executeWriteTool(c *gin.Context, token util.JWTMessage, toolName string, rawArgs json.RawMessage) (any, error) {
	switch toolName {
	case agentToolDeleteJob:
		return mgr.toolDeleteJob(c, token, rawArgs)
	case agentToolStopJob:
		return mgr.toolStopJob(c, token, rawArgs)
	case agentToolResubmitJob:
		return mgr.toolResubmitJob(c, token, rawArgs)
	case agentToolCreateJupyter:
		return mgr.toolCreateJupyterJob(c, token, rawArgs)
	case agentToolCreateWebIDE:
		return mgr.toolCreateWebIDEJob(c, token, rawArgs)
	case agentToolCreateTrain:
		return mgr.toolCreateTrainingJob(c, token, rawArgs)
	case agentToolCreateCustom:
		return mgr.toolCreateCustomJob(c, token, rawArgs)
	case agentToolCreatePytorch:
		return mgr.toolCreatePytorchJob(c, token, rawArgs)
	case agentToolCreateTensorflow:
		return mgr.toolCreateTensorflowJob(c, token, rawArgs)
	case agentToolCreateImage:
		return mgr.toolCreateImageBuild(c, token, rawArgs)
	case agentToolManageBuild:
		return mgr.toolManageImageBuild(c, token, rawArgs)
	case agentToolRegisterImage:
		return mgr.toolRegisterExternalImage(c, token, rawArgs)
	case agentToolManageAccess:
		return mgr.toolManageImageAccess(c, token, rawArgs)
	case agentToolCordonNode:
		return mgr.toolCordonNode(c, token, rawArgs)
	case agentToolUncordonNode:
		return mgr.toolUncordonNode(c, token, rawArgs)
	case agentToolDrainNode:
		return mgr.toolDrainNode(c, token, rawArgs)
	case agentToolDeletePod:
		return mgr.toolDeletePod(c, token, rawArgs)
	case agentToolRestartWL:
		return mgr.toolRestartWorkload(c, token, rawArgs)
	case toolMarkAuditHandled:
		return mgr.toolMarkAuditHandled(c, token, rawArgs)
	case toolBatchStopJobs:
		return mgr.toolBatchStopJobs(c, token, rawArgs)
	case toolNotifyJobOwner:
		return mgr.toolNotifyJobOwner(c, token, rawArgs)
	default:
		return nil, fmt.Errorf("write tool '%s' is not supported", toolName)
	}
}
