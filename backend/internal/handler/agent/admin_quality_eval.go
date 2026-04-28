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

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
)

type triggerSessionQualityEvalRequest struct {
	TurnID            string `json:"turnId,omitempty"`
	EvalScope         string `json:"evalScope,omitempty"`
	EvalType          string `json:"evalType,omitempty"`
	DialogueModelRole string `json:"dialogueModelRole,omitempty"`
	TaskModelRole     string `json:"taskModelRole,omitempty"`
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
		service.AgentQualityEvalTriggerOptions{
			SessionID:         sessionID,
			TurnID:            req.TurnID,
			TriggerSource:     "manual",
			EvalScope:         req.EvalScope,
			EvalType:          req.EvalType,
			DialogueModelRole: req.DialogueModelRole,
			TaskModelRole:     req.TaskModelRole,
		},
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
	go mgr.dispatchManualQualityEval(eval, req.DialogueModelRole, req.TaskModelRole)

	resputil.Success(c, gin.H{
		"evalId":        eval.ID,
		"sessionId":     sessionID,
		"turnId":        eval.TurnID,
		"evalScope":     eval.EvalScope,
		"evalType":      eval.EvalType,
		"evalStatus":    eval.EvalStatus,
		"triggerSource": eval.TriggerSource,
		"createdAt":     eval.CreatedAt,
	})
}

func (mgr *AgentMgr) dispatchManualQualityEval(eval *model.AgentQualityEval, dialogueModelRole, taskModelRole string) {
	payload := map[string]any{
		"eval_id":             eval.ID,
		"session_id":          eval.SessionID,
		"turn_id":             eval.TurnID,
		"eval_scope":          eval.EvalScope,
		"eval_type":           eval.EvalType,
		"dialogue_model_role": dialogueModelRole,
		"task_model_role":     taskModelRole,
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
		msg := fmt.Sprintf("build crater-agent request failed: %v", err)
		_ = mgr.agentService.FailQualityEval(context.Background(), eval.ID, msg)
		klog.Warningf("[AgentMgr] dispatchManualQualityEval: %s", msg)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Internal-Token", mgr.getPythonAgentInternalToken())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("call crater-agent failed: %v", err)
		_ = mgr.agentService.FailQualityEval(context.Background(), eval.ID, msg)
		klog.Warningf("[AgentMgr] dispatchManualQualityEval: session %s %s", eval.SessionID, msg)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("crater-agent returned status %d", resp.StatusCode)
		_ = mgr.agentService.FailQualityEval(context.Background(), eval.ID, msg)
		klog.Warningf("[AgentMgr] dispatchManualQualityEval: %s for session %s", msg, eval.SessionID)
		return
	}
	_ = mgr.agentService.SetQualityEvalStatus(context.Background(), eval.ID, "running", "")
}
