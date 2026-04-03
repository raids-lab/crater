package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/monitor"
	pkgutils "github.com/raids-lab/crater/pkg/utils"
)

//nolint:gochecknoinits // Handler managers are registered during package initialization.
func init() {
	Registers = append(Registers, NewAgentMgr)
}

const (
	agentDefaultPythonServiceURL = "http://localhost:8000"
	agentChatMessageMaxRunes     = 4000

	agentToolStatusSuccess              = "success"
	agentToolStatusError                = "error"
	agentToolStatusConfirmationRequired = "confirmation_required"
	agentToolStatusAwaitConfirm         = "await_confirm"

	// Read-only tool names
	agentToolGetJobDetail      = "get_job_detail"
	agentToolGetJobEvents      = "get_job_events"
	agentToolGetJobLogs        = "get_job_logs"
	agentToolDiagnoseJob       = "diagnose_job"
	agentToolGetDiagnosticCtx  = "get_diagnostic_context"
	agentToolSearchSimilarFail = "search_similar_failures"
	agentToolQueryJobMetrics   = "query_job_metrics"
	agentToolAnalyzeQueue      = "analyze_queue_status"
	agentToolRealtimeCapacity  = "get_realtime_capacity"
	agentToolListImages        = "list_available_images"
	agentToolListCudaBase      = "list_cuda_base_images"
	agentToolListGPUModels     = "list_available_gpu_models"
	agentToolRecommendImages   = "recommend_training_images"
	agentToolCheckQuota        = "check_quota"
	agentToolGetHealthOverview = "get_health_overview"
	agentToolListUserJobs      = "list_user_jobs"
	agentToolGetClusterHealth  = "get_cluster_health_overview"
	agentToolListClusterJobs   = "list_cluster_jobs"
	agentToolListClusterNodes  = "list_cluster_nodes"

	// Write tools that require user confirmation before execution
	agentToolResubmitJob   = "resubmit_job"
	agentToolStopJob       = "stop_job"
	agentToolDeleteJob     = "delete_job"
	agentToolCreateJupyter = "create_jupyter_job"
	agentToolCreateTrain   = "create_training_job"
)

// AgentMgr handles agent-related API endpoints.
type AgentMgr struct {
	name         string
	client       client.Client
	kubeClient   kubernetes.Interface
	nodeClient   *crclient.NodeClient
	promClient   monitor.PrometheusInterface
	agentService *service.AgentService
	jobSubmitter JobMutationSubmitter
	httpClient   *http.Client
}

// NewAgentMgr creates a new AgentMgr.
func NewAgentMgr(conf *RegisterConfig) Manager {
	return &AgentMgr{
		name:         "agent",
		client:       conf.Client,
		kubeClient:   conf.KubeClient,
		nodeClient:   &crclient.NodeClient{Client: conf.Client, KubeClient: conf.KubeClient, PrometheusClient: conf.PrometheusClient},
		promClient:   conf.PrometheusClient,
		agentService: service.NewAgentService(),
		jobSubmitter: NewJobMutationSubmitter(conf),
		httpClient:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (mgr *AgentMgr) GetName() string { return mgr.name }

func (mgr *AgentMgr) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/tools/execute", mgr.ExecuteTool)
}

func (mgr *AgentMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("/chat", mgr.Chat)
	g.POST("/chat/confirm", mgr.ConfirmToolExecution)
	g.POST("/chat/resume", mgr.ResumeAfterConfirmation)
	g.GET("/config-summary", mgr.GetAgentConfigSummary)
	g.GET("/sessions", mgr.ListSessions)
	g.PUT("/sessions/:sessionId/pin", mgr.UpdateSessionPin)
	g.DELETE("/sessions/:sessionId", mgr.DeleteSession)
	g.GET("/sessions/:sessionId/messages", mgr.GetSessionMessages)
	g.GET("/sessions/:sessionId/tool-calls", mgr.GetSessionToolCalls)
	g.GET("/sessions/:sessionId/turns", mgr.GetSessionTurns)
	g.GET("/turns/:turnId/events", mgr.GetTurnEvents)
}

func (mgr *AgentMgr) RegisterAdmin(_ *gin.RouterGroup) {}

// ─── Request / Response types ───────────────────────────────────────────────

// AgentChatRequest is the request body for POST /agent/chat.
type AgentChatRequest struct {
	Message           string          `json:"message" binding:"required"`
	SessionID         string          `json:"sessionId,omitempty"`
	RequestID         string          `json:"requestId,omitempty"`
	PageContext       json.RawMessage `json:"pageContext,omitempty"`
	ClientContext     json.RawMessage `json:"clientContext,omitempty"`
	OrchestrationMode string          `json:"orchestrationMode,omitempty"`
}

// ConfirmToolRequest is the request body for POST /agent/chat/confirm.
type ConfirmToolRequest struct {
	ConfirmID string          `json:"confirmId" binding:"required"`
	Confirmed bool            `json:"confirmed"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type AgentResumeRequest struct {
	ConfirmID string `json:"confirmId" binding:"required"`
}

type AgentSessionPinRequest struct {
	Pinned bool `json:"pinned"`
}

// ExecuteToolRequest is the request body for POST /agent/tools/execute.
// This endpoint is called by the Python Agent service.
type ExecuteToolRequest struct {
	ToolName   string          `json:"tool_name" binding:"required"`
	ToolArgs   json.RawMessage `json:"tool_args" binding:"required"`
	SessionID  string          `json:"session_id" binding:"required"`
	TurnID     string          `json:"turn_id,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	AgentID    string          `json:"agent_id,omitempty"`
	AgentRole  string          `json:"agent_role,omitempty"`
}

type AgentConfigSummary struct {
	DefaultOrchestrationMode string   `json:"defaultOrchestrationMode"`
	AvailableModes           []string `json:"availableModes,omitempty"`
}

type AgentTurnRequest struct {
	SessionID string         `json:"session_id"`
	TurnID    string         `json:"turn_id"`
	Message   string         `json:"message"`
	Context   map[string]any `json:"context"`
}

type AgentToolConfirmation struct {
	ConfirmID   string         `json:"confirm_id"`
	ToolName    string         `json:"tool_name"`
	Description string         `json:"description"`
	RiskLevel   string         `json:"risk_level"`
	Interaction string         `json:"interaction,omitempty"`
	Form        *AgentToolForm `json:"form,omitempty"`
}

type AgentToolForm struct {
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	SubmitLabel string           `json:"submitLabel,omitempty"`
	Fields      []AgentToolField `json:"fields,omitempty"`
}

type AgentToolField struct {
	Key          string                 `json:"key"`
	Label        string                 `json:"label"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Placeholder  string                 `json:"placeholder,omitempty"`
	DefaultValue any                    `json:"defaultValue,omitempty"`
	Options      []AgentToolFieldOption `json:"options,omitempty"`
}

type AgentToolFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// AgentToolResponse is the response body for POST /agent/tools/execute.
type AgentToolResponse struct {
	ToolCallID   string                 `json:"tool_call_id,omitempty"`
	Status       string                 `json:"status"`
	Result       json.RawMessage        `json:"result,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Confirmation *AgentToolConfirmation `json:"confirmation,omitempty"`
	LatencyMs    int                    `json:"latency_ms,omitempty"`
}

// ─── Handlers ────────────────────────────────────────────────────────────────

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

	// Determine session ID – create one if not provided.
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Ensure session exists in the database.
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

	// Persist the user message after history is loaded so the current turn is not duplicated
	// inside the Python request history.
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

	historyMessages, historyErr := mgr.agentService.ListMessages(c.Request.Context(), toolCall.SessionID)
	if historyErr != nil {
		resputil.Error(c, fmt.Sprintf("failed to load session history: %v", historyErr), resputil.NotSpecified)
		return
	}

	pageContext := normalizePageContext(json.RawMessage(session.PageContext))
	resumeMessage := mgr.buildConfirmationResumeMessage(toolCall)
	turnID := toolCall.TurnID
	if turnID == "" {
		turnID = uuid.New().String()
	}
	agentPayload := mgr.buildPythonAgentPayload(
		toolCall.SessionID,
		turnID,
		resumeMessage,
		token,
		pageContext,
		nil,
		session.LastOrchestrationMode,
		historyMessages,
	)
	mgr.streamPythonAgentResponse(
		c,
		toolCall.SessionID,
		turnID,
		session.LastOrchestrationMode,
		agentPayload,
		true,
	)
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
		agentToolListClusterNodes:
		return true
	default:
		return false
	}
}

func isAgentConfirmTool(toolName string) bool {
	switch toolName {
	case agentToolResubmitJob, agentToolStopJob, agentToolDeleteJob, agentToolCreateJupyter, agentToolCreateTrain:
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

	session, err := mgr.agentService.GetSession(c.Request.Context(), req.SessionID)
	if err != nil {
		resputil.HTTPError(c, http.StatusNotFound, "session not found", resputil.NotSpecified)
		return
	}
	sessionToken, tokenErr := mgr.getSessionToken(c.Request.Context(), session)
	if tokenErr != nil {
		resputil.Error(c, fmt.Sprintf("failed to resolve session actor: %v", tokenErr), resputil.NotSpecified)
		return
	}

	switch req.ToolName {
	case agentToolResubmitJob, agentToolStopJob, agentToolDeleteJob, agentToolCreateJupyter, agentToolCreateTrain:
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

	// Execute read-only tools.
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

// ─── Tool execution helpers ──────────────────────────────────────────────────

// executeReadTool dispatches a read-only tool by name.
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
	default:
		return nil, fmt.Errorf("tool '%s' is not yet implemented", req.ToolName)
	}
}

// executeWriteTool executes a confirmed write tool.
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
	default:
		return nil, fmt.Errorf("write tool '%s' is not supported", toolName)
	}
}

// ─── Individual tool implementations ─────────────────────────────────────────

type agentJobNameArgs struct {
	JobName string `json:"job_name"`
}

func (mgr *AgentMgr) findScopedJob(ctx context.Context, token util.JWTMessage, jobName string) (*model.Job, error) {
	j := query.Job
	q := j.WithContext(ctx).
		Preload(j.User).
		Preload(j.Account).
		Where(j.JobName.Eq(jobName))
	if token.RolePlatform != model.RoleAdmin {
		q = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	job, err := q.First()
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	return job, nil
}

func buildJobDetailResponse(job *model.Job) map[string]any {
	resp := map[string]any{
		"jobName":            job.JobName,
		"name":               job.Name,
		"status":             job.Status,
		"jobType":            job.JobType,
		"creationTimestamp":  job.CreationTimestamp,
		"runningTimestamp":   job.RunningTimestamp,
		"completedTimestamp": job.CompletedTimestamp,
		"resources":          job.Resources.Data(),
	}
	if job.Nodes.Data() != nil {
		resp["nodes"] = job.Nodes.Data()
	}
	if job.ScheduleData != nil {
		resp["scheduleData"] = job.ScheduleData.Data()
	}
	if job.ProfileData != nil {
		resp["profileData"] = job.ProfileData.Data()
	}
	if job.TerminatedStates != nil {
		resp["terminatedStates"] = job.TerminatedStates.Data()
	}
	return resp
}

func getPrimaryContainerImage(job *model.Job) string {
	if job == nil || job.Attributes.Data() == nil {
		return ""
	}
	for i := range job.Attributes.Data().Spec.Tasks {
		task := &job.Attributes.Data().Spec.Tasks[i]
		for j := range task.Template.Spec.Containers {
			if image := task.Template.Spec.Containers[j].Image; image != "" {
				return image
			}
		}
	}
	return ""
}

func getFirstTerminatedState(job *model.Job) *v1.ContainerStateTerminated {
	if job == nil || job.TerminatedStates == nil {
		return nil
	}
	states := job.TerminatedStates.Data()
	if len(states) == 0 {
		return nil
	}
	return &states[0]
}

func filterLogByKeyword(logContent, keyword string) (string, error) {
	if keyword == "" {
		return logContent, nil
	}
	re, err := regexp.Compile(keyword)
	if err != nil {
		return "", fmt.Errorf("invalid keyword regex: %w", err)
	}
	lines := strings.Split(logContent, "\n")
	matched := make([]string, 0, len(lines))
	for _, line := range lines {
		if re.MatchString(line) {
			matched = append(matched, line)
		}
	}
	return strings.Join(matched, "\n"), nil
}

func getMetricAlias(metric string) string {
	switch strings.TrimSpace(strings.ToLower(metric)) {
	case "gpu_mem_used":
		return "gpu_mem"
	case "cpu_mem_used":
		return "mem_usage"
	default:
		return strings.TrimSpace(strings.ToLower(metric))
	}
}

func normalizeMetricSelection(metrics []string) []string {
	if len(metrics) == 0 {
		return []string{"gpu_util", "gpu_mem", "cpu_usage", "mem_usage"}
	}
	seen := make(map[string]struct{}, len(metrics))
	normalized := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		alias := getMetricAlias(metric)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		normalized = append(normalized, alias)
	}
	if len(normalized) == 0 {
		return []string{"gpu_util", "gpu_mem", "cpu_usage", "mem_usage"}
	}
	return normalized
}

func parseToolTimeRange(input string) time.Duration {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "", "last_2h":
		return 2 * time.Hour
	case "last_1h":
		return time.Hour
	case "last_6h":
		return 6 * time.Hour
	case "last_12h":
		return 12 * time.Hour
	case "last_24h":
		return 24 * time.Hour
	default:
		return 2 * time.Hour
	}
}

func buildMetricValueMap(profile *monitor.ProfileData, selected []string) map[string]any {
	if profile == nil {
		return map[string]any{}
	}
	get := func(v *float32) any {
		if v == nil {
			return nil
		}
		return *v
	}
	result := make(map[string]any, len(selected))
	for _, metric := range selected {
		switch metric {
		case "gpu_util":
			result[metric] = map[string]any{
				"avg": get(profile.GPUUtilAvg),
				"max": get(profile.GPUUtilMax),
				"std": get(profile.GPUUtilStd),
			}
		case "gpu_mem":
			result[metric] = map[string]any{
				"avg":   get(profile.GPUMemAvg),
				"max":   get(profile.GPUMemMax),
				"std":   get(profile.GPUMemStd),
				"total": get(profile.GPUMemTotal),
			}
		case "cpu_usage":
			result[metric] = map[string]any{
				"avg":     get(profile.CPUUsageAvg),
				"max":     get(profile.CPUUsageMax),
				"std":     get(profile.CPUUsageStd),
				"request": get(profile.CPURequest),
				"limit":   get(profile.CPULimit),
			}
		case "mem_usage":
			result[metric] = map[string]any{
				"avg":     get(profile.CPUMemAvg),
				"max":     get(profile.CPUMemMax),
				"std":     get(profile.CPUMemStd),
				"request": get(profile.MemRequest),
				"limit":   get(profile.MemLimit),
			}
		}
	}
	return result
}

func getJobNamespace(job *model.Job) string {
	if job != nil && job.Attributes.Data() != nil && job.Attributes.Data().Namespace != "" {
		return job.Attributes.Data().Namespace
	}
	return pkgconfig.GetConfig().Namespaces.Job
}

func getPodNameFromJob(job *model.Job) string {
	if job == nil || job.Attributes.Data() == nil {
		return ""
	}
	for i := range job.Attributes.Data().Spec.Tasks {
		task := &job.Attributes.Data().Spec.Tasks[i]
		if task.Name != "" {
			return fmt.Sprintf("%s-%s-0", job.JobName, task.Name)
		}
	}
	return ""
}

func (mgr *AgentMgr) readJobLogPayload(ctx context.Context, job *model.Job, tailLines int64, keyword string) (map[string]string, error) {
	if tailLines <= 0 {
		tailLines = 100
	}
	namespace := getJobNamespace(job)
	labelSelector := fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, job.JobName)
	if job.Attributes.Data() != nil {
		if labelVal, ok := job.Attributes.Data().Labels[crclient.LabelKeyBaseURL]; ok && labelVal != "" {
			labelSelector = fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, labelVal)
		}
	}
	podList, podErr := mgr.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if podErr != nil || len(podList.Items) == 0 {
		return map[string]string{"log": "Pod not found or no live logs available."}, nil
	}

	pod := podList.Items[0]
	containerName := ""
	if len(pod.Spec.Containers) > 0 {
		containerName = pod.Spec.Containers[0].Name
	}

	logBytes, logErr := mgr.kubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}).DoRaw(ctx)
	if logErr != nil {
		return map[string]string{"log": fmt.Sprintf("Failed to retrieve logs: %v", logErr)}, nil
	}

	logContent, filterErr := filterLogByKeyword(string(logBytes), keyword)
	if filterErr != nil {
		return nil, filterErr
	}

	payload := map[string]string{
		"podName":   pod.Name,
		"container": containerName,
		"log":       logContent,
	}
	if keyword != "" {
		payload["keyword"] = keyword
	}
	return payload, nil
}

func getFailureCategory(job *model.Job) string {
	return categorizeFailure(job).typeName
}

func getFailureSimilarityScore(target, candidate *model.Job) int {
	score := 0
	if getFailureCategory(candidate) == getFailureCategory(target) {
		score += 5
	}
	targetTerminated := getFirstTerminatedState(target)
	candidateTerminated := getFirstTerminatedState(candidate)
	if targetTerminated != nil && candidateTerminated != nil {
		if targetTerminated.ExitCode != 0 && targetTerminated.ExitCode == candidateTerminated.ExitCode {
			score += 3
		}
		if targetTerminated.Reason != "" && strings.EqualFold(targetTerminated.Reason, candidateTerminated.Reason) {
			score += 2
		}
	}
	if target.JobType == candidate.JobType {
		score += 2
	}
	targetImage := getPrimaryContainerImage(target)
	candidateImage := getPrimaryContainerImage(candidate)
	if targetImage != "" && targetImage == candidateImage {
		score += 1
	}
	return score
}

func buildSimilarFailureEntry(job *model.Job, score int) map[string]any {
	entry := map[string]any{
		"jobName":            job.JobName,
		"name":               job.Name,
		"status":             job.Status,
		"jobType":            job.JobType,
		"category":           getFailureCategory(job),
		"similarityScore":    score,
		"completedTimestamp": job.CompletedTimestamp,
	}
	if ts := getFirstTerminatedState(job); ts != nil {
		entry["exitCode"] = ts.ExitCode
		entry["exitReason"] = ts.Reason
	}
	if image := getPrimaryContainerImage(job); image != "" {
		entry["image"] = image
	}
	return entry
}

// toolGetJobDetail returns job detail from the database.
func (mgr *AgentMgr) toolGetJobDetail(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	return buildJobDetailResponse(job), nil
}

// toolGetJobEvents returns events for a job stored in the database cache.
func (mgr *AgentMgr) toolGetJobEvents(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	if job.Events == nil {
		return []v1.Event{}, nil
	}
	return job.Events.Data(), nil
}

// toolGetJobLogs retrieves recent log lines for a job's first pod via the Kubernetes API.
func (mgr *AgentMgr) toolGetJobLogs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName   string `json:"job_name"`
		Tail      int64  `json:"tail"`
		TailLines int64  `json:"tail_lines"`
		Keyword   string `json:"keyword"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	if args.TailLines <= 0 {
		args.TailLines = args.Tail
	}
	if args.TailLines <= 0 {
		args.TailLines = 100
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	return mgr.readJobLogPayload(c.Request.Context(), job, args.TailLines, args.Keyword)
}

// toolDiagnoseJob runs the existing rule-based diagnosis on a job.
func (mgr *AgentMgr) toolDiagnoseJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}
	return performDiagnosis(job), nil
}

func (mgr *AgentMgr) toolGetDiagnosticContext(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName    string `json:"job_name"`
		IncludeLog *bool  `json:"include_log"`
		TailLines  int64  `json:"tail_lines"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	if args.TailLines <= 0 {
		args.TailLines = 200
	}
	includeLog := true
	if args.IncludeLog != nil {
		includeLog = *args.IncludeLog
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	resp := JobContextResp{}
	resp.Meta.Name = job.Name
	resp.Meta.JobName = job.JobName
	resp.Meta.Namespace = getJobNamespace(job)
	if job.User.ID != 0 {
		resp.Meta.User = job.User.Name
	}
	if job.Account.ID != 0 {
		resp.Meta.Queue = job.Account.Nickname
	}
	resp.Meta.JobType = job.JobType
	resp.Meta.Status = job.Status
	resp.Meta.CreationTimestamp = job.CreationTimestamp
	resp.Meta.RunningTimestamp = job.RunningTimestamp
	resp.Meta.CompletedTimestamp = job.CompletedTimestamp
	if job.Nodes.Data() != nil {
		resp.Meta.Nodes = job.Nodes.Data()
	}
	resp.Meta.Resources = job.Resources.Data()

	if job.ProfileData != nil {
		resp.DB.ProfileData = job.ProfileData.Data()
	}
	if job.ScheduleData != nil {
		resp.DB.ScheduleData = job.ScheduleData.Data()
	}
	if job.Events != nil {
		resp.DB.Events = job.Events.Data()
	}
	if job.TerminatedStates != nil {
		resp.DB.TerminatedStates = job.TerminatedStates.Data()
	}

	if includeLog {
		logPayload, logErr := mgr.readJobLogPayload(c.Request.Context(), job, args.TailLines, "")
		if logErr != nil {
			return nil, logErr
		}
		resp.Log.Container = logPayload["container"]
		resp.Log.Tail = logPayload["log"]
	}

	return resp, nil
}

func (mgr *AgentMgr) toolQueryJobMetrics(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName   string   `json:"job_name"`
		Metrics   []string `json:"metrics"`
		TimeRange string   `json:"time_range"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	selectedMetrics := normalizeMetricSelection(args.Metrics)
	timeRange := strings.TrimSpace(args.TimeRange)
	if timeRange == "" {
		timeRange = "last_2h"
	}

	var profile *monitor.ProfileData
	source := "persisted_profile"
	if mgr.promClient != nil {
		namespace := getJobNamespace(job)
		podName := getPodNameFromJob(job)
		if namespace != "" && podName != "" {
			profile = mgr.promClient.QueryProfileData(types.NamespacedName{
				Namespace: namespace,
				Name:      podName,
			}, time.Now().Add(-parseToolTimeRange(timeRange)))
			if profile != nil {
				source = "prometheus_live"
			}
		}
	}
	if profile == nil && job.ProfileData != nil {
		profile = job.ProfileData.Data()
		source = "persisted_profile"
	}

	return map[string]any{
		"jobName":   job.JobName,
		"timeRange": timeRange,
		"source":    source,
		"metrics":   buildMetricValueMap(profile, selectedMetrics),
	}, nil
}

func (mgr *AgentMgr) toolSearchSimilarFailures(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName string `json:"job_name"`
		Days    int    `json:"days"`
		Limit   int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	if args.Days <= 0 {
		args.Days = 30
	}
	if args.Limit <= 0 || args.Limit > 20 {
		args.Limit = 5
	}

	targetJob, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	j := query.Job
	q := j.WithContext(c).
		Preload(j.User).
		Preload(j.Account).
		Where(j.JobName.Neq(targetJob.JobName)).
		Where(j.Status.Eq(string(batch.Failed))).
		Where(j.CompletedTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days)))
	if token.RolePlatform != model.RoleAdmin {
		q = q.Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID))
	}
	candidates, err := q.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query similar failures: %w", err)
	}

	type scoredFailure struct {
		job   *model.Job
		score int
	}
	scored := make([]scoredFailure, 0, len(candidates))
	for _, candidate := range candidates {
		score := getFailureSimilarityScore(targetJob, candidate)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredFailure{job: candidate, score: score})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].job.CompletedTimestamp.After(scored[j].job.CompletedTimestamp)
		}
		return scored[i].score > scored[j].score
	})
	if len(scored) > args.Limit {
		scored = scored[:args.Limit]
	}

	items := make([]map[string]any, 0, len(scored))
	for _, entry := range scored {
		items = append(items, buildSimilarFailureEntry(entry.job, entry.score))
	}

	return map[string]any{
		"jobName":         targetJob.JobName,
		"targetCategory":  getFailureCategory(targetJob),
		"lookbackDays":    args.Days,
		"matches":         items,
		"targetJobType":   targetJob.JobType,
		"targetExitState": getFirstTerminatedState(targetJob),
	}, nil
}

func (mgr *AgentMgr) toolGetRealtimeCapacity(c *gin.Context, _ util.JWTMessage, _ json.RawMessage) (any, error) {
	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	statusCount := make(map[string]int)
	items := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		statusCount[string(node.Status)]++
		items = append(items, map[string]any{
			"name":        node.Name,
			"status":      node.Status,
			"role":        node.Role,
			"vendor":      node.Vendor,
			"workloads":   node.Workloads,
			"capacity":    node.Capacity,
			"allocatable": node.Allocatable,
			"used":        node.Used,
		})
	}
	return map[string]any{
		"scope":       "cluster",
		"statusCount": statusCount,
		"nodeCount":   len(nodes),
		"nodes":       items,
	}, nil
}

type agentAccessibleImage struct {
	Image       *model.Image
	ShareStatus model.ImageShareType
}

func (mgr *AgentMgr) toolListAvailableImages(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobType string `json:"job_type"`
		Keyword string `json:"keyword"`
		Limit   int    `json:"limit"`
	}
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid args: %w", err)
		}
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 20
	}

	images, err := mgr.listAccessibleImages(c, token)
	if err != nil {
		return nil, err
	}

	filtered := make([]agentAccessibleImage, 0, len(images))
	for _, item := range images {
		if !matchesImageJobType(item.Image.TaskType, args.JobType) {
			continue
		}
		if !matchesImageKeyword(item.Image, args.Keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Image.CreatedAt.After(filtered[j].Image.CreatedAt)
	})
	if len(filtered) > args.Limit {
		filtered = filtered[:args.Limit]
	}

	items := make([]map[string]any, 0, len(filtered))
	for _, item := range filtered {
		items = append(items, buildAgentImageSummary(item))
	}

	return map[string]any{
		"images":            items,
		"count":             len(items),
		"requestedJobType":  strings.TrimSpace(args.JobType),
		"requestedKeyword":  strings.TrimSpace(args.Keyword),
		"supportsRealImage": true,
	}, nil
}

func (mgr *AgentMgr) toolListCudaBaseImages(c *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid args: %w", err)
		}
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 20
	}

	cudaQuery := query.CudaBaseImage
	images, err := cudaQuery.WithContext(c).
		Order(cudaQuery.CreatedAt.Desc()).
		Limit(args.Limit).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list cuda base images: %w", err)
	}

	items := make([]map[string]any, 0, len(images))
	for _, item := range images {
		items = append(items, map[string]any{
			"id":         item.ID,
			"label":      item.Label,
			"imageLabel": item.ImageLabel,
			"value":      item.Value,
			"createdAt":  item.CreatedAt,
		})
	}

	return map[string]any{
		"images": items,
		"count":  len(items),
	}, nil
}

func (mgr *AgentMgr) toolListAvailableGPUModels(c *gin.Context, _ util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("invalid args: %w", err)
		}
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 20
	}

	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	type gpuSummary struct {
		ResourceName string
		GPUModel     string
		Vendor       string
		NodeCount    int
		Total        int64
		Used         int64
		Free         int64
	}

	summaryByName := make(map[string]*gpuSummary)
	for _, node := range nodes {
		for resourceName, quantity := range node.Allocatable {
			name := string(resourceName)
			if !isGPUResourceName(name) {
				continue
			}
			total := quantity.Value()
			used := int64(0)
			if usedQuantity, ok := node.Used[resourceName]; ok {
				used = usedQuantity.Value()
			}
			entry := summaryByName[name]
			if entry == nil {
				vendor := ""
				modelName := extractGPUModelFromResourceName(name)
				if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
					vendor = parts[0]
				}
				entry = &gpuSummary{
					ResourceName: name,
					GPUModel:     modelName,
					Vendor:       vendor,
				}
				summaryByName[name] = entry
			}
			entry.NodeCount++
			entry.Total += total
			entry.Used += used
			entry.Free += total - used
		}
	}

	items := make([]map[string]any, 0, len(summaryByName))
	for _, item := range summaryByName {
		items = append(items, map[string]any{
			"resourceName": item.ResourceName,
			"gpuModel":     item.GPUModel,
			"vendor":       item.Vendor,
			"nodeCount":    item.NodeCount,
			"total":        item.Total,
			"used":         item.Used,
			"free":         item.Free,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		leftFree, _ := items[i]["free"].(int64)
		rightFree, _ := items[j]["free"].(int64)
		if leftFree == rightFree {
			leftTotal, _ := items[i]["total"].(int64)
			rightTotal, _ := items[j]["total"].(int64)
			return leftTotal > rightTotal
		}
		return leftFree > rightFree
	})
	if len(items) > args.Limit {
		items = items[:args.Limit]
	}

	return map[string]any{
		"gpuModels": items,
		"count":     len(items),
	}, nil
}

func (mgr *AgentMgr) toolRecommendTrainingImages(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		TaskDescription string `json:"task_description"`
		Framework       string `json:"framework"`
		Limit           int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.TaskDescription) == "" {
		return nil, fmt.Errorf("task_description is required")
	}
	if args.Limit <= 0 || args.Limit > 20 {
		args.Limit = 5
	}

	images, err := mgr.listAccessibleImages(c, token)
	if err != nil {
		return nil, err
	}

	keywords := buildTrainingImageKeywords(args.TaskDescription, args.Framework)
	type scoredImage struct {
		Item    agentAccessibleImage
		Score   int
		Reasons []string
	}
	scored := make([]scoredImage, 0, len(images))
	for _, item := range images {
		score, reasons := scoreTrainingImage(item, keywords)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredImage{
			Item:    item,
			Score:   score,
			Reasons: reasons,
		})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].Item.Image.CreatedAt.After(scored[j].Item.Image.CreatedAt)
		}
		return scored[i].Score > scored[j].Score
	})
	if len(scored) > args.Limit {
		scored = scored[:args.Limit]
	}

	items := make([]map[string]any, 0, len(scored))
	for _, entry := range scored {
		summary := buildAgentImageSummary(entry.Item)
		summary["score"] = entry.Score
		summary["reasons"] = entry.Reasons
		summary["confidence"] = recommendationConfidence(entry.Score)
		items = append(items, summary)
	}

	return map[string]any{
		"taskDescription": args.TaskDescription,
		"framework":       strings.TrimSpace(args.Framework),
		"recommendations": items,
		"count":           len(items),
		"grounded":        "Recommendations are based only on currently visible Crater images",
	}, nil
}

func (mgr *AgentMgr) toolAnalyzeQueueStatus(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	job, err := mgr.findScopedJob(c, token, args.JobName)
	if err != nil {
		return nil, err
	}

	pendingReasons := make([]string, 0)
	if job.Events != nil {
		for _, event := range job.Events.Data() {
			if event.Reason == "FailedScheduling" && event.Message != "" {
				pendingReasons = append(pendingReasons, event.Message)
			}
		}
	}

	diagnosis := performDiagnosis(job)
	suggestions := make([]string, 0, 3)
	if diagnosis.Solution != "" {
		suggestions = append(suggestions, diagnosis.Solution)
	}
	if len(pendingReasons) == 0 && job.Status == batch.Pending {
		suggestions = append(suggestions, "暂无明确调度事件，建议先查看配额与节点实时容量。")
	}

	capacity, capacityErr := mgr.toolGetRealtimeCapacity(c, token, nil)
	quota, quotaErr := mgr.toolCheckQuota(c, token, nil)
	resp := map[string]any{
		"jobName":        job.JobName,
		"status":         job.Status,
		"category":       diagnosis.Category,
		"diagnosis":      diagnosis.Diagnosis,
		"pendingReasons": pendingReasons,
		"suggestions":    suggestions,
		"jobDetail":      buildJobDetailResponse(job),
	}
	if capacityErr == nil {
		resp["capacity"] = capacity
	}
	if quotaErr == nil {
		resp["quota"] = quota
	}
	return resp, nil
}

// toolCheckQuota returns the current resource quota for the user.
func (mgr *AgentMgr) toolCheckQuota(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	accountID := token.AccountID
	var args struct {
		AccountID *uint `json:"account_id"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.AccountID != nil {
		if token.RolePlatform != model.RoleAdmin && *args.AccountID != token.AccountID {
			return nil, fmt.Errorf("account_id is not accessible")
		}
		accountID = *args.AccountID
	}

	a := query.Account
	ua := query.UserAccount
	userAccount, err := ua.WithContext(c).
		Where(ua.AccountID.Eq(accountID), ua.UserID.Eq(token.UserID)).
		First()
	if err == nil {
		quota := userAccount.Quota.Data()
		return map[string]any{
			"accountId":  accountID,
			"source":     "user_account",
			"capability": quota.Capability,
		}, nil
	}

	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("user account not found: %w", err)
	}

	account, accountErr := a.WithContext(c).Where(a.ID.Eq(accountID)).First()
	if accountErr != nil {
		return nil, fmt.Errorf("account not found: %w", accountErr)
	}

	return map[string]any{
		"accountId":  accountID,
		"source":     "account",
		"capability": account.Quota.Data().Capability,
	}, nil
}

// toolGetHealthOverview returns a simplified health summary for the current user.
func (mgr *AgentMgr) toolGetHealthOverview(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Days int `json:"days"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}

	j := query.Job
	lookback := time.Now().AddDate(0, 0, -args.Days)
	jobs, err := j.WithContext(c).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		Where(j.CreationTimestamp.Gte(lookback)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}

	statusCount := make(map[string]int)
	for _, job := range jobs {
		statusCount[string(job.Status)]++
	}

	return map[string]any{
		"totalJobs":    len(jobs),
		"statusCount":  statusCount,
		"lookbackDays": args.Days,
	}, nil
}

func (mgr *AgentMgr) toolGetClusterHealthOverview(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("cluster overview requires admin privileges")
	}
	var args struct {
		Days int `json:"days"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}

	j := query.Job
	q := j.WithContext(c)
	if args.Days > 0 {
		q = q.Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days)))
	}
	jobs, err := q.Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query cluster jobs: %w", err)
	}

	statusCount := make(map[string]int)
	accountCount := make(map[uint]int)
	userCount := make(map[uint]int)
	for _, job := range jobs {
		statusCount[string(job.Status)]++
		accountCount[job.AccountID]++
		userCount[job.UserID]++
	}

	return map[string]any{
		"scope":        "cluster",
		"totalJobs":    len(jobs),
		"statusCount":  statusCount,
		"lookbackDays": args.Days,
		"accountCount": len(accountCount),
		"userCount":    len(userCount),
	}, nil
}

func normalizeJobStatuses(statuses []string) []string {
	normalized := make([]string, 0, len(statuses)*2)
	for _, status := range statuses {
		if trimmed := strings.TrimSpace(status); trimmed != "" {
			normalized = append(normalized, trimmed)
			normalized = append(normalized, strings.ToUpper(trimmed[:1])+strings.ToLower(trimmed[1:]))
		}
	}
	return normalized
}

func (mgr *AgentMgr) toolListUserJobs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Statuses []string `json:"statuses"`
		Days     int      `json:"days"`
		Limit    int      `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 30
	}
	if args.Limit <= 0 || args.Limit > 50 {
		args.Limit = 20
	}

	j := query.Job
	q := j.WithContext(c).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days))).
		Order(j.CreationTimestamp.Desc())

	if len(args.Statuses) > 0 {
		statuses := normalizeJobStatuses(args.Statuses)
		if len(statuses) > 0 {
			q = q.Where(j.Status.In(statuses...))
		}
	}

	jobs, err := q.Limit(args.Limit).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	items := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, map[string]any{
			"name":               job.Name,
			"jobName":            job.JobName,
			"jobType":            job.JobType,
			"status":             job.Status,
			"creationTimestamp":  job.CreationTimestamp,
			"runningTimestamp":   job.RunningTimestamp,
			"completedTimestamp": job.CompletedTimestamp,
		})
	}

	return map[string]any{
		"jobs":      items,
		"count":     len(items),
		"days":      args.Days,
		"requested": args.Statuses,
	}, nil
}

func (mgr *AgentMgr) toolListClusterJobs(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("cluster job listing requires admin privileges")
	}
	var args struct {
		Statuses []string `json:"statuses"`
		Days     int      `json:"days"`
		Limit    int      `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Days <= 0 {
		args.Days = 7
	}
	if args.Limit <= 0 || args.Limit > 100 {
		args.Limit = 30
	}

	j := query.Job
	q := j.WithContext(c).
		Where(j.CreationTimestamp.Gte(time.Now().AddDate(0, 0, -args.Days))).
		Order(j.CreationTimestamp.Desc())

	if len(args.Statuses) > 0 {
		statuses := normalizeJobStatuses(args.Statuses)
		if len(statuses) > 0 {
			q = q.Where(j.Status.In(statuses...))
		}
	}

	jobs, err := q.Limit(args.Limit).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster jobs: %w", err)
	}

	items := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, map[string]any{
			"name":               job.Name,
			"jobName":            job.JobName,
			"jobType":            job.JobType,
			"status":             job.Status,
			"userID":             job.UserID,
			"accountID":          job.AccountID,
			"creationTimestamp":  job.CreationTimestamp,
			"runningTimestamp":   job.RunningTimestamp,
			"completedTimestamp": job.CompletedTimestamp,
		})
	}

	return map[string]any{
		"scope":     "cluster",
		"jobs":      items,
		"count":     len(items),
		"days":      args.Days,
		"requested": args.Statuses,
	}, nil
}

func (mgr *AgentMgr) toolListClusterNodes(c *gin.Context, token util.JWTMessage) (any, error) {
	if token.RolePlatform != model.RoleAdmin {
		return nil, fmt.Errorf("cluster node listing requires admin privileges")
	}
	nodes, err := mgr.nodeClient.ListNodes(c.Request.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster nodes: %w", err)
	}

	statusCount := make(map[string]int)
	items := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		statusCount[string(node.Status)]++
		items = append(items, map[string]any{
			"name":      node.Name,
			"role":      node.Role,
			"status":    node.Status,
			"workloads": node.Workloads,
			"vendor":    node.Vendor,
			"address":   node.Address,
		})
	}

	return map[string]any{
		"scope":       "cluster",
		"count":       len(items),
		"statusCount": statusCount,
		"nodes":       items,
	}, nil
}

func (mgr *AgentMgr) toolDeleteJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobRecord, clusterJob, err := mgr.getOwnedJobForMutation(c, token, rawArgs)
	if err != nil {
		return nil, err
	}
	return mgr.deleteOwnedJob(c, jobRecord, clusterJob, true)
}

func (mgr *AgentMgr) toolStopJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	jobRecord, clusterJob, err := mgr.getOwnedJobForMutation(c, token, rawArgs)
	if err != nil {
		return nil, err
	}
	return mgr.stopOwnedJob(c, jobRecord, clusterJob)
}

func (mgr *AgentMgr) toolResubmitJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		JobName  string  `json:"job_name"`
		Name     *string `json:"name"`
		CPU      *string `json:"cpu"`
		Memory   *string `json:"memory"`
		GPUCount *int    `json:"gpu_count"`
		GPUModel *string `json:"gpu_model"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	jobRecord, _, err := mgr.getOwnedJobForMutation(c, token, rawArgs)
	if err != nil {
		return nil, err
	}
	sourceJob := jobRecord.Attributes.Data()
	if sourceJob == nil {
		return nil, fmt.Errorf("job spec is unavailable for resubmit")
	}
	if token.Username == "" {
		return nil, fmt.Errorf("user identity is unavailable for resubmit")
	}

	clonedJob := sourceJob.DeepCopy()
	appliedOverrides, err := applyResubmitOverrides(clonedJob, args.CPU, args.Memory, args.GPUCount, args.GPUModel)
	if err != nil {
		return nil, err
	}
	prefix := getJobNamePrefix(jobRecord.JobName)
	newJobName := pkgutils.GenerateJobName(prefix, token.Username)
	baseURL := getBaseURLFromJobName(newJobName)

	clonedJob.ObjectMeta = metav1.ObjectMeta{
		Name:        newJobName,
		Namespace:   pkgconfig.GetConfig().Namespaces.Job,
		Labels:      copyStringMap(clonedJob.Labels),
		Annotations: copyStringMap(clonedJob.Annotations),
	}
	clonedJob.Status = batch.JobStatus{}
	clonedJob.ResourceVersion = ""
	clonedJob.UID = ""
	clonedJob.CreationTimestamp = metav1.Time{}
	clonedJob.ManagedFields = nil
	clonedJob.OwnerReferences = nil
	clonedJob.Finalizers = nil
	clonedJob.DeletionTimestamp = nil

	if clonedJob.Labels == nil {
		clonedJob.Labels = map[string]string{}
	}
	clonedJob.Labels[crclient.LabelKeyBaseURL] = baseURL
	if clonedJob.Annotations == nil {
		clonedJob.Annotations = map[string]string{}
	}
	if args.Name != nil && strings.TrimSpace(*args.Name) != "" {
		clonedJob.Annotations["crater.raids.io/task-name"] = strings.TrimSpace(*args.Name)
		appliedOverrides["name"] = strings.TrimSpace(*args.Name)
	} else if clonedJob.Annotations["crater.raids.io/task-name"] == "" {
		clonedJob.Annotations["crater.raids.io/task-name"] = jobRecord.Name
	}

	for idx := range clonedJob.Spec.Tasks {
		task := &clonedJob.Spec.Tasks[idx]
		task.Template.ResourceVersion = ""
		task.Template.UID = ""
		task.Template.CreationTimestamp = metav1.Time{}
		task.Template.ManagedFields = nil
		if task.Template.Labels == nil {
			task.Template.Labels = map[string]string{}
		}
		task.Template.Labels[crclient.LabelKeyBaseURL] = baseURL
		task.Template.Labels[crclient.LabelKeyTaskType] = clonedJob.Labels[crclient.LabelKeyTaskType]
		task.Template.Labels[crclient.LabelKeyTaskUser] = clonedJob.Labels[crclient.LabelKeyTaskUser]
		if accountName := clonedJob.Labels[crclient.LalbeKeyTaskAccount]; accountName != "" {
			task.Template.Labels[crclient.LalbeKeyTaskAccount] = accountName
		}
	}

	if err := mgr.client.Create(c, clonedJob); err != nil {
		return nil, fmt.Errorf("failed to create resubmitted job: %w", err)
	}

	if err := mgr.ensureAgentResubmitAccess(c, clonedJob); err != nil {
		return map[string]any{
			"sourceJobName": jobRecord.JobName,
			"jobName":       newJobName,
			"status":        "created",
			"warning":       err.Error(),
		}, nil
	}

	return map[string]any{
		"sourceJobName": jobRecord.JobName,
		"jobName":       newJobName,
		"displayName":   clonedJob.Annotations["crater.raids.io/task-name"],
		"status":        "created",
		"overrides":     appliedOverrides,
	}, nil
}

func (mgr *AgentMgr) toolCreateJupyterJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Name      string  `json:"name"`
		ImageLink string  `json:"image_link"`
		CPU       string  `json:"cpu"`
		Memory    string  `json:"memory"`
		GPUCount  *int    `json:"gpu_count"`
		GPUModel  *string `json:"gpu_model"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if args.ImageLink == "" {
		return nil, fmt.Errorf("image_link is required")
	}
	if args.CPU == "" {
		args.CPU = "2"
	}
	if args.Memory == "" {
		args.Memory = "8Gi"
	}

	if mgr.jobSubmitter == nil {
		return nil, fmt.Errorf("job submitter is not configured")
	}

	resourceMap := map[string]string{
		"cpu":    args.CPU,
		"memory": args.Memory,
	}
	if args.GPUCount != nil && *args.GPUCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", "gpu")
		if args.GPUModel != nil && strings.TrimSpace(*args.GPUModel) != "" {
			gpuResourceName = normalizeGPUResourceName(gpuResourceName, *args.GPUModel)
		}
		resourceMap[string(gpuResourceName)] = strconv.Itoa(*args.GPUCount)
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":     args.Name,
		"resource": resourceMap,
		"image": map[string]any{
			"imageLink": args.ImageLink,
			"archs":     []string{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jupyter request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitJupyterJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

func (mgr *AgentMgr) toolCreateTrainingJob(c *gin.Context, token util.JWTMessage, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Name       string  `json:"name"`
		ImageLink  string  `json:"image_link"`
		Command    string  `json:"command"`
		WorkingDir string  `json:"working_dir"`
		CPU        string  `json:"cpu"`
		Memory     string  `json:"memory"`
		GPUCount   *int    `json:"gpu_count"`
		GPUModel   *string `json:"gpu_model"`
		Shell      string  `json:"shell"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(args.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(args.ImageLink) == "" {
		return nil, fmt.Errorf("image_link is required")
	}
	if strings.TrimSpace(args.Command) == "" {
		return nil, fmt.Errorf("command is required")
	}
	if strings.TrimSpace(args.WorkingDir) == "" {
		return nil, fmt.Errorf("working_dir is required")
	}
	if strings.TrimSpace(args.CPU) == "" {
		args.CPU = "4"
	}
	if strings.TrimSpace(args.Memory) == "" {
		args.Memory = "16Gi"
	}
	if strings.TrimSpace(args.Shell) == "" {
		args.Shell = "bash"
	}

	resourceMap := map[string]string{
		"cpu":    args.CPU,
		"memory": args.Memory,
	}
	if args.GPUCount != nil && *args.GPUCount > 0 {
		gpuResourceName := normalizeGPUResourceName("", "gpu")
		if args.GPUModel != nil && strings.TrimSpace(*args.GPUModel) != "" {
			gpuResourceName = normalizeGPUResourceName("", *args.GPUModel)
		}
		resourceMap[string(gpuResourceName)] = strconv.Itoa(*args.GPUCount)
	}

	if mgr.jobSubmitter == nil {
		return nil, fmt.Errorf("job submitter is not configured")
	}

	requestBody, err := json.Marshal(map[string]any{
		"name":       args.Name,
		"resource":   resourceMap,
		"workingDir": args.WorkingDir,
		"command":    args.Command,
		"shell":      args.Shell,
		"image": map[string]any{
			"imageLink": args.ImageLink,
			"archs":     []string{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal training request: %w", err)
	}

	result, err := mgr.jobSubmitter.SubmitTrainingJob(c, token, requestBody)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"status": "created",
		"job":    result,
	}, nil
}

func (mgr *AgentMgr) getOwnedJobForMutation(
	c *gin.Context,
	token util.JWTMessage,
	rawArgs json.RawMessage,
) (*model.Job, *batch.Job, error) {
	var args agentJobNameArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, nil, fmt.Errorf("job_name is required")
	}

	j := query.Job
	jobRecord, err := j.WithContext(c).
		Where(j.JobName.Eq(args.JobName)).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		First()
	if err != nil {
		return nil, nil, fmt.Errorf("job not found")
	}

	clusterJob := &batch.Job{}
	if err := mgr.client.Get(
		c,
		client.ObjectKey{
			Name:      args.JobName,
			Namespace: pkgconfig.GetConfig().Namespaces.Job,
		},
		clusterJob,
	); err != nil {
		if k8serrors.IsNotFound(err) {
			return jobRecord, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to load cluster job: %w", err)
	}

	return jobRecord, clusterJob, nil
}

func (mgr *AgentMgr) deleteOwnedJob(
	c *gin.Context,
	jobRecord *model.Job,
	clusterJob *batch.Job,
	deleteTerminalRecord bool,
) (any, error) {
	j := query.Job
	shouldDeleteRecord := clusterJob == nil
	shouldDeleteJob := clusterJob != nil

	if clusterJob != nil {
		phase := clusterJob.Status.State.Phase
		if deleteTerminalRecord && (phase == batch.Failed ||
			phase == batch.Completed ||
			phase == batch.Aborted ||
			phase == batch.Terminated) {
			shouldDeleteRecord = true
		}
	}

	if shouldDeleteRecord {
		if _, err := j.WithContext(c).Where(j.JobName.Eq(jobRecord.JobName)).Delete(); err != nil {
			return nil, fmt.Errorf("failed to delete job record: %w", err)
		}
	} else {
		if _, err := j.WithContext(c).Where(j.JobName.Eq(jobRecord.JobName)).Updates(model.Job{
			Status:             model.Deleted,
			CompletedTimestamp: time.Now(),
		}); err != nil {
			return nil, fmt.Errorf("failed to update job status: %w", err)
		}
	}

	if shouldDeleteJob {
		if err := mgr.client.Delete(c, clusterJob); err != nil && !k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to delete cluster job: %w", err)
		}
	}

	return map[string]any{
		"jobName":       jobRecord.JobName,
		"status":        "deleted",
		"recordDeleted": shouldDeleteRecord,
	}, nil
}

func (mgr *AgentMgr) stopOwnedJob(c *gin.Context, jobRecord *model.Job, clusterJob *batch.Job) (any, error) {
	if clusterJob == nil {
		return map[string]any{
			"jobName": jobRecord.JobName,
			"status":  "already_stopped",
		}, nil
	}

	phase := clusterJob.Status.State.Phase
	if phase == batch.Completed || phase == batch.Failed || phase == batch.Aborted || phase == batch.Terminated {
		return map[string]any{
			"jobName": jobRecord.JobName,
			"status":  "already_finished",
		}, nil
	}

	if err := mgr.client.Delete(c, clusterJob); err != nil && !k8serrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to delete cluster job: %w", err)
	}
	j := query.Job
	if _, err := j.WithContext(c).Where(j.JobName.Eq(jobRecord.JobName)).Updates(model.Job{
		Status:             batch.Aborted,
		CompletedTimestamp: time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}
	return map[string]any{
		"jobName": jobRecord.JobName,
		"status":  "stopped",
	}, nil
}

func (mgr *AgentMgr) ensureAgentResubmitAccess(c *gin.Context, job *batch.Job) error {
	serviceManager := crclient.NewServiceManager(mgr.client, mgr.kubeClient)
	labels := copyStringMap(job.Labels)
	if len(job.Spec.Tasks) == 0 {
		return nil
	}

	taskType := labels[crclient.LabelKeyTaskType]
	baseURL := labels[crclient.LabelKeyBaseURL]
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(job, batch.SchemeGroupVersion.WithKind("Job")),
	}
	switch taskType {
	case string(model.JobTypeJupyter):
		_, err := serviceManager.CreateIngressWithPrefix(
			c,
			ownerRefs,
			labels,
			&v1.ServicePort{
				Name:       "notebook",
				Port:       8888,
				TargetPort: intstrFromInt(8888),
				Protocol:   v1.ProtocolTCP,
			},
			pkgconfig.GetConfig().Host,
			baseURL,
		)
		return err
	case string(model.JobTypeWebIDE):
		username := labels[crclient.LabelKeyTaskUser]
		randomPrefix := uuid.New().String()[:5]
		_, err := serviceManager.CreateNamedIngress(
			c,
			ownerRefs,
			labels,
			&v1.ServicePort{
				Name:       "webide",
				Port:       8888,
				TargetPort: intstrFromInt(8888),
				Protocol:   v1.ProtocolTCP,
			},
			pkgconfig.GetConfig().Host,
			username,
			randomPrefix,
		)
		return err
	default:
		return nil
	}
}

func getJobNamePrefix(jobName string) string {
	parts := strings.SplitN(jobName, "-", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "job"
}

func getBaseURLFromJobName(jobName string) string {
	parts := strings.SplitN(jobName, "-", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[1]
	}
	return jobName
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func mergeToolArgsWithPayload(baseArgs json.RawMessage, payload json.RawMessage) (json.RawMessage, error) {
	if len(payload) == 0 || string(payload) == "null" {
		return baseArgs, nil
	}

	base := make(map[string]any)
	if len(baseArgs) > 0 {
		if err := json.Unmarshal(baseArgs, &base); err != nil {
			return nil, fmt.Errorf("invalid stored tool args: %w", err)
		}
	}

	incoming := make(map[string]any)
	if err := json.Unmarshal(payload, &incoming); err != nil {
		return nil, fmt.Errorf("invalid confirmation payload: %w", err)
	}

	for key, value := range incoming {
		base[key] = value
	}

	merged, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("failed to merge confirmation payload: %w", err)
	}
	return merged, nil
}

func applyResubmitOverrides(job *batch.Job, cpu *string, memory *string, gpuCount *int, gpuModel *string) (map[string]any, error) {
	if job == nil {
		return nil, fmt.Errorf("job spec is unavailable for override")
	}

	applied := make(map[string]any)
	for taskIdx := range job.Spec.Tasks {
		task := &job.Spec.Tasks[taskIdx]
		for containerIdx := range task.Template.Spec.Containers {
			container := &task.Template.Spec.Containers[containerIdx]
			if cpu != nil {
				quantity, err := resource.ParseQuantity(strings.TrimSpace(*cpu))
				if err != nil {
					return nil, fmt.Errorf("invalid cpu override: %w", err)
				}
				if container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				container.Resources.Requests[v1.ResourceCPU] = quantity
				if container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				container.Resources.Limits[v1.ResourceCPU] = quantity
				applied["cpu"] = quantity.String()
			}
			if memory != nil {
				quantity, err := resource.ParseQuantity(strings.TrimSpace(*memory))
				if err != nil {
					return nil, fmt.Errorf("invalid memory override: %w", err)
				}
				if container.Resources.Requests == nil {
					container.Resources.Requests = v1.ResourceList{}
				}
				container.Resources.Requests[v1.ResourceMemory] = quantity
				if container.Resources.Limits == nil {
					container.Resources.Limits = v1.ResourceList{}
				}
				container.Resources.Limits[v1.ResourceMemory] = quantity
				applied["memory"] = quantity.String()
			}

			gpuResourceName, changed, err := overrideGPUResourceRequirements(
				&container.Resources,
				gpuCount,
				gpuModel,
			)
			if err != nil {
				return nil, err
			}
			if changed {
				if gpuCount != nil {
					applied["gpu_count"] = *gpuCount
				}
				if gpuResourceName != "" {
					applied["gpu_resource_name"] = gpuResourceName
					if gpuModel != nil && strings.TrimSpace(*gpuModel) != "" {
						applied["gpu_model"] = normalizeGPUModelName(*gpuModel)
					}
				}
			}
		}
	}

	if len(applied) == 0 {
		applied["inherit"] = "original_spec"
	}
	return applied, nil
}

func overrideGPUResourceRequirements(
	requirements *v1.ResourceRequirements,
	gpuCount *int,
	gpuModel *string,
) (string, bool, error) {
	if requirements == nil {
		return "", false, nil
	}
	currentGPUKey := detectGPUResourceName(requirements.Requests)
	if currentGPUKey == "" {
		currentGPUKey = detectGPUResourceName(requirements.Limits)
	}
	if currentGPUKey == "" && gpuCount == nil && gpuModel == nil {
		return "", false, nil
	}

	targetGPUKey := currentGPUKey
	if gpuModel != nil && strings.TrimSpace(*gpuModel) != "" {
		targetGPUKey = normalizeGPUResourceName(currentGPUKey, *gpuModel)
	}
	if targetGPUKey == "" && gpuCount != nil && *gpuCount > 0 {
		targetGPUKey = normalizeGPUResourceName(currentGPUKey, "gpu")
	}
	if targetGPUKey == "" {
		return "", false, nil
	}

	changed := false
	if requirements.Requests == nil {
		requirements.Requests = v1.ResourceList{}
	}
	if requirements.Limits == nil {
		requirements.Limits = v1.ResourceList{}
	}

	if currentGPUKey != "" && currentGPUKey != targetGPUKey {
		moveResourceQuantity(requirements.Requests, currentGPUKey, targetGPUKey)
		moveResourceQuantity(requirements.Limits, currentGPUKey, targetGPUKey)
		changed = true
	}

	if gpuCount != nil {
		if *gpuCount < 0 {
			return "", false, fmt.Errorf("gpu_count must be non-negative")
		}
		if *gpuCount == 0 {
			if _, ok := requirements.Requests[targetGPUKey]; ok {
				delete(requirements.Requests, targetGPUKey)
				changed = true
			}
			if _, ok := requirements.Limits[targetGPUKey]; ok {
				delete(requirements.Limits, targetGPUKey)
				changed = true
			}
			return "", changed, nil
		}
		quantity := *resource.NewQuantity(int64(*gpuCount), resource.DecimalSI)
		requirements.Requests[targetGPUKey] = quantity
		requirements.Limits[targetGPUKey] = quantity
		changed = true
	}

	return string(targetGPUKey), changed, nil
}

func detectGPUResourceName(resources v1.ResourceList) v1.ResourceName {
	for name := range resources {
		nameStr := strings.ToLower(string(name))
		if strings.Contains(nameStr, "/gpu") || strings.Contains(nameStr, "gpu") {
			return name
		}
	}
	return ""
}

func normalizeGPUModelName(input string) string {
	model := strings.TrimSpace(strings.ToLower(input))
	model = strings.ReplaceAll(model, " ", "-")
	return model
}

func normalizeGPUResourceName(current v1.ResourceName, gpuModel string) v1.ResourceName {
	model := normalizeGPUModelName(gpuModel)
	if model == "" {
		return current
	}
	if strings.Contains(model, "/") {
		return v1.ResourceName(model)
	}

	vendor := "nvidia.com"
	if current != "" {
		parts := strings.SplitN(string(current), "/", 2)
		if len(parts) == 2 && parts[0] != "" {
			vendor = parts[0]
		}
	}
	return v1.ResourceName(fmt.Sprintf("%s/%s", vendor, model))
}

func moveResourceQuantity(resources v1.ResourceList, oldName, newName v1.ResourceName) {
	if resources == nil || oldName == "" || oldName == newName {
		return
	}
	if quantity, ok := resources[oldName]; ok {
		resources[newName] = quantity
		delete(resources, oldName)
	}
}

// ─── Utility helpers ──────────────────────────────────────────────────────────

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

func agentMessageRequestID(msg *model.AgentMessage) string {
	if msg == nil || len(msg.Metadata) == 0 {
		return ""
	}
	var metadata map[string]any
	if err := json.Unmarshal(msg.Metadata, &metadata); err != nil {
		return ""
	}
	requestID, _ := metadata["requestId"].(string)
	return requestID
}

func historyContainsRequestID(messages []*model.AgentMessage, requestID string) bool {
	if requestID == "" {
		return false
	}
	for _, msg := range messages {
		if agentMessageRequestID(msg) == requestID {
			return true
		}
	}
	return false
}

func filterHistoryMessagesForRequest(messages []*model.AgentMessage, requestID string) []*model.AgentMessage {
	if requestID == "" {
		return messages
	}
	filtered := make([]*model.AgentMessage, 0, len(messages))
	for _, msg := range messages {
		if agentMessageRequestID(msg) == requestID {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func buildAgentHistory(messages []*model.AgentMessage) []map[string]any {
	history := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		if msg == nil || msg.Content == "" {
			continue
		}
		history = append(history, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return history
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

func messageContainsAny(target string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(target, part) {
			return true
		}
	}
	return false
}

func buildRecentHistoryHintText(history []*model.AgentMessage, limit int) string {
	if len(history) == 0 || limit <= 0 {
		return ""
	}
	start := len(history) - limit
	if start < 0 {
		start = 0
	}
	fragments := make([]string, 0, len(history)-start)
	for _, msg := range history[start:] {
		if msg == nil {
			continue
		}
		content := strings.TrimSpace(strings.ToLower(msg.Content))
		if content == "" {
			continue
		}
		fragments = append(fragments, content)
	}
	return strings.Join(fragments, "\n")
}

func isAgentContinuationIntent(messageHint string) bool {
	if messageHint == "" {
		return false
	}
	normalized := strings.TrimSpace(messageHint)
	switch normalized {
	case "确认", "好的", "好", "行", "可以", "继续", "提交", "执行", "就这样", "是的", "yes", "ok", "okay":
		return true
	}
	return messageContainsAny(
		normalized,
		"确认", "继续", "执行", "提交", "就这样", "改成", "改名", "换成", "名字", "叫", "rename", "resubmit",
	)
}

func buildAgentCapabilities(
	token util.JWTMessage,
	page map[string]any,
	message string,
	history []*model.AgentMessage,
) map[string]any {
	route, _ := page["route"].(string)
	url, _ := page["url"].(string)
	jobName, _ := page["job_name"].(string)
	routeHint := route
	if routeHint == "" {
		routeHint = url
	}
	messageHint := strings.ToLower(strings.TrimSpace(message))
	historyHint := buildRecentHistoryHintText(history, 6)
	continuationIntent := isAgentContinuationIntent(messageHint)

	enabledSet := map[string]struct{}{}
	addTools := func(names ...string) {
		for _, name := range names {
			if strings.TrimSpace(name) == "" {
				continue
			}
			enabledSet[name] = struct{}{}
		}
	}

	addTools(agentToolListUserJobs, agentToolGetHealthOverview)

	isImagePage := messageContainsAny(routeHint, "/env/images")
	isAdminPage := strings.HasPrefix(routeHint, "/admin")
	isJobPage := jobName != "" || messageContainsAny(routeHint, "/jobs", "/vcjobs", "/job/")
	isCreateIntent := messageContainsAny(
		messageHint,
		"创建", "新建", "提交", "训练", "jupyter", "镜像", "image", "gpu", "intent", "模型",
	)
	isJobIntent := isJobPage || messageContainsAny(
		messageHint,
		"失败", "oom", "日志", "事件", "诊断", "排队", "pending", "重提", "删除", "停止", "job", "作业",
	) || regexp.MustCompile(`sg-[a-z0-9-]+`).MatchString(messageHint)
	if continuationIntent && historyHint != "" {
		if messageContainsAny(
			historyHint,
			"创建", "新建", "jupyter", "交互式", "notebook", "训练作业", "自定义作业", "create_jupyter_job", "create_training_job",
		) {
			isCreateIntent = true
		}
		if messageContainsAny(
			historyHint,
			"重提", "重新提交", "resubmit", "停止", "删除", "失败", "日志", "事件", "诊断", "排队", "job", "作业",
		) {
			isJobIntent = true
		}
	}

	if isImagePage || isCreateIntent || routeHint == "" {
		addTools(
			agentToolListImages,
			agentToolListCudaBase,
			agentToolListGPUModels,
			agentToolRecommendImages,
			agentToolCreateJupyter,
			agentToolCreateTrain,
			agentToolRealtimeCapacity,
		)
	}
	if continuationIntent && !isCreateIntent && !isJobIntent {
		addTools(
			agentToolResubmitJob,
			agentToolStopJob,
			agentToolDeleteJob,
			agentToolCreateJupyter,
			agentToolCreateTrain,
		)
	}

	if isJobIntent {
		addTools(
			agentToolGetJobDetail,
			agentToolGetJobEvents,
			agentToolGetJobLogs,
			agentToolDiagnoseJob,
			agentToolGetDiagnosticCtx,
			agentToolSearchSimilarFail,
			agentToolQueryJobMetrics,
			agentToolAnalyzeQueue,
			agentToolResubmitJob,
			agentToolStopJob,
			agentToolDeleteJob,
			agentToolRealtimeCapacity,
		)
	}

	if token.RolePlatform == model.RoleAdmin && isAdminPage {
		addTools(
			agentToolGetClusterHealth,
			agentToolListClusterJobs,
			agentToolListClusterNodes,
			agentToolListGPUModels,
			agentToolRealtimeCapacity,
		)
	}

	enabledTools := make([]string, 0, len(enabledSet))
	for name := range enabledSet {
		enabledTools = append(enabledTools, name)
	}
	sort.Strings(enabledTools)

	confirmSet := map[string]struct{}{
		agentToolResubmitJob:   {},
		agentToolStopJob:       {},
		agentToolDeleteJob:     {},
		agentToolCreateJupyter: {},
		agentToolCreateTrain:   {},
	}
	confirmTools := make([]string, 0, len(confirmSet))
	for _, name := range enabledTools {
		if _, ok := confirmSet[name]; ok {
			confirmTools = append(confirmTools, name)
		}
	}

	return map[string]any{
		"enabled_tools": enabledTools,
		"confirm_tools": confirmTools,
		"tool_catalog":  buildAgentToolCatalog(enabledTools),
		"role_policies": map[string]any{
			"coordinator": "可做路由、汇总，必要时允许少量只读取证",
			"planner":     "只读规划，可参考上下文和可用工具目录，不得执行写工具",
			"explorer":    "只读探索与检索，不得执行写工具",
			"executor":    "负责真正工具执行，写工具必须走确认流",
			"verifier":    "只读验证与挑战结论，不得执行写工具",
			"guide":       "帮助/说明型回答，不执行写工具",
			"general":     "通用平台回答，默认不执行写工具",
		},
	}
}

func (mgr *AgentMgr) listAccessibleImages(ctx context.Context, token util.JWTMessage) ([]agentAccessibleImage, error) {
	imageQuery := query.Image
	results := make([]agentAccessibleImage, 0)
	seen := make(map[uint]struct{})
	appendUnique := func(images []*model.Image, shareStatus model.ImageShareType) {
		for _, image := range images {
			if image == nil || image.ID == 0 || image.ImageLink == "" {
				continue
			}
			if _, ok := seen[image.ID]; ok {
				continue
			}
			seen[image.ID] = struct{}{}
			results = append(results, agentAccessibleImage{
				Image:       image,
				ShareStatus: shareStatus,
			})
		}
	}

	oldPublicImages, err := imageQuery.WithContext(ctx).
		Preload(imageQuery.User).
		Where(imageQuery.IsPublic.Is(true)).
		Order(imageQuery.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list public images: %w", err)
	}
	appendUnique(oldPublicImages, model.Public)

	imageAccountQuery := query.ImageAccount
	newPublicShares, err := imageAccountQuery.WithContext(ctx).
		Preload(imageAccountQuery.Image).
		Preload(imageAccountQuery.Image.User).
		Where(imageAccountQuery.AccountID.Eq(model.DefaultAccountID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list shared public images: %w", err)
	}
	newPublicImages := make([]*model.Image, 0, len(newPublicShares))
	for _, share := range newPublicShares {
		newPublicImages = append(newPublicImages, &share.Image)
	}
	appendUnique(newPublicImages, model.Public)

	accountShares, err := imageAccountQuery.WithContext(ctx).
		Preload(imageAccountQuery.Image).
		Preload(imageAccountQuery.Image.User).
		Where(imageAccountQuery.AccountID.Eq(token.AccountID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list account images: %w", err)
	}
	accountImages := make([]*model.Image, 0, len(accountShares))
	for _, share := range accountShares {
		accountImages = append(accountImages, &share.Image)
	}
	appendUnique(accountImages, model.AccountShare)

	privateImages, err := imageQuery.WithContext(ctx).
		Preload(imageQuery.User).
		Where(imageQuery.UserID.Eq(token.UserID)).
		Order(imageQuery.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list private images: %w", err)
	}
	appendUnique(privateImages, model.Private)

	imageUserQuery := query.ImageUser
	userShares, err := imageUserQuery.WithContext(ctx).
		Preload(imageUserQuery.Image).
		Preload(imageUserQuery.Image.User).
		Where(imageUserQuery.UserID.Eq(token.UserID)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list user-shared images: %w", err)
	}
	userImages := make([]*model.Image, 0, len(userShares))
	for _, share := range userShares {
		userImages = append(userImages, &share.Image)
	}
	appendUnique(userImages, model.UserShare)

	return results, nil
}

func buildAgentImageSummary(item agentAccessibleImage) map[string]any {
	description := ""
	if item.Image.Description != nil {
		description = strings.TrimSpace(*item.Image.Description)
	}
	imagePackName := ""
	if item.Image.ImagePackName != nil {
		imagePackName = *item.Image.ImagePackName
	}

	archs := item.Image.Archs.Data()
	if len(archs) == 0 {
		archs = []string{"linux/amd64"}
	}

	return map[string]any{
		"id":            item.Image.ID,
		"imageLink":     item.Image.ImageLink,
		"description":   description,
		"taskType":      item.Image.TaskType,
		"shareStatus":   item.ShareStatus,
		"imageSource":   item.Image.ImageSource.String(),
		"tags":          item.Image.Tags.Data(),
		"archs":         archs,
		"imagePackName": imagePackName,
		"createdAt":     item.Image.CreatedAt,
		"owner": map[string]any{
			"userID":   item.Image.User.ID,
			"username": item.Image.User.Name,
			"nickname": item.Image.User.Nickname,
		},
	}
}

func matchesImageJobType(taskType model.JobType, requested string) bool {
	requested = strings.TrimSpace(strings.ToLower(requested))
	if requested == "" || requested == string(model.JobTypeAll) {
		return true
	}
	switch requested {
	case "training":
		switch taskType {
		case model.JobTypeCustom, model.JobTypePytorch, model.JobTypeTensorflow, model.JobTypeDeepSpeed, model.JobTypeOpenMPI:
			return true
		default:
			return false
		}
	default:
		return strings.EqualFold(string(taskType), requested)
	}
}

func matchesImageKeyword(image *model.Image, keyword string) bool {
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if keyword == "" {
		return true
	}
	text := strings.ToLower(image.ImageLink)
	if image.Description != nil {
		text += " " + strings.ToLower(*image.Description)
	}
	if image.ImagePackName != nil {
		text += " " + strings.ToLower(*image.ImagePackName)
	}
	for _, tag := range image.Tags.Data() {
		text += " " + strings.ToLower(tag)
	}
	return strings.Contains(text, keyword)
}

func buildTrainingImageKeywords(taskDescription, framework string) []string {
	text := strings.ToLower(strings.TrimSpace(taskDescription + " " + framework))
	keywords := []string{}
	add := func(values ...string) {
		for _, value := range values {
			if value == "" {
				continue
			}
			already := false
			for _, existing := range keywords {
				if existing == value {
					already = true
					break
				}
			}
			if !already {
				keywords = append(keywords, value)
			}
		}
	}

	if strings.Contains(text, "pytorch") || strings.Contains(text, "torch") {
		add("pytorch", "torch")
	}
	if strings.Contains(text, "tensorflow") || strings.Contains(text, "tf") {
		add("tensorflow", "tf")
	}
	if strings.Contains(text, "意图") || strings.Contains(text, "nlp") || strings.Contains(text, "文本") ||
		strings.Contains(text, "分类") || strings.Contains(text, "bert") || strings.Contains(text, "transformer") {
		add("transformers", "bert", "nlp", "pytorch", "torch")
	}
	if strings.Contains(text, "jupyter") {
		add("jupyter")
	}
	if len(keywords) == 0 {
		add("python", "envd", "conda")
	}
	return keywords
}

func scoreTrainingImage(item agentAccessibleImage, keywords []string) (int, []string) {
	text := strings.ToLower(item.Image.ImageLink)
	if item.Image.Description != nil {
		text += " " + strings.ToLower(*item.Image.Description)
	}
	if item.Image.ImagePackName != nil {
		text += " " + strings.ToLower(*item.Image.ImagePackName)
	}
	for _, tag := range item.Image.Tags.Data() {
		text += " " + strings.ToLower(tag)
	}

	score := 0
	reasons := make([]string, 0, 4)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			score += 3
			reasons = append(reasons, fmt.Sprintf("命中关键词 %s", keyword))
		}
	}
	switch item.Image.TaskType {
	case model.JobTypePytorch, model.JobTypeCustom:
		score += 2
		reasons = append(reasons, fmt.Sprintf("任务类型为 %s，适合作为训练镜像", item.Image.TaskType))
	case model.JobTypeJupyter:
		score += 1
		reasons = append(reasons, "适合作为交互式实验镜像")
	}
	switch item.ShareStatus {
	case model.Public, model.AccountShare:
		score++
		reasons = append(reasons, "当前账户可直接复用")
	}
	if item.Image.Description != nil && strings.TrimSpace(*item.Image.Description) != "" {
		score++
	}
	return score, uniqueStrings(reasons)
}

func recommendationConfidence(score int) string {
	switch {
	case score >= 8:
		return "high"
	case score >= 5:
		return "medium"
	default:
		return "low"
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func isGPUResourceName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized != "" && (strings.Contains(normalized, "/gpu") || strings.Contains(normalized, "gpu") ||
		strings.Contains(normalized, "/a100") || strings.Contains(normalized, "/v100"))
}

func extractGPUModelFromResourceName(name string) string {
	parts := strings.SplitN(strings.TrimSpace(name), "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return name
}

func normalizePageContext(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var page map[string]any
	if err := json.Unmarshal(raw, &page); err != nil {
		return map[string]any{}
	}
	if jobName, ok := page["jobName"]; ok {
		page["job_name"] = jobName
	}
	if jobStatus, ok := page["jobStatus"]; ok {
		page["job_status"] = jobStatus
	}
	return page
}

func normalizeClientContext(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var clientContext map[string]any
	if err := json.Unmarshal(raw, &clientContext); err != nil {
		return map[string]any{}
	}
	return clientContext
}

func normalizeOrchestrationMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "multi_agent":
		return "multi_agent"
	default:
		return "single_agent"
	}
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
			"capabilities":  buildAgentCapabilities(token, pageContext, message, historyMessages),
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
	agentCtx, cancelAgentReq := context.WithTimeout(context.Background(), 5*time.Minute)
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

func (mgr *AgentMgr) buildToolConfirmation(token util.JWTMessage, toolName string, rawArgs json.RawMessage) AgentToolConfirmation {
	confirmation := AgentToolConfirmation{
		ToolName:    toolName,
		RiskLevel:   "high",
		Interaction: "approval",
	}
	switch toolName {
	case agentToolResubmitJob:
		confirmation.Interaction = "form"
		confirmation.Form = buildResubmitJobForm(rawArgs)
	case agentToolCreateJupyter:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateJupyterJobForm(rawArgs)
	case agentToolCreateTrain:
		confirmation.Interaction = "form"
		confirmation.Form = buildCreateTrainingJobForm(token, rawArgs)
	}
	confirmation.Description = mgr.buildConfirmationDescription(toolName, rawArgs)
	return confirmation
}

func parseToolArgsMap(rawArgs json.RawMessage) map[string]any {
	args := map[string]any{}
	_ = json.Unmarshal(rawArgs, &args)
	return args
}

func getToolArgString(args map[string]any, key, fallback string) string {
	value, _ := args[key].(string)
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func getToolArgInt(args map[string]any, key string, fallback int) int {
	value, ok := args[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func buildResubmitJobForm(rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	var gpuCountDefault any
	if rawValue, ok := args["gpu_count"]; ok && rawValue != nil {
		gpuCountDefault = getToolArgInt(args, "gpu_count", 0)
	}

	return &AgentToolForm{
		Title:       "检查并重新提交作业",
		Description: "Agent 已定位到待重提的作业。你可以在这里修改显示名称和资源；留空的字段会沿用原配置。",
		SubmitLabel: "重新提交作业",
		Fields: []AgentToolField{
			{
				Key:          "name",
				Label:        "显示名称",
				Type:         "text",
				Description:  "留空则沿用原作业显示名称；系统内部 jobName 仍会自动生成。",
				DefaultValue: getToolArgString(args, "name", ""),
			},
			{
				Key:          "cpu",
				Label:        "CPU",
				Type:         "text",
				Description:  "留空则沿用原配置，例如 4 或 8000m。",
				DefaultValue: getToolArgString(args, "cpu", ""),
			},
			{
				Key:          "memory",
				Label:        "内存",
				Type:         "text",
				Description:  "留空则沿用原配置，例如 32Gi。",
				DefaultValue: getToolArgString(args, "memory", ""),
			},
			{
				Key:          "gpu_count",
				Label:        "GPU 数量",
				Type:         "number",
				Description:  "留空则沿用原配置；填 0 表示不申请 GPU。",
				DefaultValue: gpuCountDefault,
			},
			{
				Key:          "gpu_model",
				Label:        "GPU 型号",
				Type:         "text",
				Description:  "留空则沿用原配置，例如 v100 / a100。",
				DefaultValue: getToolArgString(args, "gpu_model", ""),
			},
		},
	}
}

func buildCreateJupyterJobForm(rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)
	var gpuCountDefault any
	if rawValue, ok := args["gpu_count"]; ok && rawValue != nil {
		gpuCountDefault = getToolArgInt(args, "gpu_count", 0)
	}

	return &AgentToolForm{
		Title:       "补全 Jupyter 作业配置",
		Description: "Agent 已生成一个 Jupyter 作业草案。你可以在这里确认或补全镜像与资源配置后提交。",
		SubmitLabel: "提交 Jupyter 作业",
		Fields: []AgentToolField{
			{Key: "name", Label: "作业名称", Type: "text", Required: true, DefaultValue: getToolArgString(args, "name", "")},
			{
				Key:          "image_link",
				Label:        "镜像",
				Type:         "text",
				Required:     true,
				Placeholder:  "registry/project/image:tag",
				DefaultValue: getToolArgString(args, "image_link", ""),
			},
			{
				Key:          "cpu",
				Label:        "CPU",
				Type:         "text",
				Required:     true,
				Description:  "默认 2。",
				DefaultValue: getToolArgString(args, "cpu", "2"),
			},
			{
				Key:          "memory",
				Label:        "内存",
				Type:         "text",
				Required:     true,
				Description:  "默认 8Gi。",
				DefaultValue: getToolArgString(args, "memory", "8Gi"),
			},
			{
				Key:          "gpu_count",
				Label:        "GPU 数量",
				Type:         "number",
				Description:  "可选，填 0 或留空表示不申请 GPU。",
				DefaultValue: gpuCountDefault,
			},
			{
				Key:          "gpu_model",
				Label:        "GPU 型号",
				Type:         "text",
				Description:  "可选，例如 v100 / a100。",
				DefaultValue: getToolArgString(args, "gpu_model", ""),
			},
		},
	}
}

func buildCreateTrainingJobForm(token util.JWTMessage, rawArgs json.RawMessage) *AgentToolForm {
	args := parseToolArgsMap(rawArgs)

	defaultWorkingDir := "/workspace"
	if strings.TrimSpace(token.Username) != "" {
		defaultWorkingDir = fmt.Sprintf("/home/%s", token.Username)
	}

	return &AgentToolForm{
		Title:       "补全训练作业配置",
		Description: "Agent 已生成一个新作业草案，你可以在这里补全镜像、命令和资源后再提交。",
		SubmitLabel: "提交训练作业",
		Fields: []AgentToolField{
			{Key: "name", Label: "作业名称", Type: "text", Required: true, DefaultValue: getToolArgString(args, "name", "")},
			{Key: "image_link", Label: "镜像", Type: "text", Required: true, Placeholder: "registry/project/image:tag", DefaultValue: getToolArgString(args, "image_link", "")},
			{Key: "command", Label: "启动命令", Type: "textarea", Required: true, Placeholder: "python train.py --config ...", DefaultValue: getToolArgString(args, "command", "")},
			{Key: "working_dir", Label: "工作目录", Type: "text", Required: true, DefaultValue: getToolArgString(args, "working_dir", defaultWorkingDir)},
			{
				Key: "shell", Label: "Shell", Type: "select", DefaultValue: getToolArgString(args, "shell", "bash"),
				Options: []AgentToolFieldOption{
					{Value: "bash", Label: "bash"},
					{Value: "sh", Label: "sh"},
					{Value: "zsh", Label: "zsh"},
				},
			},
			{Key: "cpu", Label: "CPU", Type: "text", Required: true, DefaultValue: getToolArgString(args, "cpu", "4")},
			{Key: "memory", Label: "内存", Type: "text", Required: true, DefaultValue: getToolArgString(args, "memory", "16Gi")},
			{Key: "gpu_count", Label: "GPU 数量", Type: "number", DefaultValue: getToolArgInt(args, "gpu_count", 0)},
			{Key: "gpu_model", Label: "GPU 型号", Type: "text", Placeholder: "如 v100 / a100", DefaultValue: getToolArgString(args, "gpu_model", "")},
		},
	}
}

func (mgr *AgentMgr) buildConfirmationDescription(toolName string, rawArgs json.RawMessage) string {
	args := parseToolArgsMap(rawArgs)
	switch toolName {
	case agentToolStopJob:
		if jobName, ok := args["job_name"].(string); ok && jobName != "" {
			return fmt.Sprintf("停止作业 %s", jobName)
		}
		return "停止当前作业"
	case agentToolDeleteJob:
		if jobName, ok := args["job_name"].(string); ok && jobName != "" {
			return fmt.Sprintf("删除作业 %s", jobName)
		}
		return "删除当前作业"
	case agentToolResubmitJob:
		if jobName, ok := args["job_name"].(string); ok && jobName != "" {
			parts := []string{}
			if name := getToolArgString(args, "name", ""); name != "" {
				parts = append(parts, fmt.Sprintf("显示名称=%s", name))
			}
			if cpu := getToolArgString(args, "cpu", ""); cpu != "" {
				parts = append(parts, fmt.Sprintf("CPU=%s", cpu))
			}
			if memory := getToolArgString(args, "memory", ""); memory != "" {
				parts = append(parts, fmt.Sprintf("内存=%s", memory))
			}
			if gpuCount := getToolArgInt(args, "gpu_count", 0); gpuCount > 0 {
				gpuText := fmt.Sprintf("GPU=%d", gpuCount)
				if gpuModel := getToolArgString(args, "gpu_model", ""); gpuModel != "" {
					gpuText += fmt.Sprintf(" (%s)", gpuModel)
				}
				parts = append(parts, gpuText)
			}
			if len(parts) == 0 {
				return fmt.Sprintf("重新提交作业 %s", jobName)
			}
			return fmt.Sprintf("重新提交作业 %s，并应用覆盖：%s", jobName, strings.Join(parts, "，"))
		}
		return "重新提交当前作业"
	case agentToolCreateJupyter:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的 Jupyter 作业"
		}
		parts := []string{}
		if imageLink := getToolArgString(args, "image_link", ""); imageLink != "" {
			parts = append(parts, fmt.Sprintf("镜像=%s", imageLink))
		}
		if cpu := getToolArgString(args, "cpu", ""); cpu != "" {
			parts = append(parts, fmt.Sprintf("CPU=%s", cpu))
		}
		if memory := getToolArgString(args, "memory", ""); memory != "" {
			parts = append(parts, fmt.Sprintf("内存=%s", memory))
		}
		if gpuCount := getToolArgInt(args, "gpu_count", 0); gpuCount > 0 {
			gpuText := fmt.Sprintf("GPU=%d", gpuCount)
			if gpuModel := getToolArgString(args, "gpu_model", ""); gpuModel != "" {
				gpuText += fmt.Sprintf(" (%s)", gpuModel)
			}
			parts = append(parts, gpuText)
		}
		if len(parts) == 0 {
			return fmt.Sprintf("创建 Jupyter 作业 %s", name)
		}
		return fmt.Sprintf("创建 Jupyter 作业 %s：%s", name, strings.Join(parts, "，"))
	case agentToolCreateTrain:
		name := getToolArgString(args, "name", "")
		if name == "" {
			name = "新的训练作业"
		}
		return fmt.Sprintf("创建训练作业 %s", name)
	default:
		return fmt.Sprintf("执行操作 %s", toolName)
	}
}

func (mgr *AgentMgr) buildToolOutcomeMessage(toolName, status string, result any, fallback string) string {
	if status == "rejected" {
		return "已取消该操作。"
	}
	if status == agentToolStatusError {
		if fallback != "" {
			return fmt.Sprintf("%s 执行失败：%s", toolName, fallback)
		}
		return fmt.Sprintf("%s 执行失败。", toolName)
	}

	resultMap, _ := result.(map[string]any)
	switch toolName {
	case agentToolStopJob:
		if jobName, _ := resultMap["jobName"].(string); jobName != "" {
			return fmt.Sprintf("已停止作业 %s。", jobName)
		}
		return "已停止目标作业。"
	case agentToolDeleteJob:
		if jobName, _ := resultMap["jobName"].(string); jobName != "" {
			return fmt.Sprintf("已删除作业 %s。", jobName)
		}
		return "已删除目标作业。"
	case agentToolResubmitJob:
		sourceJobName, _ := resultMap["sourceJobName"].(string)
		jobName, _ := resultMap["jobName"].(string)
		if sourceJobName != "" && jobName != "" {
			return fmt.Sprintf("已基于 %s 重新提交作业 %s。", sourceJobName, jobName)
		}
		return "已重新提交作业。"
	case agentToolCreateJupyter:
		return "已提交新的 Jupyter 作业。"
	case agentToolCreateTrain:
		return "已提交新的训练作业。"
	default:
		if fallback != "" {
			return fallback
		}
		return "操作已完成。"
	}
}

func (mgr *AgentMgr) persistAssistantToolMessage(ctx context.Context, sessionID, toolName, status, content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	metadata, _ := json.Marshal(map[string]any{
		"source":   "tool_confirmation",
		"toolName": toolName,
		"status":   status,
	})
	_ = mgr.agentService.SaveMessage(ctx, &model.AgentMessage{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   content,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	})
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
