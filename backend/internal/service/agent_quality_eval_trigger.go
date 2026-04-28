package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

// ErrSessionNotFound is returned when the admin tries to trigger eval on an unknown session.
var ErrSessionNotFound = errors.New("session not found")

const (
	AgentQualityEvalScopeSession = "session"
	AgentQualityEvalScopeTurn    = "turn"

	AgentQualityEvalTypeFull     = "full"
	AgentQualityEvalTypeDialogue = "dialogue"
	AgentQualityEvalTypeTask     = "task"
)

type AgentQualityEvalTriggerOptions struct {
	SessionID          string
	TurnID             string
	TriggerSource      string
	EvalScope          string
	EvalType           string
	DialogueModelRole  string
	TaskModelRole      string
	AdditionalMetadata map[string]any
}

func normalizeAgentQualityEvalScope(scope, turnID string) string {
	switch strings.TrimSpace(strings.ToLower(scope)) {
	case AgentQualityEvalScopeTurn:
		return AgentQualityEvalScopeTurn
	case AgentQualityEvalScopeSession:
		return AgentQualityEvalScopeSession
	default:
		if strings.TrimSpace(turnID) != "" {
			return AgentQualityEvalScopeTurn
		}
		return AgentQualityEvalScopeSession
	}
}

func normalizeAgentQualityEvalType(evalType string) string {
	switch strings.TrimSpace(strings.ToLower(evalType)) {
	case AgentQualityEvalTypeDialogue:
		return AgentQualityEvalTypeDialogue
	case AgentQualityEvalTypeTask:
		return AgentQualityEvalTypeTask
	default:
		return AgentQualityEvalTypeFull
	}
}

func buildAgentQualityEvalMetadata(opts AgentQualityEvalTriggerOptions) datatypes.JSON {
	metadata := map[string]any{}
	for k, v := range opts.AdditionalMetadata {
		metadata[k] = v
	}
	if role := strings.TrimSpace(opts.DialogueModelRole); role != "" {
		metadata["dialogueModelRole"] = role
	}
	if role := strings.TrimSpace(opts.TaskModelRole); role != "" {
		metadata["taskModelRole"] = role
	}
	if len(metadata) == 0 {
		return nil
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil
	}
	return datatypes.JSON(raw)
}

// TriggerSessionQualityEval creates a pending AgentQualityEval row for the given session
// and returns it. The caller is responsible for actually dispatching the work to crater-agent.
// TurnID is optional for session-level evals and required for turn-level evals.
func (s *AgentService) TriggerSessionQualityEval(
	ctx context.Context,
	opts AgentQualityEvalTriggerOptions,
) (*model.AgentQualityEval, error) {
	sessionID := strings.TrimSpace(opts.SessionID)
	var session model.AgentSession
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	turnID := strings.TrimSpace(opts.TurnID)
	evalScope := normalizeAgentQualityEvalScope(opts.EvalScope, turnID)
	if evalScope == AgentQualityEvalScopeTurn {
		if turnID == "" {
			return nil, errors.New("turnId is required for turn-scope eval")
		}
		var turn model.AgentTurn
		if err := s.db.WithContext(ctx).
			Where("turn_id = ? AND session_id = ?", turnID, sessionID).
			First(&turn).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("turn not found in session")
			}
			return nil, err
		}
	}

	triggerSource := strings.TrimSpace(opts.TriggerSource)
	if triggerSource == "" {
		triggerSource = "manual"
	}
	evalType := normalizeAgentQualityEvalType(opts.EvalType)
	targetID := sessionID
	if evalScope == AgentQualityEvalScopeTurn {
		targetID = turnID
	}

	eval := &model.AgentQualityEval{
		SessionID:     sessionID,
		TurnID:        turnID,
		EvalScope:     evalScope,
		EvalType:      evalType,
		TargetID:      targetID,
		TriggerSource: triggerSource,
		Metadata:      buildAgentQualityEvalMetadata(opts),
	}
	if err := s.CreateQualityEval(ctx, eval); err != nil {
		return nil, err
	}
	return eval, nil
}
