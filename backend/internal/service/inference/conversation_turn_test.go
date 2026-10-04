package inference

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/raids-lab/crater/dao/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//nolint:gocyclo // Exercise reserve, replay, cancellation and recovery in the same conversation.
func TestConversationTurnReleasesConnectionAndFencesRetries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&model.KthenaChatSession{}, &model.KthenaChatMessage{}); err != nil {
		t.Fatal(err)
	}
	store := NewConversationStore(db)
	scope := &ConversationScope{UserID: 1,
		AccountID:      2,
		Namespace:      "jobs",
		ServiceName:    "model",
		ModelName:      "qwen",
		RouteModelName: "model"}
	session, _, err := store.Create(t.Context(), scope, "session", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := ConversationTurnReq{ClientTurnID: "turn", Content: "hello"}
	calls := 0
	complete := func(ctx context.Context, _ []byte) ([]byte, error) {
		calls++
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		// A one-connection pool would time out here if inference held a transaction.
		if _, err := store.Find(ctx, scope, "session"); err != nil {
			return nil, err
		}
		if _, _, err := store.CompleteTurn(ctx, scope, "session", req, nil); err == nil {
			t.Fatal("concurrent turn acquired active lease")
		}
		if err := store.Delete(ctx, scope, "session"); err == nil {
			t.Fatal("deleted active conversation")
		}
		return []byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`), nil
	}
	if _, _, err = store.CompleteTurn(t.Context(), scope, "session", req, complete); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.CompleteTurn(t.Context(), scope, "session", req, complete); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d", calls)
	}
	req.Content = "different"
	if _, _, err = store.CompleteTurn(t.Context(), scope, "session", req, complete); err == nil {
		t.Fatal("reused turn ID with different content")
	}
	req.ClientTurnID = "next"
	_,
		_,
		err = store.CompleteTurn(t.Context(),
		scope,
		"session",
		req,
		func(context.Context,
			[]byte) ([]byte,
			error) {
			return nil,
				context.Canceled
		})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	got, err := store.Find(t.Context(), scope, "session")
	if err != nil {
		t.Fatal(err)
	}
	if got.TurnLease != "" || got.TurnExpiresAt != nil || got.MessageCount != 2 {
		t.Fatalf("canceled turn changed conversation: %+v", got)
	}
	expired := time.Now().Add(-time.Minute)
	if err = db.Model(&model.KthenaChatSession{}).Where("id = ?",
		session.ID).Updates(map[string]any{"turn_lease": "expired",
		"turn_expires_at": expired}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.CompleteTurn(t.Context(), scope, "session", req, complete); err != nil {
		t.Fatal(err)
	}
}
