// src/services/api/system-config.ts
import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'

import { IResponse } from '../types'

export interface ILLMConfig {
  baseUrl: string
  apiKey: string
  modelName: string
}

export interface IUpdateLLMConfigReq extends ILLMConfig {
  validate: boolean
}

export interface IGpuAnalysisStatus {
  enabled: boolean
}

export interface IQueueResourceLimit {
  queue: string
  limits: Record<string, string>
}

export interface IUserResourceLimitConfig {
  enabled: boolean
  configs: IQueueResourceLimit[]
}

export interface IResourceLimitDetail {
  resource: string
  used: string
  limit: string
  exceeded: boolean
}

export interface IResourceLimitCheckResult {
  enabled: boolean
  exceeded: boolean
  details: IResourceLimitDetail[]
}

/** 获取 LLM 配置 */
export const apiAdminGetLLMConfig = () => apiV1Get<IResponse<ILLMConfig>>('admin/system-config/llm')

/** 更新 LLM 配置 */
export const apiAdminUpdateLLMConfig = (data: IUpdateLLMConfigReq) =>
  apiV1Put<IResponse<string>>('admin/system-config/llm', data)

/** 重置 LLM 配置 */
export const apiAdminResetLLMConfig = () =>
  apiV1Delete<IResponse<string>>('admin/system-config/llm')

/** 获取 GPU 分析开关状态 */
export const apiAdminGetGpuAnalysisStatus = () =>
  apiV1Get<IResponse<IGpuAnalysisStatus>>('admin/system-config/gpu-analysis')

/** 设置 GPU 分析开关状态 */
export const apiAdminSetGpuAnalysisStatus = (enable: boolean) =>
  apiV1Put<IResponse<string>>('admin/system-config/gpu-analysis', { enable })

/** 获取用户资源限制配置 */
export const apiAdminGetUserResourceLimitConfig = () =>
  apiV1Get<IResponse<IUserResourceLimitConfig>>('admin/system-config/user-resource-limit')

/** 更新用户资源限制配置 */
export const apiAdminUpdateUserResourceLimitConfig = (data: IUserResourceLimitConfig) =>
  apiV1Put<IResponse<string>>('admin/system-config/user-resource-limit', data)

/** 检查当前用户资源使用是否超限（含本次请求资源） */
export const apiCheckResourceLimit = (requestedResources?: Record<string, string>) =>
  apiV1Post<IResponse<IResourceLimitCheckResult>>('context/resource-limit-check', {
    requestedResources,
  })
