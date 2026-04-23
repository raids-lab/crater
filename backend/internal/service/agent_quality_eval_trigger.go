package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

// ErrSessionNotFound is returned when the admin tries to trigger eval on an unknown session.
var ErrSessionNotFound = errors.New("session not found")

// TriggerSessionQualityEval creates a pending AgentQualityEval row for the given session
// and returns it. The caller is responsible for actually dispatching the work to crater-agent.
// turnID is optional (empty string means "evaluate the whole session / latest turn").
func (s *AgentService) TriggerSessionQualityEval(
	ctx context.Context,
	sessionID, turnID, triggerSource string,
) (*model.AgentQualityEval, error) {
	var session model.AgentSession
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	if triggerSource == "" {
		triggerSource = "manual"
	}

	eval := &model.AgentQualityEval{
		SessionID:     sessionID,
		TurnID:        turnID,
		TriggerSource: triggerSource,
	}
	if err := s.CreateQualityEval(ctx, eval); err != nil {
		return nil, err
	}
	return eval, nil
}
