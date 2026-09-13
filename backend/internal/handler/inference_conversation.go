package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

const (
	defaultKthenaConversationLimit        = 30
	maxKthenaConversationLimit            = 100
	defaultKthenaConversationMessageLimit = 100
	maxKthenaConversationMessageLimit     = 500
	maxKthenaConversationMessages         = 500
	maxKthenaConversationTitleRunes       = 256
	maxKthenaConversationContentRunes     = 32768
	maxKthenaConversationTurnHistory      = 100
	maxKthenaConversationSessionIDRunes   = 128
	kthenaConversationTitlePreviewRunes   = 48
	kthenaConversationLogVerbosity        = 4
)

const (
	kthenaConversationRoleSystem    = "system"
	kthenaConversationRoleUser      = "user"
	kthenaConversationRoleAssistant = "assistant"
)

type kthenaConversationStore struct {
	db *gorm.DB
}

func newKthenaConversationStore(db *gorm.DB) *kthenaConversationStore {
	return &kthenaConversationStore{db: db}
}

// KthenaConversationMessageReq is one OpenAI-compatible message persisted in
// a conversation. Update requests replace the complete ordered message list.
type KthenaConversationMessageReq struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// KthenaConversationCreateReq creates a conversation. SessionID is optional:
// existing clients can provide their own UUID, while an empty value gets a
// server-generated UUID in the response.
type KthenaConversationCreateReq struct {
	SessionID string                         `json:"sessionId"`
	Title     string                         `json:"title"`
	Messages  []KthenaConversationMessageReq `json:"messages"`
}

// KthenaConversationUpdateReq changes the title and/or replaces all messages.
// A nil Messages field means leave the current messages untouched; an empty
// array clears them.
type KthenaConversationUpdateReq struct {
	Title    *string                         `json:"title"`
	Messages *[]KthenaConversationMessageReq `json:"messages"`
}

// KthenaConversationListReq controls the bounded conversation history list.
type KthenaConversationListReq struct {
	IncludeMessages bool `form:"includeMessages"`
	Limit           int  `form:"limit"`
	MessageLimit    int  `form:"messageLimit"`
}

type kthenaConversationGetReq struct {
	MessageLimit int `form:"messageLimit"`
}

// KthenaConversationTurnReq sends one user turn atomically. The backend reads
// the persisted context, calls Kthena, and stores the user/assistant pair only
// after a successful non-streaming completion. ClientTurnID is optional but
// makes retries idempotent for an existing conversation.
type KthenaConversationTurnReq struct {
	SessionID    string   `json:"sessionId"`
	Content      string   `json:"content"`
	Temperature  *float64 `json:"temperature"`
	MaxTokens    *int64   `json:"maxTokens"`
	ClientTurnID string   `json:"clientTurnId"`
}

// KthenaConversationMessageResp is a stored message returned to the client.
type KthenaConversationMessageResp struct {
	Sequence  int       `json:"sequence"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// KthenaConversationResp is the persisted, deployment-scoped conversation.
// Messages are populated for conversation detail and when requested from list.
type KthenaConversationResp struct {
	SessionID    string                          `json:"sessionId"`
	Title        string                          `json:"title"`
	Namespace    string                          `json:"namespace"`
	ServiceName  string                          `json:"serviceName"`
	ModelName    string                          `json:"modelName"`
	BackendType  string                          `json:"backendType"`
	MessageCount int                             `json:"messageCount"`
	CreatedAt    time.Time                       `json:"createdAt"`
	UpdatedAt    time.Time                       `json:"updatedAt"`
	Messages     []KthenaConversationMessageResp `json:"messages,omitempty"`
}

// KthenaConversationTurnResp returns the canonical persisted assistant turn
// alongside the original OpenAI-compatible completion object.
type KthenaConversationTurnResp struct {
	Conversation KthenaConversationResp        `json:"conversation"`
	Assistant    KthenaConversationMessageResp `json:"assistant"`
	Completion   json.RawMessage               `json:"completion" swaggertype:"object"`
}

type kthenaConversationScope struct {
	UserID      uint
	AccountID   uint
	Username    string
	Namespace   string
	ServiceName string
	// ModelName is the served model snapshot persisted with the conversation.
	// It is intentionally distinct from RouteModelName: Kthena routes requests
	// by the ModelBooster/ModelRoute name, while a vLLM served-model-name can be
	// an entirely different user-facing identifier.
	ModelName      string
	RouteModelName string
	BackendType    string
}

// ListKthenaConversations godoc
//
//	@Summary		List model deployment conversations
//	@Description	List a user's deployment-scoped conversations; messages are omitted by default.
//	@Tags			kthena
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string	true	"Inference service name"
//	@Param			includeMessages	query	bool	false	"Include recent messages"
//	@Param			limit	query		int		false	"Conversation limit, maximum 100"
//	@Param			messageLimit	query	int		false	"Messages per conversation, maximum 500"
//	@Success		200	{object}	resputil.Response[[]KthenaConversationResp]
//	@Failure		404	{object}	resputil.Response[any]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-services/{name}/conversations [get]
func (mgr *KthenaMgr) ListKthenaConversations(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}

	var req KthenaConversationListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid conversation list query"))
		return
	}
	req.Limit = normalizeKthenaConversationLimit(
		req.Limit, defaultKthenaConversationLimit, maxKthenaConversationLimit,
	)
	req.MessageLimit = normalizeKthenaConversationLimit(
		req.MessageLimit, defaultKthenaConversationMessageLimit, maxKthenaConversationMessageLimit,
	)

	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}
	conversations, err := store.list(c.Request.Context(), &scope, req.Limit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "list conversations failed")
		return
	}

	response := make([]KthenaConversationResp, 0, len(conversations))
	for index := range conversations {
		var messages []model.KthenaChatMessage
		if req.IncludeMessages {
			messages, err = store.messages(c.Request.Context(), conversations[index].ID, req.MessageLimit)
			if err != nil {
				kthenaConversationDatabaseError(c, err, "list conversation messages failed")
				return
			}
		}
		response = append(response, kthenaConversationToResp(&conversations[index], messages))
	}
	resputil.Success(c, response)
}

// CreateKthenaConversation godoc
//
//	@Summary		Create model deployment conversation
//	@Description	Create a user/deployment-scoped conversation; empty sessionId gets a UUID and supplied UUIDs are idempotent.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string					true	"Inference service name"
//	@Param			request	body		KthenaConversationCreateReq	true	"Conversation"
//	@Success		200		{object}	resputil.Response[KthenaConversationResp]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-services/{name}/conversations [post]
func (mgr *KthenaMgr) CreateKthenaConversation(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}
	var req KthenaConversationCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid conversation request"))
		return
	}
	if err := validateKthenaConversationCreate(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}
	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}

	conversation, existed, err := store.create(
		c.Request.Context(), &scope, req.SessionID, req.Title, req.Messages,
	)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "create conversation failed")
		return
	}
	messages, err := store.messages(c.Request.Context(), conversation.ID, maxKthenaConversationMessageLimit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load conversation messages failed")
		return
	}
	if existed {
		klog.V(kthenaConversationLogVerbosity).Infof(
			"Kthena conversation create reused client session %q", conversation.ClientSessionID,
		)
	}
	resputil.Success(c, kthenaConversationToResp(&conversation, messages))
}

// GetKthenaConversation godoc
//
//	@Summary		Get model deployment conversation
//	@Description	Get one persisted conversation and its most recent ordered messages for the current user and authorized Kthena deployment.
//	@Tags			kthena
//	@Produce		json
//	@Security		Bearer
//	@Param			name		path	string	true	"Inference service name"
//	@Param			sessionId	path	string	true	"Conversation session UUID"
//	@Param			messageLimit	query	int	false	"Maximum recent messages, maximum 500"
//	@Success		200	{object}	resputil.Response[KthenaConversationResp]
//	@Failure		404	{object}	resputil.Response[any]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-services/{name}/conversations/{sessionId} [get]
func (mgr *KthenaMgr) GetKthenaConversation(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}
	sessionID, ok := kthenaConversationSessionID(c)
	if !ok {
		return
	}
	var req kthenaConversationGetReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid conversation query"))
		return
	}
	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}
	conversation, err := store.find(c.Request.Context(), &scope, sessionID)
	if err != nil {
		kthenaConversationFindError(c, err)
		return
	}
	messages, err := store.messages(
		c.Request.Context(), conversation.ID,
		normalizeKthenaConversationLimit(
			req.MessageLimit, defaultKthenaConversationMessageLimit, maxKthenaConversationMessageLimit,
		),
	)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load conversation messages failed")
		return
	}
	resputil.Success(c, kthenaConversationToResp(&conversation, messages))
}

// UpdateKthenaConversation godoc
//
//	@Summary		Update model deployment conversation
//	@Description	Update title and/or replace all messages of a current user's conversation. Sending messages: [] clears the message list.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name		path	string					true	"Inference service name"
//	@Param			sessionId	path	string					true	"Conversation session UUID"
//	@Param			request		body	KthenaConversationUpdateReq	true	"Conversation update"
//	@Success		200		{object}	resputil.Response[KthenaConversationResp]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-services/{name}/conversations/{sessionId} [patch]
func (mgr *KthenaMgr) UpdateKthenaConversation(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}
	sessionID, ok := kthenaConversationSessionID(c)
	if !ok {
		return
	}
	var req KthenaConversationUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid conversation update"))
		return
	}
	if err := validateKthenaConversationUpdate(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}
	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}
	conversation, err := store.update(c.Request.Context(), &scope, sessionID, req.Title, req.Messages)
	if err != nil {
		kthenaConversationFindOrDatabaseError(c, err, "update conversation failed")
		return
	}
	messages, err := store.messages(c.Request.Context(), conversation.ID, maxKthenaConversationMessageLimit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load conversation messages failed")
		return
	}
	resputil.Success(c, kthenaConversationToResp(&conversation, messages))
}

// DeleteKthenaConversation godoc
//
//	@Summary		Delete model deployment conversation
//	@Description	Permanently delete one current user's persisted Kthena conversation and all of its messages.
//	@Tags			kthena
//	@Produce		json
//	@Security		Bearer
//	@Param			name		path	string	true	"Inference service name"
//	@Param			sessionId	path	string	true	"Conversation session UUID"
//	@Success		200	{object}	resputil.Response[string]
//	@Failure		404	{object}	resputil.Response[any]
//	@Failure		500	{object}	resputil.Response[any]
//	@Router			/v1/kthena/inference-services/{name}/conversations/{sessionId} [delete]
func (mgr *KthenaMgr) DeleteKthenaConversation(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}
	sessionID, ok := kthenaConversationSessionID(c)
	if !ok {
		return
	}
	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}
	if err := store.delete(c.Request.Context(), &scope, sessionID); err != nil {
		kthenaConversationFindOrDatabaseError(c, err, "delete conversation failed")
		return
	}
	resputil.Success(c, "conversation deleted")
}

// CreateKthenaConversationTurn godoc
//
//	@Summary		Send a persisted model deployment conversation turn
//	@Description	Use stored context to call Kthena and atomically save successful user and assistant messages.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name		path	string					true	"Inference service name"
//	@Param			sessionId	path	string					true	"Conversation session UUID"
//	@Param			request		body	KthenaConversationTurnReq	true	"User turn"
//	@Success		200		{object}	resputil.Response[KthenaConversationTurnResp]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Failure		502		{object}	any
//	@Router			/v1/kthena/inference-services/{name}/conversations/{sessionId}/turns [post]
//
//nolint:gocyclo // The handler keeps request validation, proxy failure passthrough, and atomic persistence visible at the API boundary.
func (mgr *KthenaMgr) CreateKthenaConversationTurn(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}
	var req KthenaConversationTurnReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid conversation turn"))
		return
	}
	if err := validateKthenaConversationTurn(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}
	sessionID := strings.TrimSpace(c.Param("sessionID"))
	if sessionID != "" && req.SessionID != "" && sessionID != req.SessionID {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("sessionId path and request body differ"))
		return
	}
	if sessionID == "" {
		sessionID = req.SessionID
	}
	if err := validateKthenaConversationSessionID(sessionID); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}

	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}
	conversation, _, err := store.create(c.Request.Context(), &scope, sessionID, "", nil)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "prepare conversation failed")
		return
	}

	if req.ClientTurnID != "" {
		userMessage, assistantMessage, found, findErr := store.findTurn(
			c.Request.Context(), conversation.ID, req.ClientTurnID,
		)
		if findErr != nil {
			kthenaConversationDatabaseError(c, findErr, "find prior conversation turn failed")
			return
		}
		if found {
			_ = userMessage
			mgr.respondKthenaConversationTurn(c, store, &conversation, &assistantMessage)
			return
		}
	}

	history, err := store.messages(c.Request.Context(), conversation.ID, maxKthenaConversationTurnHistory)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load conversation context failed")
		return
	}
	body, err := buildKthenaConversationTurnBody(scope.RouteModelName, history, req)
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}
	rawCompletion, err := mgr.proxyKthenaRouter(
		c.Request.Context(), http.MethodPost, "v1/chat/completions", body, c.Request.Header,
	)
	if err != nil {
		klog.Errorf("proxy persisted inference conversation turn failed: %v", err)
		if len(rawCompletion) > 0 {
			c.Data(kthenaProxyHTTPStatus(err), "application/json", rawCompletion)
			return
		}
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "proxy inference request failed"))
		return
	}
	assistant, err := kthenaAssistantMessageFromCompletion(rawCompletion)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "invalid inference completion response"))
		return
	}

	conversation, assistantMessage, alreadySaved, err := store.appendTurn(
		c.Request.Context(), &scope, conversation.ClientSessionID, req, assistant, rawCompletion,
	)
	if err != nil {
		kthenaConversationFindOrDatabaseError(c, err, "persist conversation turn failed")
		return
	}
	if alreadySaved {
		klog.V(kthenaConversationLogVerbosity).Infof(
			"Kthena conversation turn %q was concurrently persisted", req.ClientTurnID,
		)
	}
	mgr.respondKthenaConversationTurn(c, store, &conversation, &assistantMessage)
}

// CreateKthenaConversationTurnWithoutSession godoc
//
//	@Summary		Send a new or existing persisted model deployment conversation turn
//	@Description	Send an atomic turn without a path sessionId. Provide a body sessionId to reuse a UUID, or leave it empty for a new UUID.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path	string					true	"Inference service name"
//	@Param			request	body	KthenaConversationTurnReq	true	"User turn"
//	@Success		200		{object}	resputil.Response[KthenaConversationTurnResp]
//	@Failure		400		{object}	resputil.Response[any]
//	@Failure		404		{object}	resputil.Response[any]
//	@Failure		500		{object}	resputil.Response[any]
//	@Failure		502		{object}	any
//	@Router			/v1/kthena/inference-services/{name}/conversations/turns [post]
func (mgr *KthenaMgr) CreateKthenaConversationTurnWithoutSession(c *gin.Context) {
	mgr.CreateKthenaConversationTurn(c)
}

func (mgr *KthenaMgr) respondKthenaConversationTurn(
	c *gin.Context,
	store *kthenaConversationStore,
	conversation *model.KthenaChatSession,
	assistant *model.KthenaChatMessage,
) {
	messages, err := store.messages(c.Request.Context(), conversation.ID, maxKthenaConversationMessageLimit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load persisted conversation failed")
		return
	}
	completion := json.RawMessage(assistant.ResponseJSON)
	if len(completion) == 0 || !json.Valid(completion) {
		completion = nil
	}
	resputil.Success(c, KthenaConversationTurnResp{
		Conversation: kthenaConversationToResp(conversation, messages),
		Assistant:    kthenaConversationMessageToResp(assistant),
		Completion:   completion,
	})
}

func (mgr *KthenaMgr) loadKthenaConversationScope(
	c *gin.Context,
) (kthenaConversationScope, bool) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return kthenaConversationScope{}, false
	}
	scope, err := kthenaConversationScopeFromModelBooster(c, obj)
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return kthenaConversationScope{}, false
	}
	return scope, true
}

func (mgr *KthenaMgr) kthenaConversationStore(c *gin.Context) (*kthenaConversationStore, bool) {
	if mgr.conversationStore == nil || mgr.conversationStore.db == nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.New("Kthena conversation storage is not initialized"))
		return nil, false
	}
	return mgr.conversationStore, true
}

func kthenaConversationScopeFromModelBooster(
	c *gin.Context, obj *unstructured.Unstructured,
) (kthenaConversationScope, error) {
	if obj == nil {
		return kthenaConversationScope{}, bizerr.BadRequest.ParameterError.New("inference service is required")
	}
	token := util.GetToken(c)
	if token.UserID == 0 {
		return kthenaConversationScope{}, bizerr.BadRequest.ParameterError.New("current user is required")
	}
	modelName := strings.TrimSpace(servedModelFromModelBooster(obj))
	if modelName == "" {
		modelName = obj.GetName()
	}
	routeModelName := strings.TrimSpace(obj.GetName())
	if routeModelName == "" {
		routeModelName = modelName
	}
	backend, _, _ := unstructured.NestedMap(obj.Object, "spec", "backend")
	return kthenaConversationScope{
		UserID:         token.UserID,
		AccountID:      token.AccountID,
		Username:       strings.TrimSpace(token.Username),
		Namespace:      obj.GetNamespace(),
		ServiceName:    obj.GetName(),
		ModelName:      modelName,
		RouteModelName: routeModelName,
		BackendType:    strings.TrimSpace(stringValue(backend[kthenaSpecTypeKey])),
	}, nil
}

func kthenaConversationSessionID(c *gin.Context) (string, bool) {
	sessionID := strings.TrimSpace(c.Param("sessionID"))
	if err := validateKthenaConversationSessionID(sessionID); err != nil || sessionID == "" {
		if err == nil {
			err = bizerr.BadRequest.ParameterError.New("sessionId is required")
		}
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return "", false
	}
	return sessionID, true
}

func validateKthenaConversationCreate(req *KthenaConversationCreateReq) error {
	if req == nil {
		return bizerr.BadRequest.ParameterError.New("conversation request is required")
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Title = strings.TrimSpace(req.Title)
	if err := validateKthenaConversationSessionID(req.SessionID); err != nil {
		return err
	}
	if err := validateKthenaConversationTitle(req.Title); err != nil {
		return err
	}
	messages, err := normalizeKthenaConversationMessages(req.Messages)
	if err != nil {
		return err
	}
	req.Messages = messages
	return nil
}

func validateKthenaConversationUpdate(req *KthenaConversationUpdateReq) error {
	if req == nil || (req.Title == nil && req.Messages == nil) {
		return bizerr.BadRequest.ParameterError.New("title or messages is required")
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if err := validateKthenaConversationTitle(title); err != nil {
			return err
		}
		req.Title = &title
	}
	if req.Messages != nil {
		messages, err := normalizeKthenaConversationMessages(*req.Messages)
		if err != nil {
			return err
		}
		req.Messages = &messages
	}
	return nil
}

func validateKthenaConversationTurn(req *KthenaConversationTurnReq) error {
	if req == nil {
		return bizerr.BadRequest.ParameterError.New("conversation turn is required")
	}
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ClientTurnID = strings.TrimSpace(req.ClientTurnID)
	req.Content = strings.TrimSpace(req.Content)
	if err := validateKthenaConversationSessionID(req.SessionID); err != nil {
		return err
	}
	if err := validateKthenaConversationSessionID(req.ClientTurnID); err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "invalid clientTurnId")
	}
	if req.Content == "" {
		return bizerr.BadRequest.ParameterError.New("content is required")
	}
	if utf8.RuneCountInString(req.Content) > maxKthenaConversationContentRunes {
		return bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("content must not exceed %d characters", maxKthenaConversationContentRunes),
		)
	}
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return bizerr.BadRequest.ParameterError.New("temperature must be between 0 and 2")
	}
	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		return bizerr.BadRequest.ParameterError.New("maxTokens must be greater than 0")
	}
	return nil
}

func validateKthenaConversationSessionID(value string) error {
	if utf8.RuneCountInString(value) > maxKthenaConversationSessionIDRunes {
		return bizerr.BadRequest.ParameterError.New("sessionId must not exceed 128 characters")
	}
	return nil
}

func validateKthenaConversationTitle(value string) error {
	if utf8.RuneCountInString(value) > maxKthenaConversationTitleRunes {
		return bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("title must not exceed %d characters", maxKthenaConversationTitleRunes),
		)
	}
	return nil
}

func normalizeKthenaConversationMessages(
	messages []KthenaConversationMessageReq,
) ([]KthenaConversationMessageReq, error) {
	if len(messages) > maxKthenaConversationMessages {
		return nil, bizerr.BadRequest.ParameterError.New(
			fmt.Sprintf("messages must not exceed %d entries", maxKthenaConversationMessages),
		)
	}
	normalized := make([]KthenaConversationMessageReq, len(messages))
	for index, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		switch role {
		case kthenaConversationRoleSystem, kthenaConversationRoleUser, kthenaConversationRoleAssistant:
		default:
			return nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf("messages[%d].role must be system, user, or assistant", index),
			)
		}
		content := strings.TrimSpace(message.Content)
		if role != kthenaConversationRoleAssistant && content == "" {
			return nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf("messages[%d].content is required", index),
			)
		}
		if utf8.RuneCountInString(content) > maxKthenaConversationContentRunes {
			return nil, bizerr.BadRequest.ParameterError.New(
				fmt.Sprintf(
					"messages[%d].content must not exceed %d characters", index, maxKthenaConversationContentRunes,
				),
			)
		}
		normalized[index] = KthenaConversationMessageReq{Role: role, Content: content}
	}
	return normalized, nil
}

func normalizeKthenaConversationLimit(value, fallback, maximum int) int {
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func kthenaConversationTitle(messages []KthenaConversationMessageReq) string {
	for _, message := range messages {
		if message.Role != kthenaConversationRoleUser || message.Content == "" {
			continue
		}
		title := strings.Join(strings.Fields(message.Content), " ")
		runes := []rune(title)
		if len(runes) > kthenaConversationTitlePreviewRunes {
			return string(runes[:kthenaConversationTitlePreviewRunes]) + "…"
		}
		return title
	}
	return ""
}

func kthenaConversationToResp(
	conversation *model.KthenaChatSession, messages []model.KthenaChatMessage,
) KthenaConversationResp {
	response := KthenaConversationResp{
		SessionID:    conversation.ClientSessionID,
		Title:        conversation.Title,
		Namespace:    conversation.Namespace,
		ServiceName:  conversation.ServiceName,
		ModelName:    conversation.ModelName,
		BackendType:  conversation.BackendType,
		MessageCount: conversation.MessageCount,
		CreatedAt:    conversation.CreatedAt,
		UpdatedAt:    conversation.UpdatedAt,
	}
	if messages != nil {
		response.Messages = make([]KthenaConversationMessageResp, 0, len(messages))
		for index := range messages {
			response.Messages = append(response.Messages, kthenaConversationMessageToResp(&messages[index]))
		}
	}
	return response
}

func kthenaConversationMessageToResp(message *model.KthenaChatMessage) KthenaConversationMessageResp {
	return KthenaConversationMessageResp{
		Sequence:  message.Sequence,
		Role:      message.Role,
		Content:   message.Content,
		CreatedAt: message.CreatedAt,
	}
}

func (store *kthenaConversationStore) list(
	ctx context.Context, scope *kthenaConversationScope, limit int,
) ([]model.KthenaChatSession, error) {
	conversations := make([]model.KthenaChatSession, 0)
	err := store.scoped(store.db.WithContext(ctx), scope).
		Order("updated_at DESC, id DESC").
		Limit(limit).
		Find(&conversations).Error
	return conversations, err
}

func (store *kthenaConversationStore) find(
	ctx context.Context, scope *kthenaConversationScope, clientSessionID string,
) (model.KthenaChatSession, error) {
	var conversation model.KthenaChatSession
	err := store.scoped(store.db.WithContext(ctx), scope).
		Where("client_session_id = ?", clientSessionID).
		First(&conversation).Error
	return conversation, err
}

func (store *kthenaConversationStore) messages(
	ctx context.Context, sessionID uint, limit int,
) ([]model.KthenaChatMessage, error) {
	messages := make([]model.KthenaChatMessage, 0)
	err := store.db.WithContext(ctx).
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

func (store *kthenaConversationStore) create(
	ctx context.Context,
	scope *kthenaConversationScope,
	clientSessionID, title string,
	messages []KthenaConversationMessageReq,
) (model.KthenaChatSession, bool, error) {
	if clientSessionID == "" {
		clientSessionID = uuid.NewString()
	}
	now := time.Now().UTC()
	if title == "" {
		title = kthenaConversationTitle(messages)
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
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		return tx.Create(kthenaConversationMessages(conversation.ID, messages, now)).Error
	})
	return conversation, existed, err
}

func (store *kthenaConversationStore) update(
	ctx context.Context,
	scope *kthenaConversationScope,
	clientSessionID string,
	title *string,
	messages *[]KthenaConversationMessageReq,
) (model.KthenaChatSession, error) {
	var conversation model.KthenaChatSession
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}), scope).
			Where("client_session_id = ?", clientSessionID).
			First(&conversation).Error; err != nil {
			return err
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
				if err := tx.Create(kthenaConversationMessages(conversation.ID, *messages, now)).Error; err != nil {
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
				conversation.Title = kthenaConversationTitle(*messages)
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

func (store *kthenaConversationStore) delete(
	ctx context.Context, scope *kthenaConversationScope, clientSessionID string,
) error {
	return store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation model.KthenaChatSession
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}), scope).
			Where("client_session_id = ?", clientSessionID).
			First(&conversation).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", conversation.ID).Delete(&model.KthenaChatMessage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&conversation).Error
	})
}

func (store *kthenaConversationStore) findTurn(
	ctx context.Context, sessionID uint, clientTurnID string,
) (userMessage, assistantMessage model.KthenaChatMessage, found bool, err error) {
	err = store.db.WithContext(ctx).
		Where("session_id = ? AND client_turn_id = ?", sessionID, clientTurnID).
		First(&userMessage).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.KthenaChatMessage{}, model.KthenaChatMessage{}, false, nil
	}
	if err != nil {
		return model.KthenaChatMessage{}, model.KthenaChatMessage{}, false, err
	}
	err = store.db.WithContext(ctx).
		Where("session_id = ? AND sequence = ?", sessionID, userMessage.Sequence+1).
		First(&assistantMessage).Error
	if err != nil {
		return model.KthenaChatMessage{}, model.KthenaChatMessage{}, false, err
	}
	return userMessage, assistantMessage, true, nil
}

func (store *kthenaConversationStore) appendTurn(
	ctx context.Context,
	scope *kthenaConversationScope,
	clientSessionID string,
	req KthenaConversationTurnReq,
	assistant ChatMessage,
	rawCompletion []byte,
) (model.KthenaChatSession, model.KthenaChatMessage, bool, error) {
	var conversation model.KthenaChatSession
	var assistantMessage model.KthenaChatMessage
	var alreadySaved bool
	err := store.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := store.scoped(tx.Clauses(clause.Locking{Strength: "UPDATE"}), scope).
			Where("client_session_id = ?", clientSessionID).
			First(&conversation).Error; err != nil {
			return err
		}
		if req.ClientTurnID != "" {
			var existingUser model.KthenaChatMessage
			err := tx.Where("session_id = ? AND client_turn_id = ?", conversation.ID, req.ClientTurnID).
				First(&existingUser).Error
			if err == nil {
				if err := tx.Where("session_id = ? AND sequence = ?", conversation.ID, existingUser.Sequence+1).
					First(&assistantMessage).Error; err != nil {
					return err
				}
				alreadySaved = true
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
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
			Role:         kthenaConversationRoleUser,
			Content:      req.Content,
			ClientTurnID: clientTurnID,
			CreatedAt:    now,
		}
		assistantMessage = model.KthenaChatMessage{
			SessionID:    conversation.ID,
			Sequence:     lastSequence + 2,
			Role:         nonEmpty(assistant.Role, kthenaConversationRoleAssistant),
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
			conversation.Title = kthenaConversationTitle([]KthenaConversationMessageReq{{
				Role: kthenaConversationRoleUser, Content: req.Content,
			}})
		}
		if err := tx.Model(&conversation).Updates(map[string]any{
			"title":           conversation.Title,
			"message_count":   conversation.MessageCount,
			"last_message_at": now,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}
		conversation.UpdatedAt = now
		return nil
	})
	return conversation, assistantMessage, alreadySaved, err
}

func (store *kthenaConversationStore) scoped(
	db *gorm.DB, scope *kthenaConversationScope,
) *gorm.DB {
	return db.Where(
		"user_id = ? AND account_id = ? AND namespace = ? AND service_name = ? AND model_name = ?",
		scope.UserID, scope.AccountID, scope.Namespace, scope.ServiceName, scope.ModelName,
	)
}

func kthenaConversationMessages(
	sessionID uint, messages []KthenaConversationMessageReq, createdAt time.Time,
) []model.KthenaChatMessage {
	rows := make([]model.KthenaChatMessage, 0, len(messages))
	for index, message := range messages {
		rows = append(rows, model.KthenaChatMessage{
			SessionID: sessionID,
			Sequence:  index + 1,
			Role:      message.Role,
			Content:   message.Content,
			CreatedAt: createdAt,
		})
	}
	return rows
}

func buildKthenaConversationTurnBody(
	modelName string, history []model.KthenaChatMessage, req KthenaConversationTurnReq,
) ([]byte, error) {
	messages := make([]ChatMessage, 0, len(history)+1)
	for _, message := range history {
		messages = append(messages, ChatMessage{Role: message.Role, Content: message.Content})
	}
	messages = append(messages, ChatMessage{Role: kthenaConversationRoleUser, Content: req.Content})
	return json.Marshal(KthenaProxyReq{
		Model:       modelName,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
}

func kthenaAssistantMessageFromCompletion(rawCompletion []byte) (ChatMessage, error) {
	var completion struct {
		Choices []struct {
			Message *ChatMessage `json:"message"`
			Text    string       `json:"text"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rawCompletion, &completion); err != nil {
		return ChatMessage{}, err
	}
	if len(completion.Choices) == 0 {
		return ChatMessage{}, bizerr.Internal.K8sServiceError.New("completion has no choices")
	}
	choice := completion.Choices[0]
	if choice.Message != nil {
		return ChatMessage{
			Role:    nonEmpty(choice.Message.Role, kthenaConversationRoleAssistant),
			Content: choice.Message.Content,
		}, nil
	}
	return ChatMessage{Role: kthenaConversationRoleAssistant, Content: choice.Text}, nil
}

func kthenaConversationFindError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("conversation not found"))
		return
	}
	kthenaConversationDatabaseError(c, err, "load conversation failed")
}

func kthenaConversationFindOrDatabaseError(c *gin.Context, err error, message string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		resputil.HandleError(c, bizerr.NotFound.DataBaseNotFound.New("conversation not found"))
		return
	}
	kthenaConversationDatabaseError(c, err, message)
}

func kthenaConversationDatabaseError(c *gin.Context, err error, message string) {
	klog.Errorf("Kthena conversation database error: %v", err)
	resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, message))
}
