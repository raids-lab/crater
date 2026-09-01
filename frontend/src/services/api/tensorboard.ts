import { apiV1Delete, apiV1Get, apiV1Post } from '@/services/client'

export const MAX_TENSORBOARD_SOURCE_JOBS = 10
export const MAX_ACTIVE_TENSORBOARDS = 10

export interface TensorboardSourceJobReq {
  jobName: string
  logDir?: string
}

export interface CreateTensorboardReq {
  sourceJobName?: string
  sourceJobNames?: string[]
  sourceJobs?: TensorboardSourceJobReq[]
  logDir: string
  ttlHours: number
}

export interface CreateTensorboardResp {
  tensorboardId: string
  accessPath: string
}

export type TensorboardStatus = 'starting' | 'ready' | 'failed'

export type TensorboardStatusReason =
  | 'deployment_failed'
  | 'status_check_pending'
  | 'pod_list_pending'
  | 'image_pull_failed'
  | 'container_start_failed'
  | 'container_exited'
  | 'deployment_starting'
  | 'network_config_incomplete'
  | 'service_missing'
  | 'network_check_pending'
  | 'service_misconfigured'
  | 'ingress_missing'
  | 'ingress_misconfigured'
  | 'endpoint_pending'
  | 'ready'

export interface TensorboardInfo {
  id: string
  expiration: string
  createdAt: string
  accessPath: string
  status: TensorboardStatus
  statusReason?: TensorboardStatusReason
  statusMessage: string
}

export interface TensorboardSourceConfig {
  logDir: string
}

export function apiTensorboardList() {
  return apiV1Get<TensorboardInfo[]>('tensorboard')
}

export function apiTensorboardCreate(data: CreateTensorboardReq) {
  return apiV1Post<CreateTensorboardResp>('tensorboard', data)
}

export function apiTensorboardSourceConfig(jobName: string) {
  return apiV1Get<TensorboardSourceConfig>(`tensorboard/source/${encodeURIComponent(jobName)}`)
}

export function apiTensorboardExtendTTL(id: string, ttlHours: number) {
  return apiV1Post<string>(`tensorboard/${id}/extend`, { ttlHours })
}

export function apiTensorboardDelete(id: string) {
  return apiV1Delete<string>(`tensorboard/${id}`)
}
