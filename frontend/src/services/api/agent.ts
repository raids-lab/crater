import { getDefaultStore } from 'jotai'

import { ACCESS_TOKEN_KEY } from '@/utils/store'
import { configAPIBaseAtom } from '@/utils/store/config'

import { apiV1Get, apiV1Post } from '@/services/client'
import type { IResponse } from '../types'

// ──────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────

export interface AgentSSEEvent {
  event: 'thinking' | 'tool_call' | 'tool_result' | 'message' | 'confirmation_required' | 'error' | 'done'
  data: any
}

export interface AgentSession {
  sessionId: string
  title: string
  messageCount: number
  createdAt: string
  updatedAt: string
}

export interface AgentMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  createdAt: string
}

export interface AgentChatRequest {
  sessionId: string | null
  message: string
  pageContext: {
    url: string
    jobName?: string
    jobStatus?: string
  }
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
  message: string,
  pageContext: { url: string; jobName?: string; jobStatus?: string },
  onEvent: (event: AgentSSEEvent) => void,
  onError: (error: Error) => void,
  onDone: () => void,
): AbortController {
  const controller = new AbortController()

  const store = getDefaultStore()
  const apiBase = store.get(configAPIBaseAtom)
  const baseURL = `${apiBase ?? ''}/api`

  const token = localStorage.getItem(ACCESS_TOKEN_KEY)

  const body: AgentChatRequest = { sessionId, message, pageContext }

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

          let parsed: any
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
            onDone()
            return
          }

          if (sseEvent.event === 'error') {
            const errorMsg =
              typeof parsed === 'string'
                ? parsed
                : parsed?.message || parsed?.msg || 'Agent error'
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
            onDone()
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

// ──────────────────────────────────────────────
// REST endpoints
// ──────────────────────────────────────────────

/**
 * Confirm or reject a pending write-operation.
 */
export const apiConfirmAction = (confirmId: string, confirmed: boolean) =>
  apiV1Post<IResponse<void>>('agent/confirm', { confirmId, confirmed })

/**
 * List all agent sessions for the current user.
 */
export const apiListSessions = () =>
  apiV1Get<IResponse<AgentSession[]>>('agent/sessions')

/**
 * Fetch the message history for a given session.
 */
export const apiGetSessionMessages = (sessionId: string) =>
  apiV1Get<IResponse<AgentMessage[]>>(`agent/sessions/${encodeURIComponent(sessionId)}/messages`)
