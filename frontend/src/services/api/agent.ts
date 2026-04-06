import { getDefaultStore } from 'jotai'

import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'

import { ACCESS_TOKEN_KEY } from '@/utils/store'
import { configAPIBaseAtom } from '@/utils/store/config'

import type { IResponse } from '../types'

// ──────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────

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
    | 'resource_suggestion'
    | 'pipeline_report'
    | 'batch_confirmation'
    | 'final_answer'
    | 'error'
    | 'done'
  data: unknown
}

export interface AgentSession {
  sessionId: string
  title: string
  messageCount: number
  lastOrchestrationMode?: 'single_agent' | 'multi_agent'
  pinnedAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface AgentTurn {
  id: number
  turnId: string
  sessionId: string
  requestId?: string
  orchestrationMode: 'single_agent' | 'multi_agent'
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
  defaultOrchestrationMode: 'single_agent' | 'multi_agent'
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

export interface AgentChatRequest {
  sessionId: string | null
  requestId?: string | null
  message: string
  orchestrationMode?: 'single_agent' | 'multi_agent'
  pageContext: {
    route?: string
    url: string
    jobName?: string
    jobStatus?: string
    nodeName?: string
    entryPoint?: 'default' | 'node_analysis' | 'ops_report'
  }
  clientContext?: {
    locale?: string
    timezone?: string
  }
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

export interface ResourceSuggestionPayload {
  suggestionId: string
  context: string
  recommendations: Array<{
    gpu_model: string
    available: number
    queue_depth: number
    estimated_wait: string
    match_score: number
    reason: string
  }>
  tip: string
}

export interface PipelineReportPayload {
  reportId: string
  reportType: string
  completedAt: string
  summary: {
    total_scanned: number
    idle_detected: number
    gpu_waste_hours: number
  }
  summary_labels?: {
    total_label?: string
    middle_label?: string
    right_label?: string
  }
  categories: Array<{
    action: string
    severity: 'critical' | 'warning' | 'info'
    count: number
    items: Array<{
      job_name: string
      user: string
      gpu_util: string
      duration?: string
      gpu_requested?: number
      gpu_actual?: number
    }>
  }>
}

export interface BatchConfirmationPayload {
  batchId: string
  action: string
  description: string
  items: Array<{
    job_name: string
    user: string
    gpu_util: string
    selected: boolean
  }>
}

// ──────────────────────────────────────────────
// SSE streaming connection
// ──────────────────────────────────────────────

/**
 * Connect to the agent chat endpoint via SSE (fetch + ReadableStream).
 * Returns an AbortController that can be used to cancel the stream.
 */
export function connectAgentChat(
  sessionId: string | null,
  requestId: string,
  message: string,
  pageContext: {
    route?: string
    url: string
    jobName?: string
    jobStatus?: string
    nodeName?: string
    entryPoint?: 'default' | 'node_analysis' | 'ops_report'
  },
  orchestrationMode: 'single_agent' | 'multi_agent',
  clientContext: { locale?: string; timezone?: string } | undefined,
  onEvent: (event: AgentSSEEvent) => void,
  onError: (error: Error) => void,
  onDone: () => void,
  onSessionId?: (sessionId: string) => void
): AbortController {
  const controller = new AbortController()

  const store = getDefaultStore()
  const apiBase = store.get(configAPIBaseAtom)
  const baseURL = `${apiBase ?? ''}/api`

  const token = localStorage.getItem(ACCESS_TOKEN_KEY)
  let doneDispatched = false

  const dispatchDone = () => {
    if (doneDispatched) return
    doneDispatched = true
    onDone()
  }

  const body: AgentChatRequest = {
    sessionId,
    requestId,
    message,
    pageContext,
    orchestrationMode,
    clientContext,
  }

  fetch(`${baseURL}/v1/agent/chat`, {
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
        onError(new Error(msg))
        return
      }

      if (!response.body) {
        onError(new Error('No response body for SSE stream'))
        return
      }

      const responseSessionId = response.headers.get('X-Agent-Session-ID')
      if (responseSessionId) {
        onSessionId?.(responseSessionId)
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
            const parsedObject =
              typeof parsed === 'object' && parsed !== null
                ? (parsed as { message?: string; msg?: string })
                : null
            const errorMsg =
              typeof parsed === 'string'
                ? parsed
                : parsedObject?.message || parsedObject?.msg || 'Agent error'
            onError(new Error(errorMsg))
            return
          }

          onEvent(sseEvent)
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
          onError(err instanceof Error ? err : new Error(String(err)))
        }
      }

      await pump()
    })
    .catch((err) => {
      if (err?.name === 'AbortError') return
      onError(err instanceof Error ? err : new Error(String(err)))
    })

  return controller
}

export function connectAgentResume(
  confirmId: string,
  onEvent: (event: AgentSSEEvent) => void,
  onError: (error: Error) => void,
  onDone: () => void,
  onSessionId?: (sessionId: string) => void
): AbortController {
  const controller = new AbortController()

  const store = getDefaultStore()
  const apiBase = store.get(configAPIBaseAtom)
  const baseURL = `${apiBase ?? ''}/api`

  const token = localStorage.getItem(ACCESS_TOKEN_KEY)
  let doneDispatched = false

  const dispatchDone = () => {
    if (doneDispatched) return
    doneDispatched = true
    onDone()
  }

  const body: AgentResumeRequest = { confirmId }

  fetch(`${baseURL}/v1/agent/chat/resume`, {
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
        let msg = `HTTP ${response.status}`
        try {
          const errBody = await response.json()
          msg = errBody?.msg || errBody?.message || msg
        } catch {
          // ignore parse errors
        }
        onError(new Error(msg))
        return
      }

      if (!response.body) {
        onError(new Error('No response body for SSE stream'))
        return
      }

      const responseSessionId = response.headers.get('X-Agent-Session-ID')
      if (responseSessionId) {
        onSessionId?.(responseSessionId)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      const processBuffer = () => {
        const blocks = buffer.split('\n\n')
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
            const parsedObject =
              typeof parsed === 'object' && parsed !== null
                ? (parsed as { message?: string; msg?: string })
                : null
            const errorMsg =
              typeof parsed === 'string'
                ? parsed
                : parsedObject?.message || parsedObject?.msg || 'Agent error'
            onError(new Error(errorMsg))
            return
          }

          onEvent(sseEvent)
        }
      }

      const pump = async (): Promise<void> => {
        try {
          const { done, value } = await reader.read()
          if (done) {
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
            return
          }
          onError(err instanceof Error ? err : new Error(String(err)))
        }
      }

      await pump()
    })
    .catch((err) => {
      if (err?.name === 'AbortError') return
      onError(err instanceof Error ? err : new Error(String(err)))
    })

  return controller
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
 * List all agent sessions for the current user.
 */
export const apiListSessions = () => apiV1Get<IResponse<AgentSession[]>>('agent/sessions')

export const apiPinSession = (sessionId: string, pinned: boolean) =>
  apiV1Put<IResponse<AgentSession>>(`agent/sessions/${encodeURIComponent(sessionId)}/pin`, {
    pinned,
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
