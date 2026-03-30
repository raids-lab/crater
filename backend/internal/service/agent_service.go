package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

// AgentService encapsulates DB operations for the Agent feature.
type AgentService struct {
	db *gorm.DB
}

// NewAgentService creates a new AgentService.
func NewAgentService() *AgentService {
	return &AgentService{
		db: query.GetDB(),
	}
}

// CreateSession creates a new agent session in the database.
func (s *AgentService) CreateSession(ctx context.Context, sessionID string, userID, accountID uint, title string, pageContext json.RawMessage) (*model.AgentSession, error) {
	session := &model.AgentSession{
		SessionID: sessionID,
		UserID:    userID,
		AccountID: accountID,
		Title:     title,
	}
	if len(pageContext) > 0 {
		session.PageContext = datatypes.JSON(pageContext)
	}
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

// GetSession retrieves a session by sessionID.
func (s *AgentService) GetSession(ctx context.Context, sessionID string) (*model.AgentSession, error) {
	var session model.AgentSession
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// GetOrCreateSession retrieves an existing session or creates a new one.
func (s *AgentService) GetOrCreateSession(ctx context.Context, sessionID string, userID, accountID uint, title string, pageContext json.RawMessage) (*model.AgentSession, bool, error) {
	var session model.AgentSession
	err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error
	if err == nil {
		return &session, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	created, createErr := s.CreateSession(ctx, sessionID, userID, accountID, title, pageContext)
	if createErr != nil {
		return nil, false, createErr
	}
	return created, true, nil
}

// ListSessions returns all sessions for a given user, ordered by most recent.
func (s *AgentService) ListSessions(ctx context.Context, userID uint) ([]*model.AgentSession, error) {
	var sessions []*model.AgentSession
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// UpdateSessionTitle updates the title of a session.
func (s *AgentService) UpdateSessionTitle(ctx context.Context, sessionID string, title string) error {
	return s.db.WithContext(ctx).
		Model(&model.AgentSession{}).
		Where("session_id = ?", sessionID).
		Update("title", title).Error
}

// IncrementMessageCount increments the message count for a session.
func (s *AgentService) IncrementMessageCount(ctx context.Context, sessionID string) error {
	return s.db.WithContext(ctx).
		Model(&model.AgentSession{}).
		Where("session_id = ?", sessionID).
		UpdateColumn("message_count", gorm.Expr("message_count + 1")).Error
}

// SaveMessage saves a message to the agent_messages table.
func (s *AgentService) SaveMessage(ctx context.Context, msg *model.AgentMessage) error {
	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return err
	}
	// Increment message count in session asynchronously to avoid blocking.
	go func() {
		if err := s.IncrementMessageCount(context.Background(), msg.SessionID); err != nil {
			klog.Warningf("[AgentService] Failed to increment message count for session %s: %v", msg.SessionID, err)
		}
	}()
	return nil
}

// ListMessages returns all messages for a session ordered by creation time.
func (s *AgentService) ListMessages(ctx context.Context, sessionID string) ([]*model.AgentMessage, error) {
	var messages []*model.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// LogToolCall records a tool execution in the agent_tool_calls table.
func (s *AgentService) LogToolCall(ctx context.Context, toolCall *model.AgentToolCall) error {
	return s.db.WithContext(ctx).Create(toolCall).Error
}

// LogToolCallAsync records a tool execution asynchronously to avoid blocking the caller.
func (s *AgentService) LogToolCallAsync(sessionID, toolName string, toolArgs, toolResult json.RawMessage, resultStatus string, latencyMs int) {
	go func() {
		tc := &model.AgentToolCall{
			SessionID:    sessionID,
			ToolName:     toolName,
			ToolArgs:     datatypes.JSON(toolArgs),
			ToolResult:   datatypes.JSON(toolResult),
			ResultStatus: resultStatus,
			LatencyMs:    latencyMs,
			CreatedAt:    time.Now(),
		}
		if err := s.db.WithContext(context.Background()).Create(tc).Error; err != nil {
			klog.Warningf("[AgentService] Failed to log tool call %s for session %s: %v", toolName, sessionID, err)
		}
	}()
}
