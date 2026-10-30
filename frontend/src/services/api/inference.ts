import {
  apiClient,
  apiV1Delete,
  apiV1Get,
  apiV1Patch,
  apiV1Post,
  apiV1Put,
} from '@/services/client'

import { IResponse } from '../types'

export interface ServingPodProfile {
  secretEnv?: Record<string, { name: string; key: string }>
  image: string
  cpu: string
  memory: string
  gpu: string
  gpuModel: string
  extendedResources?: Record<string, string>
  env?: Record<string, string>
  imageArchs?: string[]
  selectors?: Array<{ key: string; operator: string; values?: string[] }>
  tolerations?: Array<{
    key?: string
    operator?: string
    value?: string
    effect?: string
    tolerationSeconds?: number
  }>
}
export interface ServingRole {
  name: 'server' | 'prefill' | 'decode'
  instances: number
  execution: 'single' | 'ray'
  nodesPerInstance: number
  tensorParallel: number
  pipelineParallel: number
  entry: ServingPodProfile
  worker?: ServingPodProfile
  engineOptions: Record<string, string>
}
export interface CreateKthenaReq {
  modelRevision?: string
  name: string
  modelSource: 'platform' | 'external'
  platformModelId?: number
  modelURI?: string
  servedModel?: string
  backendType: 'vLLM' | 'SGLang'
  layout: 'combined' | 'disaggregated'
  replicas: number
  port: number
  roles: ServingRole[]
}
export interface KthenaService extends CreateKthenaReq {
  desiredReplicas: number
  namespace: string
  owner?: string
  userInfo?: { username: string; nickname: string }
  modelURI: string
  servedModel: string
  cacheURI: string
  queue?: string
  totalResources: Record<string, string>
  phase: string
  conditions: Array<Record<string, unknown>>
  resources: KthenaResource[]
  runtimePods?: KthenaRuntimePod[]
  diagnostics?: KthenaDiagnostic[]
  access: KthenaAccess
  labels: Record<string, string>
  createdAt: string
}
export interface ServingCapability {
  images: string[]
  connector?: string
  id: string
  engine: 'vLLM' | 'SGLang'
  layout: 'combined' | 'disaggregated'
  execution: 'single' | 'ray'
  enabled: boolean
  evidence: string
  reason?: string
}
export interface ServingPreflight {
  spec: CreateKthenaReq
  totalResources: Record<string, string>
  pods: number
  queue: string
  queued: boolean
  warnings: string[]
  resources: Array<Record<string, unknown>>
}
export const apiServingCapabilities = () =>
  apiV1Get<IResponse<ServingCapability[]>>('kthena/inference-services/capabilities')
export const apiPreviewServing = (data: CreateKthenaReq, mode: 'create' | 'update' = 'create') =>
  apiV1Post<IResponse<ServingPreflight>>(`kthena/inference-services/preview?mode=${mode}`, data)

export interface KthenaResource {
  kind: string
  name: string
  namespace: string
  phase: string
  ready: boolean
  conditions: Array<Record<string, unknown>>
}

export interface KthenaRuntimePod {
  group: string
  role: string
  instance: string
  entry: boolean
  name: string
  namespace: string
  nodeName: string
  podIP?: string
  hostIP?: string
  phase: string
  ready: boolean
  restarts: number
  readyContainers: number
  totalContainers: number
}

export interface KthenaAccess {
  modelName: string
  proxyBaseURL: string
  internalBaseURL: string
  nodePortURL?: string
  routerService: string
  routeName?: string
  serverName?: string
}

export interface KthenaDiagnostic {
  level: 'warning' | 'error' | string
  reason: string
  message: string
  details?: string
  resource?: string
  pod?: string
  container?: string
  timestamp?: string
}

export type KthenaInferenceTemplateConfig = Omit<CreateKthenaReq, 'name'> & { schemaVersion: 2 }

export interface KthenaInferenceTemplate {
  id: number
  name: string
  description: string
  config: KthenaInferenceTemplateConfig
  createdAt: string
  updatedAt: string
}

export interface KthenaInferenceTemplateReq {
  name: string
  description: string
  config: KthenaInferenceTemplateConfig
}

export interface KthenaConversationMessage {
  sequence: number
  role: 'system' | 'user' | 'assistant'
  content: string
  createdAt: string
}

export interface KthenaConversation {
  sessionId: string
  title: string
  namespace: string
  serviceName: string
  modelName: string
  backendType: string
  messageCount: number
  createdAt: string
  updatedAt: string
  messages?: KthenaConversationMessage[]
}

export interface KthenaConversationUpdateReq {
  title?: string
  messages?: ChatCompletionReq['messages']
}

export interface KthenaConversationTurnReq {
  sessionId?: string
  content: string
  temperature?: number
  maxTokens?: number
  clientTurnId?: string
}

export interface KthenaConversationTurnResp {
  conversation: KthenaConversation
  assistant: KthenaConversationMessage
  completion?: ChatCompletionResp | null
}

export interface ChatCompletionReq {
  model?: string
  messages: Array<{
    role: 'system' | 'user' | 'assistant'
    content: string
  }>
  temperature?: number
  max_tokens?: number
  stream?: boolean
}

export interface ChatCompletionResp {
  id?: string
  object?: string
  created?: number
  model?: string
  choices?: Array<{
    index?: number
    message?: {
      role?: string
      content?: string
    }
    text?: string
    finish_reason?: string
  }>
  usage?: Record<string, number>
  [key: string]: unknown
}

export const apiCreateKthenaService = (data: CreateKthenaReq) =>
  apiV1Post<IResponse<KthenaService>>('kthena/inference-services', data)

export const apiListKthenaServices = () =>
  apiV1Get<IResponse<KthenaService[]>>('kthena/inference-services')

export const apiGetKthenaService = (name: string) =>
  apiV1Get<IResponse<KthenaService>>(`kthena/inference-services/${name}`)

export const apiDeleteKthenaService = (name: string) =>
  apiV1Delete<IResponse<string>>(`kthena/inference-services/${name}`)

export const apiListKthenaInferenceTemplates = () =>
  apiV1Get<IResponse<KthenaInferenceTemplate[]>>('kthena/inference-templates')

export const apiCreateKthenaInferenceTemplate = (data: KthenaInferenceTemplateReq) =>
  apiV1Post<IResponse<KthenaInferenceTemplate>>('kthena/inference-templates', data)

export const apiUpdateKthenaInferenceTemplate = (id: number, data: KthenaInferenceTemplateReq) =>
  apiV1Put<IResponse<KthenaInferenceTemplate>>(`kthena/inference-templates/${id}`, data)

export const apiDeleteKthenaInferenceTemplate = (id: number) =>
  apiV1Delete<IResponse<string>>(`kthena/inference-templates/${id}`)

export const apiListKthenaConversations = (
  name: string,
  options: { includeMessages?: boolean; limit?: number; messageLimit?: number } = {}
) =>
  apiV1Get<IResponse<KthenaConversation[]>>(`kthena/inference-services/${name}/conversations`, {
    searchParams: {
      includeMessages: String(options.includeMessages ?? false),
      ...(options.limit ? { limit: String(options.limit) } : {}),
      ...(options.messageLimit ? { messageLimit: String(options.messageLimit) } : {}),
    },
  })

export const apiUpdateKthenaConversation = (
  name: string,
  sessionID: string,
  data: KthenaConversationUpdateReq
) =>
  apiV1Patch<IResponse<KthenaConversation>>(
    `kthena/inference-services/${name}/conversations/${sessionID}`,
    data
  )

export const apiDeleteKthenaConversation = (name: string, sessionID: string) =>
  apiV1Delete<IResponse<string>>(`kthena/inference-services/${name}/conversations/${sessionID}`)

export const apiGetKthenaDiagnostics = (name: string) =>
  apiV1Get<IResponse<KthenaDiagnostic[]>>(`kthena/inference-services/${name}/diagnostics`)

export const apiRepairKthenaService = (name: string) =>
  apiV1Post<IResponse<string>>(`kthena/inference-services/${name}/reconcile`, {})

export async function apiStreamKthenaTurn(
  name: string,
  data: KthenaConversationTurnReq,
  signal: AbortSignal,
  onDelta: (text: string) => void
): Promise<KthenaConversationTurnResp> {
  const response = await apiClient.post(
    `v1/kthena/inference-services/${name}/conversations/turns`,
    {
      json: data,
      headers: { Accept: 'text/event-stream' },
      signal,
      timeout: 150_000,
      retry: 0,
    }
  )
  if (!response.body) throw new Error('Inference response has no stream')
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let result: KthenaConversationTurnResp | undefined
  try {
    for (;;) {
      const { value, done } = await reader.read()
      buffer += decoder.decode(value, { stream: !done }).replace(/\r\n/g, '\n')
      let boundary: number
      while ((boundary = buffer.indexOf('\n\n')) >= 0) {
        const event = buffer.slice(0, boundary)
        buffer = buffer.slice(boundary + 2)
        const lines = event.split('\n')
        const type = lines
          .find((line) => line.startsWith('event:'))
          ?.slice(6)
          .trim()
        const text = lines
          .filter((line) => line.startsWith('data:'))
          .map((line) => line.slice(5).trim())
          .join('\n')
        if (!text) continue
        const payload = JSON.parse(text)
        if (type === 'delta') onDelta(payload.content)
        if (type === 'error') throw new Error(payload.message)
        if (type === 'complete') result = payload as KthenaConversationTurnResp
      }
      if (done) break
    }
  } finally {
    await reader.cancel()
    reader.releaseLock()
  }
  if (!result) throw new Error('Generation was interrupted. Retry the same request.')
  return result
}

export interface KthenaMetrics {
  roles: Record<string, KthenaMetrics['metrics']>
  boundary: string
  metrics: {
    requestsPerSecond: number | null
    tokensPerSecond: number | null
    ttftP95Seconds: number | null
    latencyP95Seconds: number | null
    waitingRequests: number | null
    runningRequests: number | null
    httpErrorRatio: number | null
    observedAt: string
    windowSeconds: number
  }
  billing: { billedPoints: number; lastSettledAt: string | null; meteredPods: number }
  monitoringError?: string
}
export const apiGetKthenaMetrics = (name: string) =>
  apiV1Get<IResponse<KthenaMetrics>>(`kthena/inference-services/${name}/metrics`)

export const apiScaleServing = (name: string, replicas: number) =>
  apiV1Patch<IResponse<string>>(`kthena/inference-services/${name}/scale`, { replicas })
export const apiUpdateServing = (name: string, data: CreateKthenaReq) =>
  apiV1Put<IResponse<string>>(`kthena/inference-services/${name}`, data)
export const apiServingLogs = (name: string, pod: string, container: string) =>
  apiV1Get<IResponse<string>>(`kthena/inference-services/${name}/logs`, {
    searchParams: { pod, container },
  })
