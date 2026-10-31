package handler

import (
	"crypto/subtle"
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

type renameAgentConversationReq struct {
	Title string `json:"title" binding:"required,max=320"`
}

type saveAgentToolOutputReq struct {
	MessageID  string          `json:"messageId" binding:"required,max=128"`
	ToolCallID string          `json:"toolCallId" binding:"required,max=128"`
	State      string          `json:"state" binding:"required,oneof=output-available output-error"`
	Output     json.RawMessage `json:"output"`
	ErrorText  string          `json:"errorText" binding:"omitempty,max=2048"`
}

type adminListAgentConversationsReq struct {
	Page          int                   `form:"page" binding:"omitempty,min=1"`
	PageSize      int                   `form:"pageSize" binding:"omitempty,min=1,max=100"`
	Search        string                `form:"search" binding:"omitempty,max=120"`
	IdentityScope service.AgentIdentity `form:"identityScope" binding:"omitempty,oneof=user admin"`
	UserID        uint                  `form:"userId" binding:"omitempty,min=1"`
}

type adminListAgentSubjectsReq struct {
	Page          int                   `form:"page" binding:"omitempty,min=1"`
	PageSize      int                   `form:"pageSize" binding:"omitempty,min=1,max=100"`
	Search        string                `form:"search" binding:"omitempty,max=120"`
	IdentityScope service.AgentIdentity `form:"identityScope" binding:"omitempty,oneof=user admin"`
}

// UserListAgentConversations lists conversations for the active user and Agent identity.
//
//	@Summary		List Agent conversations
//	@Tags			Agent
//	@Produce		json
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string	false	"Agent identity scope" Enums(user,admin)
//	@Success		200					{object}	resputil.Response[[]service.AgentConversationSummary]
//	@Router			/v1/agent/conversations [get]
func (mgr *AgentMgr) UserListAgentConversations(c *gin.Context) {
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	items, err := mgr.conversations.ListConversations(
		c.Request.Context(), util.GetToken(c).UserID, identity,
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, items)
}

// UserGetAgentConversationMessages returns active messages from an owned conversation.
//
//	@Summary		Get Agent conversation messages
//	@Tags			Agent
//	@Produce		json
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string	false	"Agent identity scope" Enums(user,admin)
//	@Param			id						path		string	true	"Conversation ID"
//	@Success		200						{object}	resputil.Response[[]service.AgentUIMessage]
//	@Router			/v1/agent/conversations/{id}/messages [get]
func (mgr *AgentMgr) UserGetAgentConversationMessages(c *gin.Context) {
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	items, err := mgr.conversations.GetMessages(
		c.Request.Context(), util.GetToken(c).UserID, identity, c.Param("id"),
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, items)
}

// UserCancelAgentConversationRun cancels the active generation for an owned conversation.
//
//	@Summary		Cancel the active Agent response
//	@Tags			Agent
//	@Produce		json
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string	false	"Agent identity scope" Enums(user,admin)
//	@Param			id						path		string	true	"Conversation ID"
//	@Success		200						{object}	resputil.Response[bool]
//	@Router			/v1/agent/conversations/{id}/cancel [post]
func (mgr *AgentMgr) UserCancelAgentConversationRun(c *gin.Context) {
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	canceled, err := mgr.conversations.CancelActiveRun(
		c.Request.Context(), util.GetToken(c).UserID, identity, c.Param("id"),
	)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, canceled)
}

// UserSaveAgentToolOutput persists a user-confirmed client-side tool result.
//
//	@Summary		Save an Agent tool result
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string				false	"Agent identity scope" Enums(user,admin)
//	@Param			id						path		string				true	"Conversation ID"
//	@Param			request					body		saveAgentToolOutputReq	true	"Resolved tool result"
//	@Success		200						{object}	resputil.Response[string]
//	@Router			/v1/agent/conversations/{id}/tool-output [post]
func (mgr *AgentMgr) UserSaveAgentToolOutput(c *gin.Context) {
	var req saveAgentToolOutputReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid Agent tool output"))
		return
	}
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := mgr.conversations.SaveToolOutput(
		c.Request.Context(),
		util.GetToken(c).UserID,
		identity,
		c.Param("id"),
		&service.AgentToolOutput{
			MessageID: req.MessageID, ToolCallID: req.ToolCallID, State: req.State,
			Output: req.Output, ErrorText: req.ErrorText,
		},
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Agent tool output saved successfully")
}

// UserRenameAgentConversation updates the title of an owned conversation.
//
//	@Summary		Rename an Agent conversation
//	@Tags			Agent
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string				false	"Agent identity scope" Enums(user,admin)
//	@Param			id						path		string				true	"Conversation ID"
//	@Param			request				body		renameAgentConversationReq	true	"New conversation title"
//	@Success		200					{object}	resputil.Response[string]
//	@Router			/v1/agent/conversations/{id} [put]
func (mgr *AgentMgr) UserRenameAgentConversation(c *gin.Context) {
	var req renameAgentConversationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid conversation title"))
		return
	}
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := mgr.conversations.RenameConversation(
		c.Request.Context(), util.GetToken(c).UserID, identity, c.Param("id"), req.Title,
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Agent conversation renamed successfully")
}

// UserDeleteAgentConversation soft-deletes an owned conversation.
//
//	@Summary		Delete an Agent conversation
//	@Tags			Agent
//	@Produce		json
//	@Security		Bearer
//	@Param			X-Crater-Agent-Identity	header		string	false	"Agent identity scope" Enums(user,admin)
//	@Param			id						path		string	true	"Conversation ID"
//	@Success		200						{object}	resputil.Response[string]
//	@Router			/v1/agent/conversations/{id} [delete]
func (mgr *AgentMgr) UserDeleteAgentConversation(c *gin.Context) {
	identity, err := resolveAgentIdentity(c)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if err := mgr.conversations.DeleteConversation(
		c.Request.Context(), util.GetToken(c).UserID, identity, c.Param("id"),
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Agent conversation deleted successfully")
}

func (mgr *AgentMgr) CompleteAgentRun(c *gin.Context) {
	claims, ok := getInternalAgentClaims(c)
	if !ok {
		return
	}
	body, err := readLimitedBody(c.Request.Body, agentChatBodyLimit)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	var completion service.AgentRunCompletion
	if err := json.Unmarshal(body, &completion); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid Agent run completion"))
		return
	}
	if err := mgr.conversations.CompleteRun(
		c.Request.Context(), claims.UserID, claims.IdentityScope, c.Param("id"), completion,
	); err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, "Agent run completed successfully")
}

// AdminListAgentSubjects lists users with Agent history, separated by identity scope.
//
//	@Summary		List users with Agent conversation history
//	@Tags			Agent Admin
//	@Produce		json
//	@Security		Bearer
//	@Param			page			query		int		false	"Page number" minimum(1)
//	@Param			pageSize		query		int		false	"Items per page" minimum(1) maximum(100)
//	@Param			search			query		string	false	"Username, nickname, or user ID"
//	@Param			identityScope	query		string	false	"Agent identity scope" Enums(user,admin)
//	@Success		200				{object}	resputil.Response[service.AdminAgentSubjectPage]
//	@Router			/v1/admin/agent/subjects [get]
func (mgr *AgentMgr) AdminListAgentSubjects(c *gin.Context) {
	var req adminListAgentSubjectsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid Agent user query"))
		return
	}
	items, err := mgr.conversations.ListAdminSubjects(c.Request.Context(), service.AdminAgentSubjectQuery{
		Page: req.Page, PageSize: req.PageSize, Search: req.Search, IdentityScope: req.IdentityScope,
	})
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, items)
}

// AdminListAgentConversations lists searchable Agent conversations for administrators.
//
//	@Summary		List all Agent conversations
//	@Tags			Agent Admin
//	@Produce		json
//	@Security		Bearer
//	@Param			page			query		int		false	"Page number" minimum(1)
//	@Param			pageSize		query		int		false	"Items per page" minimum(1) maximum(100)
//	@Param			search			query		string	false	"Username, nickname, user ID, or title"
//	@Param			identityScope	query		string	false	"Agent identity scope" Enums(user,admin)
//	@Param			userId			query		int		false	"User ID" minimum(1)
//	@Success		200				{object}	resputil.Response[service.AdminAgentConversationPage]
//	@Router			/v1/admin/agent/conversations [get]
func (mgr *AgentMgr) AdminListAgentConversations(c *gin.Context) {
	var req adminListAgentConversationsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, "invalid Agent conversation query"))
		return
	}
	items, err := mgr.conversations.ListAllConversations(c.Request.Context(), service.AdminAgentConversationQuery{
		Page:          req.Page,
		PageSize:      req.PageSize,
		Search:        req.Search,
		IdentityScope: req.IdentityScope,
		UserID:        req.UserID,
	})
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, items)
}

// AdminGetAgentConversationMessages returns messages from any Agent conversation.
//
//	@Summary		Get Agent conversation messages as an administrator
//	@Tags			Agent Admin
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		string	true	"Conversation ID"
//	@Success		200	{object}	resputil.Response[[]service.AgentUIMessage]
//	@Router			/v1/admin/agent/conversations/{id}/messages [get]
func (mgr *AgentMgr) AdminGetAgentConversationMessages(c *gin.Context) {
	items, err := mgr.conversations.GetMessagesForAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	resputil.Success(c, items)
}

func constantTimeTokenEqual(received, expected string) bool {
	if received == "" || expected == "" || len(received) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(received), []byte(expected)) == 1
}
