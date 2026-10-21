package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

const (
	agentDefaultConversationTitle = ""
	agentMessageSchemaVersion     = 1
	agentMaximumTitleRunes        = 80
	agentMaximumMessageIDLength   = 128
	agentMaximumErrorLength       = 2048

	AgentRunStatusRunning   = "running"
	AgentRunStatusCompleted = "completed"
	AgentRunStatusCanceled  = "canceled"
	AgentRunStatusFailed    = "failed"

	AgentRunModeAppend  = "append"
	AgentRunModeReplace = "replace"
)

type AgentConversationService struct {
	db *gorm.DB
}

type AgentConversationSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	CustomTitle bool   `json:"customTitle"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type AdminAgentConversationSummary struct {
	AgentConversationSummary
	UserID        uint          `json:"userId"`
	Username      string        `json:"username"`
	Nickname      string        `json:"nickname"`
	IdentityScope AgentIdentity `json:"identityScope"`
	MessageCount  int64         `json:"messageCount"`
}

type AdminAgentConversationQuery struct {
	Page          int
	PageSize      int
	Search        string
	IdentityScope AgentIdentity
	UserID        uint
}

type AdminAgentConversationPage struct {
	Items    []AdminAgentConversationSummary `json:"items"`
	Total    int64                           `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"pageSize"`
}

type AdminAgentSubjectSummary struct {
	UserID            uint          `json:"userId"`
	Username          string        `json:"username"`
	Nickname          string        `json:"nickname"`
	IdentityScope     AgentIdentity `json:"identityScope"`
	ConversationCount int64         `json:"conversationCount"`
	MessageCount      int64         `json:"messageCount"`
	UpdatedAt         int64         `json:"updatedAt"`
}

type AdminAgentSubjectQuery struct {
	Page          int
	PageSize      int
	Search        string
	IdentityScope AgentIdentity
}

type AdminAgentSubjectPage struct {
	Items    []AdminAgentSubjectSummary `json:"items"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}

// AgentUIMessage is the versioned wire representation stored by Crater. Parts
// intentionally remain structured JSON so future AI SDK tool and file parts do
// not require a message-table migration.
type AgentUIMessage struct {
	ID       string          `json:"id"`
	Role     string          `json:"role"`
	Parts    json.RawMessage `json:"parts"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type PreparedAgentRun struct {
	RunID         string            `json:"runId"`
	Messages      []AgentUIMessage  `json:"messages"`
	IdentityScope AgentIdentity     `json:"identityScope"`
	PageContext   *AgentPageContext `json:"pageContext,omitempty"`
}

type AgentRunCompletion struct {
	Status       string          `json:"status"`
	Message      *AgentUIMessage `json:"message,omitempty"`
	ErrorMessage string          `json:"error,omitempty"`
}

func NewAgentConversationService(db *gorm.DB) *AgentConversationService {
	return &AgentConversationService{db: db}
}

func normalizeAdminAgentPage(page, pageSize int) (normalizedPage, normalizedPageSize int) {
	const (
		defaultPageSize = 20
		maximumPageSize = 100
	)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	} else if pageSize > maximumPageSize {
		pageSize = maximumPageSize
	}
	return page, pageSize
}

func (s *AgentConversationService) ListAdminSubjects(
	ctx context.Context,
	options AdminAgentSubjectQuery,
) (*AdminAgentSubjectPage, error) {
	options.Page, options.PageSize = normalizeAdminAgentPage(options.Page, options.PageSize)
	if options.IdentityScope != "" {
		if _, err := ParseAgentIdentity(string(options.IdentityScope)); err != nil {
			return nil, err
		}
	}

	baseQuery := func() *gorm.DB {
		query := s.db.WithContext(ctx).
			Table("agent_conversations AS conversation").
			Joins("JOIN users ON users.id = conversation.user_id").
			Where("conversation.deleted_at IS NULL")
		if search := strings.TrimSpace(options.Search); search != "" {
			pattern := "%" + search + "%"
			query = query.Where(
				`LOWER(users.name) LIKE LOWER(?) OR LOWER(users.nickname) LIKE LOWER(?)
					OR CAST(conversation.user_id AS TEXT) LIKE ?`,
				pattern, pattern, pattern,
			)
		}
		if options.IdentityScope != "" {
			query = query.Where("conversation.identity_scope = ?", options.IdentityScope)
		}
		return query
	}

	groupedSubjects := baseQuery().
		Select("conversation.user_id, conversation.identity_scope").
		Group("conversation.user_id, conversation.identity_scope")
	var total int64
	if err := s.db.WithContext(ctx).Table("(?) AS subject", groupedSubjects).Count(&total).Error; err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to count Agent users for administrators")
	}

	type adminSubjectRecord struct {
		UserID            uint
		Username          string
		Nickname          string
		IdentityScope     string
		ConversationCount int64
		MessageCount      int64
		UpdatedAt         time.Time
	}
	var records []adminSubjectRecord
	err := baseQuery().
		Select(`conversation.user_id AS user_id, users.name AS username, users.nickname AS nickname,
			conversation.identity_scope AS identity_scope,
			COUNT(DISTINCT conversation.id) AS conversation_count,
			COUNT(message.id) AS message_count, MAX(conversation.updated_at) AS updated_at`).
		Joins(`LEFT JOIN agent_messages AS message ON message.conversation_id = conversation.id
			AND message.superseded_at IS NULL`).
		Group("conversation.user_id, conversation.identity_scope, users.id").
		Order("MAX(conversation.updated_at) DESC").
		Order("conversation.user_id ASC").
		Order("conversation.identity_scope ASC").
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Scan(&records).Error
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to list Agent users for administrators")
	}

	items := make([]AdminAgentSubjectSummary, 0, len(records))
	for index := range records {
		record := &records[index]
		items = append(items, AdminAgentSubjectSummary{
			UserID:            record.UserID,
			Username:          record.Username,
			Nickname:          record.Nickname,
			IdentityScope:     AgentIdentity(record.IdentityScope),
			ConversationCount: record.ConversationCount,
			MessageCount:      record.MessageCount,
			UpdatedAt:         record.UpdatedAt.UnixMilli(),
		})
	}
	return &AdminAgentSubjectPage{
		Items: items, Total: total, Page: options.Page, PageSize: options.PageSize,
	}, nil
}

func (s *AgentConversationService) ListConversations(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
) ([]AgentConversationSummary, error) {
	var records []model.AgentConversation
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND identity_scope = ?", userID, identity).
		Order("updated_at DESC").
		Find(&records).Error; err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to list Agent conversations")
	}
	items := make([]AgentConversationSummary, 0, len(records))
	for index := range records {
		items = append(items, conversationSummary(&records[index]))
	}
	return items, nil
}

func (s *AgentConversationService) ListAllConversations(
	ctx context.Context,
	options AdminAgentConversationQuery,
) (*AdminAgentConversationPage, error) {
	options.Page, options.PageSize = normalizeAdminAgentPage(options.Page, options.PageSize)
	if options.IdentityScope != "" {
		if _, err := ParseAgentIdentity(string(options.IdentityScope)); err != nil {
			return nil, err
		}
	}

	type adminConversationRecord struct {
		model.AgentConversation
		Username     string
		Nickname     string
		MessageCount int64
	}
	baseQuery := func() *gorm.DB {
		query := s.db.WithContext(ctx).
			Table("agent_conversations AS conversation").
			Joins("JOIN users ON users.id = conversation.user_id").
			Where("conversation.deleted_at IS NULL")
		if search := strings.TrimSpace(options.Search); search != "" {
			pattern := "%" + search + "%"
			query = query.Where(
				`LOWER(conversation.title) LIKE LOWER(?) OR LOWER(users.name) LIKE LOWER(?)
					OR LOWER(users.nickname) LIKE LOWER(?) OR CAST(conversation.user_id AS TEXT) LIKE ?`,
				pattern, pattern, pattern, pattern,
			)
		}
		if options.IdentityScope != "" {
			query = query.Where("conversation.identity_scope = ?", options.IdentityScope)
		}
		if options.UserID != 0 {
			query = query.Where("conversation.user_id = ?", options.UserID)
		}
		return query
	}

	var total int64
	if err := baseQuery().Distinct("conversation.id").Count(&total).Error; err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to count Agent conversations for administrators")
	}

	var records []adminConversationRecord
	err := baseQuery().
		Select(`conversation.*, users.name AS username, users.nickname AS nickname,
			COUNT(message.id) AS message_count`).
		Joins(`LEFT JOIN agent_messages AS message ON message.conversation_id = conversation.id
			AND message.superseded_at IS NULL`).
		Group("conversation.id, users.id").
		Order("conversation.updated_at DESC").
		Order("conversation.id ASC").
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		Scan(&records).Error
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to list Agent conversations for administrators")
	}
	items := make([]AdminAgentConversationSummary, 0, len(records))
	for index := range records {
		record := &records[index]
		items = append(items, AdminAgentConversationSummary{
			AgentConversationSummary: conversationSummary(&record.AgentConversation),
			UserID:                   record.UserID,
			Username:                 record.Username,
			Nickname:                 record.Nickname,
			IdentityScope:            AgentIdentity(record.IdentityScope),
			MessageCount:             record.MessageCount,
		})
	}
	return &AdminAgentConversationPage{
		Items:    items,
		Total:    total,
		Page:     options.Page,
		PageSize: options.PageSize,
	}, nil
}

func (s *AgentConversationService) GetMessages(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	conversationID string,
) ([]AgentUIMessage, error) {
	if err := validateConversationID(conversationID); err != nil {
		return nil, err
	}
	if err := s.ensureOwnedConversation(ctx, s.db, userID, identity, conversationID); err != nil {
		return nil, err
	}
	records, err := activeAgentMessages(ctx, s.db, conversationID)
	if err != nil {
		return nil, err
	}
	return messageRecordsToDTO(records), nil
}

func (s *AgentConversationService) GetMessagesForAdmin(
	ctx context.Context,
	conversationID string,
) ([]AgentUIMessage, error) {
	if err := validateConversationID(conversationID); err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.AgentConversation{}).
		Where("id = ?", conversationID).Count(&count).Error; err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to load Agent conversation")
	}
	if count == 0 {
		return nil, agentConversationNotFound()
	}
	records, err := activeAgentMessages(ctx, s.db, conversationID)
	if err != nil {
		return nil, err
	}
	return messageRecordsToDTO(records), nil
}

func (s *AgentConversationService) RenameConversation(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	conversationID string,
	title string,
) error {
	if err := validateConversationID(conversationID); err != nil {
		return err
	}
	title, err := validateConversationTitle(title)
	if err != nil {
		return err
	}
	result := s.db.WithContext(ctx).
		Model(&model.AgentConversation{}).
		Where("id = ? AND user_id = ? AND identity_scope = ?", conversationID, userID, identity).
		Updates(map[string]any{"title": title, "custom_title": true, "updated_at": time.Now()})
	if result.Error != nil {
		return bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to rename Agent conversation")
	}
	if result.RowsAffected == 0 {
		return agentConversationNotFound()
	}
	return nil
}

func (s *AgentConversationService) UpdateGeneratedTitle(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	conversationID string,
	title string,
) error {
	if err := validateConversationID(conversationID); err != nil {
		return err
	}
	title, err := validateConversationTitle(title)
	if err != nil {
		return err
	}
	result := s.db.WithContext(ctx).
		Model(&model.AgentConversation{}).
		Where(
			"id = ? AND user_id = ? AND identity_scope = ? AND custom_title = ?",
			conversationID, userID, identity, false,
		).
		Updates(map[string]any{"title": title, "updated_at": time.Now()})
	if result.Error != nil {
		return bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to update Agent conversation title")
	}
	if result.RowsAffected == 0 {
		if findErr := s.ensureOwnedConversation(ctx, s.db, userID, identity, conversationID); findErr != nil {
			return findErr
		}
	}
	return nil
}

func (s *AgentConversationService) DeleteConversation(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	conversationID string,
) error {
	if err := validateConversationID(conversationID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where(
			"id = ? AND user_id = ? AND identity_scope = ?", conversationID, userID, identity,
		).
			Delete(&model.AgentConversation{})
		if result.Error != nil {
			return bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to delete Agent conversation")
		}
		if result.RowsAffected == 0 {
			return agentConversationNotFound()
		}
		now := time.Now()
		if err := tx.Model(&model.AgentRun{}).
			Where(
				"conversation_id = ? AND user_id = ? AND identity_scope = ? AND status = ?",
				conversationID, userID, identity, AgentRunStatusRunning,
			).
			Updates(map[string]any{"status": AgentRunStatusCanceled, "finished_at": now}).Error; err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "failed to cancel deleted Agent conversation runs")
		}
		return nil
	})
}

func (s *AgentConversationService) CancelActiveRun(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	conversationID string,
) (bool, error) {
	if err := validateConversationID(conversationID); err != nil {
		return false, err
	}
	canceled := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.ensureOwnedConversation(ctx, tx, userID, identity, conversationID); err != nil {
			return err
		}
		now := time.Now()
		result := tx.Model(&model.AgentRun{}).
			Where(
				"conversation_id = ? AND user_id = ? AND identity_scope = ? AND status = ?",
				conversationID, userID, identity, AgentRunStatusRunning,
			).
			Updates(map[string]any{
				"status":        AgentRunStatusCanceled,
				"finished_at":   now,
				"error_message": "Agent response canceled by user",
			})
		if result.Error != nil {
			return bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to cancel Agent response")
		}
		canceled = result.RowsAffected > 0
		return nil
	})
	return canceled, err
}

func (s *AgentConversationService) PrepareRun(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	conversationID string,
	parentMessageID *string,
	mode string,
	candidate *AgentUIMessage,
) (*PreparedAgentRun, error) {
	if err := validateConversationID(conversationID); err != nil {
		return nil, err
	}
	if mode != AgentRunModeAppend && mode != AgentRunModeReplace {
		return nil, bizerr.BadRequest.ParameterError.New("Agent run mode is invalid")
	}
	if candidate == nil {
		return nil, bizerr.BadRequest.MissingParameter.New("message is required")
	}
	if err := validateAgentUIMessage(candidate, "user"); err != nil {
		return nil, err
	}

	prepared := &PreparedAgentRun{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return prepareAgentRunTransaction(
			ctx, tx, userID, identity, conversationID, parentMessageID, mode, candidate, prepared,
		)
	})
	if err != nil {
		return nil, err
	}
	return prepared, nil
}

func (s *AgentConversationService) CompleteRun(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	runID string,
	completion AgentRunCompletion,
) error {
	if _, err := uuid.Parse(runID); err != nil {
		return bizerr.BadRequest.ParameterError.New("run id is invalid")
	}
	if !isTerminalRunStatus(completion.Status) {
		return bizerr.BadRequest.ParameterError.New("run status is invalid")
	}
	if completion.Message != nil && hasAgentMessageParts(completion.Message.Parts) {
		if err := validateAgentUIMessage(completion.Message, "assistant"); err != nil {
			return err
		}
	} else {
		completion.Message = nil
	}
	if completion.Status == AgentRunStatusCompleted && completion.Message == nil {
		return bizerr.BadRequest.MissingParameter.New("completed Agent run requires an assistant message")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run model.AgentRun
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ? AND identity_scope = ?", runID, userID, identity).
			First(&run).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerr.NotFound.DataBaseNotFound.New("Agent run not found")
		}
		if err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "failed to load Agent run")
		}
		if isTerminalRunStatus(run.Status) {
			return nil
		}

		if completion.Message != nil && len(bytes.TrimSpace(completion.Message.Parts)) > 0 {
			if err := saveAssistantMessage(ctx, tx, &run, completion.Message, completion.Status); err != nil {
				return err
			}
		}
		now := time.Now()
		updates := map[string]any{
			"status":        completion.Status,
			"finished_at":   now,
			"error_message": truncateString(completion.ErrorMessage, agentMaximumErrorLength),
		}
		if err := tx.Model(&run).Updates(updates).Error; err != nil {
			return bizerr.Internal.DatabaseError.Wrap(err, "failed to complete Agent run")
		}
		return nil
	})
}

func (s *AgentConversationService) FailRun(
	ctx context.Context,
	userID uint,
	identity AgentIdentity,
	runID string,
	runErr error,
) error {
	message := ""
	if runErr != nil {
		message = runErr.Error()
	}
	return s.CompleteRun(ctx, userID, identity, runID, AgentRunCompletion{
		Status:       AgentRunStatusFailed,
		ErrorMessage: message,
	})
}

const agentMaximumContextMessages = 100

func prepareAgentRunTransaction(
	ctx context.Context,
	tx *gorm.DB,
	userID uint,
	identity AgentIdentity,
	conversationID string,
	parentMessageID *string,
	mode string,
	candidate *AgentUIMessage,
	prepared *PreparedAgentRun,
) error {
	conversation, err := findOrCreateAgentConversation(ctx, tx, userID, identity, conversationID)
	if err != nil {
		return err
	}
	if err := ensureNoRunningAgentRun(ctx, tx, conversationID); err != nil {
		return err
	}
	active, err := activeAgentMessages(ctx, tx, conversationID)
	if err != nil {
		return err
	}
	prefixLength, err := resolveAgentMessagePrefix(active, parentMessageID, mode)
	if err != nil {
		return err
	}
	if err := supersedeMessages(ctx, tx, active[prefixLength:]); err != nil {
		return err
	}
	if err := ensureMessageIDAvailable(ctx, tx, candidate.ID); err != nil {
		return err
	}

	sequence, err := nextMessageSequence(ctx, tx, conversationID)
	if err != nil {
		return err
	}
	var parentID *string
	if prefixLength > 0 {
		parent := active[prefixLength-1].ID
		parentID = &parent
	}
	now := time.Now()
	userMessage := model.AgentMessage{
		ID:              candidate.ID,
		ConversationID:  conversationID,
		ParentMessageID: parentID,
		Sequence:        sequence,
		Role:            "user",
		Status:          AgentRunStatusCompleted,
		SchemaVersion:   agentMessageSchemaVersion,
		Parts:           datatypes.JSON(cloneJSON(candidate.Parts)),
		Metadata:        datatypes.JSON(cloneJSON(candidate.Metadata)),
	}
	if err := tx.Create(&userMessage).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to save Agent user message")
	}

	runID := uuid.NewString()
	run := model.AgentRun{
		ID:               runID,
		ConversationID:   conversationID,
		UserID:           userID,
		IdentityScope:    string(identity),
		TriggerMessageID: candidate.ID,
		Status:           AgentRunStatusRunning,
		StartedAt:        now,
	}
	if err := tx.Create(&run).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to create Agent run")
	}
	if err := tx.Model(conversation).Update("updated_at", now).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to update Agent conversation")
	}

	contextRecords := append(append([]model.AgentMessage{}, active[:prefixLength]...), userMessage)
	if len(contextRecords) > agentMaximumContextMessages {
		contextRecords = contextRecords[len(contextRecords)-agentMaximumContextMessages:]
	}
	prepared.RunID = runID
	prepared.Messages = messageRecordsToDTO(contextRecords)
	return nil
}

func findOrCreateAgentConversation(
	ctx context.Context,
	tx *gorm.DB,
	userID uint,
	identity AgentIdentity,
	conversationID string,
) (*model.AgentConversation, error) {
	var conversation model.AgentConversation
	result := tx.WithContext(ctx).Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", conversationID).Limit(1).Find(&conversation)
	if result.Error != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to load Agent conversation")
	}
	if result.RowsAffected > 0 {
		if conversation.UserID != userID || conversation.IdentityScope != string(identity) || conversation.DeletedAt.Valid {
			return nil, agentConversationNotFound()
		}
		return &conversation, nil
	}
	conversation = model.AgentConversation{
		ID:            conversationID,
		UserID:        userID,
		IdentityScope: string(identity),
		Title:         agentDefaultConversationTitle,
	}
	result = tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&conversation)
	if result.Error != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to create Agent conversation")
	}
	if result.RowsAffected == 0 {
		result = tx.WithContext(ctx).Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", conversationID).Limit(1).Find(&conversation)
		if result.Error != nil {
			return nil, bizerr.Internal.DatabaseError.Wrap(result.Error, "failed to load Agent conversation")
		}
		if result.RowsAffected == 0 || conversation.UserID != userID ||
			conversation.IdentityScope != string(identity) || conversation.DeletedAt.Valid {
			return nil, agentConversationNotFound()
		}
	}
	return &conversation, nil
}

func ensureNoRunningAgentRun(ctx context.Context, tx *gorm.DB, conversationID string) error {
	now := time.Now()
	if err := tx.WithContext(ctx).Model(&model.AgentRun{}).
		Where(
			"conversation_id = ? AND status = ? AND started_at < ?",
			conversationID, AgentRunStatusRunning, now.Add(-agentContextLifetime),
		).
		Updates(map[string]any{
			"status":        AgentRunStatusFailed,
			"finished_at":   now,
			"error_message": "Agent run expired before completion",
		}).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to expire stale Agent runs")
	}

	var count int64
	if err := tx.WithContext(ctx).Model(&model.AgentRun{}).
		Where("conversation_id = ? AND status = ?", conversationID, AgentRunStatusRunning).
		Count(&count).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to check active Agent runs")
	}
	if count > 0 {
		return bizerr.Conflict.ResourceStatusError.New("Agent response is still being generated")
	}
	return nil
}

func (s *AgentConversationService) ensureOwnedConversation(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	identity AgentIdentity,
	conversationID string,
) error {
	var conversation model.AgentConversation
	err := db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND identity_scope = ?", conversationID, userID, identity).
		First(&conversation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return agentConversationNotFound()
	}
	if err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to load Agent conversation")
	}
	return nil
}

func activeAgentMessages(ctx context.Context, db *gorm.DB, conversationID string) ([]model.AgentMessage, error) {
	var records []model.AgentMessage
	err := db.WithContext(ctx).
		Where("conversation_id = ? AND superseded_at IS NULL", conversationID).
		Order("sequence ASC").
		Find(&records).Error
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "failed to load Agent messages")
	}
	return records, nil
}

func resolveAgentMessagePrefix(stored []model.AgentMessage, parentMessageID *string, mode string) (int, error) {
	parentID := ""
	if parentMessageID != nil {
		parentID = strings.TrimSpace(*parentMessageID)
	}
	if mode == AgentRunModeAppend {
		if len(stored) == 0 && parentID == "" {
			return 0, nil
		}
		if len(stored) > 0 && parentID == stored[len(stored)-1].ID {
			return len(stored), nil
		}
		return 0, agentConversationChanged()
	}
	if parentID == "" {
		return 0, nil
	}
	for index := range stored {
		if stored[index].ID == parentID {
			return index + 1, nil
		}
	}
	return 0, agentConversationChanged()
}

func agentConversationChanged() error {
	return bizerr.Conflict.ResourceStatusError.New("Agent conversation changed; reload it before sending")
}

func supersedeMessages(ctx context.Context, tx *gorm.DB, messages []model.AgentMessage) error {
	if len(messages) == 0 {
		return nil
	}
	ids := make([]string, 0, len(messages))
	for index := range messages {
		ids = append(ids, messages[index].ID)
	}
	now := time.Now()
	if err := tx.WithContext(ctx).Model(&model.AgentMessage{}).
		Where("id IN ?", ids).
		Update("superseded_at", now).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to supersede Agent messages")
	}
	return nil
}

func ensureMessageIDAvailable(ctx context.Context, tx *gorm.DB, messageID string) error {
	var count int64
	if err := tx.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ?", messageID).Count(&count).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to validate Agent message id")
	}
	if count > 0 {
		return bizerr.Conflict.ResourceAlreadyExists.New("Agent message already exists")
	}
	return nil
}

func nextMessageSequence(ctx context.Context, tx *gorm.DB, conversationID string) (int64, error) {
	var result struct {
		Maximum int64
	}
	err := tx.WithContext(ctx).Model(&model.AgentMessage{}).
		Select("COALESCE(MAX(sequence), 0) AS maximum").
		Where("conversation_id = ?", conversationID).
		Scan(&result).Error
	if err != nil {
		return 0, bizerr.Internal.DatabaseError.Wrap(err, "failed to allocate Agent message sequence")
	}
	return result.Maximum + 1, nil
}

func saveAssistantMessage(
	ctx context.Context,
	tx *gorm.DB,
	run *model.AgentRun,
	message *AgentUIMessage,
	status string,
) error {
	messageID := strings.TrimSpace(message.ID)
	if messageID == "" {
		messageID = uuid.NewString()
	}
	var count int64
	if err := tx.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ?", messageID).Count(&count).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to validate Agent response id")
	}
	if count > 0 {
		messageID = uuid.NewString()
	}
	sequence, err := nextMessageSequence(ctx, tx, run.ConversationID)
	if err != nil {
		return err
	}
	runID := run.ID
	parentID := run.TriggerMessageID
	record := model.AgentMessage{
		ID:              messageID,
		ConversationID:  run.ConversationID,
		RunID:           &runID,
		ParentMessageID: &parentID,
		Sequence:        sequence,
		Role:            "assistant",
		Status:          status,
		SchemaVersion:   agentMessageSchemaVersion,
		Parts:           datatypes.JSON(cloneJSON(message.Parts)),
		Metadata:        datatypes.JSON(cloneJSON(message.Metadata)),
	}
	if err := tx.WithContext(ctx).Create(&record).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to save Agent response")
	}
	if err := tx.WithContext(ctx).Model(&model.AgentConversation{}).
		Where(
			"id = ? AND user_id = ? AND identity_scope = ?",
			run.ConversationID, run.UserID, run.IdentityScope,
		).
		Update("updated_at", time.Now()).Error; err != nil {
		return bizerr.Internal.DatabaseError.Wrap(err, "failed to update Agent conversation")
	}
	return nil
}

func validateAgentUIMessage(message *AgentUIMessage, expectedRole string) error {
	message.ID = strings.TrimSpace(message.ID)
	if (message.ID == "" && expectedRole != "assistant") || len(message.ID) > agentMaximumMessageIDLength ||
		strings.ContainsAny(message.ID, "\x00\r\n") {
		return bizerr.BadRequest.ParameterError.New("message id is invalid")
	}
	if message.Role != expectedRole {
		return bizerr.BadRequest.ParameterError.New("message role is invalid")
	}
	var parts []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message.Parts, &parts); err != nil || len(parts) == 0 {
		return bizerr.BadRequest.ParameterError.New("message parts are invalid")
	}
	for _, part := range parts {
		if strings.TrimSpace(part.Type) == "" {
			return bizerr.BadRequest.ParameterError.New("message part type is required")
		}
	}
	if len(message.Metadata) > 0 && !json.Valid(message.Metadata) {
		return bizerr.BadRequest.ParameterError.New("message metadata is invalid")
	}
	return nil
}

func hasAgentMessageParts(parts json.RawMessage) bool {
	var values []json.RawMessage
	return json.Unmarshal(parts, &values) == nil && len(values) > 0
}

func validateConversationID(conversationID string) error {
	if _, err := uuid.Parse(conversationID); err != nil {
		return bizerr.BadRequest.ParameterError.New("conversation id is invalid")
	}
	return nil
}

func validateConversationTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", bizerr.BadRequest.MissingParameter.New("conversation title is required")
	}
	if utf8.RuneCountInString(title) > agentMaximumTitleRunes {
		return "", bizerr.BadRequest.ParameterError.New("conversation title is too long")
	}
	return title, nil
}

func conversationSummary(record *model.AgentConversation) AgentConversationSummary {
	return AgentConversationSummary{
		ID:          record.ID,
		Title:       record.Title,
		CustomTitle: record.CustomTitle,
		CreatedAt:   record.CreatedAt.UnixMilli(),
		UpdatedAt:   record.UpdatedAt.UnixMilli(),
	}
}

func messageRecordsToDTO(records []model.AgentMessage) []AgentUIMessage {
	items := make([]AgentUIMessage, 0, len(records))
	for index := range records {
		record := &records[index]
		items = append(items, AgentUIMessage{
			ID:       record.ID,
			Role:     record.Role,
			Parts:    cloneJSON(record.Parts),
			Metadata: cloneJSON(record.Metadata),
		})
	}
	return items
}

func cloneJSON(value []byte) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func isTerminalRunStatus(status string) bool {
	return status == AgentRunStatusCompleted || status == AgentRunStatusCanceled || status == AgentRunStatusFailed
}

func truncateString(value string, maximum int) string {
	characters := []rune(value)
	if len(characters) <= maximum {
		return value
	}
	return string(characters[:maximum])
}

func agentConversationNotFound() error {
	return bizerr.NotFound.DataBaseNotFound.New("Agent conversation not found")
}
