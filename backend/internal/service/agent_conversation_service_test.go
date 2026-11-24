package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

func TestAgentConversationLifecycleAndEditHistory(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	service := NewAgentConversationService(db)
	ctx := context.Background()
	conversationID := uuid.NewString()

	firstUser := testAgentMessage("user-1", "user", "分析作业失败原因")
	firstRun, err := service.PrepareRun(
		ctx, 1, AgentIdentityUser, conversationID, nil, AgentRunModeAppend, &firstUser,
	)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(firstRun.Messages), 1)
	mustAgentTestNoError(t, service.CompleteRun(ctx, 1, AgentIdentityUser, firstRun.RunID, AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: testAgentAssistantPointer("assistant-1", "需要检查事件和日志"),
	}))

	secondUser := testAgentMessage("user-2", "user", "继续说明")
	secondRun, err := service.PrepareRun(
		ctx, 1, AgentIdentityUser, conversationID, agentTestStringPointer("assistant-1"), AgentRunModeAppend, &secondUser,
	)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(secondRun.Messages), 3)
	mustAgentTestNoError(t, service.CompleteRun(ctx, 1, AgentIdentityUser, secondRun.RunID, AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: testAgentAssistantPointer("assistant-2", "第二轮回答"),
	}))

	editedUser := testAgentMessage("user-edited", "user", "重新分析镜像问题")
	editedRun, err := service.PrepareRun(
		ctx, 1, AgentIdentityUser, conversationID, nil, AgentRunModeReplace, &editedUser,
	)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(editedRun.Messages), 1)
	mustAgentTestNoError(t, service.CompleteRun(ctx, 1, AgentIdentityUser, editedRun.RunID, AgentRunCompletion{
		Status: AgentRunStatusCompleted,
		Message: &AgentUIMessage{
			ID:   "assistant-edited",
			Role: "assistant",
			Parts: json.RawMessage(`[
				{"type":"text","text":"镜像拉取失败"},
				{"type":"data-table","data":{"columns":["事件"],"rows":[["ImagePullBackOff"]]}}
			]`),
		},
	}))

	messages, err := service.GetMessages(ctx, 1, AgentIdentityUser, conversationID)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(messages), 2)
	mustAgentTestEqual(t, messages[0].ID, "user-edited")
	mustAgentTestEqual(t, messages[1].ID, "assistant-edited")
	if !strings.Contains(string(messages[1].Parts), "data-table") {
		t.Fatal("structured assistant parts were not preserved")
	}

	var superseded int64
	mustAgentTestNoError(t, db.Model(&model.AgentMessage{}).
		Where("conversation_id = ? AND superseded_at IS NOT NULL", conversationID).
		Count(&superseded).Error)
	if superseded != 4 {
		t.Fatalf("superseded message count = %d, want 4", superseded)
	}

	_, err = service.GetMessages(ctx, 2, AgentIdentityUser, conversationID)
	mustAgentTestError(t, err)
	mustAgentTestNoError(t, service.DeleteConversation(ctx, 1, AgentIdentityUser, conversationID))
	_, err = service.GetMessages(ctx, 1, AgentIdentityUser, conversationID)
	mustAgentTestError(t, err)
	for _, value := range []any{&model.AgentRun{}, &model.AgentMessage{}} {
		var count int64
		mustAgentTestNoError(t, db.Unscoped().Model(value).
			Where("conversation_id = ?", conversationID).
			Count(&count).Error)
		if count != 0 {
			t.Fatalf("deleted conversation data remains for %T", value)
		}
	}
	var conversationCount int64
	mustAgentTestNoError(t, db.Unscoped().Model(&model.AgentConversation{}).
		Where("id = ?", conversationID).Count(&conversationCount).Error)
	if conversationCount != 0 {
		t.Fatal("deleted conversation record remains")
	}
}

func TestAgentContinuationRequiresAcceptedNavigation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	service := NewAgentConversationService(newAgentConversationTestDB(t))
	conversationID := uuid.NewString()
	user := testAgentMessage("user-nav", "user", "创建 Jupyter 作业")
	run, err := service.PrepareRun(ctx, 1, AgentIdentityUser, conversationID, nil, AgentRunModeAppend, &user)
	mustAgentTestNoError(t, err)
	assistant := &AgentUIMessage{
		ID: "assistant-nav", Role: "assistant",
		Parts: json.RawMessage(`[{"type":"text","text":"我会打开表单"},{"type":"tool-navigate_to_page","toolCallId":"nav-1","state":"input-available","input":{"path":"/portal/jobs/new/jupyter-job"}}]`),
	}
	mustAgentTestNoError(t, service.CompleteRun(ctx, 1, AgentIdentityUser, run.RunID, AgentRunCompletion{
		Status: AgentRunStatusCompleted, Message: assistant,
	}))
	_, err = service.PrepareRun(ctx, 1, AgentIdentityUser, conversationID, &assistant.ID, AgentRunModeContinue, assistant)
	mustAgentTestError(t, err)
	mustAgentTestNoError(t, service.SaveToolOutput(ctx, 1, AgentIdentityUser, conversationID, &AgentToolOutput{
		MessageID: assistant.ID, ToolCallID: "nav-1", State: "output-available",
		Output: json.RawMessage(`{"status":"accepted","path":"/portal/jobs/new/jupyter-job"}`),
	}))
	continued, err := service.PrepareRun(ctx, 1, AgentIdentityUser, conversationID, &assistant.ID, AgentRunModeContinue, assistant)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(continued.Messages), 2)
	if !strings.Contains(string(continued.Messages[1].Parts), `"status":"accepted"`) {
		t.Fatal("accepted navigation was not included in continuation context")
	}
	_, err = service.PrepareRun(ctx, 1, AgentIdentityUser, conversationID, &assistant.ID, AgentRunModeContinue, assistant)
	mustAgentTestError(t, err)
	continuedMessage := &AgentUIMessage{
		ID: assistant.ID, Role: "assistant",
		Parts: json.RawMessage(`[{"type":"text","text":"我会打开表单"},{"type":"tool-navigate_to_page","toolCallId":"nav-1","state":"output-available","input":{"path":"/portal/jobs/new/jupyter-job"},"output":{"status":"accepted","path":"/portal/jobs/new/jupyter-job"}},{"type":"text","text":"已提出填写建议"}]`),
	}
	mustAgentTestNoError(t, service.CompleteRun(ctx, 1, AgentIdentityUser, continued.RunID, AgentRunCompletion{
		Status: AgentRunStatusCompleted, Message: continuedMessage,
	}))
	messages, err := service.GetMessages(ctx, 1, AgentIdentityUser, conversationID)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(messages), 2)
	mustAgentTestEqual(t, messages[1].ID, assistant.ID)
	if !strings.Contains(string(messages[1].Parts), "已提出填写建议") {
		t.Fatal("continuation was not saved on the original assistant message")
	}
	nextUser := testAgentMessage("user-after-nav", "user", "继续")
	_, err = service.PrepareRun(ctx, 1, AgentIdentityUser, conversationID, &assistant.ID, AgentRunModeAppend, &nextUser)
	mustAgentTestNoError(t, err)
}

func TestTrimAgentContextBoundsMessageCountAndPayload(t *testing.T) {
	t.Parallel()
	records := make([]model.AgentMessage, agentMaximumContextMessages+2)
	for index := range records {
		records[index] = model.AgentMessage{ID: uuid.NewString(), Role: "user", Parts: []byte(`[{"type":"text"}]`)}
	}
	trimmed := trimAgentContext(records)
	mustAgentTestLength(t, len(trimmed), agentMaximumContextMessages)

	large := []model.AgentMessage{
		{ID: "old", Role: "user", Parts: make([]byte, agentMaximumContextBytes)},
		{ID: "latest", Role: "user", Parts: []byte(`[{"type":"text","text":"latest"}]`)},
	}
	trimmed = trimAgentContext(large)
	mustAgentTestLength(t, len(trimmed), 1)
	mustAgentTestEqual(t, trimmed[0].ID, "latest")
}

func TestAgentConversationTitleAndRunCompletionAreIdempotent(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	service := NewAgentConversationService(db)
	ctx := context.Background()
	conversationID := uuid.NewString()

	run, err := service.PrepareRun(
		ctx, 7, AgentIdentityUser, conversationID, nil, AgentRunModeAppend,
		agentTestUserMessagePointer("message-1", "如何创建作业"),
	)
	mustAgentTestNoError(t, err)
	completion := AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: testAgentAssistantPointer("answer-1", "打开作业创建页面"),
	}
	mustAgentTestNoError(t, service.CompleteRun(ctx, 7, AgentIdentityUser, run.RunID, completion))
	mustAgentTestNoError(t, service.CompleteRun(ctx, 7, AgentIdentityUser, run.RunID, completion))
	mustAgentTestNoError(t, service.UpdateGeneratedTitle(ctx, 7, AgentIdentityUser, conversationID, "创建交互式作业"))
	mustAgentTestNoError(t, service.RenameConversation(ctx, 7, AgentIdentityUser, conversationID, "我的作业配置"))
	mustAgentTestNoError(t, service.UpdateGeneratedTitle(ctx, 7, AgentIdentityUser, conversationID, "不应覆盖自定义标题"))

	items, err := service.ListConversations(ctx, 7, AgentIdentityUser)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(items), 1)
	mustAgentTestEqual(t, items[0].Title, "我的作业配置")
	if !items[0].CustomTitle {
		t.Fatal("conversation title should be marked as custom")
	}

	var assistantCount int64
	mustAgentTestNoError(t, db.Model(&model.AgentMessage{}).
		Where("conversation_id = ? AND role = ?", conversationID, "assistant").
		Count(&assistantCount).Error)
	if assistantCount != 1 {
		t.Fatalf("assistant message count = %d, want 1", assistantCount)
	}
}

func TestMergeAgentToolOutputAcceptsJobDetailNavigation(t *testing.T) {
	t.Parallel()
	parts := []byte(`[
		{"type":"text","text":"可以打开这个作业的详情查看日志。"},
		{"type":"tool-navigate_to_job_detail","toolCallId":"job-detail-1","state":"input-available","input":{"jobName":"sg-wangjh-260926-d52cc"}}
	]`)
	updated, err := mergeAgentToolOutput(parts, &AgentToolOutput{
		ToolCallID: "job-detail-1",
		State:      "output-available",
		Output:     json.RawMessage(`{"status":"accepted","jobName":"sg-wangjh-260926-d52cc"}`),
	})
	mustAgentTestNoError(t, err)
	if !strings.Contains(string(updated), `"state":"output-available"`) ||
		!strings.Contains(string(updated), `"jobName":"sg-wangjh-260926-d52cc"`) {
		t.Fatalf("job detail navigation output was not merged: %s", updated)
	}
}

func TestAgentRunCompletionSupportsGeneratedIDAndCancellation(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	service := NewAgentConversationService(db)
	ctx := context.Background()
	conversationID := uuid.NewString()

	run, err := service.PrepareRun(
		ctx, 12, AgentIdentityUser, conversationID, nil, AgentRunModeAppend,
		agentTestUserMessagePointer("question-1", "介绍平台"),
	)
	mustAgentTestNoError(t, err)
	mustAgentTestError(t, service.CompleteRun(ctx, 13, AgentIdentityUser, run.RunID, AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: testAgentAssistantPointer("", "平台介绍"),
	}))
	mustAgentTestNoError(t, service.CompleteRun(ctx, 12, AgentIdentityUser, run.RunID, AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: testAgentAssistantPointer("", "平台介绍"),
	}))

	messages, err := service.GetMessages(ctx, 12, AgentIdentityUser, conversationID)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(messages), 2)
	if messages[1].ID == "" {
		t.Fatal("assistant message id was not generated")
	}

	cancelRun, err := service.PrepareRun(
		ctx, 12, AgentIdentityUser, conversationID, agentTestStringPointer(messages[len(messages)-1].ID), AgentRunModeAppend,
		agentTestUserMessagePointer("question-2", "停止回答"),
	)
	mustAgentTestNoError(t, err)
	canceled, err := service.CancelActiveRun(ctx, 12, AgentIdentityUser, conversationID, "question-2")
	mustAgentTestNoError(t, err)
	if !canceled {
		t.Fatal("active Agent run was not canceled")
	}
	canceled, err = service.CancelActiveRun(ctx, 12, AgentIdentityUser, conversationID, "question-2")
	mustAgentTestNoError(t, err)
	if canceled {
		t.Fatal("canceling an already canceled Agent run should be idempotent")
	}

	// A late completion callback from the interrupted runtime must not revive the canceled run.
	mustAgentTestNoError(t, service.CompleteRun(ctx, 12, AgentIdentityUser, cancelRun.RunID, AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: testAgentAssistantPointer("late-assistant", "不应保存"),
	}))
	var storedRun model.AgentRun
	mustAgentTestNoError(t, db.First(&storedRun, "id = ?", cancelRun.RunID).Error)
	mustAgentTestEqual(t, storedRun.Status, AgentRunStatusCanceled)

	messages, err = service.GetMessages(ctx, 12, AgentIdentityUser, conversationID)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(messages), 3)
	mustAgentTestEqual(t, messages[2].ID, "question-2")
	nextRun, err := service.PrepareRun(
		ctx, 12, AgentIdentityUser, conversationID, agentTestStringPointer("question-2"), AgentRunModeAppend,
		agentTestUserMessagePointer("question-3", "继续提问"),
	)
	mustAgentTestNoError(t, err)
	canceled, err = service.CancelActiveRun(ctx, 12, AgentIdentityUser, conversationID, "question-2")
	mustAgentTestNoError(t, err)
	if canceled {
		t.Fatal("a delayed cancellation must not stop the next turn")
	}
	mustAgentTestNoError(t, service.CompleteRun(ctx, 12, AgentIdentityUser, nextRun.RunID, AgentRunCompletion{
		Status: AgentRunStatusCanceled,
	}))
}

func TestAgentPrepareRunRejectsStaleAppendAndAllowsExplicitReplacement(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	service := NewAgentConversationService(db)
	ctx := context.Background()
	conversationID := uuid.NewString()

	firstRun, err := service.PrepareRun(
		ctx, 21, AgentIdentityUser, conversationID, nil, AgentRunModeAppend,
		agentTestUserMessagePointer("user-1", "第一轮"),
	)
	mustAgentTestNoError(t, err)
	mustAgentTestNoError(t, service.CompleteRun(ctx, 21, AgentIdentityUser, firstRun.RunID, AgentRunCompletion{
		Status:  AgentRunStatusCompleted,
		Message: testAgentAssistantPointer("assistant-1", "第一轮回答"),
	}))

	_, err = service.PrepareRun(
		ctx, 21, AgentIdentityUser, conversationID, agentTestStringPointer("user-1"), AgentRunModeAppend,
		agentTestUserMessagePointer("stale-user", "过期页面发送"),
	)
	mustAgentTestError(t, err)

	replacement, err := service.PrepareRun(
		ctx, 21, AgentIdentityUser, conversationID, agentTestStringPointer("user-1"), AgentRunModeReplace,
		agentTestUserMessagePointer("replacement-user", "从这里重新开始"),
	)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(replacement.Messages), 2)
	mustAgentTestEqual(t, replacement.Messages[0].ID, "user-1")
	mustAgentTestEqual(t, replacement.Messages[1].ID, "replacement-user")
}

func TestAgentPrepareRunRejectsAnotherRunningTurn(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	service := NewAgentConversationService(db)
	ctx := context.Background()
	conversationID := uuid.NewString()

	firstRun, err := service.PrepareRun(
		ctx, 31, AgentIdentityUser, conversationID, nil, AgentRunModeAppend,
		agentTestUserMessagePointer("user-running", "第一轮仍在生成"),
	)
	mustAgentTestNoError(t, err)
	_, err = service.PrepareRun(
		ctx, 31, AgentIdentityUser, conversationID, agentTestStringPointer("user-running"), AgentRunModeAppend,
		agentTestUserMessagePointer("user-concurrent", "并发发送"),
	)
	mustAgentTestError(t, err)

	mustAgentTestNoError(t, service.CompleteRun(ctx, 31, AgentIdentityUser, firstRun.RunID, AgentRunCompletion{
		Status: AgentRunStatusCanceled,
	}))
	_, err = service.PrepareRun(
		ctx, 31, AgentIdentityUser, conversationID, agentTestStringPointer("user-running"), AgentRunModeAppend,
		agentTestUserMessagePointer("user-after-cancel", "取消后重试"),
	)
	mustAgentTestNoError(t, err)
}

func TestAgentPrepareRunExpiresUnrecoverableTurn(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	service := NewAgentConversationService(db)
	ctx := context.Background()
	conversationID := uuid.NewString()

	staleRun, err := service.PrepareRun(
		ctx, 32, AgentIdentityUser, conversationID, nil, AgentRunModeAppend,
		agentTestUserMessagePointer("user-stale", "不会完成的请求"),
	)
	mustAgentTestNoError(t, err)
	mustAgentTestNoError(t, db.Model(&model.AgentRun{}).
		Where("id = ?", staleRun.RunID).
		Update("started_at", time.Now().Add(-agentContextLifetime-time.Minute)).Error)

	_, err = service.PrepareRun(
		ctx, 32, AgentIdentityUser, conversationID, agentTestStringPointer("user-stale"), AgentRunModeAppend,
		agentTestUserMessagePointer("user-after-expiry", "继续提问"),
	)
	mustAgentTestNoError(t, err)

	var storedRun model.AgentRun
	mustAgentTestNoError(t, db.First(&storedRun, "id = ?", staleRun.RunID).Error)
	mustAgentTestEqual(t, storedRun.Status, AgentRunStatusFailed)
}

func TestAgentFailRunAfterRequestCancellationAllowsNextTurn(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	service := NewAgentConversationService(db)
	conversationID := uuid.NewString()
	run, err := service.PrepareRun(
		context.Background(), 33, AgentIdentityUser, conversationID, nil, AgentRunModeAppend,
		agentTestUserMessagePointer("user-canceled", "Analyze this job"),
	)
	mustAgentTestNoError(t, err)
	checkedCleanup := false
	mustAgentTestNoError(t, db.Callback().Update().Before("gorm:update").Register("check_agent_cleanup_context", func(tx *gorm.DB) {
		deadline, ok := tx.Statement.Context.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > agentRunFailureTimeout || tx.Statement.Context.Err() != nil {
			t.Error("run cleanup must use an independent, bounded context")
		}
		checkedCleanup = true
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mustAgentTestNoError(t, service.FailRun(ctx, 33, AgentIdentityUser, run.RunID, ctx.Err()))
	mustAgentTestNoError(t, db.Callback().Update().Remove("check_agent_cleanup_context"))
	if !checkedCleanup {
		t.Fatal("run cleanup did not update the database")
	}
	var storedRun model.AgentRun
	mustAgentTestNoError(t, db.First(&storedRun, "id = ?", run.RunID).Error)
	mustAgentTestEqual(t, storedRun.Status, AgentRunStatusFailed)
	mustAgentTestEqual(t, storedRun.ErrorMessage, context.Canceled.Error())
	nextRun, err := service.PrepareRun(
		context.Background(), 33, AgentIdentityUser, conversationID, agentTestStringPointer("user-canceled"), AgentRunModeAppend,
		agentTestUserMessagePointer("user-retry", "Hello"),
	)
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(nextRun.Messages), 1)
	mustAgentTestEqual(t, nextRun.Messages[0].ID, "user-retry")
}

func TestAdminAgentConversationListSupportsPaginationAndFiltering(t *testing.T) {
	t.Parallel()
	db := newAgentConversationTestDB(t)
	mustAgentTestNoError(t, db.Exec(`
		CREATE TABLE users (
			id integer PRIMARY KEY,
			name text NOT NULL,
			nickname text NOT NULL
		)
	`).Error)
	mustAgentTestNoError(t, db.Exec(`
		INSERT INTO users (id, name, nickname) VALUES
			(41, 'alice', 'Alice'),
			(42, 'bob', 'Bob'),
			(43, 'charlie', 'Charlie')
	`).Error)
	now := time.Now()
	mustAgentTestNoError(t, db.Create(&[]model.AgentConversation{
		{ID: uuid.NewString(), UserID: 41, IdentityScope: string(AgentIdentityUser), Title: "Dataset help", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.NewString(), UserID: 41, IdentityScope: string(AgentIdentityUser), Title: "Job help", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.NewString(), UserID: 41, IdentityScope: string(AgentIdentityAdmin), Title: "Admin review", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.NewString(), UserID: 42, IdentityScope: string(AgentIdentityAdmin), Title: "Cluster review", CreatedAt: now, UpdatedAt: now},
	}).Error)

	service := NewAgentConversationService(db)
	page, err := service.ListAllConversations(context.Background(), AdminAgentConversationQuery{
		Page: 1, PageSize: 1, Search: "bob", IdentityScope: AgentIdentityAdmin,
	})
	mustAgentTestNoError(t, err)
	mustAgentTestLength(t, len(page.Items), 1)
	if page.Total != 1 || page.Items[0].Username != "bob" || page.Items[0].IdentityScope != AgentIdentityAdmin {
		t.Fatalf("unexpected admin conversation page: %+v", page)
	}
	aliceUserPage, err := service.ListAllConversations(context.Background(), AdminAgentConversationQuery{
		Page: 1, PageSize: 20, UserID: 41, IdentityScope: AgentIdentityUser,
	})
	mustAgentTestNoError(t, err)
	if aliceUserPage.Total != 2 || len(aliceUserPage.Items) != 2 {
		t.Fatalf("unexpected user-scoped conversation page: %+v", aliceUserPage)
	}
	for _, item := range aliceUserPage.Items {
		if item.UserID != 41 || item.IdentityScope != AgentIdentityUser {
			t.Fatalf("conversation escaped the selected user identity: %+v", item)
		}
	}

	subjects, err := service.ListAdminSubjects(context.Background(), AdminAgentSubjectQuery{
		Page: 1, PageSize: 20, Search: "alice",
	})
	mustAgentTestNoError(t, err)
	if subjects.Total != 2 || len(subjects.Items) != 2 {
		t.Fatalf("expected separate user and admin subjects for alice: %+v", subjects)
	}

	userSubjects, err := service.ListAdminSubjects(context.Background(), AdminAgentSubjectQuery{
		Page: 1, PageSize: 20, IdentityScope: AgentIdentityUser,
	})
	mustAgentTestNoError(t, err)
	if userSubjects.Total != 1 || len(userSubjects.Items) != 1 ||
		userSubjects.Items[0].UserID != 41 || userSubjects.Items[0].ConversationCount != 2 {
		t.Fatalf("unexpected grouped user subjects: %+v", userSubjects)
	}
}

func newAgentConversationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	mustAgentTestNoError(t, err)
	mustAgentTestNoError(t, db.AutoMigrate(
		&model.AgentConversation{},
		&model.AgentRun{},
		&model.AgentMessage{},
	))
	return db
}

func testAgentMessage(id, role, text string) AgentUIMessage {
	parts, _ := json.Marshal([]map[string]string{{"type": "text", "text": text}})
	return AgentUIMessage{ID: id, Role: role, Parts: parts}
}

func testAgentAssistantPointer(id, text string) *AgentUIMessage {
	message := testAgentMessage(id, "assistant", text)
	return &message
}

func agentTestUserMessagePointer(id, text string) *AgentUIMessage {
	message := testAgentMessage(id, "user", text)
	return &message
}

func agentTestStringPointer(value string) *string {
	return &value
}

func mustAgentTestNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustAgentTestError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
}

func mustAgentTestLength(t *testing.T, actual, expected int) {
	t.Helper()
	if actual != expected {
		t.Fatalf("length = %d, want %d", actual, expected)
	}
}

func mustAgentTestEqual(t *testing.T, actual, expected string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("value = %q, want %q", actual, expected)
	}
}
