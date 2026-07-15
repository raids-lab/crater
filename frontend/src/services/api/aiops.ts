import { getDefaultStore } from 'jotai'
import type { Options } from 'ky'

import { apiV1Get, apiV1Post } from '@/services/client'

import { ACCESS_TOKEN_KEY } from '@/utils/store'
import { configAPIBaseAtom } from '@/utils/store/config'

import type { IResponse } from '../types'

/**
 * Health Overview Response
 */
export interface IHealthOverview {
  totalJobs: number
  failedJobs: number
  pendingJobs: number
  runningJobs: number
  failureRate: number
  failureTrend: Array<{
    date: string
    count: number
  }>
  topFailureReasons: Array<{
    reason: string
    count: number
  }>
}

/**
 * Diagnosis Response
 */
export interface IDiagnosis {
  jobName: string
  status: string
  category: string
  diagnosis: string
  solution: string
  confidence: 'high' | 'medium' | 'low'
  severity: 'critical' | 'error' | 'warning' | 'info'
  evidence: {
    exitCode?: number
    exitReason?: string
    events?: string[]
  }
}

/**
 * Chat Request & Response
 */
export interface IChatRequest {
  message: string
  jobName?: string
}

export interface IChatResponse {
  message: string
  type: 'text' | 'diagnosis' | 'suggestion'
  data?: IDiagnosis | { engine?: string; mode?: string; adminHint?: boolean }
}

/**
 * Get health overview
 */
export const apiGetHealthOverview = (days?: number) =>
  apiV1Get<IResponse<IHealthOverview>>('aiops/health-overview', {
    searchParams: { days: days ?? 7 },
    timeout: 30000,
  })

/**
 * Get health overview (admin)
 */
export const apiGetHealthOverviewAdmin = (days?: number) =>
  apiV1Get<IResponse<IHealthOverview>>('admin/aiops/health-overview', {
    searchParams: { days: days ?? 7 },
    timeout: 30000,
  })

/**
 * Diagnose a specific job
 */
export const apiDiagnoseJob = (jobName: string) =>
  apiV1Get<IResponse<IDiagnosis>>(`aiops/diagnose/${encodeURIComponent(jobName)}`)

/**
 * Chat with AI assistant
 */
export const apiChatMessage = (request: IChatRequest, options?: Options) =>
  apiV1Post<IResponse<IChatResponse>>('aiops/chat', request, options)

export const apiChatMessageLLM = (request: IChatRequest, options?: Options) =>
  apiV1Post<IResponse<IChatResponse>>('aiops/llmchat', request, { timeout: 150000, ...options })

export const apiAdminChatMessage = (request: IChatRequest, options?: Options) =>
  apiV1Post<IResponse<IChatResponse>>('admin/aiops/chat', request, options)

export const apiAdminChatMessageLLM = (request: IChatRequest, options?: Options) =>
  apiV1Post<IResponse<IChatResponse>>('admin/aiops/llmchat', request, {
    timeout: 150000,
    ...options,
  })

export interface LLMStreamEvent {
  event: 'delta' | 'error' | 'done'
  data: unknown
}

export function connectLLMChatStream(
  request: IChatRequest,
  isAdmin: boolean,
  onDelta: (delta: string) => void,
  onError: (error: Error) => void,
  onDone: (data?: IChatResponse['data']) => void
): AbortController {
  const controller = new AbortController()
  const store = getDefaultStore()
  const apiBase = store.get(configAPIBaseAtom)
  const baseURL = `${apiBase ?? ''}/api`
  const token = localStorage.getItem(ACCESS_TOKEN_KEY)
  const path = isAdmin ? 'admin/aiops/llmchat/stream' : 'aiops/llmchat/stream'
  let doneDispatched = false

  const dispatchDone = (data?: IChatResponse['data']) => {
    if (doneDispatched) return
    doneDispatched = true
    onDone(data)
  }

  fetch(`${baseURL}/v1/${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(request),
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
        onError(new Error('No response body for LLM stream'))
        return
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      const processBuffer = () => {
        const blocks = buffer.split('\n\n')
        buffer = blocks.pop() ?? ''
        for (const block of blocks) {
          if (!block.trim()) continue
          let eventType: LLMStreamEvent['event'] = 'delta'
          let dataStr = ''
          for (const line of block.split('\n')) {
            if (line.startsWith('event:')) {
              eventType = line.slice(6).trim() as LLMStreamEvent['event']
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
          if (eventType === 'delta') {
            const delta =
              typeof parsed === 'object' && parsed !== null
                ? String((parsed as { delta?: string }).delta ?? '')
                : String(parsed)
            if (delta) onDelta(delta)
            continue
          }
          if (eventType === 'error') {
            const msg =
              typeof parsed === 'object' && parsed !== null
                ? (parsed as { message?: string; msg?: string }).message ||
                  (parsed as { message?: string; msg?: string }).msg ||
                  'LLM stream error'
                : String(parsed)
            onError(new Error(msg))
            dispatchDone()
            return
          }
          if (eventType === 'done') {
            const data =
              typeof parsed === 'object' && parsed !== null
                ? ((parsed as { data?: IChatResponse['data'] }).data ?? undefined)
                : undefined
            dispatchDone(data)
            return
          }
        }
      }

      const pump = async (): Promise<void> => {
        try {
          const { done, value } = await reader.read()
          if (done) {
            if (buffer.trim()) processBuffer()
            dispatchDone()
            return
          }
          buffer += decoder.decode(value, { stream: true })
          processBuffer()
          return pump()
        } catch (err) {
          if ((err as Error)?.name === 'AbortError') return
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
