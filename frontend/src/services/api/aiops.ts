import { apiClient, apiRequest, apiV1Get, apiV1Post } from '@/services/client'

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
  })

/**
 * Get health overview (admin)
 */
export const apiGetHealthOverviewAdmin = (days?: number) =>
  apiV1Get<IResponse<IHealthOverview>>('admin/aiops/health-overview', {
    searchParams: { days: days ?? 7 },
  })

/**
 * Diagnose a specific job
 */
export const apiDiagnoseJob = (jobName: string) =>
  apiV1Get<IResponse<IDiagnosis>>(`aiops/diagnose/${encodeURIComponent(jobName)}`)

/**
 * Chat with AI assistant
 */
export const apiChatMessage = (request: IChatRequest) =>
  apiV1Post<IResponse<IChatResponse>>('aiops/chat', request)

export const apiChatMessageLLM = (request: IChatRequest) =>
  apiRequest(() =>
    apiClient.post('v1/aiops/llmchat', { json: request, timeout: 150000 }).json<IResponse<IChatResponse>>()
  )

export const apiAdminChatMessage = (request: IChatRequest) =>
  apiV1Post<IResponse<IChatResponse>>('admin/aiops/chat', request)

export const apiAdminChatMessageLLM = (request: IChatRequest) =>
  apiRequest(() =>
    apiClient
      .post('v1/admin/aiops/llmchat', { json: request, timeout: 150000 })
      .json<IResponse<IChatResponse>>()
  )
