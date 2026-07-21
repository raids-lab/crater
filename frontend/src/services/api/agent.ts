import { getDefaultStore } from 'jotai'

import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'

import { ACCESS_TOKEN_KEY } from '@/utils/store'
import { configAPIBaseAtom } from '@/utils/store/config'

import type { IResponse } from '../types'

// ──────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────

export type AgentSurface = 'user' | 'admin'

export interface AgentSSEEvent {
  event:
    | 'agent_run_started'
    | 'agent_status'
    | 'agent_handoff'
    | 'thinking'
    | 'tool_call'
    | 'tool_result'
    | 'message'
    | 'confirmation_required'
    | 'tool_call_started'
    | 'tool_call_completed'
    | 'tool_call_confirmation_required'
    | 'parameter_review'
    | 'final_answer'
    | 'error'
    | 'done'
  data: unknown
}

export type AgentOrchestrationMode = 'single_agent' | 'ask'

export interface AgentSession {
  sessionId: string
  title: string
  source?: 'chat' | 'ops_audit' | 'system'
  messageCount: number
  lastOrchestrationMode?: AgentOrchestrationMode
  pinnedAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface AgentTurn {
  id: number
  turnId: string
  sessionId: string
  requestId?: string
  orchestrationMode: AgentOrchestrationMode
  rootAgentId?: string
  status: string
  finalMessageId?: number | null
  metadata?: unknown
  startedAt: string
  endedAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface AgentEvent {
  id: number
  turnId: string
  sessionId: string
  agentId: string
  parentAgentId?: string
  agentRole: string
  eventType: string
  eventStatus?: string
  title?: string
  content?: string
  metadata?: unknown
  sequence: number
  startedAt?: string
  endedAt?: string
  createdAt: string
}

export interface AgentConfigSummary {
  defaultOrchestrationMode: 'single_agent'
}

export interface AgentMessage {
  id: string
  sessionId?: string
  role: 'user' | 'assistant'
  content: string
  metadata?: unknown
  createdAt: string
}

export interface AgentToolCall {
  id: string
  turnId?: string
  toolCallId?: string
  agentId?: string
  agentRole?: string
  source?: 'backend' | 'local'
  toolName: string
  toolArgs?: unknown
  toolResult?: unknown
  resultStatus: string
  userConfirmed?: boolean | null
  createdAt: string
}

export interface AgentConfirmationFieldOption {
  value: string
  label: string
}

export interface AgentConfirmationField {
  key: string
  label: string
  type: 'text' | 'textarea' | 'number' | 'select'
  required?: boolean
  description?: string
  placeholder?: string
  defaultValue?: unknown
  options?: AgentConfirmationFieldOption[]
}

export interface AgentConfirmationForm {
  title?: string
  description?: string
  submitLabel?: string
  fields?: AgentConfirmationField[]
}

export interface AgentConfirmResponseData {
  status: string
  result?: unknown
  message?: string
}

type AgentPageContext = {
  route?: string
  url: string
  jobName?: string
  jobStatus?: string
  nodeName?: string
  entryPoint?: 'default' | 'node_analysis'
  surface?: AgentSurface
}

export interface AgentChatRequest {
  sessionId: string | null
  requestId?: string | null
  message: string
  orchestrationMode?: 'single_agent'
  pageContext: AgentPageContext
  clientContext?: {
    locale?: string
    timezone?: string
  }
}

export interface AgentAskRequest {
  sessionId: string | null
  requestId?: string | null
  message: string
  jobName?: string
  pageContext: AgentChatRequest['pageContext']
  clientContext?: AgentChatRequest['clientContext']
}

export interface AgentResumeRequest {
  confirmId: string
}

export interface ParameterReviewPayload {
  reviewId: string
  scenario: string
  complexity: 'simple' | 'complex'
  step: number
  totalSteps: number
  title: string
  description: string
  parameters: Array<{
    key: string
    label: string
    value: unknown
    source: 'recommended' | 'default' | 'user'
    editable: boolean
    type: 'text' | 'number' | 'select' | 'textarea'
    options?: Array<{ label: string; value: string }>
    constraints?: { min?: number; max?: number }
    hint?: string
  }>
}

// ──────────────────────────────────────────────
// SSE streaming connection
// ──────────────────────────────────────────────

function getAgentStreamBaseURL() {
  const store = getDefaultStore()
  const apiBase = store.get(configAPIBaseAtom)
  return `${apiBase ?? ''}/api`
}

function getAgentStreamError(parsed: unknown, fallback: string) {
  if (typeof parsed === 'string') return parsed
  if (typeof parsed === 'object' && parsed !== null) {
    const body = parsed as { message?: string; msg?: string }
    return body.message || body.msg || fallback
  }
  return fallback
}

function connectAgentStream<TBody>(
  path: string,
  body: TBody,
  options: {
    noBodyMessage: string
    fallbackStreamError: string
    onEvent: (event: AgentSSEEvent) => void
    onError: (error: Error) => void
    onDone: () => void
    onSessionId?: (sessionId: string) => void
  }
): AbortController {
  const controller = new AbortController()
  const token = localStorage.getItem(ACCESS_TOKEN_KEY)
  let doneDispatched = false

  const dispatchDone = () => {
    if (doneDispatched) return
    doneDispatched = true
    options.onDone()
  }

  const dispatchStreamError = (error: Error) => {
    if (doneDispatched) return
    doneDispatched = true
    options.onError(error)
  }

  fetch(`${getAgentStreamBaseURL()}/v1/${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
    signal: controller.signal,
  })
    .then(async (response) => {
      if (!response.ok) {
        // Try to parse error body
        let msg = `HTTP ${response.status}`
        try {
          const errBody = await response.json()
          msg = errBody?.msg || errBody?.message || msg
        } catch {
          // ignore parse errors
        }
        dispatchStreamError(new Error(msg))
        return
      }

      if (!response.body) {
        dispatchStreamError(new Error(options.noBodyMessage))
        return
      }

      const responseSessionId = response.headers.get('X-Agent-Session-ID')
      if (responseSessionId) {
        options.onSessionId?.(responseSessionId)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      const processBuffer = () => {
        // SSE blocks are separated by double newlines
        const blocks = buffer.split('\n\n')
        // Keep the last (potentially incomplete) block in the buffer
        buffer = blocks.pop() ?? ''

        for (const block of blocks) {
          if (!block.trim()) continue

          let eventType = 'message'
          let dataStr = ''

          for (const line of block.split('\n')) {
            if (line.startsWith('event:')) {
              eventType = line.slice(6).trim()
            } else if (line.startsWith('data:')) {
              dataStr = line.slice(5).trim()
            }
          }

          if (!dataStr) continue

          let parsed: unknown
          try {
            parsed = JSON.parse(dataStr)
          } catch {
            parsed = dataStr
          }

          const sseEvent: AgentSSEEvent = {
            event: eventType as AgentSSEEvent['event'],
            data: parsed,
          }

          if (sseEvent.event === 'done') {
            dispatchDone()
            return
          }

          if (sseEvent.event === 'error') {
            dispatchStreamError(new Error(getAgentStreamError(parsed, options.fallbackStreamError)))
            return
          }

          options.onEvent(sseEvent)
        }
      }

      const pump = async (): Promise<void> => {
        try {
          const { done, value } = await reader.read()
          if (done) {
            // Flush remaining buffer
            if (buffer.trim()) {
              processBuffer()
            }
            dispatchDone()
            return
          }
          buffer += decoder.decode(value, { stream: true })
          processBuffer()
          return pump()
        } catch (err) {
          if ((err as Error)?.name === 'AbortError') {
            // Silently ignore user-initiated aborts
            return
          }
          dispatchStreamError(err instanceof Error ? err : new Error(String(err)))
        }
      }

      await pump()
    })
    .catch((err) => {
      if (err?.name === 'AbortError') return
      dispatchStreamError(err instanceof Error ? err : new Error(String(err)))
    })

  return controller
}

/**
 * Connect to the agent chat endpoint via SSE (fetch + ReadableStream).
 * Returns an AbortController that can be used to cancel the stream.
 */
export function connectAgentChat(
  sessionId: string | null,
  requestId: string,
  message: string,
  pageContext: AgentPageContext,
  orchestrationMode: 'single_agent',
  clientContext: { locale?: string; timezone?: string } | undefined,
  onEvent: (event: AgentSSEEvent) => void,
  onError: (error: Error) => void,
  onDone: () => void,
  onSessionId?: (sessionId: string) => void
): AbortController {
  return connectAgentStream<AgentChatRequest>(
    'agent/chat',
    {
      sessionId,
      requestId,
      message,
      pageContext,
      orchestrationMode,
      clientContext,
    },
    {
      noBodyMessage: 'No response body for SSE stream',
      fallbackStreamError: 'Agent error',
      onEvent,
      onError,
      onDone,
      onSessionId,
    }
  )
}

export function connectAgentAskStream(
  sessionId: string | null,
  requestId: string,
  message: string,
  pageContext: AgentChatRequest['pageContext'],
  clientContext: AgentChatRequest['clientContext'] | undefined,
  jobName: string | undefined,
  onEvent: (event: AgentSSEEvent) => void,
  onError: (error: Error) => void,
  onDone: () => void,
  onSessionId?: (sessionId: string) => void
): AbortController {
  return connectAgentStream<AgentAskRequest>(
    'agent/ask/stream',
    {
      sessionId,
      requestId,
      message,
      jobName,
      pageContext,
      clientContext,
    },
    {
      noBodyMessage: 'No response body for ask stream',
      fallbackStreamError: 'ask error',
      onEvent,
      onError,
      onDone,
      onSessionId,
    }
  )
}

export function connectAgentResume(
  confirmId: string,
  onEvent: (event: AgentSSEEvent) => void,
  onError: (error: Error) => void,
  onDone: () => void,
  onSessionId?: (sessionId: string) => void
): AbortController {
  return connectAgentStream<AgentResumeRequest>(
    'agent/chat/resume',
    { confirmId },
    {
      noBodyMessage: 'No response body for SSE stream',
      fallbackStreamError: 'Agent error',
      onEvent,
      onError,
      onDone,
      onSessionId,
    }
  )
}

// ──────────────────────────────────────────────
// REST endpoints
// ──────────────────────────────────────────────

/**
 * Confirm or reject a pending write-operation.
 */
export const apiConfirmAction = (
  confirmId: string,
  confirmed: boolean,
  payload?: Record<string, unknown>
) =>
  apiV1Post<IResponse<AgentConfirmResponseData>>('agent/chat/confirm', {
    confirmId,
    confirmed,
    payload,
  })

/**
 * List agent sessions for the current user and current surface.
 */
export const apiListSessions = (surface?: AgentSurface) =>
  apiV1Get<IResponse<AgentSession[]>>('agent/sessions', {
    searchParams: surface ? { surface } : undefined,
  })

export const apiPinSession = (sessionId: string, pinned: boolean) =>
  apiV1Put<IResponse<AgentSession>>(`agent/sessions/${encodeURIComponent(sessionId)}/pin`, {
    pinned,
  })

export const apiRenameSession = (sessionId: string, title: string) =>
  apiV1Put<IResponse<AgentSession>>(`agent/sessions/${encodeURIComponent(sessionId)}/title`, {
    title,
  })

export const apiDeleteSession = (sessionId: string) =>
  apiV1Delete<IResponse<string>>(`agent/sessions/${encodeURIComponent(sessionId)}`)

/**
 * Fetch the message history for a given session.
 */
export const apiGetSessionMessages = (sessionId: string) =>
  apiV1Get<IResponse<AgentMessage[]>>(`agent/sessions/${encodeURIComponent(sessionId)}/messages`)

export const apiGetSessionToolCalls = (sessionId: string) =>
  apiV1Get<IResponse<AgentToolCall[]>>(`agent/sessions/${encodeURIComponent(sessionId)}/tool-calls`)

export const apiGetSessionTurns = (sessionId: string) =>
  apiV1Get<IResponse<AgentTurn[]>>(`agent/sessions/${encodeURIComponent(sessionId)}/turns`)

export const apiGetTurnEvents = (turnId: string) =>
  apiV1Get<IResponse<AgentEvent[]>>(`agent/turns/${encodeURIComponent(turnId)}/events`)

export const apiGetAgentConfigSummary = () =>
  apiV1Get<IResponse<AgentConfigSummary>>('agent/config-summary')

/**
 * Submit parameter review result (confirm or modify).
 */
export const apiParameterUpdate = (
  sessionId: string,
  reviewId: string,
  action: 'confirm' | 'modify',
  parameters: Record<string, unknown>
) =>
  apiV1Post<IResponse<{ status: string }>>('agent/chat/parameter-update', {
    session_id: sessionId,
    review_id: reviewId,
    action,
    parameters,
  })

// ──────────────────────────────────────────────
// Feedback
// ──────────────────────────────────────────────

export interface AgentFeedback {
  id: number
  sessionId: string
  userId: number
  accountId: number
  targetType: 'message' | 'turn'
  targetId: string
  rating: 1 | -1
  tags?: string[]
  dimensions?: Record<string, number>
  comment?: string
  status: 'draft' | 'submitted'
  submittedAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface FeedbackUpsertRequest {
  sessionId: string
  targetType: 'message' | 'turn'
  targetId: string
  rating: 1 | -1
  tags?: string[]
  dimensions?: Record<string, number>
  comment?: string
}

export const apiUpsertFeedback = (req: FeedbackUpsertRequest) =>
  apiV1Put<IResponse<AgentFeedback>>('agent/feedbacks', req)

export const apiSubmitFeedback = (sessionId: string, targetType: string, targetId: string) =>
  apiV1Post<IResponse<AgentFeedback>>('agent/feedbacks/submit', {
    sessionId,
    targetType,
    targetId,
  })

export const apiQuickSubmitFeedback = (req: {
  sessionId: string
  targetType: 'message' | 'turn'
  targetId: string
  rating: 1 | -1
  tags?: string[]
  dimensions?: Record<string, number>
  comment?: string
}) => apiV1Post<IResponse<AgentFeedback>>('agent/feedbacks/quick-submit', req)

export const apiEnrichFeedback = (req: {
  sessionId: string
  targetType: 'message' | 'turn'
  targetId: string
  tags?: string[]
  dimensions?: Record<string, number>
  comment?: string
}) => apiV1Put<IResponse<AgentFeedback>>('agent/feedbacks/enrich', req)

export const apiListFeedbacks = (sessionId: string) =>
  apiV1Get<IResponse<AgentFeedback[]>>(`agent/feedbacks?sessionId=${encodeURIComponent(sessionId)}`)
