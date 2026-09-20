package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	controllerfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

const (
	kthenaConversationTestNamespace = "crater-workspace"
	kthenaConversationTestService   = "qwen"
	kthenaConversationTestModel     = "Qwen/Qwen3-4B"
	kthenaConversationTestUsername  = "alice"
	kthenaConversationTestOtherUser = "bob"
	kthenaConversationServedModel   = "served-model-name"
)

//nolint:gocyclo // This API-level test intentionally exercises create, list, update, and access isolation in one user flow.
func TestKthenaConversationHandlersPersistAndScopeByDeploymentUser(t *testing.T) {
	router, setToken, db := newKthenaConversationTestRouter(t, nil)

	created := requestKthenaConversation(t, router, http.MethodPost,
		"/v1/kthena/inference-services/"+kthenaConversationTestService+"/conversations",
		`{"sessionId":"client-conversation-a","messages":[{"role":"user","content":"介绍一下 Crater"}]}`,
	)
	if created.SessionID != "client-conversation-a" {
		t.Fatalf("sessionId = %q", created.SessionID)
	}
	if created.Namespace != kthenaConversationTestNamespace || created.ServiceName != kthenaConversationTestService ||
		created.ModelName != kthenaConversationTestModel || created.BackendType != "vLLM" {
		t.Fatalf("unexpected conversation scope: %+v", created)
	}
	if created.MessageCount != 1 || len(created.Messages) != 1 || created.Title != "介绍一下 Crater" {
		t.Fatalf("created conversation = %+v", created)
	}

	// A same client UUID retry is idempotent inside the current user/deployment/model scope.
	retry := requestKthenaConversation(t, router, http.MethodPost,
		"/v1/kthena/inference-services/"+kthenaConversationTestService+"/conversations",
		`{"sessionId":"client-conversation-a","title":"should be ignored"}`,
	)
	if retry.Title != created.Title || retry.MessageCount != 1 {
		t.Fatalf("idempotent create changed conversation: %+v", retry)
	}

	// Seed a foreign-user session in the same deployment scope. The list must not expose it.
	foreign := model.KthenaChatSession{
		UserID:          999,
		AccountID:       2,
		Username:        "other-user",
		Namespace:       kthenaConversationTestNamespace,
		ServiceName:     kthenaConversationTestService,
		ModelName:       kthenaConversationTestModel,
		BackendType:     "vLLM",
		ClientSessionID: "foreign-conversation",
	}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatal(err)
	}

	listed := requestKthenaConversationList(t, router,
		"/v1/kthena/inference-services/"+kthenaConversationTestService+"/conversations?includeMessages=true")
	if len(listed) != 1 || listed[0].SessionID != created.SessionID || len(listed[0].Messages) != 1 {
		t.Fatalf("list leaked foreign sessions: %+v", listed)
	}

	updated := requestKthenaConversation(t, router, http.MethodPatch,
		"/v1/kthena/inference-services/"+kthenaConversationTestService+"/conversations/client-conversation-a",
		`{"title":"部署说明","messages":[{"role":"system","content":"简洁回答"},{"role":"user","content":"模型在哪运行"},{"role":"assistant","content":"运行在 Kthena。"}]}`,
	)
	if updated.Title != "部署说明" || updated.MessageCount != 3 || len(updated.Messages) != 3 ||
		updated.Messages[2].Content != "运行在 Kthena。" {
		t.Fatalf("updated conversation = %+v", updated)
	}

	// The same deployment is not even discoverable by a different authenticated user.
	setToken(util.JWTMessage{UserID: 2, AccountID: 2, Username: kthenaConversationTestOtherUser})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/v1/kthena/inference-services/"+kthenaConversationTestService+"/conversations/client-conversation-a", http.NoBody))
	if response.Code != http.StatusNotFound {
		t.Fatalf("foreign user status = %d, body = %s", response.Code, response.Body.String())
	}
}

//nolint:gocyclo // This API-level test intentionally verifies request forwarding and retry idempotency together.
func TestKthenaConversationTurnPersistsAtomicallyAndRetriesByClientTurnID(t *testing.T) {
	var proxyCalls int
	router, _, _ := newKthenaConversationTestRouter(t, func(
		_ context.Context, method, path string, body []byte, _ http.Header,
	) ([]byte, error) {
		proxyCalls++
		if method != http.MethodPost || path != "v1/chat/completions" {
			t.Fatalf("proxy target = %s %s", method, path)
		}
		var request KthenaProxyReq
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatal(err)
		}
		if request.Model != kthenaConversationTestService || len(request.Messages) != 1 ||
			request.Messages[0].Role != kthenaConversationRoleUser || request.Messages[0].Content != "你好" {
			t.Fatalf("unexpected proxy request: %+v", request)
		}
		return []byte(`{"id":"completion-1","choices":[{"message":{"role":"assistant","content":"你好，我是部署模型。"}}]}`), nil
	})
	turn := requestKthenaConversationTurn(t, router,
		"/v1/kthena/inference-services/"+kthenaConversationTestService+"/conversations/turns",
		`{"content":"你好","clientTurnId":"turn-a"}`,
	)
	if turn.Conversation.SessionID == "" || turn.Conversation.MessageCount != 2 ||
		len(turn.Conversation.Messages) != 2 || turn.Assistant.Content != "你好，我是部署模型。" ||
		string(turn.Completion) == "" {
		t.Fatalf("persisted turn = %+v", turn)
	}
	if turn.Conversation.ModelName != kthenaConversationTestModel {
		t.Fatalf("stored served model = %q", turn.Conversation.ModelName)
	}

	// The second request is served from storage and never calls the model again.
	retry := requestKthenaConversationTurn(t, router,
		"/v1/kthena/inference-services/"+kthenaConversationTestService+"/conversations/"+turn.Conversation.SessionID+"/turns",
		`{"content":"你好","clientTurnId":"turn-a"}`,
	)
	if proxyCalls != 1 || retry.Conversation.MessageCount != 2 ||
		retry.Assistant.Content != turn.Assistant.Content || string(retry.Completion) == "" {
		t.Fatalf("idempotent turn result = %+v, calls = %d", retry, proxyCalls)
	}
}

func TestKthenaConversationRoutesRequireFeatureGate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:kthena_conversation_gate?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}, &model.KthenaChatSession{}, &model.KthenaChatMessage{}); err != nil {
		t.Fatal(err)
	}
	manager := &KthenaMgr{
		configService:     service.NewConfigService(query.Use(db)),
		conversationStore: newKthenaConversationStore(db),
	}
	router := gin.New()
	manager.RegisterProtected(router.Group("/v1/kthena"))
	for _, request := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/kthena/inference-services/qwen/conversations"},
		{http.MethodGet, "/v1/kthena/inference-services/qwen/conversations/client-a"},
		{http.MethodPost, "/v1/kthena/inference-services/qwen/conversations/turns"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), request.method, request.path, http.NoBody))
		if response.Code != http.StatusConflict {
			t.Fatalf("feature-gated %s %s returned %d: %s", request.method, request.path, response.Code, response.Body.String())
		}
	}
}

func newKthenaConversationTestRouter(
	t *testing.T,
	proxy func(context.Context, string, string, []byte, http.Header) ([]byte, error),
) (*gin.Engine, func(util.JWTMessage), *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:kthena_conversation_handlers_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}, &model.KthenaChatSession{}, &model.KthenaChatMessage{}); err != nil {
		t.Fatal(err)
	}
	configService := service.NewConfigService(query.Use(db))
	if err := configService.SetKthenaInferenceEnabled(t.Context(), true); err != nil {
		t.Fatal(err)
	}

	booster := newKthenaConversationTestModelBooster()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(modelBoosterGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(modelBoosterListGVK, &unstructured.UnstructuredList{})
	manager := &KthenaMgr{
		client:              controllerfake.NewClientBuilder().WithScheme(scheme).WithObjects(booster).Build(),
		configService:       configService,
		conversationStore:   newKthenaConversationStore(db),
		proxyKthenaRouterFn: proxy,
		namespace:           "crater-workspace",
	}

	currentToken := util.JWTMessage{UserID: 1, AccountID: 2, Username: kthenaConversationTestUsername}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, currentToken)
		c.Next()
	})
	manager.RegisterProtected(router.Group("/v1/kthena"))
	return router, func(token util.JWTMessage) { currentToken = token }, db
}

func newKthenaConversationTestModelBooster() *unstructured.Unstructured {
	booster := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{
			"backend": map[string]any{
				kthenaSpecTypeKey: "vLLM",
				"modelURI":        "hf://" + kthenaConversationTestModel,
				"workers": []any{map[string]any{
					"config": map[string]any{kthenaConversationServedModel: kthenaConversationTestModel},
				}},
			},
		},
	}}
	booster.SetGroupVersionKind(modelBoosterGVK)
	booster.SetName(kthenaConversationTestService)
	booster.SetNamespace(kthenaConversationTestNamespace)
	booster.SetLabels(map[string]string{
		inferenceServiceLabelUserID:    "1",
		inferenceServiceLabelAccountID: "2",
	})
	return booster
}

func requestKthenaConversation(
	t *testing.T, router http.Handler, method, path, body string,
) KthenaConversationResp {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s returned %d: %s", method, path, response.Code, response.Body.String())
	}
	var payload struct {
		Data KthenaConversationResp `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
}

func requestKthenaConversationList(
	t *testing.T, router http.Handler, path string,
) []KthenaConversationResp {
	t.Helper()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s returned %d: %s", path, response.Code, response.Body.String())
	}
	var payload struct {
		Data []KthenaConversationResp `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
}

func requestKthenaConversationTurn(
	t *testing.T, router http.Handler, path, body string,
) KthenaConversationTurnResp {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("POST %s returned %d: %s", path, response.Code, response.Body.String())
	}
	var payload struct {
		Data KthenaConversationTurnResp `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
}
