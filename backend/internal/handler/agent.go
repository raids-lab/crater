package handler

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

const (
	agentChatBodyLimit         = 2 << 20
	agentTitleMessageLimit     = 8 << 10
	agentUpstreamErrorLimit    = 8 << 10
	agentStreamBufferSize      = 32 << 10
	agentRuntimeRequestTimeout = 5 * time.Minute
	llmRequestTimeout          = 5 * time.Minute
	agentIdentityHeader        = "X-Crater-Agent-Identity"
)

//nolint:gochecknoinits // Managers register through the repository's standard init mechanism.
func init() {
	Registers = append(Registers, NewAgentMgr)
}

type AgentMgr struct {
	name          string
	service       *service.AgentConfigService
	conversations *service.AgentConversationService
	diagnostics   *service.AgentJobDiagnosticService
	platformRead  *service.AgentPlatformReadService
}

func NewAgentMgr(conf *RegisterConfig) Manager {
	return &AgentMgr{
		name:          "agent",
		service:       conf.AgentConfigService,
		conversations: conf.AgentConversationService,
		diagnostics:   service.NewAgentJobDiagnosticService(query.GetDB(), conf.KubeClient),
		platformRead:  service.NewAgentPlatformReadService(query.GetDB(), conf.KubeClient),
	}
}

func (mgr *AgentMgr) GetName() string { return mgr.name }

func (mgr *AgentMgr) RegisterPublic(g *gin.RouterGroup) {
	g.POST("/llm/chat/completions", mgr.ProxyLLMChatCompletion)
	g.POST("/runs/:id/complete", mgr.CompleteAgentRun)
	g.GET("/jobs/:name/diagnostic", mgr.GetAgentJobDiagnostic)
	g.GET("/jobs/:name/scheduling-diagnosis", mgr.GetAgentJobSchedulingDiagnosis)
	g.GET("/jobs", mgr.ListAgentJobs)
	g.GET("/nodes", mgr.ListAgentNodes)
	g.GET("/nodes/:name/capacity", mgr.GetAgentNodeCapacity)
	g.GET("/capacity", mgr.GetAgentClusterCapacity)
}

func (mgr *AgentMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("/llm-config", mgr.UserGetPersonalLLMConfig)
	g.PUT("/llm-config", mgr.UserUpdatePersonalLLMConfig)
	g.DELETE("/llm-config", mgr.UserDeletePersonalLLMConfig)
	g.POST("/llm-config/test", mgr.UserTestPersonalLLMConfig)
	g.POST("/chat", mgr.UserChat)
	g.POST("/title", mgr.UserGenerateTitle)
	mgr.registerConversationRoutes(g)
}

func (mgr *AgentMgr) registerConversationRoutes(g *gin.RouterGroup) {
	g.GET("/conversations", mgr.UserListAgentConversations)
	g.GET("/conversations/:id/messages", mgr.UserGetAgentConversationMessages)
	g.POST("/conversations/:id/cancel", mgr.UserCancelAgentConversationRun)
	g.POST("/conversations/:id/tool-output", mgr.UserSaveAgentToolOutput)
	g.PUT("/conversations/:id", mgr.UserRenameAgentConversation)
	g.DELETE("/conversations/:id", mgr.UserDeleteAgentConversation)
}

func (mgr *AgentMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.GET("/subjects", mgr.AdminListAgentSubjects)
	g.GET("/conversations", mgr.AdminListAgentConversations)
	g.GET("/conversations/:id/messages", mgr.AdminGetAgentConversationMessages)
}

func resolveAgentIdentity(c *gin.Context) (service.AgentIdentity, error) {
	rawIdentity := strings.TrimSpace(c.GetHeader(agentIdentityHeader))
	if rawIdentity == "" {
		rawIdentity = string(service.AgentIdentityUser)
	}
	identity, err := service.ParseAgentIdentity(rawIdentity)
	if err != nil {
		return "", err
	}
	if identity == service.AgentIdentityAdmin && util.GetToken(c).RolePlatform != model.RoleAdmin {
		return "", bizerr.Forbidden.PermissionDenied.New("administrator Agent identity is not permitted")
	}
	return identity, nil
}

type personalLLMConfigResp struct {
	BaseURL            string                  `json:"baseUrl"`
	ModelName          string                  `json:"modelName"`
	APIKey             string                  `json:"apiKey"`
	PersonalConfigured bool                    `json:"personalConfigured"`
	PlatformConfigured bool                    `json:"platformConfigured"`
	EffectiveSource    service.LLMConfigSource `json:"effectiveSource"`
}

type updatePersonalLLMConfigReq struct {
	BaseURL   string `json:"baseUrl" binding:"required,max=1024"`
	ModelName string `json:"modelName" binding:"required,max=256"`
	APIKey    string `json:"apiKey" binding:"required,max=8192"`
	Validate  bool   `json:"validate"`
}

// UserGetPersonalLLMConfig returns the current user's personal and effective LLM configuration status.
//
//	@Summary		Get personal LLM configuration
//	@Description	Returns the masked personal configuration and whether a platform fallback is available.
//	@Tags			Agent
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[personalLLMConfigResp]
//	@Router			/v1/agent/llm-config [get]
func (mgr *AgentMgr) UserGetPersonalLLMConfig(c *gin.Context) {
	status, err := mgr.service.GetStatus(
		c.Request.Context(), util.GetToken(c).UserID, service.AgentIdentityUser,
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resp := personalLLMConfigResp{
		BaseURL:            status.BaseURL,
		ModelName:          status.ModelName,
		PersonalConfigured: status.PersonalConfigured,
		PlatformConfigured: status.PlatformConfigured,
		EffectiveSource:    status.EffectiveSource,
	}
	if status.PersonalConfigured {
		resp.APIKey = service.MaskedAPIKeyPlaceholder
	}
	resputil.Success(c, resp)
}

// UserUpdatePersonalLLMConfig validates and stores the current user's personal LLM configuration.
//
//	@Summary		Update personal LLM configuration
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		updatePersonalLLMConfigReq	true	"Personal LLM configuration"
//	@Success		200		{object}	resputil.Response[string]
//	@Router			/v1/agent/llm-config [put]
func (mgr *AgentMgr) UserUpdatePersonalLLMConfig(c *gin.Context) {
	req, ok := bindPersonalLLMConfig(c)
	if !ok {
		return
	}
	err := mgr.service.SavePersonalConfig(
		c.Request.Context(),
		util.GetToken(c).UserID,
		service.AgentIdentityUser,
		toServiceLLMConfig(req),
		req.Validate,
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Personal LLM configuration updated successfully")
}

// UserDeletePersonalLLMConfig removes the current user's personal LLM configuration.
//
//	@Summary		Delete personal LLM configuration
//	@Tags			Agent
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[string]
//	@Router			/v1/agent/llm-config [delete]
func (mgr *AgentMgr) UserDeletePersonalLLMConfig(c *gin.Context) {
	if err := mgr.service.DeletePersonalConfig(
		c.Request.Context(), util.GetToken(c).UserID, service.AgentIdentityUser,
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Personal LLM configuration deleted successfully")
}

// UserTestPersonalLLMConfig validates a personal LLM configuration without storing it.
//
//	@Summary		Test personal LLM configuration
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			request	body		updatePersonalLLMConfigReq	true	"Personal LLM configuration"
//	@Success		200		{object}	resputil.Response[string]
//	@Router			/v1/agent/llm-config/test [post]
func (mgr *AgentMgr) UserTestPersonalLLMConfig(c *gin.Context) {
	req, ok := bindPersonalLLMConfig(c)
	if !ok {
		return
	}
	if err := mgr.service.CheckPersonalConfig(
		c.Request.Context(), util.GetToken(c).UserID, service.AgentIdentityUser, toServiceLLMConfig(req),
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Personal LLM connection succeeded")
}

func bindPersonalLLMConfig(c *gin.Context) (*updatePersonalLLMConfigReq, bool) {
	var req updatePersonalLLMConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid personal LLM configuration"))
		return nil, false
	}
	return &req, true
}

func toServiceLLMConfig(req *updatePersonalLLMConfigReq) *service.LLMConfig {
	return &service.LLMConfig{BaseURL: req.BaseURL, ModelName: req.ModelName, APIKey: req.APIKey}
}

// UserChat starts an Agent turn and streams the response.
//
//	@Summary		Chat with Crater-Agent
//	@Tags			Agent
//	@Accept			json
//	@Produce		text/event-stream
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string		false	"Agent identity scope" Enums(user,admin)
//	@Param			request				body		agentChatReq	true	"Agent chat request"
//	@Success		200					{string}	string
//	@Router			/v1/agent/chat [post]
func (mgr *AgentMgr) UserChat(c *gin.Context) {
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	body, err := readLimitedBody(c.Request.Body, agentChatBodyLimit)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	var req agentChatReq
	if err := json.Unmarshal(body, &req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid Agent chat request"))
		return
	}
	if err := service.ValidateAgentPageContext(req.PageContext, identity); err != nil {
		resputil.HandleError(c, err)
		return
	}
	userID := util.GetToken(c).UserID
	prepared, err := mgr.conversations.PrepareRun(
		c.Request.Context(), userID, identity, req.ConversationID, req.ParentMessageID, req.Mode, req.Message,
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	prepared.IdentityScope = identity
	prepared.PageContext = req.PageContext
	body, err = json.Marshal(prepared)
	if err != nil {
		_ = mgr.conversations.FailRun(c.Request.Context(), userID, identity, prepared.RunID, err)
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "failed to encode Agent chat request"))
		return
	}
	resp, err := mgr.requestAgentRuntime(c, identity, body, "/chat")
	if err != nil {
		_ = mgr.conversations.FailRun(c.Request.Context(), userID, identity, prepared.RunID, err)
		resputil.HandleError(c, err)
		return
	}
	defer resp.Body.Close()
	if err := validateAgentRuntimeResponse(resp); err != nil {
		_ = mgr.conversations.FailRun(c.Request.Context(), userID, identity, prepared.RunID, err)
		resputil.HandleError(c, err)
		return
	}
	copyStreamingResponse(c, resp)
}

type agentChatReq struct {
	ConversationID  string                    `json:"conversationId"`
	ParentMessageID *string                   `json:"parentMessageId"`
	Mode            string                    `json:"mode"`
	Message         *service.AgentUIMessage   `json:"message"`
	PageContext     *service.AgentPageContext `json:"pageContext,omitempty"`
}

type generateAgentTitleReq struct {
	ConversationID string `json:"conversationId"`
	FirstMessage   string `json:"firstMessage"`
}

type generateAgentTitleResp struct {
	Title string `json:"title"`
}

// UserGenerateTitle creates and persists a title for an Agent conversation.
//
//	@Summary		Generate an Agent conversation title
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string				false	"Agent identity scope" Enums(user,admin)
//	@Param			request				body		generateAgentTitleReq	true	"Title generation request"
//	@Success		200					{object}	resputil.Response[generateAgentTitleResp]
//	@Router			/v1/agent/title [post]
func (mgr *AgentMgr) UserGenerateTitle(c *gin.Context) {
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	body, err := readLimitedBody(c.Request.Body, agentTitleMessageLimit)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	var req generateAgentTitleReq
	if err := json.Unmarshal(body, &req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid title request"))
		return
	}
	req.FirstMessage = strings.TrimSpace(req.FirstMessage)
	if req.FirstMessage == "" {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("firstMessage is required"))
		return
	}
	userID := util.GetToken(c).UserID
	if _, err := mgr.conversations.GetMessages(
		c.Request.Context(), userID, identity, req.ConversationID,
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	body, err = json.Marshal(req)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "failed to encode title request"))
		return
	}
	resp, err := mgr.requestAgentRuntime(c, identity, body, "/title")
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	defer resp.Body.Close()
	if err := validateAgentRuntimeResponse(resp); err != nil {
		resputil.HandleError(c, err)
		return
	}
	var result generateAgentTitleResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, agentUpstreamErrorLimit)).Decode(&result); err != nil {
		resputil.HandleError(c, bizerr.BadGateway.BadGateway.Wrap(err, "invalid Crater-Agent title response"))
		return
	}
	result.Title = strings.TrimSpace(result.Title)
	if result.Title == "" {
		resputil.HandleError(c, bizerr.BadGateway.BadGateway.New("Crater-Agent returned an empty title"))
		return
	}
	if err := mgr.conversations.UpdateGeneratedTitle(
		c.Request.Context(), userID, identity, req.ConversationID, result.Title,
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, result)
}

func (mgr *AgentMgr) requestAgentRuntime(
	c *gin.Context,
	identity service.AgentIdentity,
	body []byte,
	endpoint string,
) (*http.Response, error) {
	serviceToken := strings.TrimSpace(os.Getenv(service.AgentInternalTokenEnv))
	contextSigningKey := strings.TrimSpace(os.Getenv(service.AgentContextSigningKeyEnv))
	token := util.GetToken(c)
	contextToken, err := service.SignAgentContext(token.UserID, token.AccountID, identity, contextSigningKey)
	if err != nil {
		return nil, err
	}
	runtimeURL, err := buildAgentRuntimeURL(os.Getenv("CRATER_AGENT_URL"), endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, runtimeURL, bytes.NewReader(body))
	if err != nil {
		return nil, bizerr.Internal.ServiceError.Wrap(err, "failed to build Crater-Agent request")
	}
	req.Header.Set("Authorization", "Bearer "+serviceToken)
	req.Header.Set("X-Crater-Agent-Context", contextToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: agentRuntimeRequestTimeout}).Do(req)
	if err != nil {
		return nil, bizerr.BadGateway.BadGateway.Wrap(err, "Crater-Agent service is unavailable")
	}
	return resp, nil
}

func validateAgentRuntimeResponse(resp *http.Response) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, agentUpstreamErrorLimit))
		return bizerr.BadGateway.BadGateway.New(sanitizeUpstreamError(message))
	}
	return nil
}

func (mgr *AgentMgr) ProxyLLMChatCompletion(c *gin.Context) {
	contextToken := bearerToken(c.GetHeader("Authorization"))
	contextSigningKey := strings.TrimSpace(os.Getenv(service.AgentContextSigningKeyEnv))
	claims, err := service.ParseAgentContext(contextToken, contextSigningKey)
	if err != nil {
		writeOpenAIError(c, http.StatusUnauthorized, "invalid Agent context", "authentication_error")
		return
	}
	body, err := readLimitedBody(c.Request.Body, agentChatBodyLimit)
	if err != nil {
		writeOpenAIError(c, http.StatusRequestEntityTooLarge, "model request is too large", "invalid_request_error")
		return
	}
	cfg, err := mgr.service.ResolveEffectiveConfig(
		c.Request.Context(), claims.UserID, claims.IdentityScope,
	)
	if err != nil {
		writeOpenAIError(c, http.StatusServiceUnavailable, err.Error(), "configuration_error")
		return
	}
	body, err = overrideModel(body, cfg.ModelName)
	if err != nil {
		writeOpenAIError(c, http.StatusBadRequest, "invalid model request", "invalid_request_error")
		return
	}
	req, err := http.NewRequestWithContext(
		c.Request.Context(), http.MethodPost, cfg.GetChatCompletionURL(), bytes.NewReader(body),
	)
	if err != nil {
		writeOpenAIError(c, http.StatusInternalServerError, "failed to build model request", "server_error")
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	client := service.NewLLMHTTPClient(cfg.Source, llmRequestTimeout)
	resp, err := client.Do(req)
	if err != nil {
		writeOpenAIError(c, http.StatusBadGateway, "configured model endpoint is unavailable", "upstream_error")
		return
	}
	defer resp.Body.Close()
	copyStreamingResponse(c, resp)
}

func readLimitedBody(body io.ReadCloser, limit int64) ([]byte, error) {
	reader := io.LimitReader(body, limit+1)
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, bizerr.BadRequest.InvalidRequest.Wrap(err, "failed to read request body")
	}
	if int64(len(payload)) > limit {
		return nil, bizerr.PayloadTooLarge.PayloadTooLarge.New("request body is too large")
	}
	return payload, nil
}

func buildAgentRuntimeURL(raw, endpoint string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", bizerr.ServiceError.ServiceUnavailable.New("CRATER_AGENT_URL is not configured")
	}
	parsed, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", bizerr.ServiceError.ServiceUnavailable.New("CRATER_AGENT_URL is invalid")
	}
	return parsed.String() + endpoint, nil
}

func overrideModel(body []byte, modelName string) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	payload["model"] = modelName
	return json.Marshal(payload)
}

func copyStreamingResponse(c *gin.Context, resp *http.Response) {
	for _, header := range []string{"Content-Type", "Cache-Control", "X-Vercel-AI-UI-Message-Stream"} {
		if value := resp.Header.Get(header); value != "" {
			c.Header(header, value)
		}
	}
	c.Status(resp.StatusCode)
	buffer := make([]byte, agentStreamBufferSize)
	for {
		count, err := resp.Body.Read(buffer)
		if count > 0 {
			_, _ = c.Writer.Write(buffer[:count])
			c.Writer.Flush()
		}
		if err != nil {
			return
		}
	}
}

func writeOpenAIError(c *gin.Context, status int, message, errorType string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"message": message, "type": errorType}})
}

func bearerToken(header string) string {
	prefix := "Bearer "
	if len(header) <= len(prefix) || subtle.ConstantTimeCompare([]byte(header[:len(prefix)]), []byte(prefix)) != 1 {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func sanitizeUpstreamError(message []byte) string {
	trimmed := strings.TrimSpace(string(message))
	if trimmed == "" {
		return "Crater-Agent returned an error"
	}
	return "Crater-Agent returned an error: " + trimmed
}
