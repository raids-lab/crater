package agent

import (
	"bytes"
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
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

// Chat godoc
// @Summary Agent chat (SSE)
// @Description Create or continue an agent chat session; streams SSE events from the Python Agent service.
// @Tags agent
// @Accept json
// @Produce text/event-stream
// @Param request body AgentChatRequest true "Chat request"
// @Router /api/v1/agent/chat [post]
func (mgr *AgentMgr) Chat(c *gin.Context) {
	var req AgentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Message)) > agentChatMessageMaxRunes {
		resputil.BadRequestError(c, fmt.Sprintf("message exceeds %d characters", agentChatMessageMaxRunes))
		return
	}

	token := util.GetToken(c)
	orchestrationMode := normalizeOrchestrationMode(req.OrchestrationMode)

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, _, err := mgr.agentService.GetOrCreateSession(
		c.Request.Context(),
		sessionID,
		token.UserID,
		token.AccountID,
		req.Message,
		req.PageContext,
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create/load session: %v", err), resputil.NotSpecified)
		return
	}
	if session.UserID != token.UserID || session.AccountID != token.AccountID {
		resputil.HTTPError(c, http.StatusForbidden, "session not found", resputil.TokenInvalid)
		return
	}
	_ = mgr.agentService.UpdateSessionOrchestrationMode(c.Request.Context(), sessionID, orchestrationMode)

	historyMessages, historyErr := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if historyErr != nil {
		resputil.Error(c, fmt.Sprintf("failed to load session history: %v", historyErr), resputil.NotSpecified)
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

	pageContext := normalizePageContext(req.PageContext)
	continuationContext := mgr.buildAgentContinuation(c.Request.Context(), sessionID)
	turnID := uuid.New().String()
	_, err = mgr.agentService.CreateTurn(c.Request.Context(), &model.AgentTurn{
		TurnID:            turnID,
		SessionID:         sessionID,
		RequestID:         req.RequestID,
		OrchestrationMode: orchestrationMode,
		Status:            "running",
		StartedAt:         time.Now(),
		Metadata:          datatypes.JSON(req.ClientContext),
	})
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create agent turn: %v", err), resputil.NotSpecified)
		return
	}
	agentPayload := mgr.buildPythonAgentPayload(
		sessionID,
		turnID,
		req.Message,
		token,
		pageContext,
		normalizeClientContext(req.ClientContext),
		orchestrationMode,
		historyForPrompt,
		continuationContext,
	)
	mgr.streamPythonAgentResponse(c, sessionID, turnID, orchestrationMode, agentPayload, true)
}

// ResumeAfterConfirmation godoc
// @Summary Resume an agent turn after a confirmation result
// @Description Starts a hidden follow-up turn so the agent can explain the execution result.
// @Tags agent
// @Accept json
// @Produce text/event-stream
// @Param request body AgentResumeRequest true "Resume request"
// @Router /api/v1/agent/chat/resume [post]
func (mgr *AgentMgr) ResumeAfterConfirmation(c *gin.Context) {
	var req AgentResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	confirmID, err := strconv.ParseUint(req.ConfirmID, 10, 64)
	if err != nil {
		resputil.BadRequestError(c, "invalid confirmId")
		return
	}
	toolCall, err := mgr.agentService.GetToolCallByID(c.Request.Context(), uint(confirmID))
	if err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "confirmation result not found", resputil.NotSpecified)
		return
	}
	if toolCall.ResultStatus == agentToolStatusAwaitConfirm {
		resputil.BadRequestError(c, "confirmation has not completed yet")
		return
	}

	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), toolCall.SessionID, token.UserID)
	if err != nil || session.AccountID != token.AccountID {
		resputil.HTTPError(c, http.StatusForbidden, "confirmation result not found", resputil.TokenInvalid)
		return
	}

	sourceTurn, sourceTurnErr := mgr.agentService.GetTurn(c.Request.Context(), toolCall.TurnID)
	if sourceTurnErr == nil && normalizeOrchestrationMode(sourceTurn.OrchestrationMode) == "multi_agent" {
		sessionToken, tokenErr := mgr.getSessionToken(c.Request.Context(), session)
		if tokenErr != nil {
			resputil.Error(c, fmt.Sprintf("failed to resolve session actor: %v", tokenErr), resputil.NotSpecified)
			return
		}
		historyMessages, historyErr := mgr.agentService.ListMessages(c.Request.Context(), toolCall.SessionID)
		if historyErr != nil {
			resputil.Error(c, fmt.Sprintf("failed to load session history: %v", historyErr), resputil.NotSpecified)
			return
		}

		resumeTurnID := uuid.New().String()
		turnMetadata, _ := json.Marshal(map[string]any{
			"resumeFromConfirmId": req.ConfirmID,
			"sourceTurnId":        sourceTurn.TurnID,
		})
		if _, err := mgr.agentService.CreateTurn(c.Request.Context(), &model.AgentTurn{
			TurnID:            resumeTurnID,
			SessionID:         toolCall.SessionID,
			OrchestrationMode: "multi_agent",
			Status:            "running",
			StartedAt:         time.Now(),
			Metadata:          datatypes.JSON(turnMetadata),
		}); err != nil {
			resputil.Error(c, fmt.Sprintf("failed to create resume turn: %v", err), resputil.NotSpecified)
			return
		}
		_ = mgr.agentService.UpdateTurnStatus(c.Request.Context(), sourceTurn.TurnID, "completed", nil, nil)

		agentPayload := mgr.buildPythonAgentPayload(
			toolCall.SessionID,
			resumeTurnID,
			"继续完成上一轮计划",
			sessionToken,
			normalizePageContext(json.RawMessage(session.PageContext)),
			normalizeClientContext(json.RawMessage(sourceTurn.Metadata)),
			"multi_agent",
			historyMessages,
			mgr.buildResumeContinuation(c.Request.Context(), sourceTurn, toolCall),
		)
		mgr.streamPythonAgentResponse(c, toolCall.SessionID, resumeTurnID, "multi_agent", agentPayload, true)
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
func (mgr *AgentMgr) ConfirmToolExecution(c *gin.Context) {
	var req ConfirmToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	confirmID, err := strconv.ParseUint(req.ConfirmID, 10, 64)
	if err != nil {
		resputil.BadRequestError(c, "invalid confirmId")
		return
	}
	toolCall, err := mgr.agentService.GetToolCallByID(c.Request.Context(), uint(confirmID))
	if err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "pending action not found", resputil.NotSpecified)
		return
	}
	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), toolCall.SessionID, token.UserID)
	if err != nil || session.AccountID != token.AccountID {
		resputil.HTTPError(c, http.StatusForbidden, "pending action not found", resputil.TokenInvalid)
		return
	}
	if toolCall.ResultStatus != agentToolStatusAwaitConfirm {
		resputil.BadRequestError(c, "pending action is no longer awaiting confirmation")
		return
	}

	if !req.Confirmed {
		summary := mgr.buildToolOutcomeMessage(toolCall.ToolName, "rejected", nil, "Operation rejected by user.")
		confirmed := false
		if updateErr := mgr.agentService.UpdateToolCallOutcome(
			c.Request.Context(),
			toolCall.ID,
			"rejected",
			json.RawMessage(toolCall.ToolResult),
			&confirmed,
		); updateErr != nil {
			resputil.Error(c, fmt.Sprintf("failed to update pending action: %v", updateErr), resputil.NotSpecified)
			return
		}
		resultBytes, _ := json.Marshal(map[string]any{
			"confirmId": req.ConfirmID,
			"confirmed": false,
		})
		resputil.Success(c, AgentToolResponse{
			Status:  "rejected",
			Result:  resultBytes,
			Message: summary,
		})
		mgr.persistAssistantToolMessage(c.Request.Context(), toolCall.SessionID, toolCall.ToolName, "rejected", summary)
		return
	}

	sessionToken, tokenErr := mgr.getSessionToken(c.Request.Context(), session)
	if tokenErr != nil {
		resputil.Error(c, fmt.Sprintf("failed to resolve session actor: %v", tokenErr), resputil.NotSpecified)
		return
	}

	mergedArgs, mergeErr := mergeToolArgsWithPayload(json.RawMessage(toolCall.ToolArgs), req.Payload)
	if mergeErr != nil {
		resputil.BadRequestError(c, mergeErr.Error())
		return
	}
	if len(req.Payload) > 0 && string(req.Payload) != "null" {
		if updateErr := mgr.agentService.UpdateToolCallArgs(c.Request.Context(), toolCall.ID, mergedArgs); updateErr != nil {
			resputil.Error(c, fmt.Sprintf("failed to persist confirmation payload: %v", updateErr), resputil.NotSpecified)
			return
		}
	}

	start := time.Now()
	result, execErr := mgr.executeWriteTool(c, sessionToken, toolCall.ToolName, mergedArgs)
	latencyMs := int(time.Since(start).Milliseconds())

	status := agentToolStatusSuccess
	var resultBytes json.RawMessage
	var responseMsg string

	if execErr != nil {
		status = agentToolStatusError
		responseMsg = mgr.buildToolOutcomeMessage(toolCall.ToolName, status, nil, execErr.Error())
		errJSON, _ := json.Marshal(map[string]string{"error": execErr.Error()})
		resultBytes = errJSON
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
		resputil.Error(c, fmt.Sprintf("failed to update pending action: %v", updateErr), resputil.NotSpecified)
		return
	}

	resputil.Success(c, AgentToolResponse{
		Status:    status,
		Result:    resultBytes,
		Message:   responseMsg,
		LatencyMs: latencyMs,
	})
	mgr.persistAssistantToolMessage(c.Request.Context(), toolCall.SessionID, toolCall.ToolName, status, responseMsg)
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
		resputil.Error(c, fmt.Sprintf("failed to list sessions: %v", err), resputil.NotSpecified)
		return
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
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	var req AgentSessionPinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "session not found", resputil.TokenInvalid)
		return
	}
	if err := mgr.agentService.UpdateSessionPinned(c.Request.Context(), sessionID, req.Pinned); err != nil {
		if errors.Is(err, service.ErrAgentSessionPinningUnavailable) {
			resputil.Error(c, "session pinning requires a completed database migration", resputil.NotSpecified)
			return
		}
		resputil.Error(c, fmt.Sprintf("failed to update session pin: %v", err), resputil.NotSpecified)
		return
	}
	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID)
	if err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "session not found", resputil.NotSpecified)
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
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "session not found", resputil.TokenInvalid)
		return
	}
	if err := mgr.agentService.DeleteSession(c.Request.Context(), sessionID); err != nil {
		resputil.Error(c, fmt.Sprintf("failed to delete session: %v", err), resputil.NotSpecified)
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
		DefaultOrchestrationMode: "single_agent",
		AvailableModes:           []string{"single_agent", "multi_agent"},
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
					if normalizeOrchestrationMode(agentSummary.DefaultOrchestrationMode) == "multi_agent" {
						summary.DefaultOrchestrationMode = "multi_agent"
					}
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
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	sessionID, _ := payload["sessionId"].(string)
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "session not found", resputil.TokenInvalid)
		return
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to marshal payload: %v", err), resputil.NotSpecified)
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
		resputil.Error(c, fmt.Sprintf("failed to create agent request: %v", err), resputil.NotSpecified)
		return
	}
	agentReq.Header.Set("Content-Type", "application/json")
	if internalToken := mgr.getPythonAgentInternalToken(); internalToken != "" {
		agentReq.Header.Set("X-Agent-Internal-Token", internalToken)
	}

	resp, err := mgr.httpClient.Do(agentReq)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("agent service unavailable: %v", err), resputil.ServiceError)
		return
	}
	defer resp.Body.Close()

	var result any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		resputil.Error(c, fmt.Sprintf("failed to decode agent response: %v", decodeErr), resputil.NotSpecified)
		return
	}

	if resp.StatusCode >= http.StatusBadRequest {
		resputil.Error(c, fmt.Sprintf("agent service returned status %d", resp.StatusCode), resputil.ServiceError)
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
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "session not found", resputil.TokenInvalid)
		return
	}

	messages, err := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list messages: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, messages)
}

// GetSessionToolCalls godoc
// @Summary Get tool calls for a specific agent session
// @Tags agent
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions/{sessionId}/tool-calls [get]
func (mgr *AgentMgr) GetSessionToolCalls(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "session not found", resputil.TokenInvalid)
		return
	}

	toolCalls, err := mgr.agentService.ListToolCalls(c.Request.Context(), sessionID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list tool calls: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, toolCalls)
}

// GetSessionTurns godoc
// @Summary Get turns for a specific agent session
// @Tags agent
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/agent/sessions/{sessionId}/turns [get]
func (mgr *AgentMgr) GetSessionTurns(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	token := util.GetToken(c)
	if _, err := mgr.agentService.GetOwnedSession(c.Request.Context(), sessionID, token.UserID); err != nil {
		resputil.HTTPError(c, http.StatusForbidden, "session not found", resputil.TokenInvalid)
		return
	}

	turns, err := mgr.agentService.ListTurns(c.Request.Context(), sessionID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list turns: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, turns)
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
		resputil.BadRequestError(c, "turnId is required")
		return
	}

	token := util.GetToken(c)
	turn, err := mgr.agentService.GetTurn(c.Request.Context(), turnID)
	if err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "turn not found", resputil.NotSpecified)
		return
	}
	session, err := mgr.agentService.GetOwnedSession(c.Request.Context(), turn.SessionID, token.UserID)
	if err != nil || session.AccountID != token.AccountID {
		resputil.HTTPError(c, http.StatusForbidden, "turn not found", resputil.TokenInvalid)
		return
	}
	events, err := mgr.agentService.ListRunEvents(c.Request.Context(), turnID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list turn events: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, events)
}
