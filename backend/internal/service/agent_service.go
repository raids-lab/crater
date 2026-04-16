package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

type toolCallAuditMetadata struct {
	ExecutionBackend  string
	SandboxJobName    string
	ScriptName        string
	ResultArtifactRef string
	EgressDomains     []string
}

var agentToolAuditCompatFields = []string{
	"ExecutionBackend",
	"SandboxJobName",
	"ScriptName",
	"ResultArtifactRef",
	"EgressDomains",
}

var agentToolAuditCompatColumns = []string{
	"execution_backend",
	"sandbox_job_name",
	"script_name",
	"result_artifact_ref",
	"egress_domains",
}

func parseStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					result = append(result, s)
				}
			}
		}
		return result
	default:
		return nil
	}
}

func parseToolCallAuditMeta(toolArgs, toolResult json.RawMessage) toolCallAuditMetadata {
	meta := toolCallAuditMetadata{}
	argsMap := map[string]any{}
	if len(toolArgs) > 0 {
		_ = json.Unmarshal(toolArgs, &argsMap)
		if scriptName, ok := argsMap["script_name"].(string); ok {
			meta.ScriptName = strings.TrimSpace(scriptName)
		}
	}

	resultMap := map[string]any{}
	if len(toolResult) > 0 {
		_ = json.Unmarshal(toolResult, &resultMap)
	}
	readMeta := func(source map[string]any) {
		if source == nil {
			return
		}
		if v, ok := source["execution_backend"].(string); ok && strings.TrimSpace(v) != "" {
			meta.ExecutionBackend = strings.TrimSpace(v)
		}
		if v, ok := source["sandbox_job_name"].(string); ok && strings.TrimSpace(v) != "" {
			meta.SandboxJobName = strings.TrimSpace(v)
		}
		if v, ok := source["script_name"].(string); ok && strings.TrimSpace(v) != "" {
			meta.ScriptName = strings.TrimSpace(v)
		}
		if v, ok := source["result_artifact_ref"].(string); ok && strings.TrimSpace(v) != "" {
			meta.ResultArtifactRef = strings.TrimSpace(v)
		}
		if domains := parseStringSlice(source["egress_domains"]); len(domains) > 0 {
			meta.EgressDomains = append(meta.EgressDomains, domains...)
		}
	}
	readMeta(resultMap)
	if nested, ok := resultMap["_audit"].(map[string]any); ok {
		readMeta(nested)
	}
	if len(meta.EgressDomains) > 0 {
		seen := make(map[string]struct{}, len(meta.EgressDomains))
		uniq := make([]string, 0, len(meta.EgressDomains))
		for _, domain := range meta.EgressDomains {
			trimmed := strings.TrimSpace(domain)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			uniq = append(uniq, trimmed)
		}
		sort.Strings(uniq)
		meta.EgressDomains = uniq
	}
	return meta
}

func isMissingAgentToolAuditColumnError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "agent_tool_calls") || !strings.Contains(message, "does not exist") {
		return false
	}
	for _, column := range agentToolAuditCompatColumns {
		if strings.Contains(message, fmt.Sprintf(`column "%s"`, column)) {
			return true
		}
	}
	return false
}

func stripUnsupportedAgentToolAuditUpdates(updates map[string]any) map[string]any {
	if len(updates) == 0 {
		return updates
	}
	compat := make(map[string]any, len(updates))
	for key, value := range updates {
		compat[key] = value
	}
	for _, column := range agentToolAuditCompatColumns {
		delete(compat, column)
	}
	return compat
}

func (s *AgentService) createToolCallWithCompat(ctx context.Context, toolCall *model.AgentToolCall) error {
	err := s.db.WithContext(ctx).Create(toolCall).Error
	if err == nil || !isMissingAgentToolAuditColumnError(err) {
		return err
	}
	return s.db.WithContext(ctx).Omit(agentToolAuditCompatFields...).Create(toolCall).Error
}

// AgentService encapsulates DB operations for the Agent feature.
type AgentService struct {
	db *gorm.DB
}

var ErrAgentSessionPinningUnavailable = errors.New(
	"agent session pinning is unavailable until database migration completes",
)

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

func (s *AgentService) GetOwnedSession(ctx context.Context, sessionID string, userID uint) (*model.AgentSession, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
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
	query := s.db.WithContext(ctx).Where("user_id = ?", userID)
	if s.db.WithContext(ctx).Migrator().HasColumn(&model.AgentSession{}, "PinnedAt") {
		query = query.
			Order("CASE WHEN pinned_at IS NULL THEN 1 ELSE 0 END ASC").
			Order("pinned_at DESC")
	}
	if err := query.Order("updated_at DESC").Find(&sessions).Error; err != nil {
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

func (s *AgentService) UpdateSessionOrchestrationMode(
	ctx context.Context,
	sessionID string,
	orchestrationMode string,
) error {
	if orchestrationMode == "" {
		return nil
	}
	return s.db.WithContext(ctx).
		Model(&model.AgentSession{}).
		Where("session_id = ?", sessionID).
		Update("last_orchestration_mode", orchestrationMode).Error
}

func (s *AgentService) UpdateSessionPinned(ctx context.Context, sessionID string, pinned bool) error {
	if !s.db.WithContext(ctx).Migrator().HasColumn(&model.AgentSession{}, "PinnedAt") {
		return ErrAgentSessionPinningUnavailable
	}
	if pinned {
		now := time.Now()
		return s.db.WithContext(ctx).
			Model(&model.AgentSession{}).
			Where("session_id = ?", sessionID).
			Updates(map[string]any{
				"pinned_at":  &now,
				"updated_at": time.Now(),
			}).Error
	}
	// Unpin: only clear pinned_at, do NOT touch updated_at so the session
	// returns to its original position in the time-sorted list.
	return s.db.WithContext(ctx).
		Model(&model.AgentSession{}).
		Where("session_id = ?", sessionID).
		Update("pinned_at", nil).Error
}

func (s *AgentService) DeleteSession(ctx context.Context, sessionID string) error {
	return s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&model.AgentSession{}).Error
}

// IncrementMessageCount increments the message count for a session.
func (s *AgentService) IncrementMessageCount(ctx context.Context, sessionID string) error {
	return s.db.WithContext(ctx).
		Model(&model.AgentSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]any{
			"message_count": gorm.Expr("message_count + 1"),
			"updated_at":    time.Now(),
		}).Error
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

func (s *AgentService) ListToolCalls(ctx context.Context, sessionID string) ([]*model.AgentToolCall, error) {
	var toolCalls []*model.AgentToolCall
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&toolCalls).Error; err != nil {
		return nil, err
	}
	return toolCalls, nil
}

func (s *AgentService) ListToolCallsByTurn(ctx context.Context, turnID string) ([]*model.AgentToolCall, error) {
	var toolCalls []*model.AgentToolCall
	if err := s.db.WithContext(ctx).
		Where("turn_id = ?", turnID).
		Order("created_at ASC").
		Find(&toolCalls).Error; err != nil {
		return nil, err
	}
	return toolCalls, nil
}

// LogToolCall records a tool execution in the agent_tool_calls table.
func (s *AgentService) LogToolCall(ctx context.Context, toolCall *model.AgentToolCall) error {
	return s.createToolCallWithCompat(ctx, toolCall)
}

func (s *AgentService) CreateToolCall(ctx context.Context, toolCall *model.AgentToolCall) (*model.AgentToolCall, error) {
	if err := s.createToolCallWithCompat(ctx, toolCall); err != nil {
		return nil, err
	}
	return toolCall, nil
}

func (s *AgentService) GetToolCallByID(ctx context.Context, id uint) (*model.AgentToolCall, error) {
	var toolCall model.AgentToolCall
	if err := s.db.WithContext(ctx).First(&toolCall, id).Error; err != nil {
		return nil, err
	}
	return &toolCall, nil
}

func (s *AgentService) UpdateToolCallOutcome(
	ctx context.Context,
	id uint,
	resultStatus string,
	toolResult json.RawMessage,
	userConfirmed *bool,
) error {
	meta := parseToolCallAuditMeta(nil, toolResult)
	updates := map[string]any{
		"result_status": resultStatus,
	}
	if toolResult != nil {
		updates["tool_result"] = datatypes.JSON(toolResult)
	}
	if userConfirmed != nil {
		updates["user_confirmed"] = *userConfirmed
	}
	if meta.ExecutionBackend != "" {
		updates["execution_backend"] = meta.ExecutionBackend
	}
	if meta.SandboxJobName != "" {
		updates["sandbox_job_name"] = meta.SandboxJobName
	}
	if meta.ScriptName != "" {
		updates["script_name"] = meta.ScriptName
	}
	if meta.ResultArtifactRef != "" {
		updates["result_artifact_ref"] = meta.ResultArtifactRef
	}
	if len(meta.EgressDomains) > 0 {
		if payload, err := json.Marshal(meta.EgressDomains); err == nil {
			updates["egress_domains"] = datatypes.JSON(payload)
		}
	}
	db := s.db.WithContext(ctx).
		Model(&model.AgentToolCall{}).
		Where("id = ?", id)
	err := db.Updates(updates).Error
	if err == nil || !isMissingAgentToolAuditColumnError(err) {
		return err
	}
	return db.Updates(stripUnsupportedAgentToolAuditUpdates(updates)).Error
}

func (s *AgentService) UpdateToolCallArgs(ctx context.Context, id uint, toolArgs json.RawMessage) error {
	return s.db.WithContext(ctx).
		Model(&model.AgentToolCall{}).
		Where("id = ?", id).
		Update("tool_args", datatypes.JSON(toolArgs)).Error
}

func (s *AgentService) GetToolCallByToolCallID(ctx context.Context, toolCallID string) (*model.AgentToolCall, error) {
	var toolCall model.AgentToolCall
	if err := s.db.WithContext(ctx).Where("tool_call_id = ?", toolCallID).First(&toolCall).Error; err != nil {
		return nil, err
	}
	return &toolCall, nil
}

func (s *AgentService) CreateTurn(ctx context.Context, turn *model.AgentTurn) (*model.AgentTurn, error) {
	if err := s.db.WithContext(ctx).Create(turn).Error; err != nil {
		return nil, err
	}
	return turn, nil
}

func (s *AgentService) GetTurn(ctx context.Context, turnID string) (*model.AgentTurn, error) {
	var turn model.AgentTurn
	if err := s.db.WithContext(ctx).Where("turn_id = ?", turnID).First(&turn).Error; err != nil {
		return nil, err
	}
	return &turn, nil
}

func (s *AgentService) ListTurns(ctx context.Context, sessionID string) ([]*model.AgentTurn, error) {
	var turns []*model.AgentTurn
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("started_at DESC").
		Find(&turns).Error; err != nil {
		return nil, err
	}
	return turns, nil
}

func (s *AgentService) UpdateTurnStatus(
	ctx context.Context,
	turnID string,
	status string,
	finalMessageID *uint,
	metadata json.RawMessage,
) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	now := time.Now()
	if status == "completed" || status == "failed" || status == "cancelled" {
		updates["ended_at"] = &now
	}
	if finalMessageID != nil {
		updates["final_message_id"] = *finalMessageID
	}
	if metadata != nil {
		updates["metadata"] = datatypes.JSON(metadata)
	}
	return s.db.WithContext(ctx).
		Model(&model.AgentTurn{}).
		Where("turn_id = ?", turnID).
		Updates(updates).Error
}

func (s *AgentService) CreateRunEvent(ctx context.Context, event *model.AgentRunEvent) (*model.AgentRunEvent, error) {
	if event.Sequence == 0 {
		nextSequence, err := s.NextRunEventSequence(ctx, event.TurnID)
		if err != nil {
			return nil, err
		}
		event.Sequence = nextSequence
	}
	if err := s.db.WithContext(ctx).Create(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (s *AgentService) NextRunEventSequence(ctx context.Context, turnID string) (int, error) {
	var maxSequence int
	row := s.db.WithContext(ctx).
		Model(&model.AgentRunEvent{}).
		Where("turn_id = ?", turnID).
		Select("COALESCE(MAX(sequence), 0)").
		Row()
	if err := row.Scan(&maxSequence); err != nil {
		return 0, err
	}
	return maxSequence + 1, nil
}

func (s *AgentService) ListRunEvents(ctx context.Context, turnID string) ([]*model.AgentRunEvent, error) {
	var events []*model.AgentRunEvent
	if err := s.db.WithContext(ctx).
		Where("turn_id = ?", turnID).
		Order("sequence ASC, created_at ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// LogToolCallAsync records a tool execution asynchronously to avoid blocking the caller.
func (s *AgentService) LogToolCallAsync(
	sessionID,
	toolName string,
	toolArgs,
	toolResult json.RawMessage,
	resultStatus string,
	latencyMs int,
	turnID,
	toolCallID,
	agentID,
	agentRole string,
) {
	go func() {
		meta := parseToolCallAuditMeta(toolArgs, toolResult)
		tc := &model.AgentToolCall{
			SessionID:    sessionID,
			TurnID:       turnID,
			ToolCallID:   toolCallID,
			AgentID:      agentID,
			AgentRole:    agentRole,
			ToolName:     toolName,
			ToolArgs:     datatypes.JSON(toolArgs),
			ToolResult:   datatypes.JSON(toolResult),
			ResultStatus: resultStatus,
			LatencyMs:    latencyMs,
			CreatedAt:    time.Now(),
		}
		if meta.ExecutionBackend != "" {
			tc.ExecutionBackend = meta.ExecutionBackend
		}
		if meta.SandboxJobName != "" {
			tc.SandboxJobName = meta.SandboxJobName
		}
		if meta.ScriptName != "" {
			tc.ScriptName = meta.ScriptName
		}
		if meta.ResultArtifactRef != "" {
			tc.ResultArtifactRef = meta.ResultArtifactRef
		}
		if len(meta.EgressDomains) > 0 {
			if payload, err := json.Marshal(meta.EgressDomains); err == nil {
				tc.EgressDomains = datatypes.JSON(payload)
			}
		}
		if err := s.createToolCallWithCompat(context.Background(), tc); err != nil {
			klog.Warningf("[AgentService] Failed to log tool call %s for session %s: %v", toolName, sessionID, err)
		}
	}()
}
