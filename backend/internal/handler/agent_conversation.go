package handler

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

type renameAgentConversationReq struct {
	Title string `json:"title" binding:"required,max=320"`
}

type adminListAgentConversationsReq struct {
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
	secret := strings.TrimSpace(os.Getenv("CRATER_AGENT_INTERNAL_TOKEN"))
	if !constantTimeTokenEqual(bearerToken(c.GetHeader("Authorization")), secret) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Agent service token"})
		return
	}
	claims, err := service.ParseAgentContext(c.GetHeader("X-Crater-Agent-Context"), secret)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Agent context"})
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
