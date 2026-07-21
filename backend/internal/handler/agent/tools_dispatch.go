package agent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func agentDispatchErrorf(format string, args ...any) error {
	return bizerr.BadRequest.ParameterError.New(fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...))
}

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
		agentToolListGPUModels,
		agentToolCheckQuota,
		agentToolListUserJobs,
		agentToolGetJobTemplates,
		agentToolResourceRecommend:
		return true
	default:
		return false
	}
}

func isAgentConfirmTool(toolName string) bool {
	switch toolName {
	case agentToolResubmitJob, agentToolStopJob, agentToolDeleteJob,
		agentToolCreateJupyter, agentToolCreateWebIDE, agentToolCreateCustom,
		agentToolCreatePytorch, agentToolCreateTensorflow:
		return true
	default:
		return false
	}
}

func isAgentAutoActionTool(toolName string) bool {
	_ = toolName
	return false
}

func isAgentAdminOnlyTool(_ string) bool {
	return false
}

func normalizeAgentRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case agentRoleCoordinator, "planner", "explorer", "executor", "verifier", "guide", "general", agentRoleSingleAgent:
		return strings.TrimSpace(strings.ToLower(role))
	default:
		return agentRoleSingleAgent
	}
}

func normalizeInternalToolRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case agentSessionSourceAdmin, "system_admin", "platform_admin":
		return agentSessionSourceAdmin
	default:
		return ""
	}
}

func normalizeRequestedSessionSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case agentSessionSourceOpsAudit:
		return agentSessionSourceOpsAudit
	case agentSessionSourceSystem:
		return agentSessionSourceSystem
	default:
		return agentSessionSourceChat
	}
}

func defaultInternalSessionTitle(source, toolName, providedTitle string) string {
	title := strings.TrimSpace(providedTitle)
	if title != "" {
		return title
	}

	prefix := "[system]"
	if source == agentSessionSourceOpsAudit {
		prefix = "[audit]"
	}

	name := strings.TrimSpace(toolName)
	if name == "" {
		name = "internal-task"
	}
	return fmt.Sprintf("%s %s", prefix, name)
}

func toolCallAuditSourceForSessionSource(sessionSource string) string {
	_ = normalizeRequestedSessionSource(sessionSource)
	return agentToolAuditSourceBackend
}

func toolCallAuditSourceForExecution(sessionSource, executionBackend string) string {
	_ = normalizeRequestedSessionSource(sessionSource)
	_ = executionBackend
	return agentToolAuditSourceBackend
}

func normalizeExecutionBackend(toolName, executionBackend string) string {
	normalized := strings.TrimSpace(strings.ToLower(executionBackend))
	if !isAgentConfirmTool(toolName) {
		return ""
	}
	if normalized == "" || normalized == agentToolAuditSourceBackend {
		return agentToolAuditSourceBackend
	}
	return agentToolAuditSourceBackend
}

func agentSessionAllowsAdminTools(session *model.AgentSession) bool {
	if session == nil {
		return false
	}
	switch normalizeRequestedSessionSource(session.Source) {
	case agentSessionSourceOpsAudit, agentSessionSourceSystem:
		return true
	}

	return agentSessionSurface(session) == agentSessionSourceAdmin
}

func normalizeAgentSurface(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case agentSessionSourceAdmin, "administrator", "management":
		return agentSessionSourceAdmin
	default:
		return agentSessionSourceUser
	}
}

func agentPageContextSurface(page map[string]any) string {
	if page == nil {
		return agentSessionSourceUser
	}
	if raw, _ := page["surface"].(string); normalizeAgentSurface(raw) == agentSessionSourceAdmin {
		return agentSessionSourceAdmin
	}
	for _, key := range []string{"route", "url"} {
		raw, _ := page[key].(string)
		if isAdminAgentRoute(raw) {
			return agentSessionSourceAdmin
		}
	}
	return agentSessionSourceUser
}

func agentPageScopeForToken(token util.JWTMessage, page map[string]any) string {
	if token.RolePlatform != model.RoleAdmin {
		return agentSessionSourceUser
	}
	return agentPageContextSurface(page)
}

func agentSessionSurface(session *model.AgentSession) string {
	if session == nil {
		return agentSessionSourceUser
	}
	switch normalizeRequestedSessionSource(session.Source) {
	case agentSessionSourceOpsAudit, agentSessionSourceSystem:
		return agentSessionSourceAdmin
	}
	return agentPageContextSurface(normalizePageContext(json.RawMessage(session.PageContext)))
}

func agentSessionMatchesSurface(session *model.AgentSession, surface string) bool {
	return agentSessionSurface(session) == normalizeAgentSurface(surface)
}

func isAdminAgentRoute(raw string) bool {
	route := strings.TrimSpace(raw)
	if route == "" {
		return false
	}
	if parsed, err := url.Parse(route); err == nil && parsed.Path != "" {
		route = parsed.Path
	}
	route = strings.TrimSpace(strings.ToLower(route))
	return route == "/admin" || strings.HasPrefix(route, "/admin/")
}

func effectiveAgentSessionToken(session *model.AgentSession, token util.JWTMessage) util.JWTMessage {
	if token.RolePlatform == model.RoleAdmin && !agentSessionAllowsAdminTools(session) {
		token.RolePlatform = model.RoleUser
	}
	return token
}

func authorizeAgentToolForSession(session *model.AgentSession, token util.JWTMessage, toolName string) error {
	if !isAgentAdminOnlyTool(toolName) {
		return nil
	}
	if token.RolePlatform != model.RoleAdmin {
		return agentDispatchErrorf("你当前没有管理员权限，不能执行该运维操作；如确需处理，请联系平台管理员或切换到管理员页面后再操作")
	}
	if !agentSessionAllowsAdminTools(session) {
		return agentDispatchErrorf("该运维操作只能在管理员页面执行；用户端会话不能发起节点、Pod 或集群级写操作")
	}
	return nil
}

func (mgr *AgentMgr) ensureInternalAuditSession(c *gin.Context, req *ExecuteToolRequest) error {
	if req.InternalContext == nil {
		return nil
	}

	source := normalizeRequestedSessionSource(req.SessionSource)
	if source == agentSessionSourceChat {
		source = agentSessionSourceSystem
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

func (mgr *AgentMgr) resolveToolExecutionToken(c *gin.Context, req *ExecuteToolRequest) (util.JWTMessage, error) {
	if req.InternalContext != nil {
		if normalizeInternalToolRole(req.InternalContext.Role) != agentSessionSourceAdmin {
			return util.JWTMessage{}, agentDispatchErrorf("unsupported internal tool role")
		}
		username := strings.TrimSpace(req.InternalContext.Username)
		if username == "" {
			username = "agent-pipeline"
		}
		accountName := strings.TrimSpace(req.InternalContext.AccountName)
		if accountName == "" {
			accountName = agentSessionSourceSystem
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
		return util.JWTMessage{}, agentDispatchErrorf("session not found")
	}
	return mgr.getSessionToken(c.Request.Context(), session)
}

func validateAgentToolAccess(agentRole, toolName string) error {
	role := normalizeAgentRole(agentRole)

	switch role {
	case agentRoleCoordinator, "planner", "explorer", "verifier", "guide", "general":
		if isAgentReadOnlyTool(toolName) {
			return nil
		}
		if isAgentAutoActionTool(toolName) {
			return agentDispatchErrorf("agent role '%s' cannot execute auto-action tools", role)
		}
		if isAgentConfirmTool(toolName) {
			return agentDispatchErrorf("agent role '%s' cannot execute confirmation tools", role)
		}
		return agentDispatchErrorf("agent role '%s' can only execute read-only tools", role)
	case "executor", agentRoleSingleAgent:
		if isAgentReadOnlyTool(toolName) || isAgentConfirmTool(toolName) || isAgentAutoActionTool(toolName) {
			return nil
		}
		return agentDispatchErrorf("tool '%s' is not supported", toolName)
	default:
		return agentDispatchErrorf("agent role '%s' is not allowed to execute tools", role)
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
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("invalid internal agent token"))
		return
	}

	var req ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		agentBadRequest(c, err.Error())
		return
	}
	req.AgentRole = normalizeAgentRole(req.AgentRole)
	if accessErr := validateAgentToolAccess(req.AgentRole, req.ToolName); accessErr != nil {
		agentForbidden(c, accessErr.Error())
		return
	}
	if err := mgr.ensureInternalAuditSession(c, &req); err != nil {
		agentInternalError(c, fmt.Sprintf("failed to create internal audit session: %v", err))
		return
	}

	sessionToken, tokenErr := mgr.resolveToolExecutionToken(c, &req)
	if tokenErr != nil {
		agentInternalError(c, fmt.Sprintf("failed to resolve tool actor: %v", tokenErr))
		return
	}
	if req.InternalContext == nil {
		session, sessionErr := mgr.agentService.GetSession(c.Request.Context(), req.SessionID)
		if sessionErr != nil {
			agentForbidden(c, "session not found")
			return
		}
		if authErr := authorizeAgentToolForSession(session, sessionToken, req.ToolName); authErr != nil {
			agentForbidden(c, authErr.Error())
			return
		}
	} else if isAgentAdminOnlyTool(req.ToolName) && sessionToken.RolePlatform != model.RoleAdmin {
		agentForbidden(c, "你当前没有管理员权限，不能执行该运维操作；如确需处理，请联系平台管理员或切换到管理员页面后再操作")
		return
	}

	if isAgentConfirmTool(req.ToolName) {
		if req.InternalContext == nil {
			if preflightErr := mgr.validateOwnedJobMutationBeforeConfirmation(c, sessionToken, req.ToolName, req.ToolArgs); preflightErr != nil {
				agentForbidden(c, preflightErr.Error())
				return
			}
		}
		start := time.Now()
		executionBackend := normalizeExecutionBackend(req.ToolName, req.ExecutionBackend)
		confirmation := mgr.buildToolConfirmation(sessionToken, req.ToolName, req.ToolArgs)
		pendingResult, _ := json.Marshal(map[string]any{
			"description":           confirmation.Description,
			"riskLevel":             confirmation.RiskLevel,
			"permissionExplanation": confirmation.PermissionExplanation,
			"riskExplanation":       confirmation.RiskExplanation,
			"affectedResources":     confirmation.AffectedResources,
			"interaction":           confirmation.Interaction,
			"form":                  confirmation.Form,
			"execution_backend":     executionBackend,
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
			agentInternalError(c, fmt.Sprintf("failed to create pending tool call: %v", createErr))
			return
		}
		resputil.Success(c, AgentToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     agentToolStatusConfirmationRequired,
			Confirmation: &AgentToolConfirmation{
				ConfirmID:             strconv.FormatUint(uint64(toolCall.ID), 10),
				ToolName:              req.ToolName,
				Description:           confirmation.Description,
				RiskLevel:             confirmation.RiskLevel,
				PermissionExplanation: confirmation.PermissionExplanation,
				RiskExplanation:       confirmation.RiskExplanation,
				AffectedResources:     confirmation.AffectedResources,
				Interaction:           confirmation.Interaction,
				Form:                  confirmation.Form,
			},
			LatencyMs: int(time.Since(start).Milliseconds()),
		})
		return
	}

	start := time.Now()

	result, execErr := mgr.executeReadTool(c, sessionToken, &req)
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

//nolint:gocyclo // Tool dispatch intentionally maps all read-only tools in one place.
func (mgr *AgentMgr) executeReadTool(c *gin.Context, token util.JWTMessage, req *ExecuteToolRequest) (any, error) {
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
	case agentToolListGPUModels:
		return mgr.toolListAvailableGPUModels(c, token, req.ToolArgs)
	case agentToolCheckQuota:
		return mgr.toolCheckQuota(c, token, req.ToolArgs)
	case agentToolListUserJobs:
		return mgr.toolListUserJobs(c, token, req.ToolArgs)
	case agentToolGetJobTemplates:
		return mgr.toolGetJobTemplates(c, token, req.ToolArgs)
	case agentToolResourceRecommend:
		return mgr.toolGetResourceRecommendation(c, token, req.ToolArgs)
	default:
		return nil, agentDispatchErrorf("tool '%s' is not yet implemented", req.ToolName)
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
	case agentToolCreateCustom:
		return mgr.toolCreateCustomJob(c, token, rawArgs)
	case agentToolCreatePytorch:
		return mgr.toolCreatePytorchJob(c, token, rawArgs)
	case agentToolCreateTensorflow:
		return mgr.toolCreateTensorflowJob(c, token, rawArgs)
	default:
		return nil, agentDispatchErrorf("write tool '%s' is not supported", toolName)
	}
}
