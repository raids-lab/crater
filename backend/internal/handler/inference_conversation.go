package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	inferencesvc "github.com/raids-lab/crater/internal/service/inference"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type kthenaConversationGetReq struct {
	MessageLimit int `form:"messageLimit"`
}

// UserListKthenaConversations godoc
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
func (mgr *KthenaMgr) UserListKthenaConversations(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}

	var req KthenaConversationListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid conversation list query"))
		return
	}
	req.Limit = inferencesvc.NormalizeKthenaConversationLimit(
		req.Limit, inferencesvc.DefaultKthenaConversationLimit, inferencesvc.MaxKthenaConversationLimit,
	)
	req.MessageLimit = inferencesvc.NormalizeKthenaConversationLimit(
		req.MessageLimit, inferencesvc.DefaultKthenaConversationMessageLimit, inferencesvc.MaxKthenaConversationMessageLimit,
	)

	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}
	conversations, err := store.List(c.Request.Context(), &scope, req.Limit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "list conversations failed")
		return
	}

	response := make([]KthenaConversationResp, 0, len(conversations))
	for index := range conversations {
		var messages []model.KthenaChatMessage
		if req.IncludeMessages {
			messages, err = store.Messages(c.Request.Context(), conversations[index].ID, req.MessageLimit)
			if err != nil {
				kthenaConversationDatabaseError(c, err, "list conversation messages failed")
				return
			}
		}
		response = append(response, inferencesvc.KthenaConversationToResp(&conversations[index], messages))
	}
	resputil.Success(c, response)
}

// UserCreateKthenaConversation godoc
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
func (mgr *KthenaMgr) UserCreateKthenaConversation(c *gin.Context) {
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

	conversation, existed, err := store.Create(
		c.Request.Context(), &scope, req.SessionID, req.Title, req.Messages,
	)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "create conversation failed")
		return
	}
	messages, err := store.Messages(c.Request.Context(), conversation.ID, inferencesvc.MaxKthenaConversationMessageLimit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load conversation messages failed")
		return
	}
	if existed {
		klog.V(inferencesvc.KthenaConversationLogVerbosity).Infof(
			"Kthena conversation create reused client session %q", conversation.ClientSessionID,
		)
	}
	resputil.Success(c, inferencesvc.KthenaConversationToResp(&conversation, messages))
}

// UserGetKthenaConversation godoc
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
func (mgr *KthenaMgr) UserGetKthenaConversation(c *gin.Context) {
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
	conversation, err := store.Find(c.Request.Context(), &scope, sessionID)
	if err != nil {
		kthenaConversationFindError(c, err)
		return
	}
	messages, err := store.Messages(
		c.Request.Context(), conversation.ID,
		inferencesvc.NormalizeKthenaConversationLimit(
			req.MessageLimit, inferencesvc.DefaultKthenaConversationMessageLimit, inferencesvc.MaxKthenaConversationMessageLimit,
		),
	)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load conversation messages failed")
		return
	}
	resputil.Success(c, inferencesvc.KthenaConversationToResp(&conversation, messages))
}

// UserUpdateKthenaConversation godoc
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
func (mgr *KthenaMgr) UserUpdateKthenaConversation(c *gin.Context) {
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
	conversation, err := store.Update(c.Request.Context(), &scope, sessionID, req.Title, req.Messages)
	if err != nil {
		kthenaConversationFindOrDatabaseError(c, err, "update conversation failed")
		return
	}
	messages, err := store.Messages(c.Request.Context(), conversation.ID, inferencesvc.MaxKthenaConversationMessageLimit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load conversation messages failed")
		return
	}
	resputil.Success(c, inferencesvc.KthenaConversationToResp(&conversation, messages))
}

// UserDeleteKthenaConversation godoc
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
func (mgr *KthenaMgr) UserDeleteKthenaConversation(c *gin.Context) {
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
	if err := store.Delete(c.Request.Context(), &scope, sessionID); err != nil {
		kthenaConversationFindOrDatabaseError(c, err, "delete conversation failed")
		return
	}
	resputil.Success(c, "conversation deleted")
}

// UserCreateKthenaConversationTurn godoc
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
func (mgr *KthenaMgr) UserCreateKthenaConversationTurn(c *gin.Context) {
	scope, ok := mgr.loadKthenaConversationScope(c)
	if !ok {
		return
	}
	var req KthenaConversationTurnReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid conversation turn"))
		return
	}
	if err := inferencesvc.ValidateKthenaConversationTurn(&req); err != nil {
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
	if err := inferencesvc.ValidateKthenaConversationSessionID(sessionID); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}

	store, ok := mgr.kthenaConversationStore(c)
	if !ok {
		return
	}
	conversation, _, err := store.Create(c.Request.Context(), &scope, sessionID, "", nil)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "prepare conversation failed")
		return
	}

	var assistantMessage model.KthenaChatMessage
	conversation, assistantMessage, err = store.CompleteTurn(c.Request.Context(), &scope, conversation.ClientSessionID, req,
		func(ctx context.Context, body []byte) ([]byte, error) {
			if wantsKthenaStream(c) {
				return mgr.streamKthenaTurn(ctx,
					body,
					func(text string) error {
						return writeKthenaEvent(c,
							"delta",
							map[string]string{"content": text})
					})
			}
			return mgr.proxyKthenaRouter(ctx, http.MethodPost, kthenaChatCompletionsPath, body, c.Request.Header)
		})
	if err != nil {
		handleKthenaTurnError(c, err)
		return
	}
	mgr.respondKthenaConversationTurn(c, store, &conversation, &assistantMessage)
}

// UserCreateKthenaConversationTurnWithoutSession godoc
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
func (mgr *KthenaMgr) UserCreateKthenaConversationTurnWithoutSession(c *gin.Context) {
	mgr.UserCreateKthenaConversationTurn(c)
}

func (mgr *KthenaMgr) respondKthenaConversationTurn(
	c *gin.Context,
	store *inferencesvc.ConversationStore,
	conversation *model.KthenaChatSession,
	assistant *model.KthenaChatMessage,
) {
	messages, err := store.Messages(c.Request.Context(), conversation.ID, inferencesvc.MaxKthenaConversationMessageLimit)
	if err != nil {
		kthenaConversationDatabaseError(c, err, "load persisted conversation failed")
		return
	}
	completion := json.RawMessage(assistant.ResponseJSON)
	if len(completion) == 0 || !json.Valid(completion) {
		completion = nil
	}
	response := KthenaConversationTurnResp{
		Conversation: inferencesvc.KthenaConversationToResp(conversation, messages),
		Assistant:    inferencesvc.KthenaConversationMessageToResp(assistant),
		Completion:   completion,
	}
	if wantsKthenaStream(c) {
		_ = writeKthenaEvent(c, "complete", response)
		return
	}
	resputil.Success(c, response)
}

func (mgr *KthenaMgr) loadKthenaConversationScope(
	c *gin.Context,
) (inferencesvc.ConversationScope, bool) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return inferencesvc.ConversationScope{}, false
	}
	scope, err := kthenaConversationScopeFromDeployment(c, obj)
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return inferencesvc.ConversationScope{}, false
	}
	return scope, true
}

func (mgr *KthenaMgr) kthenaConversationStore(c *gin.Context) (*inferencesvc.ConversationStore, bool) {
	if mgr.conversationStore == nil || mgr.conversationStore.DB == nil {
		resputil.HandleError(c, bizerr.Internal.DatabaseError.New("Kthena conversation storage is not initialized"))
		return nil, false
	}
	return mgr.conversationStore, true
}

func kthenaConversationScopeFromDeployment(
	c *gin.Context, obj *unstructured.Unstructured,
) (inferencesvc.ConversationScope, error) {
	if obj == nil {
		return inferencesvc.ConversationScope{}, bizerr.BadRequest.ParameterError.New("inference service is required")
	}
	token := util.GetToken(c)
	if token.UserID == 0 {
		return inferencesvc.ConversationScope{}, bizerr.BadRequest.ParameterError.New("current user is required")
	}
	modelName := strings.TrimSpace(servedModelFromDeployment(obj))
	if modelName == "" {
		modelName = obj.GetName()
	}
	routeModelName := strings.TrimSpace(obj.GetName())
	if routeModelName == "" {
		routeModelName = modelName
	}
	return inferencesvc.ConversationScope{
		UserID:         token.UserID,
		AccountID:      token.AccountID,
		Username:       strings.TrimSpace(token.Username),
		Namespace:      obj.GetNamespace(),
		ServiceName:    obj.GetName(),
		ModelName:      modelName,
		RouteModelName: routeModelName,
		BackendType:    kthenaBackendVLLM,
	}, nil
}

func kthenaConversationSessionID(c *gin.Context) (string, bool) {
	sessionID := strings.TrimSpace(c.Param("sessionID"))
	if err := inferencesvc.ValidateKthenaConversationSessionID(sessionID); err != nil || sessionID == "" {
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
	if err := inferencesvc.ValidateKthenaConversationSessionID(req.SessionID); err != nil {
		return err
	}
	if err := inferencesvc.ValidateKthenaConversationTitle(req.Title); err != nil {
		return err
	}
	messages, err := inferencesvc.NormalizeKthenaConversationMessages(req.Messages)
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
		if err := inferencesvc.ValidateKthenaConversationTitle(title); err != nil {
			return err
		}
		req.Title = &title
	}
	if req.Messages != nil {
		messages, err := inferencesvc.NormalizeKthenaConversationMessages(*req.Messages)
		if err != nil {
			return err
		}
		req.Messages = &messages
	}
	return nil
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
	var business *bizerr.BizError
	if errors.As(err, &business) {
		resputil.HandleError(c, err)
		return
	}
	klog.Errorf("Kthena conversation database error: %v", err)
	resputil.HandleError(c, bizerr.Internal.DatabaseError.Wrap(err, message))
}

func handleKthenaTurnError(c *gin.Context, err error) {
	if wantsKthenaStream(c) && c.Writer.Written() {
		_ = writeKthenaEvent(c, "error", map[string]string{"message": "Generation did not complete; retry the same request"})
		return
	}
	var completionErr *inferencesvc.CompletionError
	if errors.As(err, &completionErr) {
		if len(completionErr.Body) > 0 {
			c.Data(kthenaProxyHTTPStatus(completionErr.Cause), "application/json", completionErr.Body)
		} else {
			resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "proxy inference request failed"))
		}
		return
	}
	kthenaConversationFindOrDatabaseError(c, err, "complete conversation turn failed")
}
