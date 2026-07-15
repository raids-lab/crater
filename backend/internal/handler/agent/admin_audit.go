package agent

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
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
			resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid 'from' timestamp (expect RFC3339)"))
			return
		}
		opts.From = &t
	}
	if toStr := strings.TrimSpace(c.Query("to")); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid 'to' timestamp (expect RFC3339)"))
			return
		}
		opts.To = &t
	}
	result, err := mgr.agentService.ListAdminSessions(c.Request.Context(), opts)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list admin agent sessions"))
		return
	}
	resputil.Success(c, result)
}

// GetAdminSessionDetail godoc
// @Summary Get enriched detail for a single agent session (for admin audit detail page)
// @Tags agent-admin
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Success 200 {object} resputil.Response[service.AgentAuditSessionListItem]
// @Router /api/v1/admin/agent/sessions/{sessionId}/detail [get]
func (mgr *AgentMgr) GetAdminSessionDetail(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}
	detail, err := mgr.agentService.GetSessionDetail(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("session not found"))
			return
		}
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to load session detail"))
		return
	}
	resputil.Success(c, detail)
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
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("session not found"))
		return
	}
	messages, err := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list session messages"))
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
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("session not found"))
		return
	}
	toolCalls, err := mgr.agentService.ListToolCalls(c.Request.Context(), sessionID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list session tool calls"))
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
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("session not found"))
		return
	}
	turns, err := mgr.agentService.ListTurns(c.Request.Context(), sessionID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list session turns"))
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
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("turnId is required"))
		return
	}
	if _, err := mgr.agentService.GetTurn(c.Request.Context(), turnID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("turn not found"))
		return
	}
	events, err := mgr.agentService.ListRunEvents(c.Request.Context(), turnID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list turn events"))
		return
	}
	resputil.Success(c, events)
}

func (mgr *AgentMgr) authorizeInternalAuditRead(c *gin.Context) bool {
	if mgr.isInternalToolRequestAuthorized(c) {
		return true
	}
	resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("invalid internal agent token"))
	return false
}

func (mgr *AgentMgr) GetInternalSessionMessages(c *gin.Context) {
	if !mgr.authorizeInternalAuditRead(c) {
		return
	}
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("session not found"))
		return
	}
	messages, err := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list session messages"))
		return
	}
	resputil.Success(c, messages)
}

func (mgr *AgentMgr) GetInternalSessionToolCalls(c *gin.Context) {
	if !mgr.authorizeInternalAuditRead(c) {
		return
	}
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("session not found"))
		return
	}
	toolCalls, err := mgr.agentService.ListToolCalls(c.Request.Context(), sessionID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list session tool calls"))
		return
	}
	resputil.Success(c, toolCalls)
}

func (mgr *AgentMgr) GetInternalSessionTurns(c *gin.Context) {
	if !mgr.authorizeInternalAuditRead(c) {
		return
	}
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}
	if _, err := mgr.agentService.GetSession(c.Request.Context(), sessionID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("session not found"))
		return
	}
	turns, err := mgr.agentService.ListTurns(c.Request.Context(), sessionID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list session turns"))
		return
	}
	resputil.Success(c, turns)
}

func (mgr *AgentMgr) GetInternalTurnToolCalls(c *gin.Context) {
	if !mgr.authorizeInternalAuditRead(c) {
		return
	}
	turnID := c.Param("turnId")
	if turnID == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("turnId is required"))
		return
	}
	if _, err := mgr.agentService.GetTurn(c.Request.Context(), turnID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("turn not found"))
		return
	}
	toolCalls, err := mgr.agentService.ListToolCallsByTurn(c.Request.Context(), turnID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list turn tool calls"))
		return
	}
	resputil.Success(c, toolCalls)
}

func (mgr *AgentMgr) GetInternalTurnEvents(c *gin.Context) {
	if !mgr.authorizeInternalAuditRead(c) {
		return
	}
	turnID := c.Param("turnId")
	if turnID == "" {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("turnId is required"))
		return
	}
	if _, err := mgr.agentService.GetTurn(c.Request.Context(), turnID); err != nil {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("turn not found"))
		return
	}
	events, err := mgr.agentService.ListRunEvents(c.Request.Context(), turnID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list turn events"))
		return
	}
	resputil.Success(c, events)
}
