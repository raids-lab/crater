package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
)

const agentPythonStreamTimeout = 15 * time.Minute

// getPythonAgentURL returns the configured Python Agent service base URL.
func (mgr *AgentMgr) getPythonAgentURL() string {
	cfg := pkgconfig.GetConfig()
	if cfg.Agent.ServiceURL != "" {
		return cfg.Agent.ServiceURL
	}
	return agentDefaultPythonServiceURL
}

func (mgr *AgentMgr) getPythonAgentInternalToken() string {
	cfg := pkgconfig.GetConfig()
	if cfg.Agent.InternalToken != "" {
		return cfg.Agent.InternalToken
	}
	return os.Getenv("CRATER_AGENT_INTERNAL_TOKEN")
}

func (mgr *AgentMgr) isInternalToolRequestAuthorized(c *gin.Context) bool {
	internalToken := mgr.getPythonAgentInternalToken()
	if internalToken == "" {
		return false
	}
	return c.GetHeader("X-Agent-Internal-Token") == internalToken
}

func (mgr *AgentMgr) getSessionToken(ctx context.Context, session *model.AgentSession) (util.JWTMessage, error) {
	if session == nil || session.UserID == 0 || session.AccountID == 0 {
		return util.JWTMessage{}, fmt.Errorf("invalid session actor")
	}
	u := query.User
	a := query.Account
	ua := query.UserAccount

	user, err := u.WithContext(ctx).Where(u.ID.Eq(session.UserID)).First()
	if err != nil {
		return util.JWTMessage{}, fmt.Errorf("failed to load user")
	}
	account, err := a.WithContext(ctx).Where(a.ID.Eq(session.AccountID)).First()
	if err != nil {
		return util.JWTMessage{}, fmt.Errorf("failed to load account")
	}
	userAccount, err := ua.WithContext(ctx).
		Where(ua.UserID.Eq(session.UserID), ua.AccountID.Eq(session.AccountID)).
		First()
	if err != nil {
		return util.JWTMessage{}, fmt.Errorf("failed to load account membership")
	}

	return util.JWTMessage{
		UserID:            session.UserID,
		AccountID:         session.AccountID,
		Username:          user.Name,
		AccountName:       account.Name,
		RoleAccount:       userAccount.Role,
		RolePlatform:      user.Role,
		AccountAccessMode: userAccount.AccessMode,
	}, nil
}

func (mgr *AgentMgr) buildPythonAgentPayload(
	sessionID string,
	turnID string,
	message string,
	token util.JWTMessage,
	pageContext map[string]any,
	clientContext map[string]any,
	orchestrationMode string,
	historyMessages []*model.AgentMessage,
	continuation map[string]any,
) AgentTurnRequest {
	return AgentTurnRequest{
		SessionID: sessionID,
		TurnID:    turnID,
		Message:   message,
		Context: map[string]any{
			"actor": map[string]any{
				"user_id":      token.UserID,
				"account_id":   token.AccountID,
				"username":     token.Username,
				"account_name": token.AccountName,
				"role":         strings.ToLower(token.RolePlatform.String()),
			},
			"page":          pageContext,
			"client":        clientContext,
			"history":       buildAgentHistory(historyMessages),
			"continuation":  continuation,
			"capabilities":  buildAgentCapabilities(token, pageContext),
			"orchestration": map[string]any{"mode": normalizeOrchestrationMode(orchestrationMode)},
		},
	}
}

func (mgr *AgentMgr) streamPythonAgentResponse(
	c *gin.Context,
	sessionID string,
	turnID string,
	orchestrationMode string,
	agentPayload AgentTurnRequest,
	persistAssistant bool,
) {
	payloadBytes, err := json.Marshal(agentPayload)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to marshal agent payload: %v", err), resputil.NotSpecified)
		return
	}

	agentURL := mgr.getPythonAgentURL() + "/chat"
	// Multi-agent diagnostic runs can legitimately span many LLM/tool hops.
	// Keep a bounded request context, but avoid the previous short timeout
	// that interrupted SSE before final_answer could be emitted.
	agentCtx, cancelAgentReq := context.WithTimeout(context.Background(), agentPythonStreamTimeout)
	defer cancelAgentReq()
	agentReq, err := http.NewRequestWithContext(agentCtx, http.MethodPost, agentURL, bytes.NewReader(payloadBytes))
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create agent request: %v", err), resputil.NotSpecified)
		return
	}
	agentReq.Header.Set("Content-Type", "application/json")
	agentReq.Header.Set("Accept", "text/event-stream")

	resp, err := mgr.httpClient.Do(agentReq)
	if err != nil {
		errorMetadata, _ := json.Marshal(map[string]any{
			"errorMessage": fmt.Sprintf("agent service unavailable: %v", err),
		})
		_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, "failed", nil, errorMetadata)
		resputil.Error(c, fmt.Sprintf("agent service unavailable: %v", err), resputil.ServiceError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		errorMetadata, _ := json.Marshal(map[string]any{
			"errorMessage": fmt.Sprintf("agent service returned status %d", resp.StatusCode),
		})
		_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, "failed", nil, errorMetadata)
		resputil.Error(c, fmt.Sprintf("agent service returned status %d", resp.StatusCode), resputil.ServiceError)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Agent-Session-ID", sessionID)
	c.Header("X-Agent-Turn-ID", turnID)
	c.Header("X-Agent-Orchestration-Mode", orchestrationMode)

	var assistantContent string
	turnStatus := "running"
	var turnMetadata json.RawMessage
	eventSequence := 0
	now := func() *time.Time {
		timestamp := time.Now()
		return &timestamp
	}
	persistRunEvent := func(eventType string, rawData []byte) {
		if strings.TrimSpace(eventType) == "" || len(rawData) == 0 {
			return
		}

		eventSequence++
		var payloadMap map[string]any
		rawContent := strings.TrimSpace(string(rawData))
		if err := json.Unmarshal(rawData, &payloadMap); err != nil {
			payloadMap = map[string]any{"content": rawContent}
		}
		agentID, _ := payloadMap["agentId"].(string)
		parentAgentID, _ := payloadMap["parentAgentId"].(string)
		agentRole, _ := payloadMap["agentRole"].(string)
		eventStatus, _ := payloadMap["status"].(string)
		title, _ := payloadMap["title"].(string)
		content, _ := payloadMap["content"].(string)
		if content == "" {
			if summary, _ := payloadMap["summary"].(string); summary != "" {
				content = summary
			} else if resultSummary, _ := payloadMap["resultSummary"].(string); resultSummary != "" {
				content = resultSummary
			}
		}
		if title == "" {
			if toolName, _ := payloadMap["toolName"].(string); toolName != "" {
				title = toolName
			}
		}
		if agentRole == "" {
			switch normalizeOrchestrationMode(orchestrationMode) {
			case "multi_agent":
				agentRole = "coordinator"
			default:
				agentRole = "single_agent"
			}
		}
		if eventStatus == "" {
			switch eventType {
			case "tool_call_started", "agent_run_started":
				eventStatus = "started"
			case "tool_call_confirmation_required":
				eventStatus = "awaiting_confirmation"
			case "error":
				eventStatus = "error"
			default:
				eventStatus = "completed"
			}
		}
		metadataBytes, _ := json.Marshal(payloadMap)
		_, _ = mgr.agentService.CreateRunEvent(context.Background(), &model.AgentRunEvent{
			TurnID:        turnID,
			SessionID:     sessionID,
			AgentID:       agentID,
			ParentAgentID: parentAgentID,
			AgentRole:     agentRole,
			EventType:     eventType,
			EventStatus:   eventStatus,
			Title:         title,
			Content:       content,
			Metadata:      datatypes.JSON(metadataBytes),
			Sequence:      eventSequence,
			StartedAt:     now(),
			EndedAt:       now(),
		})

		switch eventType {
		case "tool_call_confirmation_required":
			turnStatus = "awaiting_confirmation"
		case "error":
			turnStatus = "failed"
			errorMessage := ""
			if message, _ := payloadMap["message"].(string); strings.TrimSpace(message) != "" {
				errorMessage = message
			} else if message, _ := payloadMap["msg"].(string); strings.TrimSpace(message) != "" {
				errorMessage = message
			} else if strings.TrimSpace(content) != "" {
				errorMessage = content
			} else if title != "" {
				errorMessage = title
			}
			if strings.TrimSpace(errorMessage) == "" {
				errorMessage = "agent execution failed"
			}
			turnMetadata, _ = json.Marshal(map[string]any{
				"errorMessage": errorMessage,
			})
		case "final_answer":
			if finalContent, _ := payloadMap["content"].(string); strings.TrimSpace(finalContent) != "" {
				assistantContent = finalContent
			}
			turnStatus = "completed"
		}
	}
	handleEventBlock := func(eventType string, data []byte) {
		if eventType == "" || len(data) == 0 {
			return
		}
		persistRunEvent(eventType, data)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	currentEvent := ""
	var currentData bytes.Buffer
	clientConnected := true
	for scanner.Scan() {
		line := scanner.Text()
		if clientConnected {
			if _, writeErr := fmt.Fprintf(c.Writer, "%s\n", line); writeErr != nil {
				clientConnected = false
			} else {
				c.Writer.Flush()
			}
		}
		switch {
		case line == "":
			handleEventBlock(currentEvent, currentData.Bytes())
			currentEvent = ""
			currentData.Reset()
		case len(line) > 7 && line[:7] == "event: ":
			currentEvent = line[7:]
		case len(line) > 6 && line[:6] == "data: ":
			currentData.WriteString(line[6:])
		}
	}
	if currentEvent != "" && currentData.Len() > 0 {
		handleEventBlock(currentEvent, currentData.Bytes())
	}
	if err := scanner.Err(); err != nil {
		turnStatus = "failed"
		turnMetadata, _ = json.Marshal(map[string]any{
			"errorMessage": err.Error(),
		})
	}
	if turnStatus == "running" {
		turnStatus = "failed"
		turnMetadata, _ = json.Marshal(map[string]any{
			"errorMessage": "Agent 未返回最终答复，执行可能已中断。",
		})
	}

	var finalMessageID *uint
	if persistAssistant && strings.TrimSpace(assistantContent) != "" {
		assistantMsg := &model.AgentMessage{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   assistantContent,
			CreatedAt: time.Now(),
		}
		if err := mgr.agentService.SaveMessage(context.Background(), assistantMsg); err == nil {
			finalMessageID = &assistantMsg.ID
		}
	}
	switch turnStatus {
	case "failed":
		_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, "failed", finalMessageID, turnMetadata)
	case "awaiting_confirmation":
		_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, "awaiting_confirmation", finalMessageID, nil)
	default:
		_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, "completed", finalMessageID, nil)
	}
}

func (mgr *AgentMgr) buildConfirmationResumeMessage(toolCall *model.AgentToolCall) string {
	if toolCall == nil {
		return "系统事件：上一条确认操作已经结束。请基于当前上下文继续回答用户。"
	}

	resultSummary := ""
	if len(toolCall.ToolResult) > 0 {
		var payload any
		if err := json.Unmarshal(toolCall.ToolResult, &payload); err == nil {
			switch typed := payload.(type) {
			case map[string]any:
				if message, _ := typed["message"].(string); message != "" {
					resultSummary = message
				} else if errorMessage, _ := typed["error"].(string); errorMessage != "" {
					resultSummary = errorMessage
				} else {
					if marshaled, marshalErr := json.Marshal(typed); marshalErr == nil {
						resultSummary = string(marshaled)
					}
				}
			default:
				resultSummary = fmt.Sprintf("%v", typed)
			}
		}
	}
	if resultSummary == "" {
		resultSummary = "无额外结果详情"
	}
	resultSummary = strings.TrimSpace(resultSummary)
	if utf8.RuneCountInString(resultSummary) > 600 {
		runes := []rune(resultSummary)
		resultSummary = string(runes[:600])
	}

	return fmt.Sprintf(
		"系统事件：刚才的待确认操作已经结束。工具=%s，状态=%s，结果摘要=%s。请基于这个执行结果继续面向用户输出一段简短自然语言回复。成功时说明已完成什么、关键对象是什么、下一步可做什么；失败时解释原因并给出下一步建议；拒绝时说明操作已取消并提示可选后续。不要要求用户再次确认，不要输出原始 JSON，也不要把这条系统事件原样复述给用户。",
		toolCall.ToolName,
		toolCall.ResultStatus,
		resultSummary,
	)
}

func parseToolCallResult(toolCall *model.AgentToolCall) (any, string) {
	if toolCall == nil || len(toolCall.ToolResult) == 0 {
		return nil, ""
	}

	var payload any
	if err := json.Unmarshal(toolCall.ToolResult, &payload); err != nil {
		return nil, ""
	}

	resultMap, _ := payload.(map[string]any)
	if resultMap == nil {
		return payload, ""
	}
	if errorMessage, _ := resultMap["error"].(string); strings.TrimSpace(errorMessage) != "" {
		return payload, strings.TrimSpace(errorMessage)
	}
	if message, _ := resultMap["message"].(string); strings.TrimSpace(message) != "" {
		return payload, strings.TrimSpace(message)
	}
	return payload, ""
}

func (mgr *AgentMgr) buildConfirmationFinalAnswer(toolCall *model.AgentToolCall) string {
	if toolCall == nil {
		return "刚才的确认操作已经结束。"
	}

	result, fallback := parseToolCallResult(toolCall)
	answer := strings.TrimSpace(mgr.buildToolOutcomeMessage(toolCall.ToolName, toolCall.ResultStatus, result, fallback))
	if answer == "" {
		answer = "刚才的确认操作已经结束。"
	}
	if toolCall.ResultStatus == agentToolStatusError {
		switch toolCall.ToolName {
		case agentToolResubmitJob:
			answer += " 你可以调整资源配置后再试，或先查看原作业详情确认失败原因。"
		case agentToolCreateJupyter, agentToolCreateTrain:
			answer += " 你可以修改表单参数后再试一次。"
		}
	}
	return answer
}

func (mgr *AgentMgr) persistSyntheticRunEvent(
	turnID string,
	sessionID string,
	agentID string,
	agentRole string,
	eventType string,
	eventStatus string,
	content string,
	metadata map[string]any,
) {
	if turnID == "" || sessionID == "" || eventType == "" {
		return
	}
	metadataBytes, _ := json.Marshal(metadata)
	timestamp := time.Now()
	_, _ = mgr.agentService.CreateRunEvent(context.Background(), &model.AgentRunEvent{
		TurnID:      turnID,
		SessionID:   sessionID,
		AgentID:     agentID,
		AgentRole:   agentRole,
		EventType:   eventType,
		EventStatus: eventStatus,
		Content:     content,
		Metadata:    datatypes.JSON(metadataBytes),
		StartedAt:   &timestamp,
		EndedAt:     &timestamp,
		CreatedAt:   timestamp,
	})
}

func writeSyntheticSSEEvent(c *gin.Context, eventType string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventType, data); err != nil {
		return err
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func (mgr *AgentMgr) streamConfirmationOutcome(
	c *gin.Context,
	sessionID string,
	turnID string,
	toolCall *model.AgentToolCall,
) {
	finalAnswer := mgr.buildConfirmationFinalAnswer(toolCall)
	agentID := "coordinator-1"
	agentRole := "coordinator"

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Agent-Session-ID", sessionID)
	c.Header("X-Agent-Turn-ID", turnID)
	c.Header("X-Agent-Orchestration-Mode", "confirmation_result")
	c.Status(http.StatusOK)

	startPayload := map[string]any{
		"turnId":     turnID,
		"sessionId":  sessionID,
		"agentId":    agentID,
		"agentRole":  agentRole,
		"status":     "completed",
		"summary":    "确认操作已完成，正在整理结果",
		"source":     "confirmation_resume",
		"toolName":   toolCall.ToolName,
		"toolStatus": toolCall.ResultStatus,
	}
	finalPayload := map[string]any{
		"turnId":    turnID,
		"sessionId": sessionID,
		"agentId":   agentID,
		"agentRole": agentRole,
		"content":   finalAnswer,
		"source":    "confirmation_resume",
	}

	mgr.persistSyntheticRunEvent(
		turnID,
		sessionID,
		agentID,
		agentRole,
		"agent_run_started",
		"completed",
		"确认操作已完成，正在整理结果",
		startPayload,
	)
	mgr.persistSyntheticRunEvent(
		turnID,
		sessionID,
		agentID,
		agentRole,
		"final_answer",
		"completed",
		finalAnswer,
		finalPayload,
	)
	_ = mgr.agentService.UpdateTurnStatus(context.Background(), turnID, "completed", nil, nil)

	_ = writeSyntheticSSEEvent(c, "agent_run_started", startPayload)
	_ = writeSyntheticSSEEvent(c, "final_answer", finalPayload)
	_ = writeSyntheticSSEEvent(c, "done", map[string]any{})
}

// proxySSEStream is a utility that copies an SSE response from an upstream URL to the Gin client.
// It is kept for future use when direct proxying is preferred over buffered streaming.
//
//nolint:unused // reserved for future direct-proxy SSE support
func proxySSEStream(c *gin.Context, httpClient *http.Client, upstreamURL string, body io.Reader) error {
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, body)
	if err != nil {
		return fmt.Errorf("failed to create upstream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if _, writeErr := fmt.Fprintf(c.Writer, "%s\n", line); writeErr != nil {
			return writeErr
		}
		c.Writer.Flush()
	}
	return scanner.Err()
}
