package agent

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/internal/bizerr"
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
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid feedback upsert request"))
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
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("feedback already submitted"))
			return
		}
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to upsert feedback"))
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
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid feedback submit request"))
		return
	}

	token := util.GetToken(c)

	result, err := mgr.agentService.SubmitFeedback(c.Request.Context(), token.UserID, req.TargetType, req.TargetID)
	if err != nil {
		if errors.Is(err, service.ErrFeedbackAlreadySubmitted) {
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("feedback already submitted"))
			return
		}
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to submit feedback"))
		return
	}

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
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("sessionId is required"))
		return
	}

	token := util.GetToken(c)

	feedbacks, err := mgr.agentService.ListFeedbacks(c.Request.Context(), sessionID, token.UserID)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to list feedbacks"))
		return
	}

	resputil.Success(c, feedbacks)
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
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid quick feedback submit request"))
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
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to quick submit feedback"))
		return
	}

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
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid feedback enrich request"))
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
			resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("feedback not found"))
			return
		}
		resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, "failed to enrich feedback"))
		return
	}

	resputil.Success(c, result)
}
