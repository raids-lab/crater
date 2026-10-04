package inference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

const kthenaTurnTimeout = 2 * time.Minute
const kthenaTurnLeaseDuration = 3 * time.Minute

type CompletionError struct {
	Body  []byte
	Cause error
}

func (err *CompletionError) Error() string { return err.Cause.Error() }
func (err *CompletionError) Unwrap() error { return err.Cause }

// completeTurn reserves the conversation in a short transaction. No database
// connection or row lock is held while the inference request is running.
func (store *ConversationStore) CompleteTurn(
	ctx context.Context, scope *ConversationScope, sessionID string,
	req ConversationTurnReq, complete func(context.Context, []byte) ([]byte, error),
) (model.KthenaChatSession, model.KthenaChatMessage, error) {
	var conversation model.KthenaChatSession
	var assistant model.KthenaChatMessage
	var body []byte
	var completed bool
	lease := uuid.NewString()
	req.SessionID = sessionID
	rawReq, err := json.Marshal(req)
	if err != nil {
		return conversation, assistant, err
	}
	sum := sha256.Sum256(rawReq)
	requestHash := hex.EncodeToString(sum[:])
	err = store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}),
			scope).Where("client_session_id = ?",
			sessionID).First(&conversation).Error; err != nil {
			return err
		}
		locked := NewConversationStore(tx)
		if req.ClientTurnID != "" {
			user, prior, found, err := locked.FindTurn(ctx, conversation.ID, req.ClientTurnID)
			if err != nil {
				return err
			}
			if found {
				if user.RequestHash != requestHash {
					return bizerr.Conflict.ResourceStatusError.New("clientTurnId was already used for a different request")
				}
				assistant, completed = prior, true
				return nil
			}
		}
		if conversation.TurnExpiresAt != nil && conversation.TurnExpiresAt.After(time.Now()) {
			return bizerr.Conflict.ResourceStatusError.New("conversation has an active turn; retry the same request after it completes")
		}
		history, err := locked.Messages(ctx, conversation.ID, MaxKthenaConversationTurnHistory)
		if err != nil {
			return err
		}
		body, err = BuildKthenaConversationTurnBody(scope.RouteModelName, history, req)
		if err != nil {
			return err
		}
		return tx.Model(&conversation).Updates(map[string]any{"turn_lease": lease,
			"turn_expires_at": time.Now().Add(kthenaTurnLeaseDuration)}).Error
	})
	if err != nil || completed {
		return conversation, assistant, err
	}
	defer store.releaseTurnLease(ctx, conversation.ID, lease)
	inferenceCtx, cancel := context.WithTimeout(ctx, kthenaTurnTimeout)
	defer cancel()
	raw, err := complete(inferenceCtx, body)
	if err != nil {
		return conversation, assistant, &CompletionError{Body: raw, Cause: err}
	}
	message, err := KthenaAssistantMessageFromCompletion(raw)
	if err != nil {
		return conversation, assistant, bizerr.Internal.ServiceError.Wrap(err, "invalid inference completion response")
	}
	// Fence expired/crashed workers before appending anything to the conversation.
	conversation, assistant, err = store.appendTurn(ctx, scope, sessionID, req, message, raw, lease, requestHash)
	return conversation, assistant, err
}

func (store *ConversationStore) releaseTurnLease(ctx context.Context, sessionID uint, lease string) {
	const cleanupTimeout = 10 * time.Second
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	if err := store.DB.WithContext(cleanupCtx).Model(&model.KthenaChatSession{}).Where("id = ? AND turn_lease = ?", sessionID, lease).
		Updates(map[string]any{"turn_lease": "", "turn_expires_at": nil}).Error; err != nil {
		klog.Errorf("release inference turn lease: %v", err)
	}
}
