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
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

type triggerSessionQualityEvalRequest struct {
	TurnID string `json:"turnId,omitempty"`
}

// TriggerSessionQualityEval godoc
// @Summary Admin: manually trigger a quality eval for a session
// @Tags agent-admin
// @Accept json
// @Produce json
// @Param sessionId path string true "Session ID (UUID)"
// @Router /api/v1/admin/agent/sessions/{sessionId}/trigger-eval [post]
func (mgr *AgentMgr) TriggerSessionQualityEval(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		resputil.BadRequestError(c, "sessionId is required")
		return
	}

	var req triggerSessionQualityEvalRequest
	_ = c.ShouldBindJSON(&req) // body is optional

	eval, err := mgr.agentService.TriggerSessionQualityEval(
		c.Request.Context(),
		sessionID,
		req.TurnID,
		"manual",
	)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			resputil.HTTPError(c, http.StatusNotFound, "session not found", resputil.NotSpecified)
			return
		}
		resputil.Error(c, fmt.Sprintf("failed to create quality eval: %v", err), resputil.NotSpecified)
		return
	}

	// Fire and forget: call crater-agent so the analyzer starts running.
	go mgr.dispatchManualQualityEval(eval.ID, sessionID, req.TurnID)

	resputil.Success(c, gin.H{
		"evalId":        eval.ID,
		"sessionId":     sessionID,
		"turnId":        req.TurnID,
		"evalStatus":    eval.EvalStatus,
		"triggerSource": eval.TriggerSource,
		"createdAt":     eval.CreatedAt,
	})
}

func (mgr *AgentMgr) dispatchManualQualityEval(evalID uint, sessionID, turnID string) {
	payload := map[string]any{
		"eval_id":    evalID,
		"session_id": sessionID,
		"turn_id":    turnID,
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serviceURL := mgr.getPythonAgentURL()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		serviceURL+"/eval/quality/session",
		bytes.NewReader(body),
	)
	if err != nil {
		klog.Warningf("[AgentMgr] dispatchManualQualityEval: build request for session %s failed: %v", sessionID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Internal-Token", mgr.getPythonAgentInternalToken())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		klog.Warningf("[AgentMgr] dispatchManualQualityEval: call crater-agent for session %s failed: %v", sessionID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		klog.Warningf("[AgentMgr] dispatchManualQualityEval: crater-agent returned %d for session %s", resp.StatusCode, sessionID)
	}
}
