/**
 * Copyright 2025 RAIDS Lab
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
import { IResponse } from '@/services/types'

import { IUserInfo } from './vcjob'

export interface CreateModelDownloadReq {
  name: string
  revision?: string
  source?: string
  category: 'model' | 'dataset'
  token?: string
}

export interface ModelDownloadActionReq {
  id: number
  token?: string
}

export type ModelDownloadStatus = 'Pending' | 'Downloading' | 'Paused' | 'Ready' | 'Failed'

export interface ModelDownload {
  id: number
  name: string
  source: string
  category: 'model' | 'dataset'
  revision: string
  path: string
  sizeBytes: number
  downloadedBytes: number
  downloadSpeed: string
  status: ModelDownloadStatus
  message: string
  jobName: string
  creatorId: number
  referenceCount: number
  createdAt: string
  updatedAt: string
  sourceUpdatedAt?: string
  userInfo: IUserInfo
  canManage: boolean
  canViewLogs: boolean
  sourceUrl: string
  displayName: string
  license: string
  task: string
  library: string
  modelType: string
  parameterCount: number
  sourceCreatedAt?: string
}

export interface ModelDownloadListParams {
  page: number
  pageSize: number
  category?: 'model' | 'dataset'
  status?: ModelDownloadStatus
  search?: string
}

export interface ModelDownloadListResp {
  total: number
  items: ModelDownload[]
  summary: Partial<Record<ModelDownloadStatus, number>>
}

export const apiCreateModelDownload = (data: CreateModelDownloadReq) =>
  apiV1Post<IResponse<ModelDownload>>('model-download/models/download', data)

export const apiListModelDownloads = (category?: 'model' | 'dataset') =>
  apiV1Get<IResponse<ModelDownload[]>>(
    'model-download/models/downloads',
    category ? { searchParams: { category } } : undefined
  )

export const apiListModelDownloadsPaged = (params: ModelDownloadListParams) => {
  const searchParams: Record<string, string | number> = {
    page: params.page,
    pageSize: params.pageSize,
  }
  if (params.category) searchParams.category = params.category
  if (params.status) searchParams.status = params.status
  if (params.search) searchParams.search = params.search
  return apiV1Get<IResponse<ModelDownloadListResp>>('model-download/models/downloads', {
    searchParams,
  })
}

export const apiGetModelDownload = (id: number) =>
  apiV1Get<IResponse<ModelDownload>>(`model-download/models/downloads/${id}`)

export const apiRetryModelDownload = ({ id, token }: ModelDownloadActionReq) =>
  apiV1Post<IResponse<ModelDownload>>(`model-download/models/downloads/${id}/retry`, { token })

export const apiPauseModelDownload = (id: number) =>
  apiV1Post<IResponse<ModelDownload>>(`model-download/models/downloads/${id}/pause`, {})

export const apiResumeModelDownload = ({ id, token }: ModelDownloadActionReq) =>
  apiV1Post<IResponse<ModelDownload>>(`model-download/models/downloads/${id}/resume`, { token })

export const apiDeleteModelDownload = (id: number) =>
  apiV1Delete<IResponse<string>>(`model-download/models/downloads/${id}`)

export const apiGetModelDownloadLogs = (id: number) =>
  apiV1Get<IResponse<string>>(`model-download/models/downloads/${id}/logs`)

// Admin APIs
export const apiAdminListModelDownloads = () =>
  apiV1Get<IResponse<ModelDownload[]>>('admin/model-download/models/downloads')

export const apiAdminDeleteModelDownload = (id: number) =>
  apiV1Delete<IResponse<string>>(`admin/model-download/models/downloads/${id}`)
