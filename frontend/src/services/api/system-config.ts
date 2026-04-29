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

export interface IPrequeueConfig {
  backfillEnabled: boolean
  queueQuotaEnabled: boolean
  normalJobWaitingToleranceSeconds: number
  activateTickerIntervalSeconds: number
  maxTotalActivationsPerRound: number
  prequeueCandidateSize: number
}

export interface IBillingStatus {
  featureEnabled: boolean
  active: boolean
  runningSettlementEnabled: boolean
  runningSettlementIntervalMinutes: number
  jobFreeMinutes: number
  defaultIssueAmount: number
  defaultIssuePeriodMinutes: number
  accountIssueAmountOverrideEnabled: boolean
  accountIssuePeriodOverrideEnabled: boolean
  baseLoopCronStatus: string
  baseLoopCronEnabled: boolean
}

export interface ISetBillingStatusReq {
  featureEnabled?: boolean
  active?: boolean
  runningSettlementEnabled?: boolean
  runningSettlementIntervalMinutes?: number
  jobFreeMinutes?: number
  defaultIssueAmount?: number
  defaultIssuePeriodMinutes?: number
  accountIssueAmountOverrideEnabled?: boolean
  accountIssuePeriodOverrideEnabled?: boolean
}

export interface IResetAllBillingBalancesResp {
  accountsAffected: number
  userAccountsAffected: number
  issuedAt: string
}

export interface IGrantAllUsersExtraBalanceReq {
  delta: number
  reason?: string
}

export interface IGrantAllUsersExtraBalanceResp {
  usersAffected: number
  delta: number
  reason?: string
  issuedAt: string
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

/** 获取 Billing 开关状态 */
export const apiGetBillingStatus = () =>
  apiV1Get<IResponse<IBillingStatus>>('system-config/billing')

/** 获取 Billing 开关状态 */
export const apiAdminGetBillingStatus = () =>
  apiV1Get<IResponse<IBillingStatus>>('admin/system-config/billing')

/** 设置 Billing 开关状态 */
export const apiAdminSetBillingStatus = (data: ISetBillingStatusReq) =>
  apiV1Put<IResponse<string>>('admin/system-config/billing', data)

/** 手动执行一次 Billing 基础循环 */
export const apiAdminTriggerBillingReconcile = () =>
  apiV1Post<IResponse<unknown>>('admin/system-config/billing/reconcile')

/** 重置全平台所有账户成员的免费额度 */
export const apiAdminResetAllBillingBalances = () =>
  apiV1Post<IResponse<IResetAllBillingBalancesResp>>('admin/system-config/billing/reset-all')

/** 给所有用户发放额外点数 */
export const apiAdminGrantAllUsersExtraBalance = (data: IGrantAllUsersExtraBalanceReq) =>
  apiV1Post<IResponse<IGrantAllUsersExtraBalanceResp>>(
    'admin/system-config/billing/extra-balance-all',
    data
  )
