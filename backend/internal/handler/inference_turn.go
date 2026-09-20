package handler

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

type kthenaCompletionError struct {
	body  []byte
	cause error
}

func (err *kthenaCompletionError) Error() string { return err.cause.Error() }
func (err *kthenaCompletionError) Unwrap() error { return err.cause }

// Lock the conversation before reading history or invoking inference. The row
// lock serializes retries across backend replicas and also orders distinct turns.
// The response is sent only after the transaction commits successfully.
func (store *kthenaConversationStore) completeTurn(
	ctx context.Context, scope *kthenaConversationScope, sessionID string,
	req KthenaConversationTurnReq, complete func([]byte) ([]byte, error),
) (model.KthenaChatSession, model.KthenaChatMessage, error) {
	var conversation model.KthenaChatSession
	var assistant model.KthenaChatMessage
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}), scope).
			Where("client_session_id = ?", sessionID).First(&conversation).Error; err != nil {
			return err
		}
		locked := newKthenaConversationStore(tx)
		if req.ClientTurnID != "" {
			_, prior, found, err := locked.findTurn(ctx, conversation.ID, req.ClientTurnID)
			if err != nil {
				return err
			}
			if found {
				assistant = prior
				return nil
			}
		}
		history, err := locked.messages(ctx, conversation.ID, maxKthenaConversationTurnHistory)
		if err != nil {
			return err
		}
		body, err := buildKthenaConversationTurnBody(scope.RouteModelName, history, req)
		if err != nil {
			return err
		}
		raw, err := complete(body)
		if err != nil {
			return &kthenaCompletionError{body: raw, cause: err}
		}
		message, err := kthenaAssistantMessageFromCompletion(raw)
		if err != nil {
			return bizerr.Internal.ServiceError.Wrap(err, "invalid inference completion response")
		}
		conversation, assistant, _, err = locked.appendTurn(ctx, scope, sessionID, req, message, raw)
		return err
	})
	return conversation, assistant, err
}
