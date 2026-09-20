import { HTTPError } from 'ky'

import {
  apiClient,
  apiV1Delete,
  apiV1Get,
  apiV1Patch,
  apiV1Post,
  apiV1Put,
} from '@/services/client'

import { IResponse } from '../types'

// A model's first response may take longer than the generic API timeout,
// especially after the runtime has just been scheduled or warmed up.
const KTHENA_INFERENCE_REQUEST_TIMEOUT = 120_000

export interface KthenaWorkerReq {
  image: string
  replicas?: number
  pods?: number
  cpu?: string
  memory?: string
  gpu?: string
  gpuModel?: string
  config?: Record<string, string>
}

export interface CreateKthenaReq {
  name: string
  modelSource?: 'platform' | 'external'
  platformModelId?: number
  modelURI?: string
  servedModel?: string
  backendType?: string
  cacheURI?: string
  replicas?: number
  env?: Record<string, string>
  worker: KthenaWorkerReq
  selectors?: Array<{
    key: string
    operator: string
    values?: string[]
  }>
}

export interface KthenaService {
  name: string
  namespace: string
  owner?: string
  userInfo?: {
    username: string
    nickname: string
  }
  modelSource: 'platform' | 'external'
  platformModelId: number
  modelURI: string
  servedModel: string
  backendType: string
  cacheURI: string
  replicas: number
  workerImage: string
  workerReplicas: number
  workerCPU: string
  workerMemory: string
  workerGPU: string
  workerGPUModel: string
  env: Record<string, string>
  workerConfig: Record<string, string>
  phase: string
  conditions: Array<Record<string, unknown>>
  resources: KthenaResource[]
  runtimePods?: KthenaRuntimePod[]
  diagnostics?: KthenaDiagnostic[]
  access: KthenaAccess
  labels: Record<string, string>
  createdAt: string
}

export interface KthenaResource {
  kind: string
  name: string
  namespace: string
  phase: string
  ready: boolean
  conditions: Array<Record<string, unknown>>
}

export interface KthenaRuntimePod {
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

export interface KthenaInferenceTemplateConfig {
  modelSource?: 'platform' | 'external'
  platformModelId?: number
  modelURI?: string
  servedModel?: string
  backendType: 'vLLM'
  cacheURI?: string
  imageSource?: 'platform' | 'manual'
  image?: string
  platformImage?: {
    imageLink?: string
    archs?: string[]
  }
  resource?: {
    cpu?: number
    memory?: number
    gpu?: {
      count?: number
      model?: string
    }
  }
  replicas?: number
  envs?: Array<{
    name: string
    value: string
  }>
  configItems?: Array<{
    key: string
    value: string
  }>
  nodeSelector?: {
    enable?: boolean
    mode?: string
    nodes?: string[]
  }
}

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

export interface KthenaConversationCreateReq {
  sessionId?: string
  title?: string
  messages?: ChatCompletionReq['messages']
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

export const apiCreateKthenaConversation = (name: string, data: KthenaConversationCreateReq) =>
  apiV1Post<IResponse<KthenaConversation>>(`kthena/inference-services/${name}/conversations`, data)

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

export const apiCreateKthenaConversationTurn = (name: string, data: KthenaConversationTurnReq) =>
  apiV1Post<IResponse<KthenaConversationTurnResp>>(
    `kthena/inference-services/${name}/conversations/turns`,
    data,
    { timeout: KTHENA_INFERENCE_REQUEST_TIMEOUT }
  )

export const apiChatKthenaService = async (name: string, data: ChatCompletionReq) => {
  try {
    return await apiClient
      .post(`v1/kthena/inference-services/${name}/openai/v1/chat/completions`, {
        json: data,
        timeout: KTHENA_INFERENCE_REQUEST_TIMEOUT,
      })
      .json<ChatCompletionResp>()
  } catch (error) {
    if (error instanceof HTTPError) {
      const body = (await error.response.text()).trim()
      let message = body
      try {
        const parsed = JSON.parse(body) as
          | string
          | { error?: string | { message?: string }; message?: string; msg?: string }
        if (typeof parsed === 'string') {
          message = parsed
        } else if (typeof parsed.error === 'string') {
          message = parsed.error
        } else {
          message = parsed.error?.message || parsed.message || parsed.msg || body
        }
      } catch {
        // Keep a non-JSON router response as-is.
      }
      throw new Error(`[HTTP ${error.response.status}] ${message || error.message}`)
    }
    throw error
  }
}
