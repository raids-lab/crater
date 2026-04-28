package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

type toolCallAuditMetadata struct {
	ExecutionBackend string
}

var agentToolAuditCompatFields = []string{
	"ExecutionBackend",
}

var agentToolAuditCompatColumns = []string{
	"execution_backend",
}

func parseToolCallAuditMeta(_ json.RawMessage, toolResult json.RawMessage) toolCallAuditMetadata {
	meta := toolCallAuditMetadata{}

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
	}
	readMeta(resultMap)
	if nested, ok := resultMap["_audit"].(map[string]any); ok {
		readMeta(nested)
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
func (s *AgentService) CreateSession(ctx context.Context, sessionID string, userID, accountID uint, title string, pageContext json.RawMessage, source string) (*model.AgentSession, error) {
	// Title is varchar(255); truncate long messages to fit.
	if runeTitle := []rune(title); len(runeTitle) > 100 {
		title = string(runeTitle[:100]) + "..."
	}
	source = normalizeAgentSessionSource(source)
	session := &model.AgentSession{
		SessionID: sessionID,
		UserID:    userID,
		AccountID: accountID,
		Title:     title,
		Source:    source,
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
	created, createErr := s.CreateSession(ctx, sessionID, userID, accountID, title, pageContext, "chat")
	if createErr != nil {
		return nil, false, createErr
	}
	return created, true, nil
}

// ListSessions returns all chat sessions for a given user, ordered by most recent.
// Only returns source='chat' sessions; ops_audit/system/benchmark sessions are excluded from the UI.
func (s *AgentService) ListSessions(ctx context.Context, userID uint) ([]*model.AgentSession, error) {
	var sessions []*model.AgentSession
	query := s.db.WithContext(ctx).Where("user_id = ?", userID)
	// Exclude non-chat sessions from the UI listing.
	if s.db.WithContext(ctx).Migrator().HasColumn(&model.AgentSession{}, "Source") {
		query = query.Where("source = ? OR source IS NULL OR source = ''", "chat")
	}
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
	if runeTitle := []rune(title); len(runeTitle) > 100 {
		title = string(runeTitle[:100]) + "..."
	}
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
	updateWithFields := func(fields map[string]any) error {
		return s.db.WithContext(ctx).
			Model(&model.AgentToolCall{}).
			Where("id = ?", id).
			Updates(fields).Error
	}
	err := updateWithFields(updates)
	if err == nil || !isMissingAgentToolAuditColumnError(err) {
		return err
	}
	return updateWithFields(stripUnsupportedAgentToolAuditUpdates(updates))
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
	agentRole,
	source string,
) {
	go func() {
		meta := parseToolCallAuditMeta(toolArgs, toolResult)
		tc := &model.AgentToolCall{
			SessionID:    sessionID,
			TurnID:       turnID,
			ToolCallID:   toolCallID,
			AgentID:      agentID,
			AgentRole:    agentRole,
			Source:       normalizeAgentToolCallSource(source),
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
		if err := s.createToolCallWithCompat(context.Background(), tc); err != nil {
			klog.Warningf("[AgentService] Failed to log tool call %s for session %s: %v", toolName, sessionID, err)
		}
	}()
}

// ── Feedback ────────────────────────────────────────────────────────────────

var ErrFeedbackAlreadySubmitted = errors.New("feedback already submitted and cannot be modified")

// UpsertFeedback creates or updates a draft feedback.
// Returns the feedback record and a boolean indicating if it was newly created.
func (s *AgentService) UpsertFeedback(ctx context.Context, fb *model.AgentFeedback) (*model.AgentFeedback, bool, error) {
	var existing model.AgentFeedback
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", fb.UserID, fb.TargetType, fb.TargetID).
		First(&existing).Error

	if err == nil {
		// Record exists — check immutability
		if existing.Status == "submitted" {
			return nil, false, ErrFeedbackAlreadySubmitted
		}
		updates := map[string]any{
			"rating":     fb.Rating,
			"tags":       fb.Tags,
			"dimensions": fb.Dimensions,
			"comment":    fb.Comment,
			"updated_at": time.Now(),
		}
		if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, false, err
		}
		existing.Rating = fb.Rating
		existing.Tags = fb.Tags
		existing.Dimensions = fb.Dimensions
		existing.Comment = fb.Comment
		return &existing, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	// New record
	fb.Status = "draft"
	if err := s.db.WithContext(ctx).Create(fb).Error; err != nil {
		return nil, false, err
	}
	return fb, true, nil
}

// SubmitFeedback transitions a draft feedback to submitted (immutable).
func (s *AgentService) SubmitFeedback(ctx context.Context, userID uint, targetType, targetID string) (*model.AgentFeedback, error) {
	var fb model.AgentFeedback
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		First(&fb).Error
	if err != nil {
		return nil, err
	}
	if fb.Status == "submitted" {
		return nil, ErrFeedbackAlreadySubmitted
	}
	now := time.Now()
	updates := map[string]any{
		"status":       "submitted",
		"submitted_at": &now,
		"updated_at":   now,
	}
	if err := s.db.WithContext(ctx).Model(&fb).Updates(updates).Error; err != nil {
		return nil, err
	}
	fb.Status = "submitted"
	fb.SubmittedAt = &now
	return &fb, nil
}

// ListFeedbacks returns all feedbacks for a user in a given session.
func (s *AgentService) ListFeedbacks(ctx context.Context, sessionID string, userID uint) ([]*model.AgentFeedback, error) {
	var feedbacks []*model.AgentFeedback
	if err := s.db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Order("created_at ASC").
		Find(&feedbacks).Error; err != nil {
		return nil, err
	}
	return feedbacks, nil
}

// FeedbackStats holds aggregated feedback statistics.
type FeedbackStats struct {
	Total         int64              `json:"total"`
	ThumbsUp      int64              `json:"thumbsUp"`
	ThumbsDown    int64              `json:"thumbsDown"`
	AvgDimensions map[string]float64 `json:"avgDimensions"`
	TopTags       []TagCount         `json:"topTags"`
}

// TagCount represents a tag and its occurrence count.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

// GetFeedbackStats returns aggregated feedback statistics for admins.
func (s *AgentService) GetFeedbackStats(ctx context.Context, from, to *time.Time) (*FeedbackStats, error) {
	db := s.db.WithContext(ctx).Model(&model.AgentFeedback{}).Where("status = ?", "submitted")
	if from != nil {
		db = db.Where("submitted_at >= ?", *from)
	}
	if to != nil {
		db = db.Where("submitted_at <= ?", *to)
	}

	stats := &FeedbackStats{AvgDimensions: map[string]float64{}}

	// Total / thumbs up / thumbs down
	db.Count(&stats.Total)
	db.Where("rating = 1").Count(&stats.ThumbsUp)
	// Re-scope for thumbs down
	db2 := s.db.WithContext(ctx).Model(&model.AgentFeedback{}).Where("status = ?", "submitted")
	if from != nil {
		db2 = db2.Where("submitted_at >= ?", *from)
	}
	if to != nil {
		db2 = db2.Where("submitted_at <= ?", *to)
	}
	db2.Where("rating = -1").Count(&stats.ThumbsDown)

	// Average dimensions via raw SQL for JSONB
	type dimAvg struct {
		Key string  `json:"key"`
		Avg float64 `json:"avg"`
	}
	var dimAvgs []dimAvg
	rawQuery := `
		SELECT kv.key, AVG((kv.value)::numeric) AS avg
		FROM agent_feedbacks, jsonb_each_text(dimensions) AS kv
		WHERE status = 'submitted' AND dimensions IS NOT NULL`
	args := []any{}
	if from != nil {
		rawQuery += " AND submitted_at >= ?"
		args = append(args, *from)
	}
	if to != nil {
		rawQuery += " AND submitted_at <= ?"
		args = append(args, *to)
	}
	rawQuery += " GROUP BY kv.key"
	s.db.WithContext(ctx).Raw(rawQuery, args...).Scan(&dimAvgs)
	for _, da := range dimAvgs {
		stats.AvgDimensions[da.Key] = da.Avg
	}

	// Top tags
	var tagCounts []TagCount
	tagQuery := `
		SELECT tag AS tag, COUNT(*) AS count
		FROM agent_feedbacks, jsonb_array_elements_text(tags) AS tag
		WHERE status = 'submitted' AND tags IS NOT NULL`
	tagArgs := []any{}
	if from != nil {
		tagQuery += " AND submitted_at >= ?"
		tagArgs = append(tagArgs, *from)
	}
	if to != nil {
		tagQuery += " AND submitted_at <= ?"
		tagArgs = append(tagArgs, *to)
	}
	tagQuery += " GROUP BY tag ORDER BY count DESC LIMIT 20"
	s.db.WithContext(ctx).Raw(tagQuery, tagArgs...).Scan(&tagCounts)
	stats.TopTags = tagCounts

	return stats, nil
}

// QuickSubmitFeedback creates a feedback record and immediately submits it (single operation).
// If a record already exists and is already submitted, returns the existing record (idempotent).
func (s *AgentService) QuickSubmitFeedback(ctx context.Context, fb *model.AgentFeedback) (*model.AgentFeedback, error) {
	var existing model.AgentFeedback
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", fb.UserID, fb.TargetType, fb.TargetID).
		First(&existing).Error

	now := time.Now()

	if err == nil {
		// Already exists — if submitted, idempotent return
		if existing.Status == "submitted" {
			return &existing, nil
		}
		// Update rating and submit
		updates := map[string]any{
			"rating":       fb.Rating,
			"status":       "submitted",
			"submitted_at": &now,
			"updated_at":   now,
		}
		if len(fb.Tags) > 0 {
			updates["tags"] = fb.Tags
		}
		if len(fb.Dimensions) > 0 {
			updates["dimensions"] = fb.Dimensions
		}
		if fb.Comment != "" {
			updates["comment"] = fb.Comment
		}
		if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}
		existing.Rating = fb.Rating
		existing.Status = "submitted"
		existing.SubmittedAt = &now
		return &existing, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Create and immediately submit
	fb.Status = "submitted"
	fb.SubmittedAt = &now
	if err := s.db.WithContext(ctx).Create(fb).Error; err != nil {
		return nil, err
	}
	return fb, nil
}

// EnrichFeedback updates the optional detail fields (tags, dimensions, comment) on any feedback
// record regardless of status. Never changes rating or status. Used for post-submit detail addition.
func (s *AgentService) EnrichFeedback(ctx context.Context, userID uint, targetType, targetID string, tags, dimensions datatypes.JSON, comment string) (*model.AgentFeedback, error) {
	var fb model.AgentFeedback
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		First(&fb).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	updates := map[string]any{
		"enriched_at": &now,
		"updated_at":  now,
	}
	if len(tags) > 0 {
		updates["tags"] = tags
	}
	if len(dimensions) > 0 {
		updates["dimensions"] = dimensions
	}
	if comment != "" {
		updates["comment"] = comment
	}

	if err := s.db.WithContext(ctx).Model(&fb).Updates(updates).Error; err != nil {
		return nil, err
	}
	fb.EnrichedAt = &now
	return &fb, nil
}

// CreateQualityEval inserts a new quality eval record with status=pending.
// TurnID is a uuid column that rejects empty string literals, so we omit it
// from the INSERT when the caller did not supply one (session-level eval).
func (s *AgentService) CreateQualityEval(ctx context.Context, eval *model.AgentQualityEval) error {
	eval.EvalStatus = "pending"
	eval.EvalScope = normalizeAgentQualityEvalScope(eval.EvalScope, eval.TurnID)
	eval.EvalType = normalizeAgentQualityEvalType(eval.EvalType)
	if strings.TrimSpace(eval.TargetID) == "" {
		if eval.EvalScope == AgentQualityEvalScopeTurn && strings.TrimSpace(eval.TurnID) != "" {
			eval.TargetID = eval.TurnID
		} else {
			eval.TargetID = eval.SessionID
		}
	}
	tx := s.db.WithContext(ctx)
	if eval.TurnID == "" {
		return tx.Omit("TurnID").Create(eval).Error
	}
	return tx.Create(eval).Error
}

func (s *AgentService) SetQualityEvalStatus(ctx context.Context, id uint, status string, summary string) error {
	now := time.Now()
	updates := map[string]any{
		"eval_status": status,
		"updated_at":  now,
	}
	if summary != "" {
		updates["summary"] = summary
	}
	if status == "completed" || status == "failed" {
		updates["completed_at"] = &now
	}
	return s.db.WithContext(ctx).Model(&model.AgentQualityEval{}).Where("id = ?", id).Updates(updates).Error
}

func (s *AgentService) FailQualityEval(ctx context.Context, id uint, summary string) error {
	return s.SetQualityEvalStatus(ctx, id, "failed", summary)
}

// UpdateQualityEvalResult updates a quality eval record with completed results from crater-agent.
func (s *AgentService) UpdateQualityEvalResult(ctx context.Context, id uint, chatScores, chainScores datatypes.JSON, chatModel, chainModel, summary string, rawChat, rawChain datatypes.JSON, artifactPath string, metadata datatypes.JSON) error {
	now := time.Now()
	updates := map[string]any{
		"eval_status":    "completed",
		"chat_scores":    chatScores,
		"chain_scores":   chainScores,
		"chat_model":     chatModel,
		"chain_model":    chainModel,
		"summary":        summary,
		"raw_chat_resp":  rawChat,
		"raw_chain_resp": rawChain,
		"artifact_path":  artifactPath,
		"completed_at":   &now,
		"updated_at":     now,
	}
	if len(metadata) > 0 && string(metadata) != "null" {
		updates["metadata"] = metadata
	}
	return s.db.WithContext(ctx).Model(&model.AgentQualityEval{}).Where("id = ?", id).Updates(updates).Error
}

// ListQualityEvals returns quality eval records, optionally filtered by session_id or trigger_source.
// Results are ordered by created_at DESC. Limit defaults to 50.
func (s *AgentService) ListQualityEvals(ctx context.Context, sessionID, triggerSource string, limit int) ([]model.AgentQualityEval, error) {
	if limit <= 0 {
		limit = 50
	}
	q := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if sessionID != "" {
		q = q.Where("session_id = ?", sessionID)
	}
	if triggerSource != "" {
		q = q.Where("trigger_source = ?", triggerSource)
	}
	var evals []model.AgentQualityEval
	if err := q.Find(&evals).Error; err != nil {
		return nil, err
	}
	return evals, nil
}
