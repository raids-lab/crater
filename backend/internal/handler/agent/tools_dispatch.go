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
		toolGetLatestAuditReport,
		toolListAuditItems,
		toolSaveAuditReport:
		return true
	default:
		return false
	}
}

func isAgentConfirmTool(toolName string) bool {
	switch toolName {
	case agentToolResubmitJob, agentToolStopJob, agentToolDeleteJob, agentToolCreateJupyter, agentToolCreateTrain,
		toolMarkAuditHandled, toolBatchStopJobs, toolNotifyJobOwner:
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
		if isAgentConfirmTool(toolName) {
			return fmt.Errorf("agent role '%s' cannot execute confirmation tools", role)
		}
		return fmt.Errorf("agent role '%s' can only execute read-only tools", role)
	case "executor", "single_agent":
		if isAgentReadOnlyTool(toolName) || isAgentConfirmTool(toolName) {
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
//nolint:gocyclo // Tool routing dispatches many named tools in one function.
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

	sessionToken, tokenErr := mgr.resolveToolExecutionToken(c, req)
	if tokenErr != nil {
		resputil.Error(c, fmt.Sprintf("failed to resolve tool actor: %v", tokenErr), resputil.NotSpecified)
		return
	}

	switch req.ToolName {
	case agentToolResubmitJob, agentToolStopJob, agentToolDeleteJob, agentToolCreateJupyter, agentToolCreateTrain,
		toolMarkAuditHandled, toolBatchStopJobs, toolNotifyJobOwner:
		start := time.Now()
		confirmation := mgr.buildToolConfirmation(sessionToken, req.ToolName, req.ToolArgs)
		pendingResult, _ := json.Marshal(map[string]any{
			"description": confirmation.Description,
			"riskLevel":   confirmation.RiskLevel,
			"interaction": confirmation.Interaction,
			"form":        confirmation.Form,
		})
		toolCall, createErr := mgr.agentService.CreateToolCall(c.Request.Context(), &model.AgentToolCall{
			SessionID:    req.SessionID,
			TurnID:       req.TurnID,
			ToolCallID:   req.ToolCallID,
			AgentID:      req.AgentID,
			AgentRole:    req.AgentRole,
			ToolName:     req.ToolName,
			ToolArgs:     datatypes.JSON(req.ToolArgs),
			ToolResult:   pendingResult,
			ResultStatus: agentToolStatusAwaitConfirm,
			CreatedAt:    time.Now(),
		})
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

	result, execErr := mgr.executeReadTool(c, sessionToken, req)
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
	)

	resputil.Success(c, AgentToolResponse{
		ToolCallID: req.ToolCallID,
		Status:     status,
		Result:     resultBytes,
		Message:    errMsg,
		LatencyMs:  latencyMs,
	})
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
	case toolGetLatestAuditReport:
		return mgr.toolGetLatestAuditReport(c, token, req.ToolArgs)
	case toolListAuditItems:
		return mgr.toolListAuditItems(c, token, req.ToolArgs)
	case toolSaveAuditReport:
		return mgr.toolSaveAuditReport(c, token, req.ToolArgs)
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
	case agentToolCreateTrain:
		return mgr.toolCreateTrainingJob(c, token, rawArgs)
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
