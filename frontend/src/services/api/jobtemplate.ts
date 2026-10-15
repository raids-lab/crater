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
import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'
import { IPage, IResponse } from '@/services/types'

import { IUserInfo } from './vcjob'

export interface JobTemplate {
  id: number
  name: string
  describe: string
  document: string
  createdAt: string
  template: string
  userInfo: IUserInfo
}
export interface JobTemplateReq {
  describe: string
  name: string
  document: string
  template: string
}

export interface JobTemplateListParams {
  page: number
  pageSize: number
  search: string
  owner: 'all' | 'mine' | 'others'
  sort: 'ascending' | 'descending'
}

export const listJobTemplate = (params: JobTemplateListParams, signal?: AbortSignal) => {
  const searchParams = new URLSearchParams({
    page: String(params.page),
    page_size: String(params.pageSize),
    owner: params.owner,
    sort: params.sort === 'ascending' ? 'createdAt' : '-createdAt',
  })
  const search = params.search.trim()
  if (search) {
    searchParams.set('search', search)
  }

  return apiV1Get<IResponse<IPage<JobTemplate>>>('jobtemplate', {
    searchParams,
    signal,
  })
}
export const createJobTemplate = (data: JobTemplateReq) => {
  return apiV1Post<IResponse<string>>('jobtemplate/create', data)
}
export const getJobTemplate = (id: number) => {
  return apiV1Get<IResponse<JobTemplate>>(`jobtemplate/${id}`)
}
export const deleteJobTemplate = (id: number) => {
  return apiV1Delete<IResponse<string>>(`jobtemplate/delete/${id}`)
}
export const updateJobTemplate = (data: JobTemplateReq & { id: number }) => {
  return apiV1Put<IResponse<string>>(`jobtemplate/update/${data.id}`, data)
}
