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

export interface CreateModelDownloadReq {
  name: string
  revision?: string
  source?: string
  category: 'model' | 'dataset'
}

export interface ModelDownload {
  id: number
  name: string
  source: string
  category: 'model' | 'dataset'
  revision: string
  path: string
  sizeBytes: number
  status: 'Pending' | 'Downloading' | 'Paused' | 'Ready' | 'Failed'
  message: string
  jobName: string
  creatorId: number
  referenceCount: number
  createdAt: string
  updatedAt: string
}

export const apiCreateModelDownload = (data: CreateModelDownloadReq) =>
  apiV1Post<IResponse<ModelDownload>>('model-download/models/download', data)

export const apiListModelDownloads = () =>
  apiV1Get<IResponse<ModelDownload[]>>('model-download/models/downloads')

export const apiGetModelDownload = (id: number) =>
  apiV1Get<IResponse<ModelDownload>>(`model-download/models/downloads/${id}`)

export const apiRetryModelDownload = (id: number) =>
  apiV1Post<IResponse<ModelDownload>>(`model-download/models/downloads/${id}/retry`, {})

export const apiPauseModelDownload = (id: number) =>
  apiV1Post<IResponse<ModelDownload>>(`model-download/models/downloads/${id}/pause`, {})

export const apiResumeModelDownload = (id: number) =>
  apiV1Post<IResponse<ModelDownload>>(`model-download/models/downloads/${id}/resume`, {})

export const apiDeleteModelDownload = (id: number) =>
  apiV1Delete<IResponse<string>>(`model-download/models/downloads/${id}`)

export const apiGetModelDownloadLogs = (id: number) =>
  apiV1Get<IResponse<string>>(`model-download/models/downloads/${id}/logs`)

// Admin APIs
export const apiAdminListModelDownloads = () =>
  apiV1Get<IResponse<ModelDownload[]>>('admin/model-download/models/downloads')

export const apiAdminDeleteModelDownload = (id: number) =>
  apiV1Delete<IResponse<string>>(`admin/model-download/models/downloads/${id}`)
