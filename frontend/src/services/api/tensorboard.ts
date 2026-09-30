/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
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
}

export interface CreateTensorboardResp {
  tensorboardId: string
  accessPath: string
}

export type TensorboardStatus = 'pending' | 'starting' | 'ready' | 'failed' | 'expired'

export type TensorboardStatusReason =
  | 'deployment_failed'
  | 'deployment_starting'
  | 'waiting_for_schedule'
  | 'pod_starting'
  | 'job_failed'
  | 'runtime_expired'
  | 'ready'

export interface TensorboardInfo {
  id: string
  expiration?: string
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

export function apiTensorboardCreateAccessSession(id: string) {
  return apiV1Post<string>(`tensorboard/${id}/access`)
}

export function apiTensorboardDelete(id: string) {
  return apiV1Delete<string>(`tensorboard/${id}`)
}
