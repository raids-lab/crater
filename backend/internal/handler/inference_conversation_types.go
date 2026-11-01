package handler

import inferencesvc "github.com/raids-lab/crater/internal/service/inference"

type KthenaConversationMessageReq = inferencesvc.ConversationMessageReq
type KthenaConversationCreateReq = inferencesvc.ConversationCreateReq
type KthenaConversationUpdateReq = inferencesvc.ConversationUpdateReq
type KthenaConversationListReq = inferencesvc.ConversationListReq
type KthenaConversationTurnReq = inferencesvc.ConversationTurnReq
type KthenaConversationMessageResp = inferencesvc.ConversationMessageResp
type KthenaConversationResp = inferencesvc.ConversationResp
type KthenaConversationTurnResp = inferencesvc.ConversationTurnResp
type KthenaProxyReq = inferencesvc.ProxyReq
type ChatMessage = inferencesvc.ChatMessage
