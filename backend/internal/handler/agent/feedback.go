package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"

	"github.com/raids-lab/crater/dao/model"
)

// ── Request types ───────────────────────────────────────────────────────────

type FeedbackUpsertRequest struct {
	SessionID  string          `json:"sessionId" binding:"required"`
	TargetType string          `json:"targetType" binding:"required,oneof=message turn"`
	TargetID   string          `json:"targetId" binding:"required"`
	Rating     int16           `json:"rating" binding:"required,oneof=1 -1"`
	Tags       json.RawMessage `json:"tags,omitempty"`
	Dimensions json.RawMessage `json:"dimensions,omitempty"`
	Comment    string          `json:"comment,omitempty"`
}

type FeedbackSubmitRequest struct {
	SessionID  string `json:"sessionId" binding:"required"`
	TargetType string `json:"targetType" binding:"required,oneof=message turn"`
	TargetID   string `json:"targetId" binding:"required"`
}

// ── Handlers ────────────────────────────────────────────────────────────────

// UpsertFeedback godoc
// @Summary Create or update a feedback (draft)
// @Tags agent
// @Accept json
// @Produce json
// @Param request body FeedbackUpsertRequest true "Feedback upsert request"
// @Router /api/v1/agent/feedbacks [put]
func (mgr *AgentMgr) UpsertFeedback(c *gin.Context) {
	var req FeedbackUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	fb := &model.AgentFeedback{
		SessionID:  req.SessionID,
		UserID:     token.UserID,
		AccountID:  token.AccountID,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	}
	if len(req.Tags) > 0 {
		fb.Tags = datatypes.JSON(req.Tags)
	}
	if len(req.Dimensions) > 0 {
		fb.Dimensions = datatypes.JSON(req.Dimensions)
	}

	result, created, err := mgr.agentService.UpsertFeedback(c.Request.Context(), fb)
	if err != nil {
		if errors.Is(err, service.ErrFeedbackAlreadySubmitted) {
			resputil.HTTPError(c, http.StatusConflict, "feedback already submitted", resputil.NotSpecified)
			return
		}
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if created {
		c.JSON(http.StatusCreated, gin.H{"code": 201, "data": result})
	} else {
		resputil.Success(c, result)
	}
}

// SubmitFeedback godoc
// @Summary Submit a draft feedback (makes it immutable)
// @Tags agent
// @Accept json
// @Produce json
// @Param request body FeedbackSubmitRequest true "Feedback submit request"
// @Router /api/v1/agent/feedbacks/submit [post]
func (mgr *AgentMgr) SubmitFeedback(c *gin.Context) {
	var req FeedbackSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	result, err := mgr.agentService.SubmitFeedback(c.Request.Context(), token.UserID, req.TargetType, req.TargetID)
	if err != nil {
		if errors.Is(err, service.ErrFeedbackAlreadySubmitted) {
			resputil.HTTPError(c, http.StatusConflict, "feedback already submitted", resputil.NotSpecified)
			return
		}
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Fire async quality eval — do not block response
	go mgr.triggerQualityEval(result)

	resputil.Success(c, result)
}

// ListFeedbacks godoc
// @Summary List feedbacks for a session
// @Tags agent
// @Produce json
// @Param sessionId query string true "Session ID"
// @Router /api/v1/agent/feedbacks [get]
func (mgr *AgentMgr) ListFeedbacks(c *gin.Context) {
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	token := util.GetToken(c)

	feedbacks, err := mgr.agentService.ListFeedbacks(c.Request.Context(), sessionID, token.UserID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, feedbacks)
}

// GetFeedbackStats godoc
// @Summary Get aggregated feedback stats (admin)
// @Tags agent
// @Produce json
// @Param from query string false "Start time (RFC3339)"
// @Param to query string false "End time (RFC3339)"
// @Router /api/v1/agent/feedbacks/stats [get]
func (mgr *AgentMgr) GetFeedbackStats(c *gin.Context) {
	var from, to *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			resputil.BadRequestError(c, "invalid 'from' timestamp")
			return
		}
		from = &t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			resputil.BadRequestError(c, "invalid 'to' timestamp")
			return
		}
		to = &t
	}

	stats, err := mgr.agentService.GetFeedbackStats(c.Request.Context(), from, to)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, stats)
}

type FeedbackQuickSubmitRequest struct {
	SessionID  string          `json:"sessionId" binding:"required"`
	TargetType string          `json:"targetType" binding:"required,oneof=message turn"`
	TargetID   string          `json:"targetId" binding:"required"`
	Rating     int16           `json:"rating" binding:"required,oneof=1 -1"`
	Tags       json.RawMessage `json:"tags,omitempty"`
	Dimensions json.RawMessage `json:"dimensions,omitempty"`
	Comment    string          `json:"comment,omitempty"`
}

// QuickSubmitFeedback godoc
// @Summary Create and immediately submit feedback (single operation, no draft step)
// @Tags agent
// @Accept json
// @Produce json
// @Router /api/v1/agent/feedbacks/quick-submit [post]
func (mgr *AgentMgr) QuickSubmitFeedback(c *gin.Context) {
	var req FeedbackQuickSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	fb := &model.AgentFeedback{
		SessionID:  req.SessionID,
		UserID:     token.UserID,
		AccountID:  token.AccountID,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	}
	if len(req.Tags) > 0 {
		fb.Tags = datatypes.JSON(req.Tags)
	}
	if len(req.Dimensions) > 0 {
		fb.Dimensions = datatypes.JSON(req.Dimensions)
	}

	result, err := mgr.agentService.QuickSubmitFeedback(c.Request.Context(), fb)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Fire async quality eval — do not block response
	go mgr.triggerQualityEval(result)

	resputil.Success(c, result)
}

type FeedbackEnrichRequest struct {
	SessionID  string          `json:"sessionId" binding:"required"`
	TargetType string          `json:"targetType" binding:"required,oneof=message turn"`
	TargetID   string          `json:"targetId" binding:"required"`
	Tags       json.RawMessage `json:"tags,omitempty"`
	Dimensions json.RawMessage `json:"dimensions,omitempty"`
	Comment    string          `json:"comment,omitempty"`
}

// EnrichFeedback godoc
// @Summary Add/update detail fields on any feedback (even submitted). Never changes rating or status.
// @Tags agent
// @Accept json
// @Produce json
// @Router /api/v1/agent/feedbacks/enrich [put]
func (mgr *AgentMgr) EnrichFeedback(c *gin.Context) {
	var req FeedbackEnrichRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	result, err := mgr.agentService.EnrichFeedback(
		c.Request.Context(),
		token.UserID,
		req.TargetType,
		req.TargetID,
		datatypes.JSON(req.Tags),
		datatypes.JSON(req.Dimensions),
		req.Comment,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resputil.HTTPError(c, http.StatusNotFound, "feedback not found", resputil.NotSpecified)
			return
		}
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, result)
}

type QualityEvalResultRequest struct {
	EvalID       uint            `json:"evalId" binding:"required"`
	ChatScores   json.RawMessage `json:"chatScores,omitempty"`
	ChainScores  json.RawMessage `json:"chainScores,omitempty"`
	ChatModel    string          `json:"chatModel,omitempty"`
	ChainModel   string          `json:"chainModel,omitempty"`
	Summary      string          `json:"summary,omitempty"`
	RawChatResp  json.RawMessage `json:"rawChatResp,omitempty"`
	RawChainResp json.RawMessage `json:"rawChainResp,omitempty"`
	ArtifactPath string          `json:"artifactPath,omitempty"`
}

// ReceiveQualityEvalResult godoc
// @Summary Internal: receive quality eval result from crater-agent
// @Tags internal
// @Accept json
// @Produce json
// @Router /internal/agent/quality-evals [post]
func (mgr *AgentMgr) ReceiveQualityEvalResult(c *gin.Context) {
	if !mgr.isInternalToolRequestAuthorized(c) {
		resputil.HTTPError(c, http.StatusUnauthorized, "invalid internal agent token", resputil.TokenInvalid)
		return
	}

	var req QualityEvalResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	err := mgr.agentService.UpdateQualityEvalResult(
		c.Request.Context(),
		req.EvalID,
		datatypes.JSON(req.ChatScores),
		datatypes.JSON(req.ChainScores),
		req.ChatModel,
		req.ChainModel,
		req.Summary,
		datatypes.JSON(req.RawChatResp),
		datatypes.JSON(req.RawChainResp),
		req.ArtifactPath,
	)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, gin.H{"ok": true})
}

// ListQualityEvals godoc
// @Summary List quality eval records (admin)
// @Tags agent
// @Produce json
// @Param sessionId query string false "Filter by session ID"
// @Param triggerSource query string false "Filter by trigger source (feedback|offline_batch|manual)"
// @Param limit query int false "Max records to return (default 50)"
// @Router /api/v1/admin/agent/quality-evals [get]
func (mgr *AgentMgr) ListQualityEvals(c *gin.Context) {
	sessionID := c.Query("sessionId")
	triggerSource := c.Query("triggerSource")
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := parseInt(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	evals, err := mgr.agentService.ListQualityEvals(c.Request.Context(), sessionID, triggerSource, limit)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, evals)
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// This must be called in a goroutine — it must never block or affect user-facing responses.
func (mgr *AgentMgr) triggerQualityEval(fb *model.AgentFeedback) {
	// First create a pending eval record so we have an ID to pass to crater-agent
	var turnID string
	if fb.TargetType == "turn" {
		turnID = fb.TargetID
	}

	eval := &model.AgentQualityEval{
		SessionID:     fb.SessionID,
		TurnID:        turnID,
		FeedbackID:    &fb.ID,
		TriggerSource: "feedback",
	}
	if err := mgr.agentService.CreateQualityEval(context.Background(), eval); err != nil {
		klog.Warningf("[AgentMgr] triggerQualityEval: failed to create quality eval record for session %s: %v", fb.SessionID, err)
		return
	}

	payload := map[string]any{
		"eval_id":     eval.ID,
		"session_id":  fb.SessionID,
		"turn_id":     turnID,
		"feedback_id": fb.ID,
		"rating":      fb.Rating,
	}
	body, _ := json.Marshal(payload)

	serviceURL := mgr.getPythonAgentURL()
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		serviceURL+"/eval/quality/feedback",
		bytes.NewReader(body),
	)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Internal-Token", mgr.getPythonAgentInternalToken())

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
