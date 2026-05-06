import { apiV1Get, apiV1Post } from '@/services/client'
import { IResponse } from '@/services/types'

export type AgentAuditSessionSource = 'chat' | 'ops_audit' | 'system' | 'benchmark'
export type AgentAuditToolCallSource = 'backend' | 'local' | 'benchmark'
export type AgentQualityEvalScope = 'session' | 'turn'
export type AgentQualityEvalType = 'full' | 'dialogue' | 'task'

export interface AgentAuditSessionSummary {
  chat: number
  opsAudit: number
  system: number
  benchmark: number
  total: number
}

export interface AgentAuditSessionListItem {
  sessionId: string
  title: string
  source: AgentAuditSessionSource
  userId: number
  username?: string
  nickname?: string
  accountId: number
  accountName?: string
  accountNickname?: string
  messageCount: number
  toolCallCount: number
  turnCount: number
  lastOrchestrationMode?: string
  orchestrationModes?: string[]
  pinnedAt?: string | null
  latestEvalId?: number | null
  latestEvalScope?: AgentQualityEvalScope | ''
  latestEvalType?: AgentQualityEvalType | ''
  latestEvalStatus?: '' | 'pending' | 'running' | 'completed' | 'failed'
  latestEvalCompletedAt?: string | null
  feedbackRating?: number | null
  hasFeedback: boolean
  createdAt: string
  updatedAt: string
}

export interface AgentAuditSessionListResult {
  total: number
  items: AgentAuditSessionListItem[]
  summary: AgentAuditSessionSummary
}

export interface AgentAuditMessage {
  id: number
  sessionId: string
  role: string
  content: string
  toolCalls?: unknown
  toolCallId?: string
  toolName?: string
  metadata?: unknown
  createdAt: string
}

export interface AgentAuditToolCall {
  id: number
  sessionId: string
  turnId?: string
  messageId?: number | null
  toolCallId?: string
  agentId?: string
  parentEventId?: number | null
  agentRole?: string
  source?: AgentAuditToolCallSource
  toolName: string
  toolArgs?: unknown
  toolResult?: unknown
  resultStatus: string
  executionBackend?: string
  sandboxJobName?: string
  scriptName?: string
  resultArtifactRef?: string
  egressDomains?: string[]
  latencyMs?: number
  tokenCount?: number
  userConfirmed?: boolean | null
  createdAt: string
}

export interface AgentAuditTurn {
  id: number
  turnId: string
  sessionId: string
  requestId?: string
  orchestrationMode: string
  rootAgentId?: string
  status: string
  finalMessageId?: number | null
  metadata?: unknown
  startedAt: string
  endedAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface AgentAuditEvent {
  id: number
  turnId: string
  sessionId: string
  agentId?: string
  parentAgentId?: string
  agentRole?: string
  eventType: string
  eventStatus?: string
  title?: string
  content?: string
  metadata?: unknown
  sequence: number
  startedAt?: string | null
  endedAt?: string | null
  createdAt: string
}

type SearchParams = Record<string, string | number | boolean>

export async function apiAdminListAgentAuditSessions(params: {
  source?: AgentAuditSessionSource | 'all'
  keyword?: string
  from?: string // RFC3339
  to?: string // RFC3339
  hasEval?: '' | 'yes' | 'no'
  limit?: number
  offset?: number
}) {
  const searchParams = Object.fromEntries(
    Object.entries(params).filter(
      ([, value]) => value !== undefined && value !== '' && value !== 'all'
    )
  ) as SearchParams

  return apiV1Get<AgentAuditSessionListResult>('admin/agent/sessions', {
    searchParams,
  })
}

export async function apiAdminGetAgentAuditSessionDetail(sessionId: string) {
  return apiV1Get<AgentAuditSessionListItem>(`admin/agent/sessions/${sessionId}/detail`)
}

export async function apiAdminGetAgentAuditSessionMessages(sessionId: string) {
  return apiV1Get<AgentAuditMessage[]>(`admin/agent/sessions/${sessionId}/messages`)
}

export async function apiAdminGetAgentAuditSessionToolCalls(sessionId: string) {
  return apiV1Get<AgentAuditToolCall[]>(`admin/agent/sessions/${sessionId}/tool-calls`)
}

export async function apiAdminGetAgentAuditSessionTurns(sessionId: string) {
  return apiV1Get<AgentAuditTurn[]>(`admin/agent/sessions/${sessionId}/turns`)
}

export async function apiAdminGetAgentAuditTurnEvents(turnId: string) {
  return apiV1Get<AgentAuditEvent[]>(`admin/agent/turns/${turnId}/events`)
}

// ─── Quality eval ──────────────────────────────────────────────────────────

export interface AgentQualityEval {
  id: number
  sessionId: string
  turnId?: string
  evalScope?: AgentQualityEvalScope
  evalType?: AgentQualityEvalType
  targetId?: string
  feedbackId?: number | null
  triggerSource: 'feedback' | 'offline_batch' | 'manual' | string
  evalStatus: 'pending' | 'running' | 'completed' | 'failed'
  chatScores?: Record<string, unknown>
  chainScores?: Record<string, unknown>
  chatModel?: string
  chainModel?: string
  summary?: string
  rawChatResp?: unknown
  rawChainResp?: unknown
  artifactPath?: string
  metadata?: Record<string, unknown>
  createdAt: string
  completedAt?: string | null
  updatedAt: string
}

export async function apiAdminListSessionQualityEvals(sessionId: string, limit = 20) {
  return apiV1Get<AgentQualityEval[]>('admin/agent/quality-evals', {
    searchParams: { sessionId, limit },
  })
}

export interface TriggerAgentQualityEvalParams {
  turnId?: string
  evalScope?: AgentQualityEvalScope
  evalType?: AgentQualityEvalType
  dialogueModelRole?: string
  taskModelRole?: string
}

export async function apiAdminTriggerSessionQualityEval(
  sessionId: string,
  params: TriggerAgentQualityEvalParams = {}
) {
  return apiV1Post<{
    evalId: number
    sessionId: string
    turnId?: string
    evalScope: AgentQualityEvalScope
    evalType: AgentQualityEvalType
    evalStatus: string
    triggerSource: string
    createdAt: string
  }>(`admin/agent/sessions/${sessionId}/trigger-eval`, {
    turnId: params.turnId ?? '',
    evalScope: params.evalScope ?? 'session',
    evalType: params.evalType ?? 'full',
    dialogueModelRole: params.dialogueModelRole ?? '',
    taskModelRole: params.taskModelRole ?? '',
  })
}

// ─── Feedback ──────────────────────────────────────────────────────────────

export interface AgentFeedback {
  id: number
  sessionId: string
  userId: number
  accountId: number
  targetType: 'message' | 'turn'
  targetId: string
  rating: number // 1 | -1
  tags?: string[] | Record<string, unknown>
  dimensions?: Record<string, number>
  comment?: string
  status: 'draft' | 'submitted'
  submittedAt?: string | null
  enrichedAt?: string | null
  createdAt: string
  updatedAt: string
}

export async function apiAdminListSessionFeedbacks(sessionId: string) {
  return apiV1Get<AgentFeedback[]>(`admin/agent/sessions/${sessionId}/feedbacks`)
}

export type AgentAuditSessionListResponse = IResponse<AgentAuditSessionListResult>
