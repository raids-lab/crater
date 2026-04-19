import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'

import { IResponse } from '../types'

export interface IQueueQuota {
  id?: number
  name: string
  enabled: boolean
  prequeueCandidateSize: number
  quota: Record<string, string>
}

export type QueueQuotaDraft = IQueueQuota & {
  savedName?: string
}

export type IQueueQuotaPayload = Omit<IQueueQuota, 'id'>

export interface IQueueQuotaConfig {
  quotas: IQueueQuota[]
}

export const apiAdminGetQueueQuotas = () =>
  apiV1Get<IResponse<IQueueQuotaConfig>>('admin/queue-quotas')

export const apiAdminCreateQueueQuota = (data: IQueueQuotaPayload) =>
  apiV1Post<IResponse<IQueueQuota>>('admin/queue-quotas', data)

export const apiAdminUpdateQueueQuota = (id: number, data: IQueueQuotaPayload) =>
  apiV1Put<IResponse<IQueueQuota>>(`admin/queue-quotas/${id}`, data)

export const apiAdminDeleteQueueQuota = (id: number) =>
  apiV1Delete<IResponse<string>>(`admin/queue-quotas/${id}`)
