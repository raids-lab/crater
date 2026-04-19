// src/services/api/system-config.ts
import { apiV1Delete, apiV1Get, apiV1Put } from '@/services/client'

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

export interface IPrequeueConfig {
  backfillEnabled: boolean
  queueQuotaEnabled: boolean
  normalJobWaitingToleranceSeconds: number
  activateTickerIntervalSeconds: number
  maxTotalActivationsPerRound: number
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

/** 获取预排队配置 */
export const apiAdminGetPrequeueConfig = () =>
  apiV1Get<IResponse<IPrequeueConfig>>('admin/system-config/prequeue')

/** 更新预排队配置 */
export const apiAdminUpdatePrequeueConfig = (data: IPrequeueConfig) =>
  apiV1Put<IResponse<string>>('admin/system-config/prequeue', data)
