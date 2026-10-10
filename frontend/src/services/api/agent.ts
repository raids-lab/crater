import type { UIMessage } from 'ai'

import { apiClient, apiRequest, apiV1Get } from '@/services/client'

import { IResponse } from '../types'

export type LLMConfigSource = 'personal' | 'platform' | ''
export type AgentIdentity = 'user' | 'admin'

export const agentIdentityHeader = 'X-Crater-Agent-Identity'

const identityOptions = (identity: AgentIdentity) => ({
  headers: { [agentIdentityHeader]: identity },
})

const agentRequest = <T>(
  method: 'delete' | 'get' | 'post' | 'put',
  url: string,
  identity: AgentIdentity,
  json?: unknown
) =>
  apiRequest(() =>
    apiClient[method](`v1/${url}`, { ...identityOptions(identity), json }).json<IResponse<T>>()
  )

export interface PersonalLLMConfigStatus {
  baseUrl: string
  modelName: string
  apiKey: string
  personalConfigured: boolean
  platformConfigured: boolean
  effectiveSource: LLMConfigSource
}

export interface PersonalLLMConfigInput {
  baseUrl: string
  modelName: string
  apiKey: string
  validate?: boolean
}

export interface AgentConversation {
  id: string
  title: string
  customTitle: boolean
  createdAt: number
  updatedAt: number
}

export interface AdminAgentConversation extends AgentConversation {
  userId: number
  username: string
  nickname: string
  identityScope: AgentIdentity
  messageCount: number
}

export interface AdminAgentConversationPage {
  items: AdminAgentConversation[]
  total: number
  page: number
  pageSize: number
}

export interface AdminAgentConversationQuery {
  page: number
  pageSize: number
  search?: string
  identityScope?: AgentIdentity
}

export const apiGetPersonalLLMConfig = () =>
  agentRequest<PersonalLLMConfigStatus>('get', 'agent/llm-config', 'user')

export const apiUpdatePersonalLLMConfig = (data: PersonalLLMConfigInput) =>
  agentRequest<string>('put', 'agent/llm-config', 'user', data)

export const apiTestPersonalLLMConfig = (data: PersonalLLMConfigInput) =>
  agentRequest<string>('post', 'agent/llm-config/test', 'user', data)

export const apiDeletePersonalLLMConfig = () =>
  agentRequest<string>('delete', 'agent/llm-config', 'user')

export const apiGetAgentConversations = (identity: AgentIdentity) =>
  agentRequest<AgentConversation[]>('get', 'agent/conversations', identity)

export const apiGetAgentConversationMessages = (identity: AgentIdentity, conversationID: string) =>
  agentRequest<UIMessage[]>('get', `agent/conversations/${conversationID}/messages`, identity)

export const apiRenameAgentConversation = (
  identity: AgentIdentity,
  conversationID: string,
  title: string
) => agentRequest<string>('put', `agent/conversations/${conversationID}`, identity, { title })

export const apiDeleteAgentConversation = (identity: AgentIdentity, conversationID: string) =>
  agentRequest<string>('delete', `agent/conversations/${conversationID}`, identity)

export const apiAdminGetAgentConversations = (params: AdminAgentConversationQuery) => {
  const searchParams: Record<string, string | number> = {
    page: params.page,
    pageSize: params.pageSize,
  }
  if (params.search) searchParams.search = params.search
  if (params.identityScope) searchParams.identityScope = params.identityScope
  return apiV1Get<IResponse<AdminAgentConversationPage>>('admin/agent/conversations', {
    searchParams,
  })
}

export const apiAdminGetAgentConversationMessages = (conversationID: string) =>
  apiV1Get<IResponse<UIMessage[]>>(`admin/agent/conversations/${conversationID}/messages`)
