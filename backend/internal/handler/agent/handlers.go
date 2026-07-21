package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/prompts"
)

const (
	agentAskTimeout           = 120 * time.Second
	agentRecentHistoryLimit   = 8
	agentSessionTitleMaxRunes = 32
)

func agentBadRequest(c *gin.Context, msg string) {
	resputil.HandleError(c, bizerr.BadRequest.ParameterError.New(msg))
}

func agentInternalError(c *gin.Context, msg string) {
	resputil.HandleError(c, bizerr.Internal.ServiceError.New(msg))
}

func agentForbidden(c *gin.Context, msg string) {
	resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New(msg))
}

func agentNotFound(c *gin.Context, msg string) {
	resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New(msg))
}

func (mgr *AgentMgr) requireConfiguredLLMAvailable(ctx context.Context, userID uint) (*service.LLMConfig, error) {
	if mgr.configService == nil {
		return nil, bizerr.Internal.ServiceError.New("LLM 配置服务未初始化")
	}

	cfg, err := mgr.configService.GetEffectiveLLMConfig(ctx, userID)
	if err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(err, "读取 LLM 配置失败")
	}
	if cfg == nil ||
		strings.TrimSpace(cfg.BaseURL) == "" ||
		strings.TrimSpace(cfg.APIKey) == "" ||
		strings.TrimSpace(cfg.ModelName) == "" {
		return nil, bizerr.BadRequest.ParameterError.New(
			"LLM 配置不完整，请先在系统配置中设置 BaseURL、API Key 和 Model",
		)
	}

	checkCtx, cancel := context.WithTimeout(ctx, agentLLMPreflightTimeout)
	defer cancel()
	if err := prompts.CheckLLMAvailable(
		mgr.httpClient,
		checkCtx,
		cfg.GetChatCompletionURL(),
		cfg.APIKey,
		cfg.ModelName,
	); err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(
			err,
			"LLM 服务不可用，请检查平台 LLM 配置、API Key、模型名或网络连通性",
		)
	}
	return cfg, nil
}

func (mgr *AgentMgr) failAgentTurn(ctx context.Context, turnID, message string) {
	errorMetadata, _ := json.Marshal(map[string]any{"errorMessage": message})
	_ = mgr.agentService.UpdateTurnStatus(ctx, turnID, agentTurnStatusFailed, nil, errorMetadata)
}

// Chat godoc
// @Summary Agent chat (SSE)
// @Description Create or continue an agent chat session; streams SSE events from the Python Agent service.
// @Tags agent
// @Accept json
// @Produce text/event-stream
// @Param request body AgentChatRequest true "Chat request"
// @Router /api/v1/agent/chat [post]
//
//nolint:gocyclo // Chat wires validation, session setup, persistence and SSE streaming in one endpoint.
func (mgr *AgentMgr) Chat(c *gin.Context) {
	var req AgentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		agentBadRequest(c, err.Error())
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Message)) > agentChatMessageMaxRunes {
		agentBadRequest(c, fmt.Sprintf("message exceeds %d characters", agentChatMessageMaxRunes))
		return
	}

	token := util.GetToken(c)
	orchestrationMode := normalizeOrchestrationMode(req.OrchestrationMode)
	requestPageContext := normalizePageContext(req.PageContext)
	requestSurface := agentPageScopeForToken(token, requestPageContext)

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, created, err := mgr.agentService.GetOrCreateSession(
		c.Request.Context(),
		sessionID,
		token.UserID,
		token.AccountID,
		buildAgentSessionTitle(req.Message, req.PageContext),
		req.PageContext,
	)
	if errors.Is(err, service.ErrAgentSessionDeleted) {
		sessionID = uuid.New().String()
		session, created, err = mgr.agentService.GetOrCreateSession(
			c.Request.Context(),
			sessionID,
			token.UserID,
			token.AccountID,
			buildAgentSessionTitle(req.Message, req.PageContext),
			req.PageContext,
		)
	}
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to create/load session: %v", err))
		return
	}
	if session.UserID != token.UserID || session.AccountID != token.AccountID {
		agentForbidden(c, "session not found")
		return
	}
	if !isChatSessionSource(session.Source) {
		agentForbidden(c, "session not found")
		return
	}
	if !created && agentSessionSurface(session) != requestSurface {
		sessionID = uuid.New().String()
		_, _, err = mgr.agentService.GetOrCreateSession(
			c.Request.Context(),
			sessionID,
			token.UserID,
			token.AccountID,
			buildAgentSessionTitle(req.Message, req.PageContext),
			req.PageContext,
		)
		if err != nil {
			agentInternalError(c, fmt.Sprintf("failed to create scoped session: %v", err))
			return
		}
	}
	_ = mgr.agentService.UpdateSessionOrchestrationMode(c.Request.Context(), sessionID, orchestrationMode)

	historyMessages, historyErr := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if historyErr != nil {
		agentInternalError(c, fmt.Sprintf("failed to load session history: %v", historyErr))
		return
	}
	historyToolCalls, toolCallErr := mgr.agentService.ListToolCalls(c.Request.Context(), sessionID)
	if toolCallErr != nil {
		agentInternalError(c, fmt.Sprintf("failed to load session tool calls: %v", toolCallErr))
		return
	}

	historyForPrompt := filterHistoryMessagesForRequest(historyMessages, req.RequestID)

	if !historyContainsRequestID(historyMessages, req.RequestID) {
		userMsg := &model.AgentMessage{
			SessionID: sessionID,
			Role:      "user",
			Content:   req.Message,
			CreatedAt: time.Now(),
		}
		if req.RequestID != "" {
			metadata, _ := json.Marshal(map[string]any{
				"requestId": req.RequestID,
			})
			userMsg.Metadata = metadata
		}
		if saveErr := mgr.agentService.SaveMessage(c.Request.Context(), userMsg); saveErr != nil {
			_ = saveErr
		}
	}

	effectiveToken := effectiveAgentSessionToken(session, token)
	continuationContext := mgr.buildAgentContinuation(c.Request.Context(), sessionID)
	turnID := uuid.New().String()
	_, err = mgr.agentService.CreateTurn(c.Request.Context(), &model.AgentTurn{
		TurnID:            turnID,
		SessionID:         sessionID,
		RequestID:         req.RequestID,
		OrchestrationMode: orchestrationMode,
		Status:            agentTurnStatusRunning,
		StartedAt:         time.Now(),
		Metadata:          datatypes.JSON(req.ClientContext),
	})
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to create agent turn: %v", err))
		return
	}
	if _, err = mgr.requireConfiguredLLMAvailable(c.Request.Context(), token.UserID); err != nil {
		mgr.failAgentTurn(context.Background(), turnID, err.Error())
		resputil.HandleError(c, err)
		return
	}
	agentPayload := mgr.buildPythonAgentPayload(
		c.Request.Context(),
		sessionID,
		turnID,
		req.Message,
		effectiveToken,
		requestPageContext,
		normalizeClientContext(req.ClientContext),
		orchestrationMode,
		historyForPrompt,
		historyToolCalls,
		continuationContext,
		token.UserID,
	)
	mgr.streamPythonAgentResponse(c, sessionID, turnID, orchestrationMode, agentPayload, true)
}

// Ask godoc
// @Summary Ask-only chat (SSE)
// @Description Create or continue an agent chat session and answer with the configured LLM without tool execution.
// @Tags agent
// @Accept json
// @Produce text/event-stream
// @Param request body AgentAskRequest true "Ask request"
// @Router /api/v1/agent/ask/stream [post]
//
//nolint:gocyclo // Ask endpoint keeps streaming, persistence and error handling together.
func (mgr *AgentMgr) Ask(c *gin.Context) {
	var req AgentAskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		agentBadRequest(c, err.Error())
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Message)) > agentChatMessageMaxRunes {
		agentBadRequest(c, fmt.Sprintf("message exceeds %d characters", agentChatMessageMaxRunes))
		return
	}
	if mgr.configService == nil {
		agentInternalError(c, "LLM 配置服务未初始化")
		return
	}

	token := util.GetToken(c)
	pageContext := normalizePageContext(req.PageContext)
	requestSurface := agentPageScopeForToken(token, pageContext)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, created, err := mgr.agentService.GetOrCreateSession(
		c.Request.Context(),
		sessionID,
		token.UserID,
		token.AccountID,
		buildAgentSessionTitle(req.Message, req.PageContext),
		req.PageContext,
	)
	if errors.Is(err, service.ErrAgentSessionDeleted) {
		sessionID = uuid.New().String()
		session, created, err = mgr.agentService.GetOrCreateSession(
			c.Request.Context(),
			sessionID,
			token.UserID,
			token.AccountID,
			buildAgentSessionTitle(req.Message, req.PageContext),
			req.PageContext,
		)
	}
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to create/load session: %v", err))
		return
	}
	if session.UserID != token.UserID || session.AccountID != token.AccountID || !isChatSessionSource(session.Source) {
		agentForbidden(c, "session not found")
		return
	}
	if !created && agentSessionSurface(session) != requestSurface {
		sessionID = uuid.New().String()
		_, _, err = mgr.agentService.GetOrCreateSession(
			c.Request.Context(),
			sessionID,
			token.UserID,
			token.AccountID,
			buildAgentSessionTitle(req.Message, req.PageContext),
			req.PageContext,
		)
		if err != nil {
			agentInternalError(c, fmt.Sprintf("failed to create scoped ask session: %v", err))
			return
		}
	}

	historyMessages, historyErr := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if historyErr != nil {
		agentInternalError(c, fmt.Sprintf("failed to load session history: %v", historyErr))
		return
	}
	if !historyContainsRequestID(historyMessages, req.RequestID) {
		userMsg := &model.AgentMessage{
			SessionID: sessionID,
			Role:      "user",
			Content:   req.Message,
			CreatedAt: time.Now(),
		}
		if req.RequestID != "" {
			metadata, _ := json.Marshal(map[string]any{
				"requestId": req.RequestID,
				"mode":      "ask",
			})
			userMsg.Metadata = metadata
		}
		_ = mgr.agentService.SaveMessage(c.Request.Context(), userMsg)
	}

	turnID := uuid.New().String()
	_, err = mgr.agentService.CreateTurn(c.Request.Context(), &model.AgentTurn{
		TurnID:            turnID,
		SessionID:         sessionID,
		RequestID:         req.RequestID,
		OrchestrationMode: "ask",
		Status:            agentTurnStatusRunning,
		StartedAt:         time.Now(),
		Metadata:          datatypes.JSON(req.ClientContext),
	})
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to create ask turn: %v", err))
		return
	}

	mgr.streamAskResponse(c, sessionID, turnID, req.Message, req.RequestID, historyMessages, token.UserID)
}

func (mgr *AgentMgr) streamAskResponse(
	c *gin.Context,
	sessionID string,
	turnID string,
	message string,
	requestID string,
	historyMessages []*model.AgentMessage,
	userID uint,
) {
	cfg, err := mgr.requireConfiguredLLMAvailable(c.Request.Context(), userID)
	if err != nil {
		mgr.failAgentTurn(context.Background(), turnID, err.Error())
		resputil.HandleError(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("X-Agent-Session-ID", sessionID)
	c.Header("X-Agent-Turn-ID", turnID)
	c.Header("X-Agent-Orchestration-Mode", "ask")
	c.Status(http.StatusOK)

	agentID := "ask-1"
	agentRole := "ask"
	startPayload := map[string]any{
		"turnId":    turnID,
		"sessionId": sessionID,
		"agentId":   agentID,
		"agentRole": agentRole,
		"status":    agentTurnStatusRunning,
		"summary":   "ask 正在回答",
	}
	_ = writeSyntheticSSEEvent(c, "agent_run_started", startPayload)

	systemPrompt := strings.Join([]string{
		"你是 Crater 平台的 ask 助手。",
		"你负责回答用户关于平台、作业、排障和使用方式的问题，但不执行写操作、不创建资源、不调用工具。",
		"如果用户请求删除、停止、创建、排空节点等操作，请说明 ask 只能解释和建议；需要执行请切换到 agent。",
		"回答要简洁、准确；不知道时说明缺少哪些信息，不要编造。",
	}, "\n")
	userPrompt := buildAskUserPrompt(message, historyMessages)
	ctx, cancel := context.WithTimeout(c.Request.Context(), agentAskTimeout)
	defer cancel()

	var full strings.Builder
	reply, err := prompts.CallLLMTextStream(
		mgr.httpClient,
		ctx,
		cfg.GetChatCompletionURL(),
		cfg.APIKey,
		cfg.ModelName,
		systemPrompt,
		userPrompt,
		func(delta string) error {
			full.WriteString(delta)
			return writeSyntheticSSEEvent(c, "message", map[string]any{
				"turnId":    turnID,
				"sessionId": sessionID,
				"agentId":   agentID,
				"agentRole": agentRole,
				"content":   delta,
				"partial":   true,
			})
		},
	)
	if err != nil {
		if ctx.Err() != nil || c.Request.Context().Err() != nil {
			_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, agentTurnStatusCancelled, nil, nil)
			return
		}
		_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, agentTurnStatusFailed, nil, nil)
		_ = writeSyntheticSSEEvent(c, "error", map[string]any{"message": fmt.Sprintf("ask 调用失败：%v", err)})
		_ = writeSyntheticSSEEvent(c, "done", map[string]any{})
		return
	}
	if strings.TrimSpace(reply) == "" {
		reply = full.String()
	}

	assistantMsg := &model.AgentMessage{
		SessionID: sessionID,
		Role:      agentMessageRoleAssistant,
		Content:   reply,
		CreatedAt: time.Now(),
	}
	if requestID != "" {
		metadata, _ := json.Marshal(map[string]any{
			"requestId": requestID,
			"mode":      "ask",
		})
		assistantMsg.Metadata = metadata
	}
	var finalMessageID *uint
	if saveErr := mgr.agentService.SaveMessage(context.Background(), assistantMsg); saveErr == nil {
		finalMessageID = &assistantMsg.ID
	}
	_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, agentTurnStatusCompleted, finalMessageID, nil)
	_ = writeSyntheticSSEEvent(c, "final_answer", map[string]any{
		"turnId":           turnID,
		"sessionId":        sessionID,
		"agentId":          agentID,
		"agentRole":        agentRole,
		"content":          reply,
		"feedbackTargetId": fmt.Sprintf("%d", assistantMsg.ID),
	})
	_ = writeSyntheticSSEEvent(c, "done", map[string]any{})
}

func buildAskUserPrompt(message string, historyMessages []*model.AgentMessage) string {
	recent := make([]string, 0, agentRecentHistoryLimit)
	start := len(historyMessages) - agentRecentHistoryLimit
	if start < 0 {
		start = 0
	}
	for _, msg := range historyMessages[start:] {
		role := strings.TrimSpace(msg.Role)
		if role == "" || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		recent = append(recent, fmt.Sprintf("%s: %s", role, strings.TrimSpace(msg.Content)))
	}
	if len(recent) == 0 {
		return fmt.Sprintf("用户问题：%s", message)
	}
	return fmt.Sprintf("最近对话：\n%s\n\n用户当前问题：%s", strings.Join(recent, "\n"), message)
}

func isChatSessionSource(source string) bool {
	normalized := strings.TrimSpace(strings.ToLower(source))
	return normalized == "" || normalized == agentSessionSourceChat
}

func buildAgentSessionTitle(message string, pageContext json.RawMessage) string {
	title := strings.TrimSpace(message)
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		title = "新的 Agent 会话"
	}

	var ctx map[string]any
	if len(pageContext) > 0 && json.Unmarshal(pageContext, &ctx) == nil {
		if jobName, ok := ctx["jobName"].(string); ok && strings.TrimSpace(jobName) != "" && !strings.Contains(title, jobName) {
			title = fmt.Sprintf("%s · %s", strings.TrimSpace(jobName), title)
		} else if nodeName, ok := ctx["nodeName"].(string); ok && strings.TrimSpace(nodeName) != "" && !strings.Contains(title, nodeName) {
			title = fmt.Sprintf("%s · %s", strings.TrimSpace(nodeName), title)
		}
	}

	runes := []rune(title)
	if len(runes) > agentSessionTitleMaxRunes {
		return string(runes[:agentSessionTitleMaxRunes]) + "…"
	}
	return title
}

func (mgr *AgentMgr) respondOwnedSessionData(
	c *gin.Context,
	errorPrefix string,
	load func(context.Context, string) (any, error),
) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		agentBadRequest(c, "sessionId is required")
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		agentForbidden(c, "session not found")
		return
	}

	data, err := load(c.Request.Context(), sessionID)
	if err != nil {
		agentInternalError(c, fmt.Sprintf("%s: %v", errorPrefix, err))
		return
	}
	resputil.Success(c, data)
}

// ResumeAfterConfirmation godoc
// @Summary Resume an agent turn after a confirmation result
// @Description Resumes a paused agent turn after the confirmation result has been recorded.
// @Tags agent
// @Accept json
// @Produce text/event-stream
// @Param request body AgentResumeRequest true "Resume request"
// @Router /api/v1/agent/chat/resume [post]
//
//nolint:gocyclo // Resume handles both streamed continuation and already-settled confirmation outcomes.
func (mgr *AgentMgr) ResumeAfterConfirmation(c *gin.Context) {
	var req AgentResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		agentBadRequest(c, err.Error())
		return
	}

	token := util.GetToken(c)
	confirmID, err := strconv.ParseUint(req.ConfirmID, 10, 64)
	if err != nil {
		agentBadRequest(c, "invalid confirmId")
		return
	}
	toolCall, err := mgr.agentService.GetToolCallByID(c.Request.Context(), uint(confirmID))
	if err != nil {
		agentNotFound(c, "confirmation result not found")
		return
	}
	if toolCall.ResultStatus == agentToolStatusAwaitConfirm {
		agentBadRequest(c, "confirmation has not completed yet")
		return
	}
	if mgr.hasOtherPendingConfirmations(c.Request.Context(), toolCall.TurnID, toolCall.ID) {
		resputil.Success(c, AgentToolResponse{
			Status:  agentToolStatusAwaitingConfirmation,
			Message: "仍有其他待确认操作，请处理完本轮所有确认卡后再继续。",
		})
		return
	}
	if toolCall.ResultStatus == agentToolStatusRejected {
		turnID := toolCall.TurnID
		if turnID == "" {
			turnID = uuid.New().String()
		}
		mgr.streamConfirmationOutcome(c, toolCall.SessionID, turnID, toolCall)
		return
	}

	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), toolCall.SessionID, token.UserID)
	if err != nil {
		agentForbidden(c, "confirmation result not found")
		return
	}

	sourceTurn, sourceTurnErr := mgr.agentService.GetTurn(c.Request.Context(), toolCall.TurnID)
	if sourceTurnErr == nil {
		orchestrationMode := normalizeOrchestrationMode(sourceTurn.OrchestrationMode)
		sessionToken, tokenErr := mgr.getSessionToken(c.Request.Context(), session)
		if tokenErr != nil {
			agentInternalError(c, fmt.Sprintf("failed to resolve session actor: %v", tokenErr))
			return
		}
		historyMessages, historyErr := mgr.agentService.ListMessages(c.Request.Context(), toolCall.SessionID)
		if historyErr != nil {
			agentInternalError(c, fmt.Sprintf("failed to load session history: %v", historyErr))
			return
		}
		historyToolCalls, toolCallErr := mgr.agentService.ListToolCalls(c.Request.Context(), toolCall.SessionID)
		if toolCallErr != nil {
			agentInternalError(c, fmt.Sprintf("failed to load session tool calls: %v", toolCallErr))
			return
		}

		if err := mgr.agentService.UpdateTurnStatus(c.Request.Context(), sourceTurn.TurnID, agentTurnStatusRunning, nil, nil); err != nil {
			agentInternalError(c, fmt.Sprintf("failed to resume source turn: %v", err))
			return
		}
		if _, err = mgr.requireConfiguredLLMAvailable(c.Request.Context(), sessionToken.UserID); err != nil {
			mgr.failAgentTurn(context.Background(), sourceTurn.TurnID, err.Error())
			resputil.HandleError(c, err)
			return
		}

		agentPayload := mgr.buildPythonAgentPayload(
			c.Request.Context(),
			toolCall.SessionID,
			sourceTurn.TurnID,
			"继续完成上一轮计划",
			sessionToken,
			normalizePageContext(json.RawMessage(session.PageContext)),
			normalizeClientContext(json.RawMessage(sourceTurn.Metadata)),
			orchestrationMode,
			historyMessages,
			historyToolCalls,
			mgr.buildResumeContinuation(c.Request.Context(), sourceTurn, toolCall),
			sessionToken.UserID,
		)
		mgr.streamPythonAgentResponse(c, toolCall.SessionID, sourceTurn.TurnID, orchestrationMode, agentPayload, true)
		return
	}

	turnID := toolCall.TurnID
	if turnID == "" {
		turnID = uuid.New().String()
	}
	mgr.streamConfirmationOutcome(c, toolCall.SessionID, turnID, toolCall)
}

// ConfirmToolExecution godoc
// @Summary Confirm or reject a write tool operation
// @Description Called by the frontend after the user confirms or rejects a write operation (e.g., stop_job).
// @Tags agent
// @Accept json
// @Produce json
// @Param request body ConfirmToolRequest true "Confirmation request"
// @Success 200 {object} resputil.Response[AgentToolResponse]
// @Router /api/v1/agent/chat/confirm [post]
//
//nolint:gocyclo // Confirmation handling validates ownership, executes/rejects and records the result atomically.
func (mgr *AgentMgr) ConfirmToolExecution(c *gin.Context) {
	var req ConfirmToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		agentBadRequest(c, err.Error())
		return
	}

	token := util.GetToken(c)
	confirmID, err := strconv.ParseUint(req.ConfirmID, 10, 64)
	if err != nil {
		agentBadRequest(c, "invalid confirmId")
		return
	}
	toolCall, err := mgr.agentService.GetToolCallByID(c.Request.Context(), uint(confirmID))
	if err != nil {
		agentNotFound(c, "pending action not found")
		return
	}
	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), toolCall.SessionID, token.UserID)
	if err != nil {
		agentForbidden(c, "pending action not found")
		return
	}
	if toolCall.ResultStatus != agentToolStatusAwaitConfirm {
		agentBadRequest(c, "pending action is no longer awaiting confirmation")
		return
	}

	if !req.Confirmed {
		summary := mgr.buildToolOutcomeMessage(toolCall.ToolName, agentToolStatusRejected, nil, "Operation rejected by user.")
		confirmed := false
		if updateErr := mgr.agentService.UpdateToolCallOutcome(
			c.Request.Context(),
			toolCall.ID,
			agentToolStatusRejected,
			json.RawMessage(toolCall.ToolResult),
			&confirmed,
		); updateErr != nil {
			agentInternalError(c, fmt.Sprintf("failed to update pending action: %v", updateErr))
			return
		}
		resultBytes, _ := json.Marshal(map[string]any{
			"confirmId": req.ConfirmID,
			"confirmed": false,
		})
		resputil.Success(c, AgentToolResponse{
			Status:  agentToolStatusRejected,
			Result:  resultBytes,
			Message: summary,
		})
		return
	}

	sessionToken, tokenErr := mgr.getSessionToken(c.Request.Context(), session)
	if tokenErr != nil {
		agentInternalError(c, fmt.Sprintf("failed to resolve session actor: %v", tokenErr))
		return
	}
	if authErr := authorizeAgentToolForSession(session, sessionToken, toolCall.ToolName); authErr != nil {
		agentForbidden(c, authErr.Error())
		return
	}

	mergedArgs, mergeErr := mergeToolArgsWithPayload(json.RawMessage(toolCall.ToolArgs), req.Payload)
	if mergeErr != nil {
		agentBadRequest(c, mergeErr.Error())
		return
	}
	if len(req.Payload) > 0 && string(req.Payload) != "null" {
		if updateErr := mgr.agentService.UpdateToolCallArgs(c.Request.Context(), toolCall.ID, mergedArgs); updateErr != nil {
			agentInternalError(c, fmt.Sprintf("failed to persist confirmation payload: %v", updateErr))
			return
		}
	}

	start := time.Now()
	executionBackend := strings.TrimSpace(toolCall.ExecutionBackend)
	if executionBackend == "" {
		executionBackend = normalizeExecutionBackend(toolCall.ToolName, "")
	}

	result, execErr := mgr.executeWriteTool(c, sessionToken, toolCall.ToolName, mergedArgs)
	latencyMs := int(time.Since(start).Milliseconds())

	status := agentToolStatusSuccess
	var resultBytes json.RawMessage
	var responseMsg string

	if execErr != nil {
		status = agentToolStatusError
		responseMsg = mgr.buildToolOutcomeMessage(toolCall.ToolName, status, nil, execErr.Error())
		if result != nil {
			resultBytes, _ = json.Marshal(result)
		}
		if len(resultBytes) == 0 {
			errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
			resultBytes = errJSON
		}
	} else {
		resultBytes, _ = json.Marshal(result)
		responseMsg = mgr.buildToolOutcomeMessage(toolCall.ToolName, status, result, "")
	}

	confirmed := true
	if updateErr := mgr.agentService.UpdateToolCallOutcome(
		c.Request.Context(),
		toolCall.ID,
		status,
		resultBytes,
		&confirmed,
	); updateErr != nil {
		agentInternalError(c, fmt.Sprintf("failed to update pending action: %v", updateErr))
		return
	}
	recordAgentMutationOperationLog(
		c,
		toolCall.ToolName,
		mergedArgs,
		result,
		execErr,
		executionBackend,
		req.ConfirmID,
	)

	resputil.Success(c, AgentToolResponse{
		Status:    status,
		Result:    resultBytes,
		Message:   responseMsg,
		LatencyMs: latencyMs,
	})
}

// ListSessions godoc
// @Summary List agent chat sessions for the current user
// @Tags agent
// @Produce json
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions [get]
func (mgr *AgentMgr) ListSessions(c *gin.Context) {
	token := util.GetToken(c)
	sessions, err := mgr.agentService.ListSessions(c.Request.Context(), token.UserID)
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to list sessions: %v", err))
		return
	}
	if rawSurface := strings.TrimSpace(c.Query("surface")); rawSurface != "" {
		surface := normalizeAgentSurface(rawSurface)
		filtered := make([]*model.AgentSession, 0, len(sessions))
		for _, session := range sessions {
			if agentSessionMatchesSurface(session, surface) {
				filtered = append(filtered, session)
			}
		}
		sessions = filtered
	}
	resputil.Success(c, sessions)
}

// UpdateSessionPin godoc
// @Summary Pin or unpin an agent session
// @Tags agent
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Param request body AgentSessionPinRequest true "Pin request"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions/{sessionId}/pin [put]
func (mgr *AgentMgr) UpdateSessionPin(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		agentBadRequest(c, "sessionId is required")
		return
	}

	var req AgentSessionPinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		agentBadRequest(c, err.Error())
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		agentForbidden(c, "session not found")
		return
	}
	if err := mgr.agentService.UpdateSessionPinned(c.Request.Context(), sessionID, req.Pinned); err != nil {
		if errors.Is(err, service.ErrAgentSessionPinningUnavailable) {
			agentInternalError(c, "session pinning requires a completed database migration")
			return
		}
		agentInternalError(c, fmt.Sprintf("failed to update session pin: %v", err))
		return
	}
	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID)
	if err != nil {
		agentNotFound(c, "session not found")
		return
	}
	resputil.Success(c, session)
}

// UpdateSessionTitle godoc
// @Summary Rename an agent session
// @Tags agent
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Param request body AgentSessionTitleRequest true "Rename request"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions/{sessionId}/title [put]
func (mgr *AgentMgr) UpdateSessionTitle(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		agentBadRequest(c, "sessionId is required")
		return
	}

	var req AgentSessionTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		agentBadRequest(c, err.Error())
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		agentForbidden(c, "session not found")
		return
	}
	if err := mgr.agentService.UpdateSessionTitle(c.Request.Context(), sessionID, req.Title); err != nil {
		agentBadRequest(c, fmt.Sprintf("failed to update session title: %v", err))
		return
	}
	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID)
	if err != nil {
		agentNotFound(c, "session not found")
		return
	}
	resputil.Success(c, session)
}

// DeleteSession godoc
// @Summary Soft delete an agent session
// @Tags agent
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[string]
// @Router /api/v1/agent/sessions/{sessionId} [delete]
func (mgr *AgentMgr) DeleteSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		agentBadRequest(c, "sessionId is required")
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		agentForbidden(c, "session not found")
		return
	}
	if err := mgr.agentService.DeleteSession(c.Request.Context(), sessionID); err != nil {
		agentInternalError(c, fmt.Sprintf("failed to delete session: %v", err))
		return
	}
	resputil.Success(c, "ok")
}

// GetAgentConfigSummary godoc
// @Summary Get agent configuration summary for the current user
// @Tags agent
// @Produce json
// @Success 200 {object} resputil.Response[AgentConfigSummary]
// @Router /api/v1/agent/config-summary [get]
func (mgr *AgentMgr) GetAgentConfigSummary(c *gin.Context) {
	summary := AgentConfigSummary{
		DefaultOrchestrationMode: agentRoleSingleAgent,
		AvailableModes:           []string{agentRoleSingleAgent},
	}
	if mgr.configService != nil {
		token := util.GetToken(c)
		if llmStatus, err := mgr.configService.GetEffectiveLLMConfigStatus(c.Request.Context(), token.UserID); err == nil && llmStatus != nil {
			llmSummary := &AgentLLMConfigSummary{
				Source:        llmStatus.Source,
				UsingConfig:   llmStatus.Complete,
				UsingPersonal: llmStatus.UsingPersonal,
				Complete:      llmStatus.Complete,
				HasAPIKey:     llmStatus.HasAPIKey,
			}
			if llmStatus.Config != nil {
				llmSummary.BaseURL = strings.TrimSpace(llmStatus.Config.BaseURL)
				llmSummary.Model = strings.TrimSpace(llmStatus.Config.ModelName)
			}
			if !llmStatus.Complete {
				llmSummary.FallbackNote = "LLM 配置不完整，Agent 不会使用 crater-agent 本地 llm-clients.json 兜底；请在个人设置或平台设置中配置 BaseURL、Model 和 API Key。"
			}
			summary.LLM = llmSummary
		}
	}
	agentReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodGet,
		mgr.getPythonAgentURL()+"/config-summary",
		http.NoBody,
	)
	if err == nil {
		resp, requestErr := mgr.httpClient.Do(agentReq)
		if requestErr == nil {
			defer resp.Body.Close()
			if resp.StatusCode < http.StatusBadRequest {
				var agentSummary AgentConfigSummary
				if decodeErr := json.NewDecoder(resp.Body).Decode(&agentSummary); decodeErr == nil {
					if len(agentSummary.AvailableModes) > 0 {
						summary.AvailableModes = agentSummary.AvailableModes
					}
				}
			}
		}
	}
	resputil.Success(c, summary)
}

// HandleParameterUpdate godoc
// @Summary Forward a parameter update to the Python Agent service
// @Description Allows the frontend to send parameter adjustments (e.g. form field changes) to the agent mid-session.
// @Tags agent
// @Accept json
// @Produce json
// @Param request body map[string]any true "Parameter update payload"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/chat/parameter-update [post]
func (mgr *AgentMgr) HandleParameterUpdate(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		agentBadRequest(c, err.Error())
		return
	}

	token := util.GetToken(c)
	sessionID, _ := payload["sessionId"].(string)
	if sessionID == "" {
		agentBadRequest(c, "sessionId is required")
		return
	}

	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		agentForbidden(c, "session not found")
		return
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to marshal payload: %v", err))
		return
	}

	agentURL := mgr.getPythonAgentURL() + "/chat/parameter-update"
	agentReq, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodPost,
		agentURL,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to create agent request: %v", err))
		return
	}
	agentReq.Header.Set("Content-Type", "application/json")
	if internalToken := mgr.getPythonAgentInternalToken(); internalToken != "" {
		agentReq.Header.Set("X-Agent-Internal-Token", internalToken)
	}

	resp, err := mgr.httpClient.Do(agentReq)
	if err != nil {
		agentInternalError(c, fmt.Sprintf("agent service unavailable: %v", err))
		return
	}
	defer resp.Body.Close()

	var result any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		agentInternalError(c, fmt.Sprintf("failed to decode agent response: %v", decodeErr))
		return
	}

	if resp.StatusCode >= http.StatusBadRequest {
		agentInternalError(c, fmt.Sprintf("agent service returned status %d", resp.StatusCode))
		return
	}

	resputil.Success(c, result)
}

// GetSessionMessages godoc
// @Summary Get messages for a specific agent session
// @Tags agent
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions/{sessionId}/messages [get]
func (mgr *AgentMgr) GetSessionMessages(c *gin.Context) {
	mgr.respondOwnedSessionData(c, "failed to list messages", func(ctx context.Context, sessionID string) (any, error) {
		return mgr.agentService.ListMessages(ctx, sessionID)
	})
}

// GetSessionToolCalls godoc
// @Summary Get tool calls for a specific agent session
// @Tags agent
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions/{sessionId}/tool-calls [get]
func (mgr *AgentMgr) GetSessionToolCalls(c *gin.Context) {
	mgr.respondOwnedSessionData(c, "failed to list tool calls", func(ctx context.Context, sessionID string) (any, error) {
		return mgr.agentService.ListToolCalls(ctx, sessionID)
	})
}

// GetSessionTurns godoc
// @Summary Get turns for a specific agent session
// @Tags agent
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions/{sessionId}/turns [get]
func (mgr *AgentMgr) GetSessionTurns(c *gin.Context) {
	mgr.respondOwnedSessionData(c, "failed to list turns", func(ctx context.Context, sessionID string) (any, error) {
		return mgr.agentService.ListTurns(ctx, sessionID)
	})
}

// GetTurnEvents godoc
// @Summary Get run events for a specific agent turn
// @Tags agent
// @Produce json
// @Param turnId path string true "Turn ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/turns/{turnId}/events [get]
func (mgr *AgentMgr) GetTurnEvents(c *gin.Context) {
	turnID := c.Param("turnId")
	if turnID == "" {
		agentBadRequest(c, "turnId is required")
		return
	}

	token := util.GetToken(c)
	turn, err := mgr.agentService.GetTurn(c.Request.Context(), turnID)
	if err != nil {
		agentNotFound(c, "turn not found")
		return
	}
	_, err = mgr.agentService.GetOwnedSession(c.Request.Context(), turn.SessionID, token.UserID)
	if err != nil {
		agentForbidden(c, "turn not found")
		return
	}
	events, err := mgr.agentService.ListRunEvents(c.Request.Context(), turnID)
	if err != nil {
		agentInternalError(c, fmt.Sprintf("failed to list turn events: %v", err))
		return
	}
	resputil.Success(c, events)
}
