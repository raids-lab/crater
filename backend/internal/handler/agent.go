package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	pkgconfig "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

//nolint:gochecknoinits // Handler managers are registered during package initialization.
func init() {
	Registers = append(Registers, NewAgentMgr)
}

const (
	agentDefaultPythonServiceURL = "http://localhost:8000"

	agentToolStatusSuccess              = "success"
	agentToolStatusError                = "error"
	agentToolStatusConfirmationRequired = "confirmation_required"

	// Read-only tool names
	agentToolGetJobDetail      = "get_job_detail"
	agentToolGetJobEvents      = "get_job_events"
	agentToolGetJobLogs        = "get_job_logs"
	agentToolDiagnoseJob       = "diagnose_job"
	agentToolCheckQuota        = "check_quota"
	agentToolGetHealthOverview = "get_health_overview"

	// Write tools that require user confirmation before execution
	agentToolResubmitJob = "resubmit_job"
	agentToolStopJob     = "stop_job"
	agentToolDeleteJob   = "delete_job"
)

// AgentMgr handles agent-related API endpoints.
type AgentMgr struct {
	name         string
	client       client.Client
	kubeClient   kubernetes.Interface
	agentService *service.AgentService
	httpClient   *http.Client
}

// NewAgentMgr creates a new AgentMgr.
func NewAgentMgr(conf *RegisterConfig) Manager {
	return &AgentMgr{
		name:         "agent",
		client:       conf.Client,
		kubeClient:   conf.KubeClient,
		agentService: service.NewAgentService(),
		httpClient:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (mgr *AgentMgr) GetName() string { return mgr.name }

func (mgr *AgentMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *AgentMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("/chat", mgr.Chat)
	g.POST("/chat/confirm", mgr.ConfirmToolExecution)
	g.GET("/sessions", mgr.ListSessions)
	g.GET("/sessions/:sessionId/messages", mgr.GetSessionMessages)
	g.POST("/tools/execute", mgr.ExecuteTool)
}

func (mgr *AgentMgr) RegisterAdmin(_ *gin.RouterGroup) {}

// ─── Request / Response types ───────────────────────────────────────────────

// AgentChatRequest is the request body for POST /agent/chat.
type AgentChatRequest struct {
	Message     string          `json:"message" binding:"required"`
	SessionID   string          `json:"sessionId,omitempty"`
	PageContext json.RawMessage `json:"pageContext,omitempty"`
}

// ConfirmToolRequest is the request body for POST /agent/chat/confirm.
type ConfirmToolRequest struct {
	SessionID  string          `json:"sessionId" binding:"required"`
	ToolCallID string          `json:"toolCallId" binding:"required"`
	ToolName   string          `json:"toolName" binding:"required"`
	ToolArgs   json.RawMessage `json:"toolArgs" binding:"required"`
	Confirmed  bool            `json:"confirmed"`
}

// ExecuteToolRequest is the request body for POST /agent/tools/execute.
// This endpoint is called by the Python Agent service.
type ExecuteToolRequest struct {
	ToolName      string          `json:"tool_name" binding:"required"`
	ToolArgs      json.RawMessage `json:"tool_args" binding:"required"`
	SessionID     string          `json:"session_id" binding:"required"`
	RequestUserID uint            `json:"request_user_id"`
}

// AgentToolResponse is the response body for POST /agent/tools/execute.
type AgentToolResponse struct {
	Status  string          `json:"status"`
	Result  json.RawMessage `json:"result,omitempty"`
	Message string          `json:"message,omitempty"`
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

	token := util.GetToken(c)

	// Determine session ID – create one if not provided.
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Ensure session exists in the database.
	_, _, err := mgr.agentService.GetOrCreateSession(
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

	// Persist the user message.
	userMsg := &model.AgentMessage{
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Message,
		CreatedAt: time.Now(),
	}
	if saveErr := mgr.agentService.SaveMessage(c.Request.Context(), userMsg); saveErr != nil {
		// Non-fatal – log and continue.
		_ = saveErr
	}

	// Build the payload for the Python Agent service.
	agentPayload := map[string]any{
		"message":    req.Message,
		"session_id": sessionID,
		"user_id":    token.UserID,
		"account_id": token.AccountID,
		"username":   token.Username,
	}
	if req.PageContext != nil {
		agentPayload["page_context"] = req.PageContext
	}

	payloadBytes, err := json.Marshal(agentPayload)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to marshal agent payload: %v", err), resputil.NotSpecified)
		return
	}

	agentURL := mgr.getPythonAgentURL() + "/chat"
	agentReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, agentURL, bytes.NewReader(payloadBytes))
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create agent request: %v", err), resputil.NotSpecified)
		return
	}
	agentReq.Header.Set("Content-Type", "application/json")
	agentReq.Header.Set("Accept", "text/event-stream")

	resp, err := mgr.httpClient.Do(agentReq)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("agent service unavailable: %v", err), resputil.ServiceError)
		return
	}
	defer resp.Body.Close()

	// Stream SSE response back to the client.
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Agent-Session-ID", sessionID)

	var assistantContent bytes.Buffer
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if _, writeErr := fmt.Fprintf(c.Writer, "%s\n", line); writeErr != nil {
			break
		}
		c.Writer.Flush()
		// Accumulate assistant text lines for persisting.
		if len(line) > 6 && line[:6] == "data: " {
			assistantContent.WriteString(line[6:])
		}
	}

	// Persist the assistant message after streaming completes.
	if assistantContent.Len() > 0 {
		assistantMsg := &model.AgentMessage{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   assistantContent.String(),
			CreatedAt: time.Now(),
		}
		_ = mgr.agentService.SaveMessage(context.Background(), assistantMsg)
	}
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

	if !req.Confirmed {
		mgr.agentService.LogToolCallAsync(
			req.SessionID, req.ToolName,
			req.ToolArgs, nil,
			"rejected", 0,
		)
		resputil.Success(c, AgentToolResponse{
			Status:  agentToolStatusError,
			Message: "Operation rejected by user.",
		})
		return
	}

	token := util.GetToken(c)

	start := time.Now()
	result, execErr := mgr.executeWriteTool(c, token, req.ToolName, req.ToolArgs)
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
		status, latencyMs,
	)

	resputil.Success(c, AgentToolResponse{
		Status:  status,
		Result:  resultBytes,
		Message: errMsg,
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
		resputil.Error(c, fmt.Sprintf("failed to list sessions: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, sessions)
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

	messages, err := mgr.agentService.ListMessages(c.Request.Context(), sessionID)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to list messages: %v", err), resputil.NotSpecified)
		return
	}
	resputil.Success(c, messages)
}

// ExecuteTool godoc
// @Summary Execute a named tool (called by the Python Agent service)
// @Description Routes tool_name to the appropriate internal handler. Write tools return confirmation_required.
// @Tags agent
// @Accept json
// @Produce json
// @Param request body ExecuteToolRequest true "Tool execution request"
// @Success 200 {object} AgentToolResponse
// @Router /api/v1/agent/tools/execute [post]
//
//nolint:gocyclo // Tool routing dispatches many named tools in one function.
func (mgr *AgentMgr) ExecuteTool(c *gin.Context) {
	var req ExecuteToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	// Write tools require user confirmation – return a prompt without executing.
	switch req.ToolName {
	case agentToolResubmitJob, agentToolStopJob, agentToolDeleteJob:
		resputil.Success(c, AgentToolResponse{
			Status:  agentToolStatusConfirmationRequired,
			Message: fmt.Sprintf("Tool '%s' requires user confirmation before execution.", req.ToolName),
		})
		return
	}

	start := time.Now()

	// Execute read-only tools.
	result, execErr := mgr.executeReadTool(c, req)
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
		status, latencyMs,
	)

	resputil.Success(c, AgentToolResponse{
		Status:  status,
		Result:  resultBytes,
		Message: errMsg,
	})
}

// ─── Tool execution helpers ──────────────────────────────────────────────────

// executeReadTool dispatches a read-only tool by name.
func (mgr *AgentMgr) executeReadTool(c *gin.Context, req ExecuteToolRequest) (any, error) {
	// Build a token from the caller. If RequestUserID is provided, override UserID so
	// lookups are scoped to the correct user (the Python Agent calls on behalf of a user).
	token := util.GetToken(c)
	if req.RequestUserID != 0 {
		token.UserID = req.RequestUserID
	}

	switch req.ToolName {
	case agentToolGetJobDetail:
		return mgr.toolGetJobDetail(c, token, req.ToolArgs)
	case agentToolGetJobEvents:
		return mgr.toolGetJobEvents(c, token, req.ToolArgs)
	case agentToolGetJobLogs:
		return mgr.toolGetJobLogs(c, token, req.ToolArgs)
	case agentToolDiagnoseJob:
		return mgr.toolDiagnoseJob(c, token, req.ToolArgs)
	case agentToolCheckQuota:
		return mgr.toolCheckQuota(c, token)
	case agentToolGetHealthOverview:
		return mgr.toolGetHealthOverview(c, token, req.ToolArgs)
	default:
		return nil, fmt.Errorf("tool '%s' is not yet implemented", req.ToolName)
	}
}

// executeWriteTool executes a confirmed write tool.
func (mgr *AgentMgr) executeWriteTool(_ *gin.Context, _ util.JWTMessage, toolName string, _ json.RawMessage) (any, error) {
	return nil, fmt.Errorf("write tool '%s' execution is not yet implemented in the Go backend", toolName)
}

// ─── Individual tool implementations ─────────────────────────────────────────

type agentJobNameArgs struct {
	JobName string `json:"job_name"`
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

	j := query.Job
	job, err := j.WithContext(c).
		Where(j.JobName.Eq(args.JobName)).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		First()
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	return map[string]any{
		"jobName":            job.JobName,
		"name":               job.Name,
		"status":             job.Status,
		"jobType":            job.JobType,
		"creationTimestamp":  job.CreationTimestamp,
		"runningTimestamp":   job.RunningTimestamp,
		"completedTimestamp": job.CompletedTimestamp,
		"resources":          job.Resources.Data(),
	}, nil
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

	j := query.Job
	job, err := j.WithContext(c).
		Where(j.JobName.Eq(args.JobName)).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		First()
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
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
		TailLines int64  `json:"tail_lines"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("invalid args: %w", err)
	}
	if args.JobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}
	if args.TailLines <= 0 {
		args.TailLines = 100
	}

	// Verify the job belongs to the caller.
	j := query.Job
	job, err := j.WithContext(c).
		Where(j.JobName.Eq(args.JobName)).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		First()
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	namespace := pkgconfig.GetConfig().Namespaces.Job
	// Use the base-url label stored in the job's k8s attributes for pod lookup.
	labelSelector := fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, job.JobName)
	if job.Attributes.Data() != nil {
		if labelVal, ok := job.Attributes.Data().Labels[crclient.LabelKeyBaseURL]; ok && labelVal != "" {
			labelSelector = fmt.Sprintf("%s=%s", crclient.LabelKeyBaseURL, labelVal)
		}
	}
	podList, podErr := mgr.kubeClient.CoreV1().Pods(namespace).List(c.Request.Context(), metav1.ListOptions{
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
		TailLines: &args.TailLines,
	}).DoRaw(c.Request.Context())
	if logErr != nil {
		return map[string]string{"log": fmt.Sprintf("Failed to retrieve logs: %v", logErr)}, nil
	}

	return map[string]string{
		"podName":   pod.Name,
		"container": containerName,
		"log":       string(logBytes),
	}, nil
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

	j := query.Job
	job, err := j.WithContext(c).
		Where(j.JobName.Eq(args.JobName)).
		Where(j.UserID.Eq(token.UserID), j.AccountID.Eq(token.AccountID)).
		First()
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	return performDiagnosis(job), nil
}

// toolCheckQuota returns the current resource quota for the user.
func (mgr *AgentMgr) toolCheckQuota(c *gin.Context, token util.JWTMessage) (any, error) {
	ua := query.UserAccount
	userAccount, err := ua.WithContext(c).
		Where(ua.AccountID.Eq(token.AccountID), ua.UserID.Eq(token.UserID)).
		First()
	if err != nil {
		return nil, fmt.Errorf("user account not found: %w", err)
	}

	quota := userAccount.Quota.Data()
	return map[string]any{
		"capability": quota.Capability,
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

// ─── Utility helpers ──────────────────────────────────────────────────────────

// getPythonAgentURL returns the configured Python Agent service base URL.
func (mgr *AgentMgr) getPythonAgentURL() string {
	cfg := pkgconfig.GetConfig()
	if cfg.Agent.ServiceURL != "" {
		return cfg.Agent.ServiceURL
	}
	return agentDefaultPythonServiceURL
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
