package inference

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

func NewConversationStore(db *gorm.DB) *ConversationStore {
	return &ConversationStore{DB: db}
}

func (store *ConversationStore) List(
	ctx context.Context, scope *ConversationScope, limit int,
) ([]model.KthenaChatSession, error) {
	conversations := make([]model.KthenaChatSession, 0)
	err := store.scoped(store.DB.WithContext(ctx), scope).
		Order("updated_at DESC, id DESC").
		Limit(limit).
		Find(&conversations).Error
	return conversations, err
}

func (store *ConversationStore) Find(
	ctx context.Context, scope *ConversationScope, clientSessionID string,
) (model.KthenaChatSession, error) {
	var conversation model.KthenaChatSession
	err := store.scoped(store.DB.WithContext(ctx), scope).
		Where("client_session_id = ?", clientSessionID).
		First(&conversation).Error
	return conversation, err
}

func (store *ConversationStore) Messages(
	ctx context.Context, sessionID uint, limit int,
) ([]model.KthenaChatMessage, error) {
	messages := make([]model.KthenaChatMessage, 0)
	err := store.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("sequence DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	sort.Slice(messages, func(left, right int) bool {
		return messages[left].Sequence < messages[right].Sequence
	})
	return messages, nil
}

func (store *ConversationStore) Create(
	ctx context.Context,
	scope *ConversationScope,
	clientSessionID, title string,
	messages []ConversationMessageReq,
) (model.KthenaChatSession, bool, error) {
	if clientSessionID == "" {
		clientSessionID = uuid.NewString()
	}
	now := time.Now().UTC()
	if title == "" {
		title = KthenaConversationTitle(messages)
	}
	conversation := model.KthenaChatSession{
		UserID:          scope.UserID,
		AccountID:       scope.AccountID,
		Username:        scope.Username,
		Namespace:       scope.Namespace,
		ServiceName:     scope.ServiceName,
		ModelName:       scope.ModelName,
		BackendType:     scope.BackendType,
		ClientSessionID: clientSessionID,
		Title:           title,
		MessageCount:    len(messages),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if len(messages) > 0 {
		conversation.LastMessageAt = &now
	}

	var existed bool
	err := store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&conversation)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			existed = true
			return store.scoped(tx, scope).
				Where("client_session_id = ?", clientSessionID).
				First(&conversation).Error
		}
		if len(messages) == 0 {
			return nil
		}
		return tx.Create(KthenaConversationMessages(conversation.ID, messages, now)).Error
	})
	return conversation, existed, err
}

func (store *ConversationStore) Update(
	ctx context.Context,
	scope *ConversationScope,
	clientSessionID string,
	title *string,
	messages *[]ConversationMessageReq,
) (model.KthenaChatSession, error) {
	var conversation model.KthenaChatSession
	err := store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}), scope).
			Where("client_session_id = ?", clientSessionID).
			First(&conversation).Error; err != nil {
			return err
		}
		if conversation.TurnExpiresAt != nil && conversation.TurnExpiresAt.After(time.Now()) {
			return bizerr.Conflict.ResourceStatusError.New("conversation has an active turn; cancel or wait before modifying it")
		}

		now := time.Now().UTC()
		updates := map[string]any{"updated_at": now}
		if title != nil {
			conversation.Title = *title
			updates["title"] = *title
		}
		if messages != nil {
			if err := tx.Where("session_id = ?", conversation.ID).Delete(&model.KthenaChatMessage{}).Error; err != nil {
				return err
			}
			if len(*messages) > 0 {
				if err := tx.Create(KthenaConversationMessages(conversation.ID, *messages, now)).Error; err != nil {
					return err
				}
			}
			conversation.MessageCount = len(*messages)
			updates["message_count"] = conversation.MessageCount
			if len(*messages) == 0 {
				conversation.LastMessageAt = nil
				updates["last_message_at"] = nil
			} else {
				conversation.LastMessageAt = &now
				updates["last_message_at"] = now
			}
			if conversation.Title == "" {
				conversation.Title = KthenaConversationTitle(*messages)
				updates["title"] = conversation.Title
			}
		}
		if err := tx.Model(&conversation).Updates(updates).Error; err != nil {
			return err
		}
		conversation.UpdatedAt = now
		return nil
	})
	return conversation, err
}

func (store *ConversationStore) Delete(
	ctx context.Context, scope *ConversationScope, clientSessionID string,
) error {
	return store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation model.KthenaChatSession
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}), scope).
			Where("client_session_id = ?", clientSessionID).
			First(&conversation).Error; err != nil {
			return err
		}
		if conversation.TurnExpiresAt != nil && conversation.TurnExpiresAt.After(time.Now()) {
			return bizerr.Conflict.ResourceStatusError.New("conversation has an active turn; cancel or wait before modifying it")
		}

		if err := tx.Where("session_id = ?", conversation.ID).Delete(&model.KthenaChatMessage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&conversation).Error
	})
}

func (store *ConversationStore) FindTurn(
	ctx context.Context, sessionID uint, clientTurnID string,
) (userMessage, assistantMessage model.KthenaChatMessage, found bool, err error) {
	err = store.DB.WithContext(ctx).
		Where("session_id = ? AND client_turn_id = ?", sessionID, clientTurnID).
		First(&userMessage).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.KthenaChatMessage{}, model.KthenaChatMessage{}, false, nil
	}
	if err != nil {
		return model.KthenaChatMessage{}, model.KthenaChatMessage{}, false, err
	}
	err = store.DB.WithContext(ctx).
		Where("session_id = ? AND sequence = ?", sessionID, userMessage.Sequence+1).
		First(&assistantMessage).Error
	if err != nil {
		return model.KthenaChatMessage{}, model.KthenaChatMessage{}, false, err
	}
	return userMessage, assistantMessage, true, nil
}

func (store *ConversationStore) appendTurn(
	ctx context.Context,
	scope *ConversationScope,
	clientSessionID string,
	req ConversationTurnReq,
	assistant ChatMessage,
	rawCompletion []byte,
	lease, requestHash string,
) (model.KthenaChatSession, model.KthenaChatMessage, error) {
	var conversation model.KthenaChatSession
	var assistantMessage model.KthenaChatMessage
	err := store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}), scope).
			Where("client_session_id = ?", clientSessionID).
			First(&conversation).Error; err != nil {
			return err
		}
		if conversation.TurnLease != lease || conversation.TurnExpiresAt == nil || !conversation.TurnExpiresAt.After(time.Now()) {
			return bizerr.Conflict.ResourceStatusError.New("turn lease expired; retry the same request")
		}

		var lastSequence int
		if err := tx.Model(&model.KthenaChatMessage{}).
			Where("session_id = ?", conversation.ID).
			Select("COALESCE(MAX(sequence), 0)").
			Scan(&lastSequence).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		var clientTurnID *string
		if req.ClientTurnID != "" {
			clientTurnID = &req.ClientTurnID
		}
		userMessage := model.KthenaChatMessage{
			SessionID:    conversation.ID,
			Sequence:     lastSequence + 1,
			Role:         KthenaConversationRoleUser,
			Content:      req.Content,
			ClientTurnID: clientTurnID,
			RequestHash:  requestHash,
			CreatedAt:    now,
		}
		assistantMessage = model.KthenaChatMessage{
			SessionID:    conversation.ID,
			Sequence:     lastSequence + 2,
			Role:         nonEmpty(assistant.Role, KthenaConversationRoleAssistant),
			Content:      assistant.Content,
			ResponseJSON: datatypes.JSON(rawCompletion),
			CreatedAt:    now,
		}
		if err := tx.Create(&[]model.KthenaChatMessage{userMessage, assistantMessage}).Error; err != nil {
			return err
		}
		conversation.MessageCount = lastSequence + 2
		conversation.LastMessageAt = &now
		if conversation.Title == "" {
			conversation.Title = KthenaConversationTitle([]ConversationMessageReq{{
				Role: KthenaConversationRoleUser, Content: req.Content,
			}})
		}
		if err := tx.Model(&conversation).Updates(map[string]any{
			"title":           conversation.Title,
			"message_count":   conversation.MessageCount,
			"last_message_at": now,
			"turn_lease":      "", "turn_expires_at": nil,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		conversation.UpdatedAt = now
		return nil
	})
	return conversation, assistantMessage, err
}

func (store *ConversationStore) scoped(
	db *gorm.DB, scope *ConversationScope,
) *gorm.DB {
	return db.Where(
		"user_id = ? AND account_id = ? AND namespace = ? AND service_name = ? AND model_name = ?",
		scope.UserID, scope.AccountID, scope.Namespace, scope.ServiceName, scope.ModelName,
	)
}
