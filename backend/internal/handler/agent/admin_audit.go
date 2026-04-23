package agent

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

func parseNonNegativeQueryInt(value string, fallback int) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

// ListAdminSessions godoc
// @Summary List agent sessions for admin audit
// @Tags agent-admin
// @Produce json
// @Success 200 {object} resputil.Response[service.AgentAuditSessionListResult]
// @Router /api/v1/admin/agent/sessions [get]
func (mgr *AgentMgr) ListAdminSessions(c *gin.Context) {
	opts := service.AgentAuditSessionListOptions{
		Source:  c.Query("source"),
		Keyword: c.Query("keyword"),
		HasEval: c.Query("hasEval"),
		Limit:   parseNonNegativeQueryInt(c.Query("limit"), 40),
		Offset:  parseNonNegativeQueryInt(c.Query("offset"), 0),
	}
	if fromStr := strings.TrimSpace(c.Query("from")); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			resputil.BadRequestError(c, "invalid 'from' timestamp (expect RFC3339)")
			return
		}
		opts.From = &t
	}
	if toStr := strings.TrimSpace(c.Query("to")); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			resputil.BadRequestError(c, "invalid 'to' timestamp (expect RFC3339)")
			return
		}
		opts.To = &t
	}
	result, err := mgr.agentService.ListAdminSessions(c.Request.Context(), opts)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list admin agent sessions: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, result)
}

// GetAdminSessionMessages godoc
// @Summary Get messages for an agent session as admin
// @Tags agent-admin
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/admin/agent/sessions/{sessionId}/messages [get]
func (mgr *AgentMgr) GetAdminSessionMessages(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "session not found", resputil.NotSpecified)
		return
	}
	messages, err := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list session messages: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, messages)
}

// GetAdminSessionToolCalls godoc
// @Summary Get tool calls for an agent session as admin
// @Tags agent-admin
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/admin/agent/sessions/{sessionId}/tool-calls [get]
func (mgr *AgentMgr) GetAdminSessionToolCalls(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "session not found", resputil.NotSpecified)
		return
	}
	toolCalls, err := mgr.agentService.ListToolCalls(c.Request.Context(), sessionID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list session tool calls: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, toolCalls)
}

// GetAdminSessionTurns godoc
// @Summary Get turns for an agent session as admin
// @Tags agent-admin
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/admin/agent/sessions/{sessionId}/turns [get]
func (mgr *AgentMgr) GetAdminSessionTurns(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "session not found", resputil.NotSpecified)
		return
	}
	turns, err := mgr.agentService.ListTurns(c.Request.Context(), sessionID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list session turns: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, turns)
}

// GetAdminTurnEvents godoc
// @Summary Get run events for an agent turn as admin
// @Tags agent-admin
// @Produce json
// @Param turnId path string true "Turn ID (UUID)"
// @Success 200 {object} resputil.Response[any]
// @Router /api/v1/admin/agent/turns/{turnId}/events [get]
func (mgr *AgentMgr) GetAdminTurnEvents(c *gin.Context) {
	turnID := c.Param("turnId")
	if turnID == "" {
		resputil.BadRequestError(c, "turnId is required")
		return
	}
	if _, err := mgr.agentService.GetTurn(c.Request.Context(), turnID); err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "turn not found", resputil.NotSpecified)
		return
	}
	events, err := mgr.agentService.ListRunEvents(c.Request.Context(), turnID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list turn events: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, events)
}
